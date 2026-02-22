package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/Atrix21/signalstream/internal/auth"
)

func newTestServer(t *testing.T) *Server {
	t.Helper()
	return &Server{
		tokenService: auth.NewTokenService("test-secret-key-minimum-16chars"),
	}
}

func TestRequireAuth_ValidToken(t *testing.T) {
	srv := newTestServer(t)
	userID := uuid.New()
	email := "user@test.com"

	token, err := srv.tokenService.GenerateToken(userID, email)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		claims := GetUserClaims(r)
		if claims == nil {
			t.Fatal("claims should be present in context")
		}
		if claims.UserID != userID {
			t.Errorf("user ID mismatch: got %v, want %v", claims.UserID, userID)
		}
		if claims.Email != email {
			t.Errorf("email mismatch: got %q, want %q", claims.Email, email)
		}
		w.WriteHeader(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

func TestRequireAuth_MissingHeader(t *testing.T) {
	srv := newTestServer(t)

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called without auth header")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}

	var body map[string]string
	json.NewDecoder(rec.Body).Decode(&body)
	if body["error"] != "missing authorization header" {
		t.Errorf("unexpected error: %q", body["error"])
	}
}

func TestRequireAuth_InvalidFormat(t *testing.T) {
	srv := newTestServer(t)

	tests := []struct {
		name   string
		header string
	}{
		{"no bearer prefix", "just-a-token"},
		{"basic auth", "Basic dXNlcjpwYXNz"},
		{"empty bearer", "Bearer "},
		{"extra parts", "Bearer token extra"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
				t.Fatal("handler should not be called with invalid format")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			req.Header.Set("Authorization", tt.header)
			rec := httptest.NewRecorder()

			handler(rec, req)

			if rec.Code != http.StatusUnauthorized {
				t.Fatalf("expected 401, got %d", rec.Code)
			}
		})
	}
}

func TestRequireAuth_InvalidToken(t *testing.T) {
	srv := newTestServer(t)

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with invalid token")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestRequireAuth_WrongSecret(t *testing.T) {
	srv := newTestServer(t)

	// Generate token with a different secret.
	otherService := auth.NewTokenService("different-secret-key-16")
	token, _ := otherService.GenerateToken(uuid.New(), "user@test.com")

	handler := srv.requireAuth(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("handler should not be called with token signed by wrong secret")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()

	handler(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestGetUserClaims_NoClaims(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	claims := GetUserClaims(req)
	if claims != nil {
		t.Fatal("expected nil claims when none set in context")
	}
}
