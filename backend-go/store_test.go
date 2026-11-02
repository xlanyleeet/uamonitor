package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestPostgresLifecycle(t *testing.T) {
	raw := os.Getenv("UM_TEST_DATABASE_URL")
	if raw == "" {
		t.Skip("set UM_TEST_DATABASE_URL for isolated PostgreSQL integration test")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	schema := fmt.Sprintf("uamonitor_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer admin.Exec(context.Background(), "DROP SCHEMA "+schema+" CASCADE")
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	store := &Store{pool: pool}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal(err)
	}
	if err = store.Migrate(ctx); err != nil {
		t.Fatal("migration not idempotent", err)
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	t.Run("applications", func(t *testing.T) { testAdmissionLifecycle(t, store) })
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		for {
			conn, e := listener.Accept()
			if e != nil {
				return
			}
			go serveFixture(conn)
		}
	}()
	port := strconv.Itoa(listener.Addr().(*net.TCPAddr).Port)
	servers := []Server{{ID: 1, Name: "Fixture", IP: "127.0.0.1", Port: port, ChangeIcon: true}, {ID: 2, Name: "No observations", IP: "example.org", Port: "25565"}}
	if err = store.Import(ctx, servers); err != nil {
		t.Fatal(err)
	}
	all, err := store.Servers(ctx, 0)
	if err != nil || len(all) != 2 || len(all[0].Statuses) != 0 {
		t.Fatal(all, err)
	}
	graph, err := store.Graphs(ctx, 0)
	if err != nil || len(graph) != 2 || len(graph[0].Data) != 0 {
		t.Fatal(graph, err)
	}
	// Restrict the actual poll to the local protocol fixture, no external traffic in tests.
	if _, err = pool.Exec(ctx, `DELETE FROM servers WHERE id=2`); err != nil {
		t.Fatal(err)
	}
	if err = pollOnce(ctx, store, NewAPI(store), 2, 30, true); err != nil {
		t.Fatal(err)
	}
	all, err = store.Servers(ctx, 1)
	if err != nil || len(all) != 1 || len(all[0].Statuses) != 1 || !all[0].Statuses[0].IsOnline || *all[0].Statuses[0].OnlineCount != 7 {
		t.Fatal(all, err)
	}
	if _, err = pool.Exec(ctx, `UPDATE statuses SET date=now()-interval '40 days'`); err != nil {
		t.Fatal(err)
	}
	// Preserve latest observation while removing older history.
	if err = store.Record(ctx, servers[0], PingResult{Online: 8, Max: 50, Version: "new"}, true); err != nil {
		t.Fatal(err)
	}
	if err = store.Prune(ctx, 30); err != nil {
		t.Fatal(err)
	}
	var count int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM statuses`).Scan(&count); err != nil || count != 1 {
		t.Fatal(count, err)
	}
	// Force favicon persistence to fail after the online insert. Transaction must roll back;
	// no offline record may be synthesized by the worker.
	if _, err = pool.Exec(ctx, `ALTER TABLE servers ADD CONSTRAINT reject_test_icon CHECK(icon IS NULL)`); err != nil {
		t.Fatal(err)
	}
	icon := "data:image/png;base64,iVBORw0KGgo="
	err = store.Record(ctx, servers[0], PingResult{Online: 9, Max: 50, Version: "new", Icon: icon}, true)
	if err == nil {
		t.Fatal("expected icon write failure")
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM statuses`).Scan(&count); err != nil || count != 1 {
		t.Fatal("partial observation persisted", count, err)
	}
	// A second replica must respect the distributed poll lock.
	lock, err := pool.Acquire(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = lock.Exec(ctx, `SELECT pg_advisory_lock(72621002)`); err != nil {
		t.Fatal(err)
	}
	err = pollOnce(ctx, store, NewAPI(store), 2, 30, true)
	lock.Exec(ctx, `SELECT pg_advisory_unlock(72621002)`)
	lock.Release()
	if err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM statuses`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate worker ran", count, err)
	}
	t.Run("uptime history", func(t *testing.T) { testUptimeHistory(t, store) })
	if !strings.HasPrefix(schema, "uamonitor_test_") {
		t.Fatal("invalid test schema")
	}
}
