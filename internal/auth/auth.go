package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTTL  = 15 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour
	// Custo 11: dentro da faixa 10-12 exigida, rápido o bastante p/ testes.
	BcryptCost = 11
)

var ErrInvalidToken = errors.New("invalid token")

type Claims struct {
	UserID string `json:"sub"`
	jwt.RegisteredClaims
}

// HashPassword gera hash bcrypt. Nunca exponha o hash em respostas.
func HashPassword(pw string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(pw), BcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// IssueAccess emite JWT HS256 de 15min com sub=userID.
func IssueAccess(secret, userID string) (string, error) {
	if secret == "" {
		return "", errors.New("empty jwt secret")
	}
	now := time.Now()
	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(AccessTTL)),
		},
	})
	return tok.SignedString([]byte(secret))
}

// ParseAccess valida o Bearer e devolve o userID.
func ParseAccess(secret, token string) (string, error) {
	tok, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil {
		return "", ErrInvalidToken
	}
	c, ok := tok.Claims.(*Claims)
	if !ok || !tok.Valid || c.UserID == "" {
		return "", ErrInvalidToken
	}
	return c.UserID, nil
}

// NewRefreshToken gera token opaco (32 bytes aleatórios, hex) para cookie httpOnly.
// Só o sha256 hex é persistido no banco.
func NewRefreshToken() (token, hash string, err error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", "", err
	}
	token = hex.EncodeToString(b[:])
	sum := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(sum[:]), nil
}

// HashRefresh deriva o hash armazenado a partir do token apresentado.
func HashRefresh(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// MatchRefresh compara em tempo constante.
func MatchRefresh(storedHash, presented string) bool {
	a, err1 := hex.DecodeString(storedHash)
	b := sha256.Sum256([]byte(presented))
	bHex, err2 := hex.DecodeString(hex.EncodeToString(b[:]))
	if err1 != nil || err2 != nil {
		return false
	}
	return subtle.ConstantTimeCompare(a, bHex) == 1
}
