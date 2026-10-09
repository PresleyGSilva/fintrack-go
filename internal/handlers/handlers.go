package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PresleyGSilva/fintrack-go/internal/auth"
)

type Handler struct {
	DB        *pgxpool.Pool
	JWTSecret string
	Limiter   *LoginLimiter
}

type ctxKey string

const userKey ctxKey = "user_id"

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func sendOK(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusOK, map[string]any{"data": data})
}

func created(w http.ResponseWriter, data any) {
	writeJSON(w, http.StatusCreated, map[string]any{"data": data})
}

func apiErr(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code, "message": message},
	})
}

func validation(w http.ResponseWriter, msg string) {
	apiErr(w, http.StatusBadRequest, "VALIDATION_ERROR", msg)
}

func unauthorized(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "unauthorized"
	}
	apiErr(w, http.StatusUnauthorized, "UNAUTHORIZED", msg)
}

func notFound(w http.ResponseWriter, msg string) {
	if msg == "" {
		msg = "not found"
	}
	apiErr(w, http.StatusNotFound, "NOT_FOUND", msg)
}

func conflict(w http.ResponseWriter, msg string) {
	apiErr(w, http.StatusConflict, "CONFLICT", msg)
}

func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		validation(w, "invalid JSON body: "+err.Error())
		return false
	}
	return true
}

// ---- auth middleware + org resolution ----

func (h *Handler) withAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" || !strings.HasPrefix(strings.ToLower(header), "bearer ") {
			unauthorized(w, "missing bearer token")
			return
		}
		token := strings.TrimSpace(header[len("Bearer "):])
		// Lenient split: "bearer " case-insensitive.
		if token == "" {
			parts := strings.SplitN(header, " ", 2)
			if len(parts) == 2 {
				token = strings.TrimSpace(parts[1])
			}
		}
		uid, err := auth.ParseAccess(h.JWTSecret, token)
		if err != nil {
			unauthorized(w, "invalid or expired token")
			return
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), userKey, uid)))
	})
}

func userID(r *http.Request) string {
	uid, _ := r.Context().Value(userKey).(string)
	return uid
}

// orgID resolve a org: header X-Org-ID (com membership) ou a primeira org do user.
func (h *Handler) orgID(w http.ResponseWriter, r *http.Request) (string, bool) {
	uid := userID(r)
	if uid == "" {
		unauthorized(w, "")
		return "", false
	}
	if org := strings.TrimSpace(r.Header.Get("X-Org-ID")); org != "" {
		var ok bool
		err := h.DB.QueryRow(r.Context(),
			`SELECT TRUE FROM memberships WHERE user_id=$1 AND org_id=$2`, uid, org).Scan(&ok)
		if err != nil {
			notFound(w, "organization not found")
			return "", false
		}
		return org, true
	}
	var org string
	err := h.DB.QueryRow(r.Context(),
		`SELECT org_id FROM memberships WHERE user_id=$1 ORDER BY created_at LIMIT 1`, uid).Scan(&org)
	if err != nil {
		notFound(w, "no organization for user")
		return "", false
	}
	return org, true
}
