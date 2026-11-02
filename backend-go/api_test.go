package main

import (
	"context"
	"errors"
	"net/http/httptest"
	"sync/atomic"
	"testing"
)

type fakeRepo struct {
	calls atomic.Int32
	fail  bool
}

func (f *fakeRepo) Ready(context.Context) error {
	if f.fail {
		return errors.New("secret connection details")
	}
	return nil
}
func (f *fakeRepo) Servers(_ context.Context, id int) ([]Server, error) {
	f.calls.Add(1)
	if f.fail {
		return nil, errors.New("secret")
	}
	if id > 1 {
		return []Server{}, nil
	}
	return []Server{{ID: 1, Port: "25565", Statuses: []Status{}}}, nil
}
func (f *fakeRepo) Graphs(_ context.Context, id int) ([]Graph, error) {
	f.calls.Add(1)
	if f.fail {
		return nil, errors.New("secret")
	}
	if id > 1 {
		return []Graph{}, nil
	}
	return []Graph{{ID: 1, Data: []Point{}}}, nil
}

func TestAPIValidationAndCompatibility(t *testing.T) {
	api := NewAPI(&fakeRepo{}).Handler()
	for _, tc := range []struct {
		path string
		code int
	}{
		{"/health", 200}, {"/ready", 200}, {"/api/servers", 200}, {"/api/graph", 200},
		{"/api/servers/1", 200}, {"/api/graph/1", 200}, {"/api/servers/2", 404}, {"/api/graph/2", 404},
		{"/api/graph/12junk", 400}, {"/api/graph/0", 400}, {"/api/graph/-1", 400}, {"/api/graph/2147483648", 400}, {"/api/missing", 404},
	} {
		t.Run(tc.path, func(t *testing.T) {
			w := httptest.NewRecorder()
			api.ServeHTTP(w, httptest.NewRequest("GET", tc.path, nil))
			if w.Code != tc.code {
				t.Fatalf("got %d: %s", w.Code, w.Body.String())
			}
		})
	}
}

func TestCacheAndRateLimit(t *testing.T) {
	repo := &fakeRepo{}
	a := NewAPI(repo)
	handler := a.Handler()
	for i := 0; i < 120; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/graph", nil))
		if w.Code != 200 {
			t.Fatal(w.Code)
		}
	}
	if repo.calls.Load() != 1 {
		t.Fatal("bulk cache missed", repo.calls.Load())
	}
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest("GET", "/api/graph", nil))
	if w.Code != 429 {
		t.Fatal("missing rate limit")
	}
	a.Invalidate()
	req := httptest.NewRequest("GET", "/api/graph", nil)
	req.RemoteAddr = "192.0.2.2:1234"
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if repo.calls.Load() != 2 {
		t.Fatal("cache did not invalidate")
	}
}

func TestDatabaseErrorDoesNotLeak(t *testing.T) {
	a := NewAPI(&fakeRepo{fail: true}).Handler()
	w := httptest.NewRecorder()
	a.ServeHTTP(w, httptest.NewRequest("GET", "/api/servers", nil))
	if w.Code != 503 || w.Body.String() != "{\"message\":\"Database temporarily unavailable\"}\n" {
		t.Fatal(w.Code, w.Body.String())
	}
}

func TestProxyHeadersRequireExplicitTrust(t *testing.T) {
	for _, trusted := range []bool{false, true} {
		a := NewAPI(&fakeRepo{})
		a.TrustProxy = trusted
		handler := a.Handler()
		for i := 0; i < 200; i++ {
			r := httptest.NewRequest("GET", "/api/graph", nil)
			if i < 100 {
				r.Header.Set("X-Real-IP", "1.1.1.1")
			} else {
				r.Header.Set("X-Real-IP", "8.8.8.8")
			}
			w := httptest.NewRecorder()
			handler.ServeHTTP(w, r)
			expected := 200
			if !trusted && i >= 120 {
				expected = 429
			}
			if w.Code != expected {
				t.Fatalf("trusted=%v request=%d status=%d", trusted, i, w.Code)
			}
		}
	}
}
