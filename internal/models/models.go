package models

import "time"

// Dinheiro SEMPRE int64 em cents. Nenhum float neste backend.

type User struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Org struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type Account struct {
	ID           string    `json:"id"`
	OrgID        string    `json:"org_id"`
	Name         string    `json:"name"`
	Type         string    `json:"type"`
	BalanceCents int64     `json:"balance_cents"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Category struct {
	ID        string    `json:"id"`
	OrgID     string    `json:"org_id"`
	Name      string    `json:"name"`
	Kind      string    `json:"kind"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Transaction struct {
	ID          string    `json:"id"`
	OrgID       string    `json:"org_id"`
	AccountID   string    `json:"account_id"`
	CategoryID  *string   `json:"category_id"`
	Kind        string    `json:"kind"`
	AmountCents int64     `json:"amount_cents"`
	Date        string    `json:"date"`
	Note        string    `json:"note"`
	ImportHash  *string   `json:"import_hash,omitempty"`
	TransferID  *string   `json:"transfer_id,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Recurrence struct {
	ID          string  `json:"id"`
	OrgID       string  `json:"org_id"`
	AccountID   string  `json:"account_id"`
	CategoryID  *string `json:"category_id"`
	Kind        string  `json:"kind"`
	AmountCents int64   `json:"amount_cents"`
	Frequency   string  `json:"frequency"`
	NextRun     string  `json:"next_run"`
	Active      bool    `json:"active"`
	Note        string  `json:"note"`
}

type Budget struct {
	ID         string `json:"id"`
	OrgID      string `json:"org_id"`
	CategoryID string `json:"category_id"`
	Month      string `json:"month"`
	LimitCents int64  `json:"limit_cents"`
}

type Goal struct {
	ID           string `json:"id"`
	OrgID        string `json:"org_id"`
	Name         string `json:"name"`
	TargetCents  int64  `json:"target_cents"`
	CurrentCents int64  `json:"current_cents"`
}

type CategoryTotal struct {
	CategoryID   *string `json:"category_id"`
	CategoryName *string `json:"category_name"`
	Kind         string  `json:"kind"`
	TotalCents   int64   `json:"total_cents"`
}

type Summary struct {
	IncomeCents  int64           `json:"income_cents"`
	ExpenseCents int64           `json:"expense_cents"`
	BalanceCents int64           `json:"balance_cents"`
	ByCategory   []CategoryTotal `json:"by_category"`
}

type MonthlyPoint struct {
	Month        string `json:"month"`
	IncomeCents  int64  `json:"income_cents"`
	ExpenseCents int64  `json:"expense_cents"`
}
