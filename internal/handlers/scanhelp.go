package handlers

import (
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// nullText adapta pgtype.Text (coluna NULLable) para *string.
type nullText struct {
	t pgtype.Text
}

func (n nullText) ptr() *string {
	if !n.t.Valid {
		return nil
	}
	s := n.t.String
	return &s
}

func fmtDate(t time.Time) string { return t.Format("2006-01-02") }

func itoa(i int) string { return strconv.Itoa(i) }
