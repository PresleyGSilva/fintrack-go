package handlers

import (
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	docs "github.com/PresleyGSilva/fintrack-go/docs"
)

// frontendOrigins lista as origens permitidas para CORS com credenciais.
// O refresh usa cookie httpOnly, então o navegador exige origem explícita
// (nunca "*"). Configurável via FRONTEND_ORIGINS (vírgulas).
func frontendOrigins() []string {
	if v := strings.TrimSpace(os.Getenv("FRONTEND_ORIGINS")); v != "" {
		parts := strings.Split(v, ",")
		out := make([]string, 0, len(parts))
		for _, p := range parts {
			if p = strings.TrimSpace(p); p != "" {
				out = append(out, p)
			}
		}
		if len(out) > 0 {
			return out
		}
	}
	return []string{"http://localhost:3004", "http://localhost:3000"}
}

// NewRouter monta todas as rotas da API.
func (h *Handler) NewRouter() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(30 * time.Second))
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   frontendOrigins(),
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Org-ID", "X-User"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		sendOK(w, map[string]any{"status": "ok"})
	})
	r.Get("/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = w.Write(docs.Spec)
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/register", h.Register)
		r.Post("/auth/login", h.Login)
		r.Post("/auth/refresh", h.Refresh)
		r.Post("/auth/logout", h.Logout)

		r.Group(func(r chi.Router) {
			r.Use(h.withAuth)
			r.Get("/me", h.Me)

			r.Route("/accounts", func(r chi.Router) {
				r.Get("/", h.ListAccounts)
				r.Post("/", h.CreateAccount)
				r.Get("/{id}", h.GetAccount)
				r.Put("/{id}", h.UpdateAccount)
				r.Delete("/{id}", h.DeleteAccount)
			})
			r.Route("/categories", func(r chi.Router) {
				r.Get("/", h.ListCategories)
				r.Post("/", h.CreateCategory)
				r.Get("/{id}", h.GetCategory)
				r.Put("/{id}", h.UpdateCategory)
				r.Delete("/{id}", h.DeleteCategory)
			})
			r.Route("/transactions", func(r chi.Router) {
				r.Get("/", h.ListTransactions)
				r.Post("/", h.CreateTransaction)
				r.Get("/{id}", h.GetTransaction)
				r.Put("/{id}", h.UpdateTransaction)
				r.Delete("/{id}", h.DeleteTransaction)
			})
			r.Post("/transfers", h.CreateTransfer)
			r.Route("/recurrences", func(r chi.Router) {
				r.Get("/", h.ListRecurrences)
				r.Post("/", h.CreateRecurrence)
				r.Get("/{id}", h.GetRecurrence)
				r.Put("/{id}", h.UpdateRecurrence)
				r.Delete("/{id}", h.DeleteRecurrence)
			})
			r.Route("/budgets", func(r chi.Router) {
				r.Get("/", h.ListBudgets)
				r.Post("/", h.CreateBudget)
				r.Get("/{id}", h.GetBudget)
				r.Put("/{id}", h.UpdateBudget)
				r.Delete("/{id}", h.DeleteBudget)
			})
			r.Route("/goals", func(r chi.Router) {
				r.Get("/", h.ListGoals)
				r.Post("/", h.CreateGoal)
				r.Get("/{id}", h.GetGoal)
				r.Put("/{id}", h.UpdateGoal)
				r.Delete("/{id}", h.DeleteGoal)
			})
			r.Get("/reports/summary", h.Summary)
			r.Get("/reports/monthly", h.Monthly)
			r.Post("/import/csv", h.ImportCSV)
			r.Get("/export/csv", h.ExportCSV)
		})
	})
	return r
}
