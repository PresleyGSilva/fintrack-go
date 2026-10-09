package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/PresleyGSilva/fintrack-go/internal/models"
)

// Summary agrega income/expense do período + total por categoria.
// Transferências (kind='transfer') não entram nas somas.
func (h *Handler) Summary(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	args := []any{org}
	where := "WHERE t.org_id=$1 AND t.kind IN ('income','expense')"
	if v := q.Get("from"); v != "" {
		if !validDate(v) {
			validation(w, "from must be YYYY-MM-DD")
			return
		}
		args = append(args, v)
		where += " AND t.date>=$" + itoa(len(args))
	}
	if v := q.Get("to"); v != "" {
		if !validDate(v) {
			validation(w, "to must be YYYY-MM-DD")
			return
		}
		args = append(args, v)
		where += " AND t.date<=$" + itoa(len(args))
	}
	var income, expense int64
	err := h.DB.QueryRow(r.Context(),
		`SELECT COALESCE(SUM(CASE WHEN t.kind='income' THEN t.amount_cents END),0),
		        COALESCE(SUM(CASE WHEN t.kind='expense' THEN t.amount_cents END),0)
		 FROM transactions t `+where, args...).Scan(&income, &expense)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot build summary")
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT t.category_id, c.name, t.kind, COALESCE(SUM(t.amount_cents),0)
		 FROM transactions t LEFT JOIN categories c ON c.id=t.category_id `+where+`
		 GROUP BY t.category_id, c.name, t.kind ORDER BY t.kind, c.name`, args...)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot build summary")
		return
	}
	defer rows.Close()
	byCat := []models.CategoryTotal{}
	for rows.Next() {
		var ct models.CategoryTotal
		var cid, cname nullText
		if err := rows.Scan(&cid.t, &cname.t, &ct.Kind, &ct.TotalCents); err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan summary")
			return
		}
		ct.CategoryID = cid.ptr()
		ct.CategoryName = cname.ptr()
		byCat = append(byCat, ct)
	}
	sendOK(w, models.Summary{
		IncomeCents: income, ExpenseCents: expense,
		BalanceCents: income - expense, ByCategory: byCat,
	})
}

// Monthly devolve 12 pontos (income/expense) do ano pedido (?year=YYYY).
func (h *Handler) Monthly(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	year := time.Now().Year()
	if v := r.URL.Query().Get("year"); v != "" {
		y, err := strconv.Atoi(v)
		if err != nil || y < 1970 || y > 2100 {
			validation(w, "year must be YYYY")
			return
		}
		year = y
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT EXTRACT(MONTH FROM date)::INT AS m, kind, COALESCE(SUM(amount_cents),0)
		 FROM transactions
		 WHERE org_id=$1 AND kind IN ('income','expense')
		   AND date >= make_date($2,1,1) AND date < make_date($2+1,1,1)
		 GROUP BY m, kind`, org, year)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot build monthly report")
		return
	}
	defer rows.Close()
	points := make([]models.MonthlyPoint, 12)
	for i := range points {
		points[i] = models.MonthlyPoint{Month: fmt.Sprintf("%04d-%02d", year, i+1)}
	}
	for rows.Next() {
		var m int
		var kind string
		var total int64
		if err := rows.Scan(&m, &kind, &total); err != nil {
			apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot scan monthly report")
			return
		}
		if m < 1 || m > 12 {
			continue
		}
		if kind == "income" {
			points[m-1].IncomeCents = total
		} else {
			points[m-1].ExpenseCents = total
		}
	}
	sendOK(w, points)
}
