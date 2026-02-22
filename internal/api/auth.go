package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/Atrix21/signalstream/internal/auth"
	"github.com/Atrix21/signalstream/internal/database"
)

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type authResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Email == "" || req.Password == "" {
		s.errorResponse(w, http.StatusBadRequest, "email and password are required")
		return
	}

	hash, err := auth.HashPassword(req.Password)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	user, err := s.db.CreateUser(r.Context(), req.Email, hash)
	if err != nil {
		if errors.Is(err, database.ErrDuplicateEmail) {
			s.errorResponse(w, http.StatusConflict, "email already registered")
			return
		}
		s.errorResponse(w, http.StatusInternalServerError, "failed to create user")
		return
	}

	token, err := s.tokenService.GenerateToken(user.ID, user.Email)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	s.jsonResponse(w, http.StatusCreated, authResponse{
		Token: token,
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{
			ID:    user.ID.String(),
			Email: user.Email,
		},
	})
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	user, err := s.db.GetUserByEmail(r.Context(), req.Email)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}

	if user == nil || !auth.CheckPasswordHash(req.Password, user.PasswordHash) {
		s.errorResponse(w, http.StatusUnauthorized, "invalid credentials")
		return
	}

	token, err := s.tokenService.GenerateToken(user.ID, user.Email)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to generate token")
		return
	}

	s.jsonResponse(w, http.StatusOK, authResponse{
		Token: token,
		User: struct {
			ID    string `json:"id"`
			Email string `json:"email"`
		}{
			ID:    user.ID.String(),
			Email: user.Email,
		},
	})
}

func (s *Server) handleGetMe(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	user, err := s.db.GetUserByID(r.Context(), claims.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}
	if user == nil {
		s.errorResponse(w, http.StatusNotFound, "user not found")
		return
	}

	s.jsonResponse(w, http.StatusOK, struct {
		ID        string `json:"id"`
		Email     string `json:"email"`
		CreatedAt string `json:"created_at"`
	}{
		ID:        user.ID.String(),
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Format(time.RFC3339),
	})
}
