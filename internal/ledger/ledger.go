package ledger

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"sort"
)

// Erros de domínio (mapeados p/ códigos da API no handler).
var (
	ErrSameAccount       = errors.New("from and to accounts must differ")
	ErrInvalidAmount     = errors.New("amount_cents must be > 0")
	ErrInsufficientFunds = errors.New("insufficient funds")
	ErrNotFound          = errors.New("account not found")
	ErrCrossOrg          = errors.New("accounts belong to different orgs")
)

// DBTX é o mínimo usado pela transferência (pool ou tx).
type DBTX interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// ApplyTransfer é a regra pura do razão: valida e calcula os novos saldos.
// Testável sem banco; a soma from+to é preservada.
func ApplyTransfer(fromBal, toBal, amount int64) (newFrom, newTo int64, err error) {
	if amount <= 0 {
		return 0, 0, ErrInvalidAmount
	}
	if fromBal < amount {
		return 0, 0, ErrInsufficientFunds
	}
	return fromBal - amount, toBal + amount, nil
}

// Summarize soma cents por kind (income - expense = balance). Inteiros, sem float.
func Summarize(kinds []string, amounts []int64) (income, expense, balance int64) {
	for i, k := range kinds {
		switch k {
		case "income":
			income += amounts[i]
		case "expense":
			expense += amounts[i]
		}
	}
	return income, expense, income - expense
}

type accountRow struct {
	id      string
	orgID   string
	balance int64
}

// Transfer move amount cents entre duas contas da mesma org, ATOMICAMENTE:
// trava as duas linhas em ordem crescente de id (evita deadlock), valida,
// lança 2 transactions kind='transfer' (expense na origem, income no destino)
// com transfer_id comum e atualiza os saldos. Devolve o transfer_id.
func Transfer(ctx context.Context, pool *pgxpool.Pool, orgID, fromID, toID string, amount int64, memo string) (string, error) {
	if fromID == toID {
		return "", ErrSameAccount
	}
	if amount <= 0 {
		return "", ErrInvalidAmount
	}
	// Ordem canônica de travamento: menor id primeiro.
	first, second := fromID, toID
	if second < first {
		first, second = second, first
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	get := func(id string) (accountRow, error) {
		var a accountRow
		err := tx.QueryRow(ctx,
			`SELECT id, org_id, balance_cents FROM accounts WHERE id=$1 FOR UPDATE`, id).
			Scan(&a.id, &a.orgID, &a.balance)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return a, fmt.Errorf("%w: %s", ErrNotFound, id)
			}
			return a, err
		}
		return a, nil
	}
	a1, err := get(first)
	if err != nil {
		return "", err
	}
	a2, err := get(second)
	if err != nil {
		return "", err
	}
	byID := map[string]accountRow{a1.id: a1, a2.id: a2}
	from, to := byID[fromID], byID[toID]
	if from.orgID != orgID || to.orgID != orgID || from.orgID != to.orgID {
		return "", ErrCrossOrg
	}
	newFrom, newTo, err := ApplyTransfer(from.balance, to.balance, amount)
	if err != nil {
		return "", err
	}
	transferID := uuid.NewString()
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET balance_cents=$1 WHERE id=$2`, newFrom, fromID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE accounts SET balance_cents=$1 WHERE id=$2`, newTo, toID); err != nil {
		return "", err
	}
	memoOut, memoIn := "transfer out", "transfer in"
	if memo != "" {
		memoOut = memo
		memoIn = memo
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO transactions(id, org_id, account_id, kind, amount_cents, date, note, transfer_id)
		 VALUES($1,$2,$3,'expense',$4,CURRENT_DATE,$5,$6)`,
		uuid.NewString(), orgID, fromID, amount, memoOut, transferID); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO transactions(id, org_id, account_id, kind, amount_cents, date, note, transfer_id)
		 VALUES($1,$2,$3,'income',$4,CURRENT_DATE,$5,$6)`,
		uuid.NewString(), orgID, toID, amount, memoIn, transferID); err != nil {
		return "", err
	}
	if err := tx.Commit(ctx); err != nil {
		return "", err
	}
	return transferID, nil
}

// LockOrder devolve os ids na ordem em que devem ser travados (teste/documentação).
func LockOrder(a, b string) (string, string) {
	s := []string{a, b}
	sort.Strings(s)
	return s[0], s[1]
}
