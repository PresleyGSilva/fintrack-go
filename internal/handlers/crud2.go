package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"github.com/PresleyGSilva/fintrack-go/internal/models"
)

func scanRecurrence(scan func(...any) error) (models.Recurrence, error) {
	var x models.Recurrence
	var cat nullText
	var nr time.Time
	err := scan(&x.ID, &x.OrgID, &x.AccountID, &cat.t, &x.Kind, &x.AmountCents,
		&x.Frequency, &nr, &x.Active, &x.Note)
	if err != nil {
		return x, err
	}
	x.CategoryID = cat.ptr()
	x.NextRun = fmtDate(nr)
	return x, nil
}

const recCols = `id,org_id,account_id,category_id,kind,amount_cents,frequency,next_run,active,note`

func (h *Handler) ListRecurrences(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT `+recCols+` FROM recurrences WHERE org_id=$1 ORDER BY next_run`, org)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot list recurrences")
		return
	}
	defer rows.Close()
	out := []models.Recurrence{}
	for rows.Next() {
		x, err := scanRecurrence(rows.Scan)
		if err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan recurrence")
			return
		}
		out = append(out, x)
	}
	sendOK(w, out)
}

func (h *Handler) CreateRecurrence(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		AccountID   string  `json:"account_id"`
		CategoryID  *string `json:"category_id"`
		Kind        string  `json:"kind"`
		AmountCents int64   `json:"amount_cents"`
		Frequency   string  `json:"frequency"`
		NextRun     string  `json:"next_run"`
		Active      *bool   `json:"active"`
		Note        string  `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	if _, found := h.getAccount(r, org, in.AccountID); !found {
		validation(w, "account not found")
		return
	}
	if in.CategoryID != nil && *in.CategoryID != "" {
		if _, found := h.getCategory(r, org, *in.CategoryID); !found {
			validation(w, "category not found")
			return
		}
	} else {
		in.CategoryID = nil
	}
	if !kinds[in.Kind] {
		validation(w, "kind must be income|expense")
		return
	}
	if in.AmountCents <= 0 {
		validation(w, "amount_cents must be > 0")
		return
	}
	if !frequencies[in.Frequency] {
		validation(w, "frequency must be daily|weekly|monthly")
		return
	}
	if !validDate(in.NextRun) {
		validation(w, "next_run must be YYYY-MM-DD")
		return
	}
	active := true
	if in.Active != nil {
		active = *in.Active
	}
	x, err := scanRecurrence(h.DB.QueryRow(r.Context(),
		`INSERT INTO recurrences(id,org_id,account_id,category_id,kind,amount_cents,frequency,next_run,active,note)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) RETURNING `+recCols,
		uuid.NewString(), org, in.AccountID, in.CategoryID, in.Kind, in.AmountCents,
		in.Frequency, in.NextRun, active, in.Note).Scan)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create recurrence")
		return
	}
	created(w, x)
}

func (h *Handler) GetRecurrence(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	x, err := scanRecurrence(h.DB.QueryRow(r.Context(),
		`SELECT `+recCols+` FROM recurrences WHERE id=$1 AND org_id=$2`,
		chi.URLParam(r, "id"), org).Scan)
	if err != nil {
		notFound(w, "recurrence not found")
		return
	}
	sendOK(w, x)
}

