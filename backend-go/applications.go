package main

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/mail"
	"net/netip"
	"net/url"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Application struct {
	Name          string `json:"name"`
	Host          string `json:"host"`
	Port          string `json:"port"`
	Versions      string `json:"versions"`
	Modes         string `json:"modes"`
	Description   string `json:"description"`
	Website       string `json:"website"`
	Community     string `json:"community"`
	Owner         string `json:"owner"`
	Role          string `json:"role"`
	Channel       string `json:"channel"`
	Contact       string `json:"contact"`
	Rules         string `json:"rules"`
	Language      string `json:"language"`
	Access        string `json:"access"`
	AccessDetails string `json:"accessDetails"`
	Note          string `json:"note"`
	Authorized    bool   `json:"authorized"`
	Accepted      bool   `json:"accepted"`
}

func (a *Application) validate() error {
	a.Host = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(a.Host), "."))
	a.Name = strings.TrimSpace(a.Name)
	fields := []struct {
		value    string
		min, max int
	}{
		{a.Name, 2, 100}, {a.Host, 1, 253}, {a.Versions, 1, 100}, {a.Modes, 1, 200},
		{a.Description, 30, 1000}, {a.Owner, 2, 100}, {a.Contact, 3, 200}, {a.Language, 10, 500},
		{a.Note, 0, 1000}, {a.AccessDetails, 0, 1000},
	}
	for _, f := range fields {
		n := utf8.RuneCountInString(strings.TrimSpace(f.value))
		if n < f.min || n > f.max {
			return errors.New("Перевірте довжину обов’язкових полів.")
		}
	}
	port, err := strconv.Atoi(a.Port)
	if err != nil || port < 1 || port > 65535 {
		return errors.New("Некоректний порт.")
	}
	a.Port = strconv.Itoa(port)
	if strings.ContainsAny(a.Host, " /\\:@?#\r\n\t") {
		return errors.New("Вкажіть публічний домен або IPv4 без порту та шляху.")
	}
	if ip := net.ParseIP(a.Host); ip != nil {
		address, _ := netip.AddrFromSlice(ip)
		if !publicAddress(address.Unmap()) {
			return errors.New("Потрібна публічна адреса сервера.")
		}
	} else {
		if !strings.Contains(a.Host, ".") {
			return errors.New("Потрібен повний домен сервера.")
		}
		for _, label := range strings.Split(a.Host, ".") {
			if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
				return errors.New("Некоректний домен.")
			}
			for _, c := range label {
				if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
					return errors.New("Використайте ASCII / punycode домен.")
				}
			}
		}
	}
	for _, s := range []string{a.Community, a.Rules} {
		if !applicationURL(s) {
			return errors.New("Потрібні дійсні HTTPS-посилання на спільноту та правила.")
		}
	}
	if a.Website != "" && !applicationURL(a.Website) {
		return errors.New("Перевірте HTTPS-адресу сайту.")
	}
	if !oneOf(a.Role, "owner", "admin") || !oneOf(a.Channel, "telegram", "discord", "email") || !oneOf(a.Access, "open", "whitelist", "paid", "other") || !a.Authorized || !a.Accepted {
		return errors.New("Підтвердьте вимоги й право представляти сервер.")
	}
	if a.Access != "open" && utf8.RuneCountInString(strings.TrimSpace(a.AccessDetails)) < 10 {
		return errors.New("Опишіть умови входу.")
	}
	if a.Channel == "email" {
		address, e := mail.ParseAddress(a.Contact)
		if e != nil || address.Address != a.Contact {
			return errors.New("Перевірте email для відповіді.")
		}
	}
	return nil
}
func oneOf(s string, values ...string) bool {
	for _, v := range values {
		if s == v {
			return true
		}
	}
	return false
}
func applicationURL(s string) bool {
	u, e := url.Parse(s)
	return e == nil && len(s) <= 500 && u.Scheme == "https" && u.Hostname() != "" && u.User == nil
}
func randomKey() string {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	return hex.EncodeToString(b[:])
}

type Admission struct {
	store              *Store
	api                *API
	webhook, publicURL string
	client             *http.Client
}

