package csvimport

import (
	"strings"
	"testing"
)

func TestParseDecimalToCents(t *testing.T) {
	cases := map[string]int64{
		"100":      10000,
		"100.00":   10000,
		"12.34":    1234,
		"12.3":     1230,
		"12":       1200,
		"0.99":     99,
		"-12.34":   -1234,
		"+5.00":    500,
		"1,234.56": 123456,
		"1234,56":  123456,
		"  7.50  ": 750,
		"0.01":     1,
		"90000.00": 9000000,
	}
	for in, want := range cases {
		got, err := ParseDecimalToCents(in)
		if err != nil {
			t.Errorf("%q: unexpected err %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("%q: got %d want %d", in, got, want)
		}
	}
	for _, bad := range []string{"", "abc", "12.345", "12..34", "--5", "1.2.3"} {
		if _, err := ParseDecimalToCents(bad); err == nil {
			t.Errorf("%q: expected error", bad)
		}
	}
}

const sampleCSV = `date,description,amount,category,account
2026-09-01,Salário,8500.00,Salário,Conta Corrente
2026-09-02,Supermercado,-423.50,Mercado,Conta Corrente
2026-09-03,Aluguel,-1800.00,Moradia,Conta Corrente
`

func TestParseAndDedup(t *testing.T) {
	r1, errs := Parse("org-1", strings.NewReader(sampleCSV))
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if len(r1) != 3 {
		t.Fatalf("got %d rows", len(r1))
	}
	if r1[0].Kind != "income" || r1[0].AmountCents != 850000 {
		t.Fatalf("row0: %+v", r1[0])
	}
	if r1[1].Kind != "expense" || r1[1].AmountCents != 42350 {
		t.Fatalf("row1: %+v", r1[1])
	}
	// Segunda importação do mesmo CSV: hashes idênticos → dedup total.
	r2, _ := Parse("org-1", strings.NewReader(sampleCSV))
	if len(r2) != len(r1) {
		t.Fatalf("reparse length mismatch")
	}
	seen := map[string]bool{}
	for _, r := range r1 {
		seen[r.Hash] = true
	}
	dups := 0
	for _, r := range r2 {
		if seen[r.Hash] {
			dups++
		}
	}
	if dups != len(r1) {
		t.Fatalf("expected all %d rows deduped, got %d", len(r1), dups)
	}
	// Hash é sensível à org: mesma linha em outra org não colide.
	r3, _ := Parse("org-2", strings.NewReader(sampleCSV))
	for i := range r1 {
		if r1[i].Hash == r3[i].Hash {
			t.Fatalf("cross-org hash collision on row %d", i)
		}
	}
}

func TestParseCentsColumn(t *testing.T) {
	in := "date,description,amount_cents,category,account\n2026-09-01,X,1500,Y,Z\n"
	rows, errs := Parse("o", strings.NewReader(in))
	if len(errs) != 0 || len(rows) != 1 {
		t.Fatalf("rows=%v errs=%v", rows, errs)
	}
	if rows[0].AmountCents != 1500 || rows[0].Kind != "income" {
		t.Fatalf("row: %+v", rows[0])
	}
}