func (h *Handler) UpdateRecurrence(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var exists bool
	_ = h.DB.QueryRow(r.Context(),
		`SELECT TRUE FROM recurrences WHERE id=$1 AND org_id=$2`, id, org).Scan(&exists)
	if !exists {
		notFound(w, "recurrence not found")
		return
	}
	var in struct {
		AccountID   *string `json:"account_id"`
		CategoryID  *string `json:"category_id"`
		Kind        *string `json:"kind"`
		AmountCents *int64  `json:"amount_cents"`
		Frequency   *string `json:"frequency"`
		NextRun     *string `json:"next_run"`
		Active      *bool   `json:"active"`
		Note        *string `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	sets, args := []string{}, []any{}
	add := func(expr string, v any) {
		args = append(args, v)
		sets = append(sets, expr+"=$"+itoa(len(args)))
	}
	if in.AccountID != nil {
		if _, found := h.getAccount(r, org, *in.AccountID); !found {
			validation(w, "account not found")
			return
		}
		add("account_id", *in.AccountID)
	}
	if in.CategoryID != nil {
		if *in.CategoryID == "" {
			add("category_id", nil)
		} else {
			if _, found := h.getCategory(r, org, *in.CategoryID); !found {
				validation(w, "category not found")
				return
			}
			add("category_id", *in.CategoryID)
		}
	}
	if in.Kind != nil {
		if !kinds[*in.Kind] {
			validation(w, "kind must be income|expense")
			return
		}
		add("kind", *in.Kind)
	}
	if in.AmountCents != nil {
		if *in.AmountCents <= 0 {
			validation(w, "amount_cents must be > 0")
			return
		}
		add("amount_cents", *in.AmountCents)
	}
	if in.Frequency != nil {
		if !frequencies[*in.Frequency] {
			validation(w, "frequency must be daily|weekly|monthly")
			return
		}
		add("frequency", *in.Frequency)
	}
	if in.NextRun != nil {
		if !validDate(*in.NextRun) {
			validation(w, "next_run must be YYYY-MM-DD")
			return
		}
		add("next_run", *in.NextRun)
	}
	if in.Active != nil {
		add("active", *in.Active)
	}
	if in.Note != nil {
		add("note", *in.Note)
	}
	if len(sets) == 0 {
		validation(w, "nothing to update")
		return
	}
	args = append(args, id, org)
	q := `UPDATE recurrences SET ` + strings.Join(sets, ", ") +
		` WHERE id=$` + itoa(len(args)-1) + ` AND org_id=$` + itoa(len(args)) +
		` RETURNING ` + recCols
	x, err := scanRecurrence(h.DB.QueryRow(r.Context(), q, args...).Scan)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update recurrence")
		return
	}
	sendOK(w, x)
}

func (h *Handler) DeleteRecurrence(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(),
		`DELETE FROM recurrences WHERE id=$1 AND org_id=$2`, chi.URLParam(r, "id"), org)
	if err != nil || tag.RowsAffected() == 0 {
		notFound(w, "recurrence not found")
		return
	}
	sendOK(w, map[string]any{"status": "deleted"})
}

// ---------- budgets ----------

func scanBudget(scan func(...any) error) (models.Budget, error) {
	var b models.Budget
	var m time.Time
	err := scan(&b.ID, &b.OrgID, &b.CategoryID, &m, &b.LimitCents)
	if err != nil {
		return b, err
	}
	b.Month = m.Format("2006-01-02")
	return b, nil
}

const budgetCols = `id,org_id,category_id,month,limit_cents`

func validMonth(s string) bool {
	t, err := time.Parse("2006-01-02", s)
	return err == nil && t.Day() == 1
}

func (h *Handler) ListBudgets(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT `+budgetCols+` FROM budgets WHERE org_id=$1 ORDER BY month DESC`, org)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot list budgets")
		return
	}
	defer rows.Close()
	out := []models.Budget{}
	for rows.Next() {
		b, err := scanBudget(rows.Scan)
		if err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan budget")
			return
		}
		out = append(out, b)
	}
	sendOK(w, out)
}

func (h *Handler) CreateBudget(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		CategoryID string `json:"category_id"`
		Month      string `json:"month"`
		LimitCents int64  `json:"limit_cents"`
	}
	if !decode(w, r, &in) {
		return
	}
	if _, found := h.getCategory(r, org, in.CategoryID); !found {
		validation(w, "category not found")
		return
	}
	if !validMonth(in.Month) {
		validation(w, "month must be YYYY-MM-01")
		return
	}
	if in.LimitCents <= 0 {
		validation(w, "limit_cents must be > 0")
		return
	}
	b, err := scanBudget(h.DB.QueryRow(r.Context(),
		`INSERT INTO budgets(id,org_id,category_id,month,limit_cents) VALUES($1,$2,$3,$4,$5)
		 RETURNING `+budgetCols,
		uuid.NewString(), org, in.CategoryID, in.Month, in.LimitCents).Scan)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			conflict(w, "budget already exists for category/month")
			return
		}
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create budget")
		return
	}
	created(w, b)
}

func (h *Handler) GetBudget(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	b, err := scanBudget(h.DB.QueryRow(r.Context(),
		`SELECT `+budgetCols+` FROM budgets WHERE id=$1 AND org_id=$2`,
		chi.URLParam(r, "id"), org).Scan)
	if err != nil {
		notFound(w, "budget not found")
		return
	}
	sendOK(w, b)
}

