package handlers

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/PresleyGSilva/fintrack-go/internal/auth"
	"github.com/PresleyGSilva/fintrack-go/internal/models"
)

// LoginLimiter é um rate-limit simples em memória para o login:
// no máximo maxAttempts por janela por IP.
type LoginLimiter struct {
	mu          sync.Mutex
	hits        map[string][]time.Time
	maxAttempts int
	window      time.Duration
}

func NewLoginLimiter(maxAttempts int, window time.Duration) *LoginLimiter {
	return &LoginLimiter{hits: map[string][]time.Time{}, maxAttempts: maxAttempts, window: window}
}

func (l *LoginLimiter) allow(ip string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := time.Now()
	cut := now.Add(-l.window)
	kept := l.hits[ip][:0]
	for _, t := range l.hits[ip] {
		if t.After(cut) {
			kept = append(kept, t)
		}
	}
	if len(kept) >= l.maxAttempts {
		l.hits[ip] = kept
		return false
	}
	l.hits[ip] = append(kept, now)
	return true
}

func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		parts := strings.Split(fwd, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

const refreshCookieName = "refresh_token"

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     refreshCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(auth.RefreshTTL.Seconds()),
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name: refreshCookieName, Value: "",
		Path: "/", HttpOnly: true, MaxAge: -1,
	})
}

func validEmail(e string) bool {
	e = strings.TrimSpace(e)
	if len(e) < 3 || len(e) > 255 || !strings.Contains(e, "@") {
		return false
	}
	parts := strings.Split(e, "@")
	return len(parts) == 2 && parts[0] != "" && strings.Contains(parts[1], ".")
}

func (h *Handler) issueSession(w http.ResponseWriter, r *http.Request, userID string) (string, bool) {
	access, err := auth.IssueAccess(h.JWTSecret, userID)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot issue token")
		return "", false
	}
	raw, hash, err := auth.NewRefreshToken()
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot issue token")
		return "", false
	}
	if _, err := h.DB.Exec(r.Context(),
		`INSERT INTO refresh_tokens(id, user_id, token_hash, expires_at) VALUES($1,$2,$3,now()+make_interval(days=>7))`,
		uuid.NewString(), userID, hash); err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot persist session")
		return "", false
	}
	setRefreshCookie(w, raw)
	return access, true
}

func (h *Handler) listOrgs(r *http.Request, userID string) ([]models.Org, error) {
	rows, err := h.DB.Query(r.Context(),
		`SELECT o.id, o.name FROM orgs o
		 JOIN memberships m ON m.org_id=o.id WHERE m.user_id=$1 ORDER BY o.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var orgs []models.Org
	for rows.Next() {
		var o models.Org
		if err := rows.Scan(&o.ID, &o.Name); err != nil {
			return nil, err
		}
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	if in.Name == "" || len(in.Name) > 120 {
		validation(w, "name is required (max 120)")
		return
	}
	if !validEmail(in.Email) {
		validation(w, "valid email is required")
		return
	}
	if len(in.Password) < 8 || len(in.Password) > 128 {
		validation(w, "password must be 8..128 chars")
		return
	}
	hash, err := auth.HashPassword(in.Password)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot hash password")
		return
	}
	userID := uuid.NewString()
	orgID := uuid.NewString()
	orgName := in.Name + "'s Org"
	if len(orgName) > 120 {
		orgName = orgName[:120]
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO users(id,name,email,password_hash) VALUES($1,$2,$3,$4)`,
		userID, in.Name, in.Email, hash); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			conflict(w, "email already registered")
			return
		}
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create user")
		return
	}
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO orgs(id,name) VALUES($1,$2)`, orgID, orgName); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create organization")
		return
	}
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO memberships(user_id,org_id,role) VALUES($1,$2,'owner')`, userID, orgID); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create membership")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot commit registration")
		return
	}
	access, ok := h.issueSession(w, r, userID)
	if !ok {
		return
	}
	created(w, map[string]any{
		"user":         models.User{ID: userID, Name: in.Name, Email: in.Email},
		"org":          models.Org{ID: orgID, Name: orgName},
		"access_token": access,
	})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	if !h.Limiter.allow(clientIP(r)) {
		apiErr(w, http.StatusTooManyRequests, "RATE_LIMITED", "too many login attempts, try again later")
		return
	}
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if !decode(w, r, &in) {
		return
	}
	email := strings.ToLower(strings.TrimSpace(in.Email))
	var u models.User
	var pwHash string
	err := h.DB.QueryRow(r.Context(),
		`SELECT id,name,email,password_hash FROM users WHERE email=$1`, email).
		Scan(&u.ID, &u.Name, &u.Email, &pwHash)
	if err != nil || !auth.CheckPassword(pwHash, in.Password) {
		unauthorized(w, "invalid credentials")
		return
	}
	orgs, err := h.listOrgs(r, u.ID)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot load organizations")
		return
	}
	access, ok := h.issueSession(w, r, u.ID)
	if !ok {
		return
	}
	sendOK(w, map[string]any{"user": u, "orgs": orgs, "access_token": access})
}

func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie(refreshCookieName)
	if err != nil || c.Value == "" {
		unauthorized(w, "missing refresh token")
		return
	}
	hash := auth.HashRefresh(c.Value)
	var id, userID string
	var expires int64
	var revoked bool
	err = h.DB.QueryRow(r.Context(),
		`SELECT id, user_id, EXTRACT(EPOCH FROM expires_at)::BIGINT, revoked
		 FROM refresh_tokens WHERE token_hash=$1`, hash).
		Scan(&id, &userID, &expires, &revoked)
	if err != nil || revoked || expires < time.Now().Unix() {
		unauthorized(w, "invalid or expired refresh token")
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot rotate session")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	if _, err := tx.Exec(r.Context(),
		`UPDATE refresh_tokens SET revoked=TRUE WHERE id=$1`, id); err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot rotate session")
		return
	}
	raw, newHash, err := auth.NewRefreshToken()
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot rotate session")
		return
	}
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO refresh_tokens(id, user_id, token_hash, expires_at) VALUES($1,$2,$3,now()+make_interval(days=>7))`,
		uuid.NewString(), userID, newHash); err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot rotate session")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot rotate session")
		return
	}
	access, err := auth.IssueAccess(h.JWTSecret, userID)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "UNAUTHORIZED", "cannot issue token")
		return
	}
	setRefreshCookie(w, raw)
	sendOK(w, map[string]any{"access_token": access})
}

func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		_, _ = h.DB.Exec(r.Context(),
			`UPDATE refresh_tokens SET revoked=TRUE WHERE token_hash=$1`, auth.HashRefresh(c.Value))
	}
	clearRefreshCookie(w)
	sendOK(w, map[string]any{"status": "logged out"})
}

func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	uid := userID(r)
	var u models.User
	if err := h.DB.QueryRow(r.Context(),
		`SELECT id,name,email,created_at FROM users WHERE id=$1`, uid).Scan(&u.ID, &u.Name, &u.Email, &u.CreatedAt); err != nil {
		unauthorized(w, "user not found")
		return
	}
	orgs, err := h.listOrgs(r, uid)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot load organizations")
		return
	}
	sendOK(w, map[string]any{"user": u, "orgs": orgs})
}
