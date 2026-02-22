package api

import (
	"context"
	"net/http"
	"strings"

	"github.com/Atrix21/signalstream/internal/auth"
)

type contextKey string

const UserClaimsKey contextKey = "user_claims"

func (s *Server) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			s.errorResponse(w, http.StatusUnauthorized, "missing authorization header")
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			s.errorResponse(w, http.StatusUnauthorized, "invalid authorization header format")
			return
		}

		tokenString := parts[1]
		claims, err := s.tokenService.ValidateToken(tokenString)
		if err != nil {
			s.errorResponse(w, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
		next(w, r.WithContext(ctx))
	}
}

func GetUserClaims(r *http.Request) *auth.Claims {
	if claims, ok := r.Context().Value(UserClaimsKey).(*auth.Claims); ok {
		return claims
	}
	return nil
}
