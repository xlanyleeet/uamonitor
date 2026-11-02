package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

type admissionTransport func(*http.Request) (*http.Response, error)

func (f admissionTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func applicationFixture() Application {
	return Application{Name: "Тестовий сервер", Host: "admissions.example.net", Port: "25565", Versions: "1.21", Modes: "Виживання", Description: "Український тестовий сервер для перевірки заявки.", Community: "https://example.net/community", Owner: "Власник", Role: "owner", Channel: "email", Contact: "owner@example.net", Rules: "https://example.net/rules", Language: "Українська мова спілкування", Access: "open", Authorized: true, Accepted: true}
}
func TestApplicationValidation(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "192.168.1.1", "localhost", "https://example.net", "example.net:25565", "-bad.example", "a..net"} {
		a := applicationFixture()
		a.Host = host
		if a.validate() == nil {
			t.Errorf("accepted host %q", host)
		}
	}
	a := applicationFixture()
	a.Host = "ADMISSIONS.EXAMPLE.NET."
	if e := a.validate(); e != nil || a.Host != "admissions.example.net" {
		t.Fatal("normalization", e)
	}
	a.Accepted = false
	if a.validate() == nil {
		t.Fatal("missing consent accepted")
	}
	if _, e := newAdmission(nil, nil, "https://attacker.invalid/api/webhooks/1/token", "https://monitoring.gradient-mc.com"); e == nil {
		t.Fatal("untrusted webhook accepted")
	}
}

