package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/PresleyGSilva/fintrack-go/internal/models"
)

var (
	accountTypes = map[string]bool{"checking": true, "savings": true, "cash": true, "credit": true, "investment": true, "other": true}
	kinds        = map[string]bool{"income": true, "expense": true}
	frequencies  = map[string]bool{"daily": true, "weekly": true, "monthly": true}
)

func validDate(s string) bool {
	_, err := time.Parse("2006-01-02", s)
	return err == nil
}

func nullStr(t pgtype.Text) *string {
	if !t.Valid {
		return nil
	}
	s := t.String
	return &s
}

func scanAccount(scan func(...any) error) (models.Account, error) {
	var a models.Account
	err := scan(&a.ID, &a.OrgID, &a.Name, &a.Type, &a.BalanceCents, &a.CreatedAt, &a.UpdatedAt)
	return a, err
}

// ---------- accounts ----------

func (h *Handler) ListAccounts(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT id,org_id,name,type,balance_cents,created_at,updated_at FROM accounts WHERE org_id=$1 ORDER BY created_at`, org)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot list accounts")
		return
	}
	defer rows.Close()
	out := []models.Account{}
	for rows.Next() {
		a, err := scanAccount(rows.Scan)
		if err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan account")
			return
		}
		out = append(out, a)
	}
	sendOK(w, out)
}

func (h *Handler) CreateAccount(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		Name         string `json:"name"`
		Type         string `json:"type"`
		BalanceCents int64  `json:"balance_cents"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		validation(w, "name is required (max 120)")
		return
	}
	if in.Type == "" {
		in.Type = "checking"
	}
	if !accountTypes[in.Type] {
		validation(w, "invalid type")
		return
	}
	if in.BalanceCents < 0 {
		validation(w, "balance_cents must be >= 0")
		return
	}
	a := models.Account{ID: uuid.NewString(), OrgID: org, Name: in.Name, Type: in.Type, BalanceCents: in.BalanceCents}
	var ts struct {
		c time.Time
		u time.Time
	}
	err := h.DB.QueryRow(r.Context(),
		`INSERT INTO accounts(id,org_id,name,type,balance_cents) VALUES($1,$2,$3,$4,$5)
		 RETURNING created_at, updated_at`,
		a.ID, a.OrgID, a.Name, a.Type, a.BalanceCents).Scan(&ts.c, &ts.u)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create account")
		return
	}
	a.CreatedAt, a.UpdatedAt = ts.c, ts.u
	created(w, a)
}

func (h *Handler) GetAccount(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	a, found := h.getAccount(r, org, chi.URLParam(r, "id"))
	if !found {
		notFound(w, "account not found")
		return
	}
	sendOK(w, a)
}

func (h *Handler) getAccount(r *http.Request, org, id string) (models.Account, bool) {
	a, err := scanAccount(h.DB.QueryRow(r.Context(),
		`SELECT id,org_id,name,type,balance_cents,created_at,updated_at FROM accounts WHERE id=$1 AND org_id=$2`,
		id, org).Scan)
	if err != nil {
		return models.Account{}, false
	}
	return a, true
}

func (h *Handler) UpdateAccount(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, found := h.getAccount(r, org, id); !found {
		notFound(w, "account not found")
		return
	}
	var in struct {
		Name string `json:"name"`
		Type string `json:"type"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		validation(w, "name is required (max 120)")
		return
	}
	if in.Type == "" {
		in.Type = "checking"
	}
	if !accountTypes[in.Type] {
		validation(w, "invalid type")
		return
	}
	a, err := scanAccount(h.DB.QueryRow(r.Context(),
		`UPDATE accounts SET name=$1, type=$2 WHERE id=$3 AND org_id=$4
		 RETURNING id,org_id,name,type,balance_cents,created_at,updated_at`,
		in.Name, in.Type, id, org).Scan)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update account")
		return
	}
	sendOK(w, a)
}

func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, found := h.getAccount(r, org, id); !found {
		notFound(w, "account not found")
		return
	}
	var n int
	_ = h.DB.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM transactions WHERE account_id=$1`, id).Scan(&n)
	if n > 0 {
		conflict(w, "account has transactions and cannot be deleted")
		return
	}
	_ = h.DB.QueryRow(r.Context(),
		`SELECT COUNT(*) FROM recurrences WHERE account_id=$1`, id).Scan(&n)
	if n > 0 {
		conflict(w, "account has recurrences and cannot be deleted")
		return
	}
	if _, err := h.DB.Exec(r.Context(),
		`DELETE FROM accounts WHERE id=$1 AND org_id=$2`, id, org); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot delete account")
		return
	}
	sendOK(w, map[string]any{"status": "deleted"})
}

