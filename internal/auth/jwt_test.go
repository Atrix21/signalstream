package auth

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

func TestTokenService_GenerateAndValidate(t *testing.T) {
	svc := NewTokenService("test-secret-key-minimum-16chars")
	userID := uuid.New()
	email := "user@example.com"

	token, err := svc.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("generate token failed: %v", err)
	}

	if token == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := svc.ValidateToken(token)
	if err != nil {
		t.Fatalf("validate token failed: %v", err)
	}

	if claims.UserID != userID {
		t.Fatalf("user ID mismatch: got %v, want %v", claims.UserID, userID)
	}
	if claims.Email != email {
		t.Fatalf("email mismatch: got %q, want %q", claims.Email, email)
	}
	if claims.Issuer != "signalstream" {
		t.Fatalf("issuer mismatch: got %q, want %q", claims.Issuer, "signalstream")
	}
}

func TestTokenService_WrongSecret(t *testing.T) {
	svc1 := NewTokenService("secret-one-is-sixteen")
	svc2 := NewTokenService("secret-two-is-sixteen")

	userID := uuid.New()
	token, _ := svc1.GenerateToken(userID, "user@example.com")

	_, err := svc2.ValidateToken(token)
	if err == nil {
		t.Fatal("validating with wrong secret should fail")
	}
}

func TestTokenService_ExpiredToken(t *testing.T) {
	svc := &TokenService{secret: []byte("test-secret-key-minimum-16chars")}
	userID := uuid.New()

	claims := Claims{
		UserID: userID,
		Email:  "expired@example.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)), // Expired 1 hour ago
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-25 * time.Hour)),
			Issuer:    "signalstream",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, _ := token.SignedString(svc.secret)

	_, err := svc.ValidateToken(tokenString)
	if err == nil {
		t.Fatal("expired token should fail validation")
	}
}

func TestTokenService_MalformedToken(t *testing.T) {
	svc := NewTokenService("test-secret-key-minimum-16chars")

	tests := []struct {
		name  string
		token string
	}{
		{"empty", ""},
		{"garbage", "not-a-jwt-token"},
		{"partial", "eyJhbGciOiJIUzI1NiJ9."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateToken(tt.token)
			if err == nil {
				t.Fatal("malformed token should fail validation")
			}
		})
	}
}

func TestTokenService_ClaimsContainExpectedFields(t *testing.T) {
	svc := NewTokenService("test-secret-key-minimum-16chars")
	userID := uuid.New()
	email := "full-claims@example.com"

	token, _ := svc.GenerateToken(userID, email)
	claims, _ := svc.ValidateToken(token)

	if claims.ExpiresAt == nil {
		t.Fatal("token should have expiry")
	}
	if claims.IssuedAt == nil {
		t.Fatal("token should have issued-at")
	}

	// Token should expire ~24h from now.
	expiresIn := time.Until(claims.ExpiresAt.Time)
	if expiresIn < 23*time.Hour || expiresIn > 25*time.Hour {
		t.Fatalf("unexpected expiry duration: %v", expiresIn)
	}
}
