package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/PresleyGSilva/fintrack-go/internal/db"
)

// testServer sobe os handlers contra Postgres real.
// Pula se DATABASE_URL não estiver definido.
func testServer(t *testing.T) (*Handler, string, string) {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("no db: %v", err)
	}
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	// Identidades únicas por execução: nunca colidem com restos de runs anteriores.
	suffix := strings.ReplaceAll(t.Name(), "/", "_") + "-" + time.Now().Format("150405.000000000")
	orgID := "test-" + suffix
	userID := "user-" + suffix
	_, err = pool.Exec(ctx, `INSERT INTO users(id,name,email,password_hash) VALUES($1,$2,$3,'x')`,
		userID, suffix, suffix+"@example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO orgs(id,name) VALUES($1,$2)`, orgID, suffix); err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO memberships(user_id,org_id,role) VALUES($1,$2,'owner')`,
		userID, orgID); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM orgs WHERE id=$1`, orgID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, userID)
		pool.Close()
	})
	h := &Handler{DB: pool, JWTSecret: testSecret, Limiter: NewLoginLimiter(1000, time.Minute)}
	return h, orgID, userID
}

func doReq(t *testing.T, h *Handler, token, org, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rdr)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	if org != "" {
		req.Header.Set("X-Org-ID", org)
	}
	rec := httptest.NewRecorder()
	h.NewRouter().ServeHTTP(rec, req)
	return rec
}

func decodeData[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	var env struct {
		Data  T `json:"data"`
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatalf("bad json: %v body=%s", err, rec.Body.String())
	}
	if rec.Code >= 400 {
		t.Fatalf("status %d code=%s msg=%s", rec.Code, env.Error.Code, env.Error.Message)
	}
	return env.Data
}

func TestIntegrationAccountsTransactionsSummary(t *testing.T) {
	h, org, uid := testServer(t)
	tok := mustToken(t, uid)

	acc := decodeData[struct {
		ID string `json:"id"`
	}](t, doReq(t, h, tok, org, "POST", "/api/v1/accounts", map[string]any{
		"name": "Corrente", "type": "checking", "balance_cents": 0,
	}))
	accID := acc.ID

	cat := decodeData[struct {
		ID string `json:"id"`
	}](t, doReq(t, h, tok, org, "POST", "/api/v1/categories", map[string]any{
		"name": "Salário", "kind": "income",
	}))
	catID := cat.ID

	doReq(t, h, tok, org, "POST", "/api/v1/transactions", map[string]any{
		"account_id": accID, "category_id": catID,
		"kind": "income", "amount_cents": 500000, "date": "2026-09-01", "note": "sal",
	})
	rec := doReq(t, h, tok, org, "GET", "/api/v1/reports/summary?from=2026-09-01&to=2026-09-30", nil)
	sum := decodeData[struct {
		IncomeCents  int64 `json:"income_cents"`
		ExpenseCents int64 `json:"expense_cents"`
		BalanceCents int64 `json:"balance_cents"`
	}](t, rec)
	if sum.IncomeCents != 500000 || sum.BalanceCents != 500000 || sum.ExpenseCents != 0 {
		t.Fatalf("summary: %+v", sum)
	}
	// Saldo da conta acompanhou.
	rec = doReq(t, h, tok, org, "GET", "/api/v1/accounts/"+accID, nil)
	got := decodeData[struct {
		BalanceCents int64 `json:"balance_cents"`
	}](t, rec)
	if got.BalanceCents != 500000 {
		t.Fatalf("balance: %+v", got)
	}
}

func TestIntegrationTransferAtomic(t *testing.T) {
	h, org, uid := testServer(t)
	tok := mustToken(t, uid)

	mk := func(name string) string {
		a := decodeData[struct {
			ID string `json:"id"`
		}](t, doReq(t, h, tok, org, "POST", "/api/v1/accounts", map[string]any{
			"name": name, "type": "checking",
		}))
		return a.ID
	}
	from, to := mk("A"), mk("B")
	doReq(t, h, tok, org, "POST", "/api/v1/transactions", map[string]any{
		"account_id": from, "kind": "income", "amount_cents": 100000,
		"date": "2026-09-01", "note": "seed",
	})
	rec := doReq(t, h, tok, org, "POST", "/api/v1/transfers", map[string]any{
		"from_account_id": from, "to_account_id": to, "amount_cents": 40000, "memo": "t",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("transfer: %d %s", rec.Code, rec.Body.String())
	}
	bal := func(id string) int64 {
		r := doReq(t, h, tok, org, "GET", "/api/v1/accounts/"+id, nil)
		return decodeData[struct {
			BalanceCents int64 `json:"balance_cents"`
		}](t, r).BalanceCents
	}
	if bal(from) != 60000 || bal(to) != 40000 {
		t.Fatalf("balances from=%v to=%v", bal(from), bal(to))
	}
	// Sem saldo → 422 INSUFFICIENT_FUNDS.
	rec = doReq(t, h, tok, org, "POST", "/api/v1/transfers", map[string]any{
		"from_account_id": from, "to_account_id": to, "amount_cents": 999999,
	})
	if rec.Code != 422 || !strings.Contains(rec.Body.String(), "INSUFFICIENT_FUNDS") {
		t.Fatalf("expected 422 INSUFFICIENT_FUNDS: %d %s", rec.Code, rec.Body.String())
	}
}

func TestIntegrationImportDedup(t *testing.T) {
	h, org, uid := testServer(t)
	tok := mustToken(t, uid)

	csvBody := "date,description,amount_cents,category,account\n" +
		"2026-09-01,Item A,10000,Cat A,Conta A\n" +
		"2026-09-02,Item B,-5000,Cat B,Conta A\n"
	upload := func() *httptest.ResponseRecorder {
		var buf bytes.Buffer
		mw := multipart.NewWriter(&buf)
		fw, err := mw.CreateFormFile("file", "in.csv")
		if err != nil {
			t.Fatal(err)
		}
		_, _ = fw.Write([]byte(csvBody))
		_ = mw.Close()
		req := httptest.NewRequest("POST", "/api/v1/import/csv", &buf)
		req.Header.Set("Content-Type", mw.FormDataContentType())
		req.Header.Set("Authorization", "Bearer "+tok)
		req.Header.Set("X-Org-ID", org)
		rec := httptest.NewRecorder()
		h.NewRouter().ServeHTTP(rec, req)
		return rec
	}
	first := decodeData[struct {
		Inserted int64 `json:"inserted"`
		Skipped  int64 `json:"skipped"`
	}](t, upload())
	if first.Inserted != 2 {
		t.Fatalf("first import: %+v", first)
	}
	second := decodeData[struct {
		Inserted int64 `json:"inserted"`
		Skipped  int64 `json:"skipped"`
	}](t, upload())
	if second.Inserted != 0 || second.Skipped != 2 {
		t.Fatalf("second import should dedup: %+v", second)
	}
}

func TestIntegrationAuthRoundtrip(t *testing.T) {
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	pool, err := db.Connect(ctx, url)
	if err != nil {
		t.Skipf("no db: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	h := &Handler{DB: pool, JWTSecret: testSecret, Limiter: NewLoginLimiter(1000, time.Minute)}
	email := "roundtrip-" + strings.ReplaceAll(t.Name(), "/", "_") + "-" + time.Now().Format("150405.000000000") + "@example.com"
	rec := doReq(t, h, "", "", "POST", "/api/v1/auth/register", map[string]any{
		"name": "RT", "email": email, "password": "supersecret1",
	})
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: %d %s", rec.Code, rec.Body.String())
	}
	var env struct {
		Data struct {
			Access string `json:"access_token"`
			Org    struct {
				ID string `json:"id"`
			} `json:"org"`
			User struct {
				ID string `json:"id"`
			} `json:"user"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM orgs WHERE id=$1`, env.Data.Org.ID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id=$1`, env.Data.User.ID)
	})
	cookies := rec.Result().Cookies()
	found := false
	for _, c := range cookies {
		if c.Name == "refresh_token" && c.HttpOnly && c.Value != "" {
			found = true
		}
	}
	if !found {
		t.Fatal("missing httpOnly refresh cookie")
	}
	// /me com o access token.
	rec = doReq(t, h, env.Data.Access, env.Data.Org.ID, "GET", "/api/v1/me", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("me: %d %s", rec.Code, rec.Body.String())
	}
	// Refresh via cookie.
	req := httptest.NewRequest("POST", "/api/v1/auth/refresh", nil)
	for _, c := range cookies {
		req.AddCookie(c)
	}
	rec2 := httptest.NewRecorder()
	h.NewRouter().ServeHTTP(rec2, req)
	if rec2.Code != http.StatusOK {
		t.Fatalf("refresh: %d %s", rec2.Code, rec2.Body.String())
	}
}