// ---------- categories ----------

func scanCategory(scan func(...any) error) (models.Category, error) {
	var c models.Category
	err := scan(&c.ID, &c.OrgID, &c.Name, &c.Kind, &c.Color, &c.CreatedAt, &c.UpdatedAt)
	return c, err
}

func (h *Handler) ListCategories(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	q := `SELECT id,org_id,name,kind,color,created_at,updated_at FROM categories WHERE org_id=$1`
	args := []any{org}
	if k := r.URL.Query().Get("kind"); k != "" {
		if !kinds[k] {
			validation(w, "invalid kind")
			return
		}
		q += ` AND kind=$2`
		args = append(args, k)
	}
	q += ` ORDER BY name`
	rows, err := h.DB.Query(r.Context(), q, args...)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot list categories")
		return
	}
	defer rows.Close()
	out := []models.Category{}
	for rows.Next() {
		c, err := scanCategory(rows.Scan)
		if err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan category")
			return
		}
		out = append(out, c)
	}
	sendOK(w, out)
}

func (h *Handler) CreateCategory(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		Name  string `json:"name"`
		Kind  string `json:"kind"`
		Color string `json:"color"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		validation(w, "name is required (max 120)")
		return
	}
	if !kinds[in.Kind] {
		validation(w, "kind must be income|expense")
		return
	}
	if in.Color == "" {
		in.Color = "#6b7280"
	}
	c := models.Category{ID: uuid.NewString(), OrgID: org, Name: in.Name, Kind: in.Kind, Color: in.Color}
	err := h.DB.QueryRow(r.Context(),
		`INSERT INTO categories(id,org_id,name,kind,color) VALUES($1,$2,$3,$4,$5)
		 RETURNING created_at, updated_at`,
		c.ID, c.OrgID, c.Name, c.Kind, c.Color).Scan(&c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			conflict(w, "category already exists")
			return
		}
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create category")
		return
	}
	created(w, c)
}

func (h *Handler) getCategory(r *http.Request, org, id string) (models.Category, bool) {
	c, err := scanCategory(h.DB.QueryRow(r.Context(),
		`SELECT id,org_id,name,kind,color,created_at,updated_at FROM categories WHERE id=$1 AND org_id=$2`,
		id, org).Scan)
	if err != nil {
		return models.Category{}, false
	}
	return c, true
}

func (h *Handler) GetCategory(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	c, found := h.getCategory(r, org, chi.URLParam(r, "id"))
	if !found {
		notFound(w, "category not found")
		return
	}
	sendOK(w, c)
}

func (h *Handler) UpdateCategory(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, found := h.getCategory(r, org, id); !found {
		notFound(w, "category not found")
		return
	}
	var in struct {
		Name  string `json:"name"`
		Color string `json:"color"`
	}
	if !decode(w, r, &in) {
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" || len(in.Name) > 120 {
		validation(w, "name is required (max 120)")
		return
	}
	if in.Color == "" {
		in.Color = "#6b7280"
	}
	c, err := scanCategory(h.DB.QueryRow(r.Context(),
		`UPDATE categories SET name=$1, color=$2 WHERE id=$3 AND org_id=$4
		 RETURNING id,org_id,name,kind,color,created_at,updated_at`,
		in.Name, in.Color, id, org).Scan)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			conflict(w, "category already exists")
			return
		}
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update category")
		return
	}
	sendOK(w, c)
}

func (h *Handler) DeleteCategory(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	if _, found := h.getCategory(r, org, id); !found {
		notFound(w, "category not found")
		return
	}
	if _, err := h.DB.Exec(r.Context(),
		`DELETE FROM categories WHERE id=$1 AND org_id=$2`, id, org); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot delete category")
		return
	}
	sendOK(w, map[string]any{"status": "deleted"})
}

// ---------- transactions ----------

func scanTx(scan func(...any) error) (models.Transaction, error) {
	var t models.Transaction
	var cat, ih, tid pgtype.Text
	var dt time.Time
	err := scan(&t.ID, &t.OrgID, &t.AccountID, &cat, &t.Kind, &t.AmountCents, &dt, &t.Note, &ih, &tid, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	t.CategoryID = nullStr(cat)
	t.ImportHash = nullStr(ih)
	t.TransferID = nullStr(tid)
	t.Date = dt.Format("2006-01-02")
	return t, nil
}

const txCols = `id,org_id,account_id,category_id,kind,amount_cents,date,note,import_hash,transfer_id,created_at,updated_at`

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	args := []any{org}
	where := "WHERE org_id=$1"
	if v := q.Get("kind"); v != "" {
		if !kinds[v] && v != "transfer" {
			validation(w, "invalid kind")
			return
		}
		args = append(args, v)
		where += " AND kind=$" + strconv.Itoa(len(args))
	}
	if v := q.Get("account_id"); v != "" {
		args = append(args, v)
		where += " AND account_id=$" + strconv.Itoa(len(args))
	}
	if v := q.Get("category_id"); v != "" {
		args = append(args, v)
		where += " AND category_id=$" + strconv.Itoa(len(args))
	}
	if v := q.Get("from"); v != "" {
		if !validDate(v) {
			validation(w, "from must be YYYY-MM-DD")
			return
		}
		args = append(args, v)
		where += " AND date>=$" + strconv.Itoa(len(args))
	}
	if v := q.Get("to"); v != "" {
		if !validDate(v) {
			validation(w, "to must be YYYY-MM-DD")
			return
		}
		args = append(args, v)
		where += " AND date<=$" + strconv.Itoa(len(args))
	}
	limit := 100
	if v := q.Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			validation(w, "limit must be 1..500")
			return
		}
		limit = n
	}
	offset := 0
	if v := q.Get("offset"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 0 {
			validation(w, "offset must be >= 0")
			return
		}
		offset = n
	}
	args = append(args, limit, offset)
	query := `SELECT ` + txCols + ` FROM transactions ` + where +
		` ORDER BY date DESC, created_at DESC LIMIT $` + strconv.Itoa(len(args)-1) +
		` OFFSET $` + strconv.Itoa(len(args))
	rows, err := h.DB.Query(r.Context(), query, args...)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot list transactions")
		return
	}
	defer rows.Close()
	out := []models.Transaction{}
	for rows.Next() {
		t, err := scanTx(rows.Scan)
		if err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan transaction")
			return
		}
		out = append(out, t)
	}
	sendOK(w, out)
}

func (h *Handler) CreateTransaction(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		AccountID   string  `json:"account_id"`
		CategoryID  *string `json:"category_id"`
		Kind        string  `json:"kind"`
		AmountCents int64   `json:"amount_cents"`
		Date        string  `json:"date"`
		Note        string  `json:"note"`
		ImportHash  *string `json:"import_hash"`
	}
	if !decode(w, r, &in) {
		return
	}
	if !kinds[in.Kind] {
		validation(w, "kind must be income|expense")
		return
	}
	if in.AmountCents <= 0 {
		validation(w, "amount_cents must be > 0")
		return
	}
	if !validDate(in.Date) {
		validation(w, "date must be YYYY-MM-DD")
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
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	id := uuid.NewString()
	if _, err := tx.Exec(r.Context(),
		`INSERT INTO transactions(id,org_id,account_id,category_id,kind,amount_cents,date,note,import_hash)
		 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		id, org, in.AccountID, in.CategoryID, in.Kind, in.AmountCents, in.Date, in.Note, in.ImportHash); err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unique") {
			conflict(w, "duplicate transaction (import_hash)")
			return
		}
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot create transaction")
		return
	}
	delta := in.AmountCents
	if in.Kind == "expense" {
		delta = -delta
	}
	if _, err := tx.Exec(r.Context(),
		`UPDATE accounts SET balance_cents=balance_cents+$1 WHERE id=$2 AND org_id=$3`,
		delta, in.AccountID, org); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update balance")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot commit transaction")
		return
	}
	t, err := scanTx(h.DB.QueryRow(r.Context(),
		`SELECT `+txCols+` FROM transactions WHERE id=$1`, id).Scan)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot read transaction")
		return
	}
	created(w, t)
}