func (h *Handler) UpdateBudget(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		LimitCents *int64 `json:"limit_cents"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.LimitCents == nil || *in.LimitCents <= 0 {
		validation(w, "limit_cents must be > 0")
		return
	}
	b, err := scanBudget(h.DB.QueryRow(r.Context(),
		`UPDATE budgets SET limit_cents=$1 WHERE id=$2 AND org_id=$3 RETURNING `+budgetCols,
		*in.LimitCents, chi.URLParam(r, "id"), org).Scan)
	if err != nil {
		notFound(w, "budget not found")
		return
	}
	sendOK(w, b)
}

func (h *Handler) DeleteBudget(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(),
		`DELETE FROM budgets WHERE id=$1 AND org_id=$2`, chi.URLParam(r, "id"), org)
	if err != nil || tag.RowsAffected() == 0 {
		notFound(w, "budget not found")
		return
	}
	sendOK(w, map[string]any{"status": "deleted"})
}

// ---------- goals ----------

func scanGoal(scan func(...any) error) (models.Goal, error) {
	var g models.Goal
	err := scan(&g.ID, &g.OrgID, &g.Name, &g.TargetCents, &g.CurrentCents)
	return g, err
}

const goalCols = `id,org_id,name,target_cents,current_cents`

func (h *Handler) ListGoals(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT `+goalCols+` FROM goals WHERE org_id=$1 ORDER BY created_at`, org)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot list goals")
		return
	}
	defer rows.Close()
	out := []models.Goal{}
	for rows.Next() {
		g, err := scanGoal(rows.Scan)
		if err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan goal")
			return
		}
		out = append(out, g)
	}
	sendOK(w, out)
}

func (h *Handler) CreateGoal(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		Name         string `json:"name"`
		TargetCents  int64  `json:"target_cents"`
		CurrentCents int64  `json:"current_cents"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		validation(w, "name is required (max 120)")
		return
	}
	if in.TargetCents <= 0 {
		validation(w, "target_cents must be > 0")
		return
	}
	if in.CurrentCents < 0 {
		validation(w, "current_cents must be >= 0")
		return
	}
	g := models.Goal{ID: uuid.NewString(), OrgID: org, Name: in.Name,
		TargetCents: in.TargetCents, CurrentCents: in.CurrentCents}
	if _, err := h.DB.Exec(r.Context(),
		`INSERT INTO goals(id,org_id,name,target_cents,current_cents) VALUES($1,$2,$3,$4,$5)`,
		g.ID, g.OrgID, g.Name, g.TargetCents, g.CurrentCents); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create goal")
		return
	}
	created(w, g)
}

func (h *Handler) GetGoal(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	g, err := scanGoal(h.DB.QueryRow(r.Context(),
		`SELECT `+goalCols+` FROM goals WHERE id=$1 AND org_id=$2`,
		chi.URLParam(r, "id"), org).Scan)
	if err != nil {
		notFound(w, "goal not found")
		return
	}
	sendOK(w, g)
}

func (h *Handler) UpdateGoal(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	var cur models.Goal
	err := h.DB.QueryRow(r.Context(),
		`SELECT id,org_id,name,target_cents,current_cents FROM goals WHERE id=$1 AND org_id=$2`,
		id, org).Scan(&cur.ID, &cur.OrgID, &cur.Name, &cur.TargetCents, &cur.CurrentCents)
	if err != nil {
		notFound(w, "goal not found")
		return
	}
	var in struct {
		Name         *string `json:"name"`
		TargetCents  *int64  `json:"target_cents"`
		CurrentCents *int64  `json:"current_cents"`
	}
	if !decode(w, r, &in) {
		return
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" || len(n) > 120 {
			validation(w, "name is required (max 120)")
			return
		}
		cur.Name = n
	}
	if in.TargetCents != nil {
		if *in.TargetCents <= 0 {
			validation(w, "target_cents must be > 0")
			return
		}
		cur.TargetCents = *in.TargetCents
	}
	if in.CurrentCents != nil {
		if *in.CurrentCents < 0 {
			validation(w, "current_cents must be >= 0")
			return
		}
		cur.CurrentCents = *in.CurrentCents
	}
	if _, err := h.DB.Exec(r.Context(),
		`UPDATE goals SET name=$1, target_cents=$2, current_cents=$3 WHERE id=$4 AND org_id=$5`,
		cur.Name, cur.TargetCents, cur.CurrentCents, id, org); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update goal")
		return
	}
	sendOK(w, cur)
}

func (h *Handler) DeleteGoal(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	tag, err := h.DB.Exec(r.Context(),
		`DELETE FROM goals WHERE id=$1 AND org_id=$2`, chi.URLParam(r, "id"), org)
	if err != nil || tag.RowsAffected() == 0 {
		notFound(w, "goal not found")
		return
	}
	sendOK(w, map[string]any{"status": "deleted"})
}
