package main

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

type Store struct {
	pool         *pgxpool.Pool
	pollInterval time.Duration
}

func (s *Store) Migrate(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(72621001)`); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (name text PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return err
	}
	files, err := migrations.ReadDir("migrations")
	if err != nil {
		return err
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Name() < files[j].Name() })
	for _, file := range files {
		var applied bool
		if err = tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM schema_migrations WHERE name=$1)`, file.Name()).Scan(&applied); err != nil {
			return err
		}
		if applied {
			continue
		}
		sql, e := migrations.ReadFile("migrations/" + file.Name())
		if e != nil {
			return e
		}
		if _, err = tx.Exec(ctx, string(sql)); err != nil {
			return fmt.Errorf("migration %s: %w", file.Name(), err)
		}
		if _, err = tx.Exec(ctx, `INSERT INTO schema_migrations(name) VALUES ($1)`, file.Name()); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Ready(ctx context.Context) error {
	var version string
	return s.pool.QueryRow(ctx, `SELECT name FROM schema_migrations ORDER BY name DESC LIMIT 1`).Scan(&version)
}

// One query; histories are aggregated in PostgreSQL and never sent to the client.
func (s *Store) Servers(ctx context.Context, id int) ([]Server, error) {
	rows, err := s.pool.Query(ctx, `SELECT s.id,s.name,s.ip,s.port::text,s.icon,s.change_icon,s.telegram,s.discord,s.youtube,s.link,s.description,
        t.date,t.is_online,t.online_count,t.max_online,t.version,u.known_seconds,u.online_seconds
        FROM servers s LEFT JOIN LATERAL (
            SELECT date,is_online,online_count,max_online,version FROM statuses
            WHERE server_id=s.id ORDER BY date DESC,id DESC LIMIT 1
        ) t ON true LEFT JOIN LATERAL (
 SELECT COALESCE(SUM(seconds),0)::float8 AS known_seconds,
        COALESCE(SUM(seconds) FILTER (WHERE is_online),0)::float8 AS online_seconds
 FROM (
   SELECT is_online, GREATEST(0,EXTRACT(EPOCH FROM (
     LEAST(COALESCE(LEAD(date) OVER (ORDER BY date,id),now()),date+make_interval(secs => $2),now())
     - GREATEST(date,now()-interval '30 days')
   ))) AS seconds
   FROM statuses WHERE server_id=s.id
     AND date >= now()-interval '30 days'-make_interval(secs => $2) AND date <= now()
 ) observations
) u ON true WHERE ($1::integer=0 OR s.id=$1) ORDER BY s.id`, id, s.observationSeconds())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Server{}
	for rows.Next() {
		var srv Server
		var knownSeconds, onlineSeconds float64
		var date *time.Time
		var online *bool
		var count, max *int
		var version *string
		if err = rows.Scan(&srv.ID, &srv.Name, &srv.IP, &srv.Port, &srv.Icon, &srv.ChangeIcon, &srv.Telegram, &srv.Discord, &srv.Youtube, &srv.Link, &srv.Description, &date, &online, &count, &max, &version, &knownSeconds, &onlineSeconds); err != nil {
			return nil, err
		}
		srv.Uptime = summarizeUptime(knownSeconds, onlineSeconds)
		srv.Statuses = []Status{}
		if date != nil {
			srv.Statuses = append(srv.Statuses, Status{*date, *online, count, max, version})
		}
		out = append(out, srv)
	}
	return out, rows.Err()
}