func (h *Handler) GetTransaction(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	t, err := scanTx(h.DB.QueryRow(r.Context(),
		`SELECT `+txCols+` FROM transactions WHERE id=$1 AND org_id=$2`,
		chi.URLParam(r, "id"), org).Scan)
	if err != nil {
		notFound(w, "transaction not found")
		return
	}
	sendOK(w, t)
}

func (h *Handler) UpdateTransaction(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	cur, err := scanTx(h.DB.QueryRow(r.Context(),
		`SELECT `+txCols+` FROM transactions WHERE id=$1 AND org_id=$2`, id, org).Scan)
	if err != nil {
		notFound(w, "transaction not found")
		return
	}
	if cur.TransferID != nil {
		conflict(w, "transfer entries must be managed via transfers")
		return
	}
	var in struct {
		CategoryID *string `json:"category_id"`
		Date       string  `json:"date"`
		Note       string  `json:"note"`
	}
	if !decode(w, r, &in) {
		return
	}
	date := cur.Date
	if in.Date != "" {
		if !validDate(in.Date) {
			validation(w, "date must be YYYY-MM-DD")
			return
		}
		date = in.Date
	}
	cat := cur.CategoryID
	if in.CategoryID != nil {
		if *in.CategoryID == "" {
			cat = nil
		} else {
			if _, found := h.getCategory(r, org, *in.CategoryID); !found {
				validation(w, "category not found")
				return
			}
			cat = in.CategoryID
		}
	}
	note := cur.Note
	if in.Note != "" || r.URL.Query().Has("note") {
		note = in.Note
	}
	t, err := scanTx(h.DB.QueryRow(r.Context(),
		`UPDATE transactions SET category_id=$1, date=$2, note=$3 WHERE id=$4 AND org_id=$5 RETURNING `+txCols,
		cat, date, note, id, org).Scan)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update transaction")
		return
	}
	sendOK(w, t)
}

func (h *Handler) DeleteTransaction(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	id := chi.URLParam(r, "id")
	cur, err := scanTx(h.DB.QueryRow(r.Context(),
		`SELECT `+txCols+` FROM transactions WHERE id=$1 AND org_id=$2`, id, org).Scan)
	if err != nil {
		notFound(w, "transaction not found")
		return
	}
	if cur.TransferID != nil {
		conflict(w, "transfer entries must be managed via transfers")
		return
	}
	tx, err := h.DB.Begin(r.Context())
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot start transaction")
		return
	}
	defer func() { _ = tx.Rollback(r.Context()) }()
	delta := -cur.AmountCents
	if cur.Kind == "expense" {
		delta = cur.AmountCents
	}
	if _, err := tx.Exec(r.Context(),
		`UPDATE accounts SET balance_cents=balance_cents+$1 WHERE id=$2 AND org_id=$3`,
		delta, cur.AccountID, org); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot update balance")
		return
	}
	if _, err := tx.Exec(r.Context(),
		`DELETE FROM transactions WHERE id=$1 AND org_id=$2`, id, org); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot delete transaction")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot commit delete")
		return
	}
	sendOK(w, map[string]any{"status": "deleted"})
}
