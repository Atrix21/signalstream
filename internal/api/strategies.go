package api

import (
	"encoding/json"
	"net/http"

	"github.com/Atrix21/signalstream/internal/database"
	"github.com/google/uuid"
)

type strategyRequest struct {
	Name                string   `json:"name"`
	Description         string   `json:"description"`
	Query               string   `json:"query"`
	Source              []string `json:"source"`
	Tickers             []string `json:"tickers"`
	SimilarityThreshold float64  `json:"similarity_threshold"`
	IsActive            bool     `json:"is_active"`
}

func (s *Server) handleListStrategies(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	strategies, err := s.db.GetUserStrategies(r.Context(), claims.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}

	s.jsonResponse(w, http.StatusOK, strategies)
}

func (s *Server) handleCreateStrategy(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req strategyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Name == "" || req.Query == "" {
		s.errorResponse(w, http.StatusBadRequest, "name and query are required")
		return
	}

	strategy := &database.Strategy{
		UserID:              claims.UserID,
		Name:                req.Name,
		Description:         req.Description,
		Query:               req.Query,
		Source:              req.Source,
		Tickers:             req.Tickers,
		SimilarityThreshold: req.SimilarityThreshold,
		IsActive:            req.IsActive,
	}

	if err := s.db.CreateStrategy(r.Context(), strategy); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to create strategy")
		return
	}

	s.jsonResponse(w, http.StatusCreated, strategy)
}

func (s *Server) handleDeleteStrategy(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid strategy id")
		return
	}

	if err := s.db.DeleteStrategy(r.Context(), id, claims.UserID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to delete strategy")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) handleToggleStrategy(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	idStr := r.URL.Query().Get("id")
	if idStr == "" {
		s.errorResponse(w, http.StatusBadRequest, "id is required")
		return
	}

	id, err := uuid.Parse(idStr)
	if err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid strategy id")
		return
	}

	strategy, err := s.db.GetStrategy(r.Context(), id, claims.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}
	if strategy == nil {
		s.errorResponse(w, http.StatusNotFound, "strategy not found")
		return
	}

	strategy.IsActive = !strategy.IsActive
	if err := s.db.UpdateStrategy(r.Context(), strategy); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to update strategy")
		return
	}

	s.jsonResponse(w, http.StatusOK, strategy)
}