// One global query; no N-times repeated history reads. Empty servers remain visible.
func (s *Store) Graphs(ctx context.Context, id int) ([]Graph, error) {
	rows, err := s.pool.Query(ctx, `SELECT s.id,t.is_online,COALESCE(t.online_count,0),t.date
        FROM servers s LEFT JOIN statuses t ON t.server_id=s.id AND t.date >= now()-interval '24 hours' AND t.date <= now()
        WHERE ($1::integer=0 OR s.id=$1) ORDER BY s.id,t.date DESC,t.id DESC`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Graph{}
	for rows.Next() {
		var sid, count int
		var online *bool
		var date *time.Time
		if err = rows.Scan(&sid, &online, &count, &date); err != nil {
			return nil, err
		}
		if len(out) == 0 || out[len(out)-1].ID != sid {
			out = append(out, Graph{ID: sid, Data: []Point{}})
		}
		if date != nil {
			n := len(out) - 1
			out[n].Data = append(out[n].Data, Point{*online, count, *date})
		}
	}
	return out, rows.Err()
}

func (s *Store) Record(ctx context.Context, srv Server, result PingResult, online bool) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var count, max *int
	var version *string
	if online {
		count = &result.Online
		max = &result.Max
		version = &result.Version
	}
	if _, err = tx.Exec(ctx, `INSERT INTO statuses(server_id,is_online,online_count,max_online,version) VALUES ($1,$2,$3,$4,$5)`, srv.ID, online, count, max, version); err != nil {
		return err
	}
	if online && srv.ChangeIcon && result.Icon != "" {
		if _, err = tx.Exec(ctx, `UPDATE servers SET icon=$2 WHERE id=$1 AND change_icon AND icon IS DISTINCT FROM $2`, srv.ID, result.Icon); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) Prune(ctx context.Context, days int) error {
	// Keep each server's latest observation even when its history is otherwise expired.
	for {
		tag, err := s.pool.Exec(ctx, `DELETE FROM statuses WHERE id IN (
            SELECT t.id FROM statuses t WHERE t.date < now()-make_interval(days => $1)
            AND t.id <> (SELECT x.id FROM statuses x WHERE x.server_id=t.server_id ORDER BY x.date DESC,x.id DESC LIMIT 1)
            ORDER BY t.date LIMIT 5000)`, days)
		if err != nil {
			return err
		}
		if tag.RowsAffected() < 5000 {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
	}
}

func (s *Store) Import(ctx context.Context, servers []Server) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	for _, srv := range servers {
		port, err := strconv.Atoi(srv.Port)
		if err != nil || port < 1 || port > 65535 || len(srv.Name) == 0 || len(srv.Name) > 200 || !validHost(srv.IP) || srv.ID < 1 || srv.ID > 2147483647 {
			return fmt.Errorf("invalid server %d", srv.ID)
		}
		for _, link := range []*string{srv.Link, srv.Youtube} {
			if link != nil && *link != "" && !safeURL(*link) {
				return fmt.Errorf("invalid URL for server %d", srv.ID)
			}
		}
		if srv.Icon != nil && *srv.Icon != "" && !validIcon(*srv.Icon) {
			return fmt.Errorf("invalid icon for server %d", srv.ID)
		}
		if srv.Description != nil && len(*srv.Description) > 32768 {
			return fmt.Errorf("description too long for server %d", srv.ID)
		}
		_, err = tx.Exec(ctx, `INSERT INTO servers(id,name,ip,port,icon,change_icon,telegram,discord,youtube,link,description)
            VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
            ON CONFLICT (id) DO UPDATE SET name=EXCLUDED.name,ip=EXCLUDED.ip,port=EXCLUDED.port,
            icon=EXCLUDED.icon,change_icon=EXCLUDED.change_icon,telegram=EXCLUDED.telegram,discord=EXCLUDED.discord,
            youtube=EXCLUDED.youtube,link=EXCLUDED.link,description=EXCLUDED.description`, srv.ID, srv.Name, srv.IP, port, srv.Icon, srv.ChangeIcon, srv.Telegram, srv.Discord, srv.Youtube, srv.Link, srv.Description)
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `SELECT setval(pg_get_serial_sequence('servers','id'),COALESCE((SELECT MAX(id) FROM servers),1),EXISTS(SELECT 1 FROM servers))`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
