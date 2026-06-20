package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort             string
	DatabaseURL         string
	Redis               RedisConfig
	AadhaarGateway      AadhaarGatewayConfig
	JWTSecret           string
	WhatsAppVerifyToken string
	WhatsAppAccessToken string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type AadhaarGatewayConfig struct {
	Provider     string
	BaseURL      string
	ClientID     string
	ClientSecret string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("REDIS_DB must be an integer: %w", err)
	}

	cfg := &Config{
		AppPort:     getEnv("APP_PORT", "8080"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		Redis: RedisConfig{
			Addr:     getEnv("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		AadhaarGateway: AadhaarGatewayConfig{
			Provider:     getEnv("AADHAAR_GATEWAY_PROVIDER", "mock"),
			BaseURL:      os.Getenv("AADHAAR_GATEWAY_BASE_URL"),
			ClientID:     os.Getenv("AADHAAR_GATEWAY_CLIENT_ID"),
			ClientSecret: os.Getenv("AADHAAR_GATEWAY_CLIENT_SECRET"),
		},
		JWTSecret:           os.Getenv("JWT_SECRET"),
		WhatsAppVerifyToken: os.Getenv("WHATSAPP_VERIFY_TOKEN"),
		WhatsAppAccessToken: os.Getenv("WHATSAPP_ACCESS_TOKEN"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
