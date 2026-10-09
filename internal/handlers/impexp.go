package handlers

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/PresleyGSilva/fintrack-go/internal/csvimport"
)

// ImportCSV recebe multipart (campo "file") com colunas
// date,description,amount_cents|amount,category,account.
// Dedup por sha256 (import_hash UNIQUE por org): a 2ª importação ignora repetidos.
func (h *Handler) ImportCSV(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		validation(w, "multipart file is required (field \"file\")")
		return
	}
	f, _, err := r.FormFile("file")
	if err != nil {
		validation(w, "multipart file is required (field \"file\")")
		return
	}
	defer f.Close()
	rows, lineErrs := csvimport.Parse(org, f)
	var inserted, skipped int64
	for _, row := range rows {
		accID, okAcc := h.resolveAccount(r, org, row.Account)
		if !okAcc {
			lineErrs = append(lineErrs, "account not resolvable: "+row.Account)
			skipped++
			continue
		}
		var catID *string
		if strings.TrimSpace(row.Category) != "" {
			c, okCat := h.resolveCategory(r, org, row.Category, row.Kind)
			if !okCat {
				lineErrs = append(lineErrs, "category not resolvable: "+row.Category)
				skipped++
				continue
			}
			catID = &c
		}
		tag, err := h.DB.Exec(r.Context(),
			`INSERT INTO transactions(id,org_id,account_id,category_id,kind,amount_cents,date,note,import_hash)
			 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			uuid.NewString(), org, accID, catID, row.Kind, row.AmountCents,
			row.Date, row.Description, row.Hash)
		if err != nil {
			lineErrs = append(lineErrs, "insert: "+err.Error())
			skipped++
			continue
		}
		if tag.RowsAffected() == 0 {
			skipped++
			continue
		}
		delta := row.AmountCents
		if row.Kind == "expense" {
			delta = -delta
		}
		if _, err := h.DB.Exec(r.Context(),
			`UPDATE accounts SET balance_cents=balance_cents+$1 WHERE id=$2 AND org_id=$3`,
			delta, accID, org); err != nil {
			lineErrs = append(lineErrs, "balance: "+err.Error())
			continue
		}
		inserted++
	}
	sendOK(w, map[string]any{
		"inserted": inserted,
		"skipped":  skipped,
		"errors":   lineErrs,
	})
}

func (h *Handler) resolveAccount(r *http.Request, org, name string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	var id string
	err := h.DB.QueryRow(r.Context(),
		`SELECT id FROM accounts WHERE org_id=$1 AND lower(name)=lower($2)`, org, name).Scan(&id)
	if err == nil {
		return id, true
	}
	id = uuid.NewString()
	if _, err := h.DB.Exec(r.Context(),
		`INSERT INTO accounts(id,org_id,name,type) VALUES($1,$2,$3,'other')`, id, org, name); err != nil {
		return "", false
	}
	return id, true
}

func (h *Handler) resolveCategory(r *http.Request, org, name, kind string) (string, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return "", false
	}
	var id string
	err := h.DB.QueryRow(r.Context(),
		`SELECT id FROM categories WHERE org_id=$1 AND lower(name)=lower($2)`, org, name).Scan(&id)
	if err == nil {
		return id, true
	}
	id = uuid.NewString()
	if _, err := h.DB.Exec(r.Context(),
		`INSERT INTO categories(id,org_id,name,kind) VALUES($1,$2,$3,$4)`, id, org, name, kind); err != nil {
		// Corrida ou nome duplicado com outro kind: tenta ler de novo.
		if err2 := h.DB.QueryRow(r.Context(),
			`SELECT id FROM categories WHERE org_id=$1 AND lower(name)=lower($2)`, org, name).Scan(&id); err2 != nil {
			return "", false
		}
	}
	return id, true
}

// ExportCSV devolve todas as transações da org em CSV com as mesmas colunas
// do import (reimportar o arquivo gera os mesmos hashes → dedup atua).
func (h *Handler) ExportCSV(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	rows, err := h.DB.Query(r.Context(),
		`SELECT t.date, t.note, t.amount_cents, t.kind, COALESCE(c.name,''), a.name
		 FROM transactions t
		 JOIN accounts a ON a.id=t.account_id
		 LEFT JOIN categories c ON c.id=t.category_id
		 WHERE t.org_id=$1 ORDER BY t.date, t.created_at`, org)
	if err != nil {
		apiErr(w, http.StatusInternalServerError, "NOT_FOUND", "cannot export")
		return
	}
	defer rows.Close()
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="transactions.csv"`)
	cw := csv.NewWriter(w)
	_ = cw.Write([]string{"date", "description", "amount_cents", "category", "account", "kind", "note"})
	for rows.Next() {
		var dt time.Time
		var note string
		var cents int64
		var kind, cat, acc string
		if err := rows.Scan(&dt, &note, &cents, &kind, &cat, &acc); err != nil {
			continue
		}
		signed := cents
		if kind == "expense" {
			signed = -cents
		}
		_ = cw.Write([]string{
			dt.Format("2006-01-02"), note,
			strconv.FormatInt(signed, 10), cat, acc, kind, note,
		})
	}
	cw.Flush()
}