func newAdmission(store *Store, api *API, webhook, publicURL string) (*Admission, error) {
	a := &Admission{store: store, api: api, webhook: webhook, publicURL: strings.TrimRight(publicURL, "/"), client: &http.Client{Timeout: 10 * time.Second, CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return http.ErrUseLastResponse }}}
	if webhook == "" {
		return a, nil
	}
	u, e := url.Parse(webhook)
	if e != nil || u.Scheme != "https" || u.Host != "discord.com" || u.User != nil || !strings.HasPrefix(u.Path, "/api/webhooks/") || u.RawQuery != "" || u.Fragment != "" {
		return nil, errors.New("UM_DISCORD_WEBHOOK_URL must be a discord.com HTTPS webhook URL")
	}
	u, e = url.Parse(a.publicURL)
	if e != nil || u.Scheme != "https" || u.Host == "" || u.User != nil || u.Path != "" || u.RawQuery != "" || u.Fragment != "" || len(a.publicURL) > 250 {
		return nil, errors.New("UM_PUBLIC_URL must be a public HTTPS origin without a path")
	}
	return a, nil
}
func (a *Admission) routes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/applications", a.submit)
	mux.HandleFunc("GET /api/applications/{id}", a.applicationStatus)
	mux.HandleFunc("GET /api/applications/config", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		writeJSON(w, 200, map[string]bool{"enabled": a.webhook != ""})
	})
	mux.HandleFunc("GET /moderation/{id}", a.moderate)
	mux.HandleFunc("POST /moderation/{id}", a.moderate)
}

