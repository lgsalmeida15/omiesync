package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	DatabaseURL        string
	JWTSecret          string
	Port               string
	AppEnv             string
	WorkerMaxConcurrent int
}

func Load() (*Config, error) {
	// Tenta carregar o arquivo .env, mas não falha se não existir
	_ = godotenv.Load()

	maxConcurrent := 20
	if v := os.Getenv("WORKER_MAX_CONCURRENT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			maxConcurrent = n
		}
	}

	cfg := &Config{
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		JWTSecret:          os.Getenv("JWT_SECRET"),
		Port:               os.Getenv("PORT"),
		AppEnv:             os.Getenv("APP_ENV"),
		WorkerMaxConcurrent: maxConcurrent,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("config.Load: DATABASE_URL é obrigatório")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("config.Load: JWT_SECRET deve ter no mínimo 32 caracteres")
	}
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.AppEnv == "" {
		cfg.AppEnv = "development"
	}

	return cfg, nil
}
