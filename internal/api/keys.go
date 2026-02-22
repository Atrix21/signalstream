package api

import (
	"encoding/json"
	"net/http"
)

type apiKeyRequest struct {
	Provider string `json:"provider"`
	Key      string `json:"key"`
}

type apiKeyResponse struct {
	Provider string `json:"provider"`
	HasKey   bool   `json:"has_key"`
}

func (s *Server) handleListAPIKeys(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	keys, err := s.db.GetAPIKeys(r.Context(), claims.UserID)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}

	response := []apiKeyResponse{
		{Provider: "polygon", HasKey: false},
		{Provider: "openai", HasKey: false},
	}

	for _, k := range keys {
		for i := range response {
			if response[i].Provider == k.Provider {
				response[i].HasKey = true
			}
		}
	}

	s.jsonResponse(w, http.StatusOK, response)
}

func (s *Server) handleUpdateAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req apiKeyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.errorResponse(w, http.StatusBadRequest, "invalid request body")
		return
	}

	if req.Provider != "polygon" && req.Provider != "openai" {
		s.errorResponse(w, http.StatusBadRequest, "invalid provider (must be 'polygon' or 'openai')")
		return
	}
	if req.Key == "" {
		s.errorResponse(w, http.StatusBadRequest, "key is required")
		return
	}

	// Encrypt the key
	encryptedKey, err := s.encryptor.Encrypt(req.Key)
	if err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "encryption failed")
		return
	}

	if err := s.db.AddAPIKey(r.Context(), claims.UserID, req.Provider, encryptedKey); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteAPIKey(w http.ResponseWriter, r *http.Request) {
	claims := GetUserClaims(r)
	if claims == nil {
		s.errorResponse(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	provider := r.URL.Query().Get("provider")
	if provider == "" {
		s.errorResponse(w, http.StatusBadRequest, "provider is required")
		return
	}

	if err := s.db.DeleteAPIKey(r.Context(), claims.UserID, provider); err != nil {
		s.errorResponse(w, http.StatusInternalServerError, "database error")
		return
	}

	s.jsonResponse(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// Internal helper to decrypt keys
func (s *Server) getUserAPIKey(r *http.Request, provider string) (string, error) {
	claims := GetUserClaims(r)
	if claims == nil {
		return "", http.ErrNoCookie // Just a placeholder error
	}
	return "", nil
}
