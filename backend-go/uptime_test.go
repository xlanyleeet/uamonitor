package main

import (
	"testing"
	"time"
)

func TestUptimeSummary(t *testing.T) {
	empty := summarizeUptime(0, 0)
	if empty.Percent != nil || empty.Coverage != 0 {
		t.Fatal(empty)
	}
	partial := summarizeUptime(1800, 1200)
	if partial.Percent == nil || *partial.Percent < 66.66 || *partial.Percent > 66.67 {
		t.Fatal(partial)
	}
	if partial.Coverage >= 1 {
		t.Fatal("gaps must stay unknown", partial)
	}
	full := summarizeUptime(uptimeWindowSeconds, uptimeWindowSeconds)
	if *full.Percent != 100 || full.Coverage != 100 {
		t.Fatal(full)
	}
	if (&Store{}).observationSeconds() != 600 || (&Store{pollInterval: time.Minute}).observationSeconds() != 60 {
		t.Fatal("poll interval ignored")
	}
}

func testUptimeHistory(t *testing.T, store *Store) {
	t.Helper()
	ctx := t.Context()
	if _, err := store.pool.Exec(ctx, `DELETE FROM statuses;
 INSERT INTO statuses(server_id,is_online,date) VALUES
 (1,false,now()-interval '40 days'),
 (1,true,now()-interval '60 minutes'),
 (1,false,now()-interval '30 minutes'),
 (1,true,now()-interval '20 minutes'),
 (1,true,now()-interval '10 minutes')`); err != nil {
		t.Fatal(err)
	}
	servers, err := store.Servers(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	uptime := servers[0].Uptime
	// The 20 minute gap after the first online observation stays unknown.
	if uptime.Percent == nil || *uptime.Percent < 74.9 || *uptime.Percent > 75.1 {
		t.Fatal(uptime)
	}
	expected := 2400.0 / uptimeWindowSeconds * 100
	if uptime.Coverage < expected-.001 || uptime.Coverage > expected+.001 {
		t.Fatal(uptime)
	}
	// Expired observations must not imply either downtime or fresh uptime.
	if _, err = store.pool.Exec(ctx, `DELETE FROM statuses WHERE date>now()-interval '30 days'`); err != nil {
		t.Fatal(err)
	}
	servers, err = store.Servers(ctx, 1)
	if err != nil || servers[0].Uptime.Percent != nil {
		t.Fatal(servers, err)
	}
}
