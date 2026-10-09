package seed

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Run popula a org com dados fictícios de demonstração:
// 2 contas, categorias, ~30 transações, 1 orçamento e 1 meta.
// Idempotente por org (só roda se a org não tiver contas).
func Run(ctx context.Context, pool *pgxpool.Pool, orgID string) error {
	var n int
	if err := pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM accounts WHERE org_id=$1`, orgID).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	exec := func(sql string, args ...any) error {
		_, err := tx.Exec(ctx, sql, args...)
		return err
	}
	qid := func() string { return uuid.NewString() }

	checking, savings := qid(), qid()
	if err := exec(`INSERT INTO accounts(id,org_id,name,type,balance_cents) VALUES($1,$2,'Conta Corrente','checking',0)`,
		checking, orgID); err != nil {
		return err
	}
	if err := exec(`INSERT INTO accounts(id,org_id,name,type,balance_cents) VALUES($1,$2,'Poupança','savings',0)`,
		savings, orgID); err != nil {
		return err
	}
	type cat struct {
		id, name, kind string
	}
	cats := []cat{
		{qid(), "Salário", "income"}, {qid(), "Freelance", "income"},
		{qid(), "Mercado", "expense"}, {qid(), "Transporte", "expense"},
		{qid(), "Lazer", "expense"}, {qid(), "Moradia", "expense"},
	}
	for _, c := range cats {
		if err := exec(`INSERT INTO categories(id,org_id,name,kind) VALUES($1,$2,$3,$4)`,
			c.id, orgID, c.name, c.kind); err != nil {
			return err
		}
	}
	byName := map[string]cat{}
	for _, c := range cats {
		byName[c.name] = c
	}
	// ~30 transações nos últimos 60 dias, valores em cents (sem float).
	type txn struct {
		daysAgo int
		cat     string
		kind    string
		cents   int64
		note    string
		acc     string
	}
	plan := []txn{
		{58, "Salário", "income", 850000, "Salário mensal", "checking"},
		{55, "Moradia", "expense", 180000, "Aluguel", "checking"},
		{52, "Mercado", "expense", 42350, "Supermercado", "checking"},
		{50, "Transporte", "expense", 22000, "Combustível", "checking"},
		{47, "Freelance", "income", 120000, "Projeto extra", "checking"},
		{45, "Mercado", "expense", 38900, "Feira", "checking"},
		{43, "Lazer", "expense", 15900, "Cinema", "checking"},
		{40, "Transporte", "expense", 12500, "Aplicativo", "checking"},
		{38, "Mercado", "expense", 45200, "Supermercado", "checking"},
		{35, "Lazer", "expense", 8900, "Streaming", "checking"},
		{33, "Transporte", "expense", 9800, "Metrô", "checking"},
		{30, "Salário", "income", 850000, "Salário mensal", "checking"},
		{29, "Moradia", "expense", 180000, "Aluguel", "checking"},
		{27, "Mercado", "expense", 51200, "Supermercado", "checking"},
		{25, "Transporte", "expense", 22000, "Combustível", "checking"},
		{23, "Freelance", "income", 75000, "Consultoria", "savings"},
		{21, "Mercado", "expense", 33400, "Feira", "checking"},
		{19, "Lazer", "expense", 24900, "Restaurante", "checking"},
		{17, "Transporte", "expense", 11000, "Aplicativo", "checking"},
		{15, "Mercado", "expense", 47800, "Supermercado", "checking"},
		{12, "Lazer", "expense", 8900, "Streaming", "checking"},
		{10, "Transporte", "expense", 22000, "Combustível", "checking"},
		{8, "Mercado", "expense", 29600, "Feira", "checking"},
		{6, "Freelance", "income", 150000, "Projeto extra", "checking"},
		{5, "Lazer", "expense", 19900, "Show", "checking"},
		{4, "Transporte", "expense", 8500, "Metrô", "checking"},
		{3, "Mercado", "expense", 36700, "Supermercado", "checking"},
		{2, "Moradia", "expense", 35000, "Condomínio", "checking"},
		{1, "Transporte", "expense", 9200, "Aplicativo", "checking"},
		{0, "Freelance", "income", 60000, "Bico", "savings"},
	}
	accID := map[string]string{"checking": checking, "savings": savings}
	bal := map[string]int64{"checking": 0, "savings": 0}
	today := time.Now()
	for _, p := range plan {
		c := byName[p.cat]
		date := today.AddDate(0, 0, -p.daysAgo).Format("2006-01-02")
		if err := exec(`INSERT INTO transactions(id,org_id,account_id,category_id,kind,amount_cents,date,note)
			VALUES($1,$2,$3,$4,$5,$6,$7,$8)`,
			qid(), orgID, accID[p.acc], c.id, p.kind, p.cents, date, p.note); err != nil {
			return err
		}
		if p.kind == "income" {
			bal[p.acc] += p.cents
		} else {
			bal[p.acc] -= p.cents
		}
	}
	for name, b := range bal {
		if err := exec(`UPDATE accounts SET balance_cents=$1 WHERE id=$2`, b, accID[name]); err != nil {
			return err
		}
	}
	month := time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.UTC).Format("2006-01-02")
	if err := exec(`INSERT INTO budgets(id,org_id,category_id,month,limit_cents) VALUES($1,$2,$3,$4,$5)`,
		qid(), orgID, byName["Mercado"].id, month, 200000); err != nil {
		return err
	}
	if err := exec(`INSERT INTO goals(id,org_id,name,target_cents,current_cents) VALUES($1,$2,$3,$4,$5)`,
		qid(), orgID, "Reserva de emergência", 1000000, 235000); err != nil {
		return err
	}
	if err := tx.Commit(ctx); err != nil {
		return err
	}
	fmt.Println("seed: demo data inserted")
	return nil
}
