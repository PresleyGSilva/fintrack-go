package handlers

import (
	"errors"
	"net/http"
	"strings"

	"github.com/PresleyGSilva/fintrack-go/internal/ledger"
)

func (h *Handler) CreateTransfer(w http.ResponseWriter, r *http.Request) {
	org, ok := h.orgID(w, r)
	if !ok {
		return
	}
	var in struct {
		FromAccountID string `json:"from_account_id"`
		ToAccountID   string `json:"to_account_id"`
		AmountCents   int64  `json:"amount_cents"`
		Memo          string `json:"memo"`
	}
	if !decode(w, r, &in) {
		return
	}
	if strings.TrimSpace(in.FromAccountID) == "" || strings.TrimSpace(in.ToAccountID) == "" {
		validation(w, "from_account_id and to_account_id are required")
		return
	}
	transferID, err := ledger.Transfer(r.Context(), h.DB, org,
		in.FromAccountID, in.ToAccountID, in.AmountCents, strings.TrimSpace(in.Memo))
	switch {
	case err == nil:
		created(w, map[string]any{"transfer_id": transferID})
	case errors.Is(err, ledger.ErrSameAccount),
		errors.Is(err, ledger.ErrInvalidAmount):
		validation(w, err.Error())
	case errors.Is(err, ledger.ErrInsufficientFunds):
		apiErr(w, http.StatusUnprocessableEntity, "INSUFFICIENT_FUNDS", err.Error())
	case errors.Is(err, ledger.ErrNotFound),
		errors.Is(err, ledger.ErrCrossOrg):
		validation(w, err.Error())
	default:
		apiErr(w, http.StatusInternalServerError, "CONFLICT", "cannot complete transfer")
	}
}
