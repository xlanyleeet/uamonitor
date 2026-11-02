package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Repository interface {
	Servers(context.Context, int) ([]Server, error)
	Graphs(context.Context, int) ([]Graph, error)
	Ready(context.Context) error
}

type cacheEntry struct {
	data    []byte
	expires time.Time
}
type visitor struct {
	start    time.Time
	requests int
}
type API struct {
	store     Repository
	Admission *Admission
	// Enabled only behind the private Compose proxy; never trust public client headers.
	TrustProxy bool
	mu         sync.Mutex
	cache      map[string]cacheEntry
	visitors   map[string]visitor
	loading    [2]sync.Mutex
}

func NewAPI(store Repository) *API {
	return &API{store: store, cache: map[string]cacheEntry{}, visitors: map[string]visitor{}}
}
func (a *API) Invalidate() { a.mu.Lock(); a.cache = map[string]cacheEntry{}; a.mu.Unlock() }

func (a *API) allowed(addr string) bool {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	now := time.Now()
	a.mu.Lock()
	defer a.mu.Unlock()
	v, ok := a.visitors[host]
	if !ok || now.Sub(v.start) >= time.Minute {
		if len(a.visitors) >= 10000 {
			for k, old := range a.visitors {
				if now.Sub(old.start) >= time.Minute {
					delete(a.visitors, k)
				}
			}
			if !ok && len(a.visitors) >= 10000 {
				return false
			}
		}
		v = visitor{start: now}
	}
	v.requests++
	a.visitors[host] = v
	return v.requests <= 120
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(value)
}

func apiError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

func (a *API) Handler() http.Handler {
	mux := http.NewServeMux()
	if a.Admission != nil {
		a.Admission.routes(mux)
	}
	mux.HandleFunc("GET /health", func(w http.ResponseWriter, r *http.Request) { writeJSON(w, 200, map[string]string{"status": "ok"}) })
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if a.store.Ready(ctx) != nil {
			apiError(w, 503, "Database is not ready")
			return
		}
		writeJSON(w, 200, map[string]string{"status": "ready"})
	})
	mux.HandleFunc("GET /api/{resource}", a.serve)
	mux.HandleFunc("GET /api/{resource}/{id}", a.serve)
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) { apiError(w, 404, "Not found") })
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				slog.Error("HTTP panic", "error", v)
				apiError(w, 500, "Internal server error")
			}
		}()
		remote := r.RemoteAddr
		if a.TrustProxy && net.ParseIP(r.Header.Get("X-Real-IP")) != nil {
			remote = net.JoinHostPort(r.Header.Get("X-Real-IP"), "0")
		}
		if strings.HasPrefix(r.URL.Path, "/api/") && !a.allowed(remote) {
			w.Header().Set("Retry-After", "60")
			apiError(w, 429, "Too many requests")
			return
		}
		mux.ServeHTTP(w, r)
	})
}

func (a *API) serve(w http.ResponseWriter, r *http.Request) {
	resource := r.PathValue("resource")
	if resource != "servers" && resource != "graph" {
		apiError(w, 404, "Not found")
		return
	}
	id := 0
	if raw := r.PathValue("id"); raw != "" {
		for _, c := range raw {
			if c < '0' || c > '9' {
				apiError(w, 400, "ID must be a positive integer")
				return
			}
		}
		value, err := strconv.ParseInt(raw, 10, 32)
		if err != nil || value < 1 {
			apiError(w, 400, "ID must be a positive integer")
			return
		}
		id = int(value)
	}
	// Cache only the two bounded bulk resources, never arbitrary IDs.
	if id == 0 {
		key := 0
		if resource == "graph" {
			key = 1
		}
		a.loading[key].Lock()
		defer a.loading[key].Unlock()
		a.mu.Lock()
		entry, ok := a.cache[resource]
		a.mu.Unlock()
		if ok && time.Now().Before(entry.expires) {
			a.send(w, entry.data)
			return
		}
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var value any
	if resource == "servers" {
		rows, err := a.store.Servers(ctx, id)
		if err != nil {
			slog.Error("servers query", "error", err)
			apiError(w, 503, "Database temporarily unavailable")
			return
		}
		if id > 0 {
			if len(rows) == 0 {
				apiError(w, 404, "Server not found")
				return
			}
			value = rows[0]
		} else {
			value = rows
		}
	} else {
		rows, err := a.store.Graphs(ctx, id)
		if err != nil {
			slog.Error("graph query", "error", err)
			apiError(w, 503, "Database temporarily unavailable")
			return
		}
		if id > 0 {
			if len(rows) == 0 {
				apiError(w, 404, "Server not found")
				return
			}
			value = rows[0]
		} else {
			value = rows
		}
	}
	data, err := json.Marshal(value)
	if err != nil {
		apiError(w, 500, "Response encoding failed")
		return
	}
	if id == 0 {
		a.mu.Lock()
		a.cache[resource] = cacheEntry{data, time.Now().Add(30 * time.Second)}
		a.mu.Unlock()
	}
	a.send(w, data)
}

func (a *API) send(w http.ResponseWriter, data []byte) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "public, max-age=15")
	w.Write(data)
}
