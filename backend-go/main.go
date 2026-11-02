package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
func envInt(key string, fallback, min, max int) (int, error) {
	value, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil || value < min || value > max {
		return 0, fmt.Errorf("%s must be between %d and %d", key, min, max)
	}
	return value, nil
}

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	migrate := flag.Bool("migrate", false, "Apply PostgreSQL migrations and exit")
	importPath := flag.String("import", "", "Import a JSON server catalog and exit (explicit IDs)")
	flag.Parse()
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	databaseURL := os.Getenv("UM_DATABASE_URL")
	if databaseURL == "" {
		return errors.New("UM_DATABASE_URL is required")
	}
	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return errors.New("invalid UM_DATABASE_URL")
	}
	config.MaxConns = 12
	config.MinConns = 1
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return err
	}
	defer pool.Close()
	store := &Store{pool: pool}
	initCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err = pool.Ping(initCtx); err != nil {
		return fmt.Errorf("database unavailable: %w", err)
	}
	if *migrate {
		return store.Migrate(initCtx)
	}
	if err = store.Ready(initCtx); err != nil {
		return errors.New("database migrations are required; run with -migrate")
	}
	if *importPath != "" {
		file, err := os.Open(*importPath)
		if err != nil {
			return err
		}
		defer file.Close()
		var servers []Server
		decoder := json.NewDecoder(io.LimitReader(file, 8<<20))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&servers); err != nil {
			return err
		}
		var extra any
		if decoder.Decode(&extra) != io.EOF {
			return errors.New("unexpected data after catalog")
		}
		if len(servers) > 10000 {
			return errors.New("catalog exceeds 10000 servers")
		}
		return store.Import(initCtx, servers)
	}
	intervalSeconds, err := envInt("UM_POLL_SECONDS", 600, 10, 86400)
	if err != nil {
		return err
	}
	store.pollInterval = time.Duration(intervalSeconds) * time.Second
	concurrency, err := envInt("UM_POLL_CONCURRENCY", 16, 1, 64)
	if err != nil {
		return err
	}
	retention, err := envInt("UM_RETENTION_DAYS", 30, 1, 3650)
	if err != nil {
		return err
	}
	private, err := strconv.ParseBool(env("UM_ALLOW_PRIVATE_TARGETS", "false"))
	if err != nil {
		return errors.New("invalid UM_ALLOW_PRIVATE_TARGETS")
	}
	if private {
		slog.Warn("private target access explicitly enabled; use only on trusted test networks")
	}
	api := NewAPI(store)
	api.Admission, err = newAdmission(store, api, os.Getenv("UM_DISCORD_WEBHOOK_URL"), os.Getenv("UM_PUBLIC_URL"))
	if err != nil {
		return err
	}
	api.TrustProxy, err = strconv.ParseBool(env("UM_TRUST_PROXY", "false"))
	if err != nil {
		return errors.New("invalid UM_TRUST_PROXY")
	}
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); api.Admission.worker(workerCtx) }()
	wg.Add(1)
	go func() {
		defer wg.Done()
		poll(workerCtx, store, api, time.Duration(intervalSeconds)*time.Second, concurrency, retention, private)
	}()
	server := &http.Server{Addr: env("UM_LISTEN", ":8080"), Handler: api.Handler(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	done := make(chan error, 1)
	go func() { slog.Info("HTTP listening", "address", server.Addr); done <- server.ListenAndServe() }()
	select {
	case <-ctx.Done():
	case err = <-done:
	}
	workerCancel()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	shutdownErr := server.Shutdown(shutdownCtx)
	wg.Wait()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return shutdownErr
}

func poll(ctx context.Context, store *Store, api *API, interval time.Duration, concurrency, retention int, private bool) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		if err := pollOnce(ctx, store, api, concurrency, retention, private); err != nil && !errors.Is(err, context.Canceled) {
			slog.Error("poll cycle failed", "error", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func pollOnce(ctx context.Context, store *Store, api *API, concurrency, retention int, private bool) error {
	conn, err := store.pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	var locked bool
	if err = conn.QueryRow(ctx, `SELECT pg_try_advisory_lock(72621002)`).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return nil
	}
	defer func() {
		unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if _, e := conn.Exec(unlockCtx, `SELECT pg_advisory_unlock(72621002)`); e != nil {
			conn.Conn().Close(unlockCtx)
		}
	}()
	servers, err := store.Servers(ctx, 0)
	if err != nil {
		return err
	}
	jobs := make(chan Server)
	var wg sync.WaitGroup
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for srv := range jobs {
				pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
				port, _ := strconv.Atoi(srv.Port)
				result, pingErr := Ping(pingCtx, srv.IP, port, private)
				cancel()
				if ctx.Err() != nil {
					return
				}
				if errors.Is(pingErr, ErrForbidden) {
					slog.Warn("target rejected", "serverId", srv.ID)
					continue
				}
				recordCtx, recordCancel := context.WithTimeout(ctx, 5*time.Second)
				// Persistence failures never generate a fabricated offline observation.
				err := store.Record(recordCtx, srv, result, pingErr == nil)
				recordCancel()
				if err != nil {
					slog.Error("status persistence failed", "serverId", srv.ID, "error", err)
				}
			}
		}()
	}
	for _, srv := range servers {
		select {
		case jobs <- srv:
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return ctx.Err()
		}
	}
	close(jobs)
	wg.Wait()
	api.Invalidate()
	pruneCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	return store.Prune(pruneCtx, retention)
}
