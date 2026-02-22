package api

import (
	"encoding/json"
	"net/http"

	"github.com/Atrix21/signalstream/internal/auth"
	"github.com/Atrix21/signalstream/internal/config"
	"github.com/Atrix21/signalstream/internal/database"
	"github.com/Atrix21/signalstream/internal/metrics"
	"github.com/Atrix21/signalstream/internal/sse"
)

type Server struct {
	cfg          config.AppConfig
	db           *database.DB
	tokenService *auth.TokenService
	encryptor    *auth.Encryptor
	broker       *sse.Broker
}

func NewServer(cfg config.AppConfig, db *database.DB, broker *sse.Broker) (*Server, error) {
	encryptor, err := auth.NewEncryptor(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:          cfg,
		db:           db,
		tokenService: auth.NewTokenService(cfg.JWTSecret),
		encryptor:    encryptor,
		broker:       broker,
	}, nil
}

func (s *Server) Routes() http.Handler {
	mux := http.NewServeMux()

	// Auth routes
	mux.HandleFunc("POST /api/v1/auth/register", s.handleRegister)
	mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)
	mux.HandleFunc("GET /api/v1/auth/me", s.requireAuth(s.handleGetMe))

	// API Key routes
	mux.HandleFunc("GET /api/v1/keys", s.requireAuth(s.handleListAPIKeys))
	mux.HandleFunc("POST /api/v1/keys", s.requireAuth(s.handleUpdateAPIKey))
	mux.HandleFunc("DELETE /api/v1/keys", s.requireAuth(s.handleDeleteAPIKey))

	// Strategy routes
	mux.HandleFunc("GET /api/v1/strategies", s.requireAuth(s.handleListStrategies))
	mux.HandleFunc("POST /api/v1/strategies", s.requireAuth(s.handleCreateStrategy))
	mux.HandleFunc("DELETE /api/v1/strategies", s.requireAuth(s.handleDeleteStrategy))
	mux.HandleFunc("PATCH /api/v1/strategies/toggle", s.requireAuth(s.handleToggleStrategy))

	// Alert routes
	mux.HandleFunc("GET /api/v1/alerts", s.requireAuth(s.handleListAlerts))
	mux.HandleFunc("PATCH /api/v1/alerts/read", s.requireAuth(s.handleMarkAlertRead))
	mux.HandleFunc("GET /api/v1/alerts/stream", s.requireAuth(s.handleAlertStream))

	// Observability
	mux.HandleFunc("GET /api/v1/metrics", metrics.Global.Handler())

	return s.corsMiddleware(mux)
}

func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", s.cfg.FrontendURL)
		w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(data); err != nil {
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func (s *Server) errorResponse(w http.ResponseWriter, status int, message string) {
	s.jsonResponse(w, status, map[string]string{"error": message})
}