// Called inside TestPostgresLifecycle's isolated schema. No Discord traffic or catalog fixtures leak.
func testAdmissionLifecycle(t *testing.T, store *Store) {
	api := NewAPI(store)
	a, e := newAdmission(store, api, "https://discord.com/api/webhooks/1/test-token", "https://monitoring.gradient-mc.com")
	if e != nil {
		t.Fatal(e)
	}
	api.Admission = a
	handler := api.Handler()
	ctx := context.Background()
	send := func(app Application, key string, origin bool) *httptest.ResponseRecorder {
		b, _ := json.Marshal(app)
		r := httptest.NewRequest("POST", "/api/applications", bytes.NewReader(b))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Idempotency-Key", key)
		if origin {
			r.Header.Set("Origin", a.publicURL)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	app := applicationFixture()
	key := randomKey()
	if w := send(app, key, false); w.Code != 403 {
		t.Fatal("cross-origin accepted", w.Code)
	}
	w := send(app, key, true)
	if w.Code != 201 {
		t.Fatal("submit", w.Code, w.Body.String())
	}
	var receipt map[string]string
	json.Unmarshal(w.Body.Bytes(), &receipt)
	id := receipt["id"]
	readStatus := func(id, key string) *httptest.ResponseRecorder {
		r := httptest.NewRequest("GET", "/api/applications/"+id, nil)
		if key != "" {
			r.Header.Set("Authorization", "Bearer "+key)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	for _, credential := range []string{"", randomKey()} {
		if w := readStatus(id, credential); w.Code != 404 {
			t.Fatal("status exposed without submission key", w.Code)
		}
	}
	w = readStatus(id, key)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"pending"`) || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("pending receipt unavailable", w.Code, w.Body.String())
	}
	for _, private := range []string{"payload", "request_key", "token", app.Contact, app.Owner} {
		if strings.Contains(w.Body.String(), private) {
			t.Fatal("status leaks private field", private)
		}
	}
	if w := send(app, key, true); w.Code != 200 {
		t.Fatal("retry not idempotent", w.Code, w.Body.String())
	}
	if w := send(app, randomKey(), true); w.Code != 409 {
		t.Fatal("pending duplicate accepted", w.Code)
	}
	app.Name = "Змінені дані"
	if w := send(app, key, true); w.Code != 409 {
		t.Fatal("key reused with changed payload", w.Code)
	}
	var count int
	store.pool.QueryRow(ctx, `SELECT count(*) FROM applications`).Scan(&count)
	if count != 1 {
		t.Fatal("duplicate application", count)
	}
	attempts := 0
	a.client = &http.Client{Transport: admissionTransport(func(r *http.Request) (*http.Response, error) {
		attempts++
		if r.URL.Query().Get("with_components") != "true" || r.URL.Query().Get("wait") != "true" {
			t.Error("missing Discord query options")
		}
		var payload map[string]json.RawMessage
		json.NewDecoder(r.Body).Decode(&payload)
		if !strings.Contains(string(payload["components"]), "/moderation/"+id) || !strings.Contains(string(payload["allowed_mentions"]), `"parse":[]`) {
			t.Error("invalid message components or mentions")
		}
		code := 500
		if attempts > 1 {
			code = 200
		}
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(`{"id":"mock-message"}`)), Header: make(http.Header)}, nil
	})}
	if e = a.deliver(ctx); e != nil {
		t.Fatal(e)
	}
	var delivered bool
	store.pool.QueryRow(ctx, `SELECT notified_at IS NOT NULL FROM applications WHERE id=$1`, id).Scan(&delivered)
	if delivered {
		t.Fatal("failed delivery acknowledged")
	}
	store.pool.Exec(ctx, `UPDATE applications SET next_attempt=now() WHERE id=$1`, id)
	if e = a.deliver(ctx); e != nil {
		t.Fatal(e)
	}
	store.pool.QueryRow(ctx, `SELECT notified_at IS NOT NULL FROM applications WHERE id=$1`, id).Scan(&delivered)
	if !delivered || attempts != 2 {
		t.Fatal("delivery retry failed")
	}
	var token string
	store.pool.QueryRow(ctx, `SELECT token FROM applications WHERE id=$1`, id).Scan(&token)
	path := "/moderation/" + id + "?token=" + token + "&action=approve"
	get := httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest("GET", path, nil))
	if get.Code != 200 || get.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("moderation page", get.Code)
	}
	store.pool.QueryRow(ctx, `SELECT count(*) FROM servers WHERE ip='admissions.example.net'`).Scan(&count)
	if count != 0 {
		t.Fatal("GET published server")
	}
	bad := httptest.NewRecorder()
	handler.ServeHTTP(bad, httptest.NewRequest("GET", "/moderation/"+id+"?token=wrong", nil))
	if bad.Code != 404 {
		t.Fatal("invalid capability accepted")
	}
	cookie := get.Result().Cookies()[0]
	decide := func(decision, reason string, csrf bool) *httptest.ResponseRecorder {
		form := url.Values{"token": {token}, "decision": {decision}, "reason": {reason}, "csrf": {cookie.Value}}
		r := httptest.NewRequest("POST", "/moderation/"+id, strings.NewReader(form.Encode()))
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		r.Header.Set("Origin", a.publicURL)
		if csrf {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)
		return w
	}
	if w := decide("approve", "", false); w.Code != 403 {
		t.Fatal("missing cookie accepted")
	}
	if w := decide("reject", "", true); !strings.Contains(w.Body.String(), "5–1000") {
		t.Fatal("rejection without reason accepted")
	}
	if w := decide("approve", "Перевірено", true); w.Code != 200 {
		t.Fatal("approve", w.Code, w.Body.String())
	}
	w = readStatus(id, key)
	if w.Code != 200 || !strings.Contains(w.Body.String(), `"status":"approved"`) || !strings.Contains(w.Body.String(), `"serverID":`) {
		t.Fatal("approved receipt unavailable", w.Code, w.Body.String())
	}
	if w := decide("approve", "Повтор", true); w.Code != 200 {
		t.Fatal("replay", w.Code)
	}
	store.pool.QueryRow(ctx, `SELECT count(*) FROM servers WHERE ip='admissions.example.net'`).Scan(&count)
	if count != 1 {
		t.Fatal("approval was not exactly once", count)
	}
	// Separate rejection, required reason, and durable per-client submission limit.
	app = applicationFixture()
	app.Host = "reject.example.net"
	w = send(app, randomKey(), true)
	if w.Code != 201 {
		t.Fatal("second submission", w.Code, w.Body.String())
	}
	json.Unmarshal(w.Body.Bytes(), &receipt)
	rejectedID := receipt["id"]
	id = rejectedID
	store.pool.QueryRow(ctx, `SELECT token FROM applications WHERE id=$1`, id).Scan(&token)
	get = httptest.NewRecorder()
	handler.ServeHTTP(get, httptest.NewRequest("GET", "/moderation/"+id+"?token="+token, nil))
	cookie = get.Result().Cookies()[0]
	if w := decide("reject", "Немає доступних правил", true); w.Code != 200 {
		t.Fatal("reject", w.Code)
	}
	var status, reason string
	store.pool.QueryRow(ctx, `SELECT status,reason FROM applications WHERE id=$1`, id).Scan(&status, &reason)
	if status != "rejected" || reason != "Немає доступних правил" {
		t.Fatal("rejection not persisted")
	}
	store.pool.QueryRow(ctx, `SELECT count(*) FROM servers WHERE ip='reject.example.net'`).Scan(&count)
	if count != 0 {
		t.Fatal("rejected server published")
	}
	app.Host = "third.example.net"
	w = send(app, randomKey(), true)
	if w.Code != 201 {
		t.Fatal("third submission", w.Code, w.Body.String())
	}
	app.Host = "fourth.example.net"
	if w := send(app, randomKey(), true); w.Code != 429 {
		t.Fatal("hourly rate limit failed", w.Code, w.Body.String())
	}
	// Remove only this test's records before the existing lifecycle checks.
	store.pool.Exec(ctx, `DELETE FROM applications WHERE host IN ('admissions.example.net','reject.example.net','third.example.net')`)
	store.pool.Exec(ctx, `DELETE FROM servers WHERE ip='admissions.example.net'`)
}
