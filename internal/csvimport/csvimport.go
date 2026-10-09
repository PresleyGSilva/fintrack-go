package csvimport

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

// Row é uma linha válida do CSV, valores em cents (int64, sem float).
type Row struct {
	Date        string // YYYY-MM-DD
	Description string
	AmountCents int64  // sempre > 0
	Kind        string // income | expense (sinal do amount define)
	Category    string
	Account     string
	Hash        string // sha256 dedup, estável por conteúdo
}

// ParseDecimalToCents converte "1234.56", "-12,34", "100" etc. em cents,
// SEM float: aritmética inteira sobre a parte fracionária (máx. 2 casas).
func ParseDecimalToCents(s string) (int64, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, errors.New("empty amount")
	}
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	} else if strings.HasPrefix(s, "+") {
		s = s[1:]
	}
	// Remove separador de milhar comum e normaliza vírgula decimal.
	if strings.Contains(s, ",") && strings.Contains(s, ".") {
		// Formato 1,234.56 → remove vírgulas.
		s = strings.ReplaceAll(s, ",", "")
	} else if strings.Contains(s, ",") {
		// Formato 1234,56 → vírgula vira ponto.
		s = strings.ReplaceAll(s, ",", ".")
	}
	intPart := s
	fracPart := ""
	if i := strings.IndexByte(s, '.'); i >= 0 {
		intPart = s[:i]
		fracPart = s[i+1:]
	}
	if intPart == "" {
		intPart = "0"
	}
	if len(fracPart) > 2 {
		return 0, fmt.Errorf("too many decimals: %q", s)
	}
	for len(fracPart) < 2 {
		fracPart += "0"
	}
	for _, c := range intPart + fracPart {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid amount: %q", s)
		}
	}
	whole, err := strconv.ParseInt(intPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %q", s)
	}
	frac, err := strconv.ParseInt(fracPart, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount: %q", s)
	}
	const maxCents = int64(9_000_000_000_000_000) // ~9e15, longe do overflow int64
	if whole > maxCents/100 {
		return 0, fmt.Errorf("amount too large: %q", s)
	}
	cents := whole*100 + frac
	if neg {
		cents = -cents
	}
	return cents, nil
}

// HashRow calcula o sha256 de dedup: org + data + descrição + cents + conta + categoria.
func HashRow(orgID string, r Row) string {
	canon := strings.Join([]string{
		orgID, r.Date, r.Description,
		strconv.FormatInt(r.AmountCents, 10), r.Kind, r.Category, r.Account,
	}, "|")
	sum := sha256.Sum256([]byte(canon))
	return hex.EncodeToString(sum[:])
}

// Parse lê o CSV (cabeçalho: date,description,amount_cents|amount,category,account)
// e devolve as linhas válidas. Erros de linha são acumulados; o chamador decide.
func Parse(orgID string, in io.Reader) (rows []Row, lineErrs []string) {
	rd := csv.NewReader(in)
	rd.TrimLeadingSpace = true
	rd.FieldsPerRecord = -1
	header, err := rd.Read()
	if err != nil {
		return nil, []string{"empty csv: " + err.Error()}
	}
	idx := map[string]int{}
	for i, h := range header {
		idx[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(rec []string, name string) string {
		i, ok := idx[name]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}
	hasCents := true
	if _, ok := idx["amount_cents"]; !ok {
		hasCents = false
		if _, ok := idx["amount"]; !ok {
			return nil, []string{"missing column: amount_cents (or amount)"}
		}
	}
	for _, need := range []string{"date", "description", "account"} {
		if _, ok := idx[need]; !ok {
			return nil, []string{"missing column: " + need}
		}
	}
	line := 1
	for {
		rec, err := rd.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		line++
		if err != nil {
			lineErrs = append(lineErrs, fmt.Sprintf("line %d: %v", line, err))
			continue
		}
		dateRaw := get(rec, "date")
		var t time.Time
		for _, layout := range []string{"2006-01-02", "02/01/2006", "01/02/2006"} {
			if tt, e := time.Parse(layout, dateRaw); e == nil {
				t = tt
				break
			}
		}
		if t.IsZero() {
			lineErrs = append(lineErrs, fmt.Sprintf("line %d: invalid date %q", line, dateRaw))
			continue
		}
		var cents int64
		if hasCents {
			// Coluna amount_cents: inteiro em cents; aceita decimal ("10.50") como fallback.
			v := strings.TrimSpace(get(rec, "amount_cents"))
			if iv, perr := strconv.ParseInt(v, 10, 64); perr == nil {
				cents = iv
			} else if cents, err = ParseDecimalToCents(v); err != nil {
				lineErrs = append(lineErrs, fmt.Sprintf("line %d: bad amount_cents %q", line, v))
				continue
			}
		} else {
			cents, err = ParseDecimalToCents(get(rec, "amount"))
			if err != nil {
				lineErrs = append(lineErrs, fmt.Sprintf("line %d: bad amount: %v", line, err))
				continue
			}
		}
		if cents == 0 {
			lineErrs = append(lineErrs, fmt.Sprintf("line %d: amount is zero", line))
			continue
		}
		kind := "income"
		if cents < 0 {
			kind = "expense"
			cents = -cents
		}
		r := Row{
			Date:        t.Format("2006-01-02"),
			Description: get(rec, "description"),
			AmountCents: cents,
			Kind:        kind,
			Category:    get(rec, "category"),
			Account:     get(rec, "account"),
		}
		if r.Account == "" {
			lineErrs = append(lineErrs, fmt.Sprintf("line %d: missing account", line))
			continue
		}
		r.Hash = HashRow(orgID, r)
		rows = append(rows, r)
	}
	return rows, lineErrs
}
