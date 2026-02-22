package api

import (
	"net/http"
	"strconv"

	"github.com/google/uuid"
)

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil {
			limit = val
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil {
			offset = val
		}
	}

	alerts, err := s.db.GetUserAlerts(r.Context(), claims.UserID, limit, offset)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}

	s.jsonResponse(w, http.StatusOK, alerts)
}

func (s *Server) handleMarkAlertRead(w http.ResponseWriter, r *http.Request) {
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
		s.errorResponse(w, http.StatusBadRequest, "invalid alert id")
		return
	}

	if err := s.db.MarkAlertRead(r.Context(), id, claims.UserID); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "failed to mark alert as read")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAlertStream(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	s.broker.ServeHTTP(w, r, claims.UserID.String())
}
