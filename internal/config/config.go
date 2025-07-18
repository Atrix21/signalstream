package config

import (
	"log"
	"sync"

	"github.com/caarlos0/env/v6"
	"github.com/joho/godotenv"
)

// AppConfig holds all configuration for the application, loaded from environment variables.
type AppConfig struct {
	LogLevel      string `env:"LOG_LEVEL" envDefault:"info"`
	PolygonAPIKey string `env:"POLYGON_API_KEY,required"`
	QdrantAddr    string `env:"QDRANT_ADDR" envDefault:"http://localhost:6333"`
	OpenAIAPIKey  string `env:"OPENAI_API_KEY,required"`
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
	})
	return config
}
