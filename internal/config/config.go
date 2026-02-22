package config

import (
	"fmt"
	"log"
	"sync"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

// AppConfig holds all configuration for the application, loaded from environment variables.
type AppConfig struct {
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`
	PolygonAPIKey string `env:"POLYGON_API_KEY"`
	OpenAIAPIKey  string `env:"OPENAI_API_KEY"`

	QdrantHost string `env:"QDRANT_HOST" envDefault:"127.0.0.1"`
	QdrantPort int    `env:"QDRANT_PORT" envDefault:"6334"`

	DatabaseURL   string `env:"DATABASE_URL" envDefault:"postgres://signalstream:dev_password@localhost:5432/signalstream"`
	JWTSecret     string `env:"JWT_SECRET,required"`
	EncryptionKey string `env:"ENCRYPTION_KEY,required"`
	ServerPort    string `env:"SERVER_PORT" envDefault:"8080"`
	FrontendURL   string `env:"FRONTEND_URL" envDefault:"http://localhost:3000"`
}

// Validate checks that the configuration is sane.
func (c AppConfig) Validate() error {
	if len(c.JWTSecret) < 16 {
		return fmt.Errorf("JWT_SECRET must be at least 16 characters")
	}
	if len(c.EncryptionKey) != 32 {
		return fmt.Errorf("ENCRYPTION_KEY must be exactly 32 bytes (got %d)", len(c.EncryptionKey))
	}
	if c.QdrantPort < 1 || c.QdrantPort > 65535 {
		return fmt.Errorf("QDRANT_PORT must be between 1 and 65535")
	}
	return nil
}

var (
	once   sync.Once
	config AppConfig
)

// Get returns the singleton config instance, loading it from the environment on first call.
func Get() AppConfig {
	once.Do(func() {
		if err := godotenv.Load(); err != nil {
			log.Println("No .env file found, using environment variables")
		}

		if err := env.Parse(&config); err != nil {
			log.Fatalf("failed to parse configuration: %+v", err)
		}

		if err := config.Validate(); err != nil {
			log.Fatalf("invalid configuration: %v", err)
		}
	})
	return config
}
