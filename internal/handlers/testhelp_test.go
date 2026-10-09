package handlers

import (
	"testing"

	"github.com/PresleyGSilva/fintrack-go/internal/auth"
)

func mustToken(t *testing.T, uid string) string {
	t.Helper()
	tok, err := auth.IssueAccess(testSecret, uid)
	if err != nil {
		t.Fatal(err)
	}
	return tok
}