// The submission key grants read-only access; moderation uses a separate secret.
func (a *Admission) applicationStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	id := r.PathValue("id")
	key := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !strings.HasPrefix(r.Header.Get("Authorization"), "Bearer ") || len(id) != 24 || len(key) != 64 {
		apiError(w, 404, "Посилання на заявку недійсне.")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	var stored, status, reason string
	var serverID *int
	e := a.store.pool.QueryRow(ctx, `SELECT request_key,status,reason,server_id FROM applications WHERE id=$1`, id).Scan(&stored, &status, &reason, &serverID)
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 503, "Не вдалося отримати статус. Спробуйте пізніше.")
		return
	}
	kh, sh := sha256.Sum256([]byte(key)), sha256.Sum256([]byte(stored))
	if e != nil || subtle.ConstantTimeCompare(kh[:], sh[:]) != 1 {
		apiError(w, 404, "Посилання на заявку недійсне.")
		return
	}
	writeJSON(w, 200, map[string]any{"id": id, "status": status, "reason": reason, "serverID": serverID})
}
func (a *Admission) submit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	if a.webhook == "" {
		apiError(w, 503, "Приймання заявок ще не підключено. Спробуйте пізніше.")
		return
	}
	if r.Header.Get("Origin") != a.publicURL {
		apiError(w, 403, "Надсилайте заявку з офіційної сторінки.")
		return
	}
	if !strings.HasPrefix(r.Header.Get("Content-Type"), "application/json") {
		apiError(w, 415, "Потрібен JSON.")
		return
	}
	key := r.Header.Get("Idempotency-Key")
	if len(key) != 64 {
		apiError(w, 400, "Оновіть сторінку й повторіть надсилання.")
		return
	}
	if _, e := hex.DecodeString(key); e != nil {
		apiError(w, 400, "Некоректний ключ запиту.")
		return
	}
	var app Application
	d := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20<<10))
	d.DisallowUnknownFields()
	if e := d.Decode(&app); e != nil {
		apiError(w, 400, "Некоректна заявка.")
		return
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		apiError(w, 400, "Некоректна заявка.")
		return
	}
	if e := app.validate(); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	body, _ := json.Marshal(app)
	port, _ := strconv.Atoi(app.Port)
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	tx, e := a.store.pool.Begin(ctx)
	if e != nil {
		apiError(w, 503, "Сховище тимчасово недоступне.")
		return
	}
	defer tx.Rollback(context.Background())
	// Serialize admission checks across instances, bounding queue spam and duplicate registrations.
	_, e = tx.Exec(ctx, `SELECT pg_advisory_xact_lock(72621003)`)
	var id string
	var previous []byte
	if e == nil {
		e = tx.QueryRow(ctx, `SELECT id,payload FROM applications WHERE request_key=$1`, key).Scan(&id, &previous)
	}
	if e != nil && !errors.Is(e, pgx.ErrNoRows) {
		apiError(w, 503, "Сховище недоступне.")
		return
	}
	if e == nil {
		var old Application
		json.Unmarshal(previous, &old)
		canonical, _ := json.Marshal(old)
		if !bytes.Equal(canonical, body) {
			apiError(w, 409, "Цей запит уже використано. Оновіть сторінку.")
			return
		}
		writeJSON(w, 200, map[string]string{"id": id, "status": "received"})
		return
	}
	var exists bool
	e = tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM servers WHERE lower(ip)=$1 AND port=$2)`, app.Host, port).Scan(&exists)
	if e != nil {
		apiError(w, 503, "Не вдалося перевірити заявку.")
		return
	}
	if exists {
		apiError(w, 409, "Цей сервер уже є в каталозі.")
		return
	}
	remote := r.RemoteAddr
	if a.api.TrustProxy && net.ParseIP(r.Header.Get("X-Real-IP")) != nil {
		remote = r.Header.Get("X-Real-IP")
	}
	if h, _, err := net.SplitHostPort(remote); err == nil {
		remote = h
	}
	iphash := sha256.Sum256([]byte(remote))
	var bucket int
	e = tx.QueryRow(ctx, `INSERT INTO application_limits(client,window_start,count) VALUES($1,now(),1) ON CONFLICT(client) DO UPDATE SET window_start=CASE WHEN application_limits.window_start<now()-interval '1 hour' THEN now() ELSE application_limits.window_start END,count=CASE WHEN application_limits.window_start<now()-interval '1 hour' THEN 1 ELSE application_limits.count+1 END RETURNING count`, hex.EncodeToString(iphash[:])).Scan(&bucket)
	if e != nil || bucket > 3 {
		apiError(w, 429, "Не більше трьох заявок на годину. Спробуйте пізніше.")
		return
	}
	if _, e = tx.Exec(ctx, `DELETE FROM application_limits WHERE window_start<now()-interval '2 hours'`); e != nil {
		apiError(w, 503, "Сховище недоступне.")
		return
	}
	var recent int
	e = tx.QueryRow(ctx, `SELECT count(*) FROM applications WHERE created_at>now()-interval '1 hour'`).Scan(&recent)
	if e != nil || recent >= 30 {
		apiError(w, 429, "Ліміт заявок досягнуто. Спробуйте пізніше.")
		return
	}
	id = randomKey()[:24]
	_, e = tx.Exec(ctx, `INSERT INTO applications(id,request_key,payload,host,port,token) VALUES($1,$2,$3,$4,$5,$6)`, id, key, body, app.Host, port, randomKey())
	if e != nil {
		var pe *pgconn.PgError
		if errors.As(e, &pe) && pe.Code == "23505" {
			apiError(w, 409, "Заявка для цієї адреси вже очікує перевірки.")
		} else {
			apiError(w, 503, "Не вдалося зберегти заявку.")
		}
		return
	}
	if tx.Commit(ctx) != nil {
		apiError(w, 503, "Не вдалося підтвердити збереження. Повторіть надсилання.")
		return
	}
	writeJSON(w, 201, map[string]string{"id": id, "status": "received"})
}

// A durable outbox. Locking keeps multiple backend instances from sending simultaneously.
// Delivery is at least once: Discord may accept a message before a connection fails.
func (a *Admission) deliver(ctx context.Context) error {
	if a.webhook == "" {
		return nil
	}
	tx, e := a.store.pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(context.Background())
	var id, token, status string
	var body []byte
	var attempts int
	e = tx.QueryRow(ctx, `SELECT id,token,payload,attempts,status FROM applications WHERE notified_at IS NULL AND next_attempt<=now() ORDER BY created_at FOR UPDATE SKIP LOCKED LIMIT 1`).Scan(&id, &token, &body, &attempts, &status)
	if e != nil {
		if errors.Is(e, pgx.ErrNoRows) {
			return nil
		}
		return e
	}
	var app Application
	if e = json.Unmarshal(body, &app); e != nil {
		return e
	}
	link := a.publicURL + "/moderation/" + id + "?token=" + token
	payload := map[string]any{"content": "Нова заявка до каталогу Ґрадієнту · " + id, "allowed_mentions": map[string]any{"parse": []string{}}, "embeds": []any{map[string]any{"title": app.Name, "description": app.Description, "color": 7506394, "fields": []any{map[string]any{"name": "Адреса", "value": app.Host + ":" + app.Port}, map[string]any{"name": "Версії · режими", "value": app.Versions + " · " + app.Modes}, map[string]any{"name": "Представник", "value": app.Owner + " (" + app.Role + ") · " + app.Channel + ": " + app.Contact}, map[string]any{"name": "Усі відповіді", "value": "Відкрийте приватну сторінку перевірки. Рішення потребує підтвердження."}}}}, "components": []any{map[string]any{"type": 1, "components": []any{map[string]any{"type": 2, "style": 5, "label": "Схвалити / перевірити", "url": link + "&action=approve"}, map[string]any{"type": 2, "style": 5, "label": "Відхилити", "url": link + "&action=reject"}}}}}
	data, _ := json.Marshal(payload)
	req, e := http.NewRequestWithContext(ctx, "POST", a.webhook+"?wait=true&with_components=true", bytes.NewReader(data))
	if e != nil {
		return errors.New("webhook request unavailable")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, e := a.client.Do(req)
	delivered := false
	if e == nil {
		var confirmation struct {
			ID string `json:"id"`
		}
		decodeErr := json.NewDecoder(io.LimitReader(resp.Body, 64<<10)).Decode(&confirmation)
		resp.Body.Close()
		delivered = resp.StatusCode == http.StatusOK && decodeErr == nil && confirmation.ID != ""
	}
	if delivered {
		_, e = tx.Exec(ctx, `UPDATE applications SET notified_at=now(),attempts=attempts+1 WHERE id=$1`, id)
	} else {
		delay := time.Duration(1<<min(attempts, 9)) * time.Minute
		_, e = tx.Exec(ctx, `UPDATE applications SET attempts=attempts+1,next_attempt=now()+$2::interval WHERE id=$1`, id, fmt.Sprintf("%d seconds", int(delay.Seconds())))
		slog.Warn("application notification deferred", "application", id)
	}
	if e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func (a *Admission) worker(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()
	for {
		c, cancel := context.WithTimeout(ctx, 13*time.Second)
		if e := a.deliver(c); e != nil && ctx.Err() == nil {
			slog.Warn("application delivery unavailable")
		}
		cancel()
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (a *Admission) moderate(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Referrer-Policy", "no-referrer")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; form-action 'self'; frame-ancestors 'none'; base-uri 'none'")
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	if r.ParseForm() != nil {
		http.Error(w, "Некоректний запит", 400)
		return
	}
	token := r.FormValue("token")
	id := r.PathValue("id")
	var stored, status, reason string
	var payload []byte
	var created time.Time
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()
	e := a.store.pool.QueryRow(ctx, `SELECT token,payload,status,reason,created_at FROM applications WHERE id=$1`, id).Scan(&stored, &payload, &status, &reason, &created)
	th, sh := sha256.Sum256([]byte(token)), sha256.Sum256([]byte(stored))
	if e != nil || token == "" || subtle.ConstantTimeCompare(th[:], sh[:]) != 1 {
		http.Error(w, "Посилання недійсне", 404)
		return
	}
	if time.Since(created) > 30*24*time.Hour {
		http.Error(w, "Посилання прострочене. Зверніться до адміністратора каталогу.", 410)
		return
	}
	var app Application
	json.Unmarshal(payload, &app)
	message := ""
	nonce := ""
	if r.Method == "POST" {
		cookie, err := r.Cookie("um_moderation")
		if r.Header.Get("Origin") != a.publicURL || err != nil || len(r.PostForm.Get("csrf")) != 64 || subtle.ConstantTimeCompare([]byte(cookie.Value), []byte(r.PostForm.Get("csrf"))) != 1 {
			http.Error(w, "Оновіть сторінку перед підтвердженням", 403)
			return
		}
		action := r.PostForm.Get("decision")
		reason = strings.TrimSpace(r.PostForm.Get("reason"))
		if !oneOf(action, "approve", "reject") || utf8.RuneCountInString(reason) > 1000 || action == "reject" && utf8.RuneCountInString(reason) < 5 {
			message = "Для відмови вкажіть причину (5–1000 символів)."
		} else {
			tx, err := a.store.pool.Begin(ctx)
			if err != nil {
				http.Error(w, "Сховище недоступне", 503)
				return
			}
			defer tx.Rollback(context.Background())
			err = tx.QueryRow(ctx, `SELECT status FROM applications WHERE id=$1 FOR UPDATE`, id).Scan(&status)
			if err != nil {
				http.Error(w, "Сховище недоступне", 503)
				return
			}
			if status != "pending" {
				message = "Рішення вже збережене."
			} else {
				var serverID *int
				port, _ := strconv.Atoi(app.Port)
				if action == "approve" {
					var n int
					err = tx.QueryRow(ctx, `INSERT INTO servers(name,ip,port,description,link) VALUES($1,$2,$3,$4,NULLIF($5,'')) RETURNING id`, app.Name, app.Host, port, app.Description, app.Website).Scan(&n)
					serverID = &n
					status = "approved"
				} else {
					status = "rejected"
				}
				if err == nil {
					_, err = tx.Exec(ctx, `UPDATE applications SET status=$2,reason=$3,server_id=$4,decided_at=now() WHERE id=$1`, id, status, reason, serverID)
				}
				if err != nil {
					http.Error(w, "Не вдалося зберегти рішення. Можливо, сервер уже є в каталозі.", 409)
					return
				}
				if tx.Commit(ctx) != nil {
					http.Error(w, "Повторіть перевірку стану заявки", 503)
					return
				}
				a.api.Invalidate()
				message = "Рішення збережено."
			}
		}
	}
	nonce = randomKey()
	http.SetCookie(w, &http.Cookie{Name: "um_moderation", Value: nonce, Path: "/moderation/" + id, Secure: strings.HasPrefix(a.publicURL, "https://"), HttpOnly: true, SameSite: http.SameSiteStrictMode, MaxAge: 1800})
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	moderationTemplate.Execute(w, map[string]any{"ID": id, "Token": token, "Nonce": nonce, "App": app, "Status": status, "Reason": reason, "Message": message, "Pending": status == "pending"})
}

var moderationTemplate = template.Must(template.New("moderation").Parse(`<!doctype html><html lang="uk"><meta charset="utf-8"><meta name="viewport" content="width=device-width"><title>Перевірка заявки · Ґрадієнт</title><style>body{font:16px system-ui;background:#101012;color:#eee;margin:0;padding:32px}main{max-width:780px;margin:auto}article,form{background:#19191d;border:1px solid #333;border-radius:16px;padding:24px;margin:20px 0}dt{color:#aaa;margin-top:18px}dd{margin:5px 0;white-space:pre-wrap;overflow-wrap:anywhere}button{padding:14px 22px;border:0;border-radius:8px;font:inherit;cursor:pointer}textarea{display:block;box-sizing:border-box;width:100%;padding:12px;background:#101012;color:#fff;border:1px solid #555;margin:12px 0}small{color:#aaa}</style><main><small>ҐРАДІЄНТ · ПРИВАТНА МОДЕРАЦІЯ</small><h1>{{.App.Name}}</h1><p>Заявка {{.ID}} · {{.Status}}</p><p role="status">{{.Message}}</p><article><dl><dt>Адреса</dt><dd>{{.App.Host}}:{{.App.Port}}</dd><dt>Опис</dt><dd>{{.App.Description}}</dd><dt>Версії та режими</dt><dd>{{.App.Versions}} · {{.App.Modes}}</dd><dt>Сайт</dt><dd>{{.App.Website}}</dd><dt>Спільнота</dt><dd>{{.App.Community}}</dd><dt>Правила</dt><dd>{{.App.Rules}}</dd><dt>Мова</dt><dd>{{.App.Language}}</dd><dt>Доступ</dt><dd>{{.App.Access}} · {{.App.AccessDetails}}</dd><dt>Представник</dt><dd>{{.App.Owner}} · {{.App.Role}} · {{.App.Channel}}: {{.App.Contact}}</dd><dt>Коментар</dt><dd>{{.App.Note}}</dd></dl></article>{{if .Pending}}<p>Перевірте всі вимоги та адресу перед рішенням. Доступ до цього посилання дає право модерації — не пересилайте його. Схвалення додає адресу до моніторингу; доступність перевіряється окремим опитуванням.</p><form method="post"><input type="hidden" name="token" value="{{.Token}}"><input type="hidden" name="csrf" value="{{.Nonce}}"><label>Причина відмови або примітка<textarea name="reason" maxlength="1000" rows="4"></textarea></label><button name="decision" value="approve">Підтвердити додавання</button> <button name="decision" value="reject">Відхилити заявку</button></form>{{else}}<article><h2>Рішення вже ухвалено</h2><p>{{.Reason}}</p><a href="/">До каталогу</a></article>{{end}}</main></html>`))
