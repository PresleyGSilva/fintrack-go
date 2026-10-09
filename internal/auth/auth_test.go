package auth

import (
	"strings"
	"testing"
)

func TestPasswordRoundtrip(t *testing.T) {
	h, err := HashPassword("correct-horse-123")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(h, "correct-horse-123") {
		t.Fatal("hash leaks password")
	}
	if !CheckPassword(h, "correct-horse-123") {
		t.Fatal("valid password rejected")
	}
	if CheckPassword(h, "wrong") {
		t.Fatal("invalid password accepted")
	}
}

func TestAccessRoundtrip(t *testing.T) {
	const secret = "test-secret-at-least-32-chars-long-xxxx"
	tok, err := IssueAccess(secret, "user-1")
	if err != nil {
		t.Fatal(err)
	}
	uid, err := ParseAccess(secret, tok)
	if err != nil || uid != "user-1" {
		t.Fatalf("got %q %v", uid, err)
	}
	if _, err := ParseAccess("other-secret", tok); err == nil {
		t.Fatal("wrong secret accepted")
	}
	if _, err := ParseAccess(secret, tok+"tampered"); err == nil {
		t.Fatal("tampered token accepted")
	}
	if _, err := IssueAccess("", "user-1"); err == nil {
		t.Fatal("empty secret accepted")
	}
}

func TestRefreshRoundtrip(t *testing.T) {
	raw, stored, err := NewRefreshToken()
	if err != nil {
		t.Fatal(err)
	}
	if raw == stored || strings.Contains(stored, raw) {
		t.Fatal("stored value must be a hash, not the token")
	}
	if HashRefresh(raw) != stored {
		t.Fatal("hash mismatch")
	}
	if !MatchRefresh(stored, raw) {
		t.Fatal("valid refresh rejected")
	}
	if MatchRefresh(stored, "other") {
		t.Fatal("invalid refresh accepted")
	}
}
