package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/PresleyGSilva/fintrack-go/internal/auth"
)

const testSecret = "test-secret-at-least-32-chars-long-xxxx"

// TestAuthMiddlewareRoundtrip exercita o Bearer via httptest sem banco:
// emite access, passa pelo middleware e chega ao handler com o user no contexto.
func TestAuthMiddlewareRoundtrip(t *testing.T) {
	h := &Handler{JWTSecret: testSecret, Limiter: NewLoginLimiter(10, 1000000000)}
	token, err := auth.IssueAccess(testSecret, "user-abc")
	if err != nil {
		t.Fatal(err)
	}
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sendOK(w, map[string]any{"uid": userID(r)})
	})
	srv := httptest.NewServer(h.withAuth(inner))
	defer srv.Close()

	// Sem token → 401 UNAUTHORIZED.
	res, err := http.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}

	// Com Bearer válido → 200 e uid ecoado.
	req, _ := http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	var body struct {
		Data map[string]string `json:"data"`
	}
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body.Data["uid"] != "user-abc" {
		t.Fatalf("uid not propagated: %v", body.Data)
	}

	// Token adulterado → 401 com envelope de erro.
	req, _ = http.NewRequest(http.MethodGet, srv.URL, nil)
	req.Header.Set("Authorization", "Bearer "+token+"x")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", res.StatusCode)
	}
	var errBody struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.NewDecoder(res.Body).Decode(&errBody); err != nil {
		t.Fatal(err)
	}
	if errBody.Error.Code != "UNAUTHORIZED" {
		t.Fatalf("code: %q", errBody.Error.Code)
	}
}
