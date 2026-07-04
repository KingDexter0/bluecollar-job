package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv              string
	AppPort             string
	CORSAllowedOrigins  []string
	DatabaseURL         string
	FrontendURL         string
	Redis               RedisConfig
	AadhaarGateway      AadhaarGatewayConfig
	NotificationWorker  NotificationWorkerConfig
	JWTTTL              time.Duration
	JWTSecret           string
	JWTIssuer           string
	AdminToken          string
	WhatsAppVerifyToken string
	WhatsAppAccessToken string
	ObjectStorage       ObjectStorageConfig
	DocumentUpload      DocumentUploadConfig
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

type NotificationWorkerConfig struct {
	Enabled             bool
	Count               int
	PollIntervalSeconds int
}

type ObjectStorageConfig struct {
	Provider        string
	LocalBasePath   string
	Bucket          string
	Region          string
	Endpoint        string
	AccessKeyID     string
	SecretAccessKey string
}

type DocumentUploadConfig struct {
	Enabled bool
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return nil, fmt.Errorf("REDIS_DB must be an integer: %w", err)
	}
	workerCount, err := strconv.Atoi(getEnv("NOTIFICATION_WORKER_COUNT", "2"))
	if err != nil {
		return nil, fmt.Errorf("NOTIFICATION_WORKER_COUNT must be an integer: %w", err)
	}
	pollIntervalSeconds, err := strconv.Atoi(getEnv("NOTIFICATION_POLL_INTERVAL_SECONDS", "5"))
	if err != nil {
		return nil, fmt.Errorf("NOTIFICATION_POLL_INTERVAL_SECONDS must be an integer: %w", err)
	}
	jwtTTLHours, err := strconv.Atoi(getEnv("JWT_TTL_HOURS", "24"))
	if err != nil || jwtTTLHours <= 0 {
		return nil, fmt.Errorf("JWT_TTL_HOURS must be a positive integer")
	}

	cfg := &Config{
		AppEnv:             strings.ToLower(strings.TrimSpace(getEnv("APP_ENV", "development"))),
		AppPort:            getEnv("APP_PORT", "8080"),
		CORSAllowedOrigins: parseCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://127.0.0.1:3000")),
		DatabaseURL:        os.Getenv("DATABASE_URL"),
		FrontendURL:        getEnv("FRONTEND_URL", "http://localhost:3000"),
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
		NotificationWorker: NotificationWorkerConfig{
			Enabled:             getEnv("NOTIFICATION_WORKER_ENABLED", "false") == "true",
			Count:               workerCount,
			PollIntervalSeconds: pollIntervalSeconds,
		},
		JWTTTL:              time.Duration(jwtTTLHours) * time.Hour,
		JWTSecret:           os.Getenv("JWT_SECRET"),
		JWTIssuer:           getEnv("JWT_ISSUER", "bluecollarjob"),
		AdminToken:          getEnv("ADMIN_TOKEN", "local-admin-token"),
		WhatsAppVerifyToken: os.Getenv("WHATSAPP_VERIFY_TOKEN"),
		WhatsAppAccessToken: os.Getenv("WHATSAPP_ACCESS_TOKEN"),
		ObjectStorage: ObjectStorageConfig{
			Provider:        getEnv("OBJECT_STORAGE_PROVIDER", "local"),
			LocalBasePath:   getEnv("OBJECT_STORAGE_LOCAL_PATH", "./var/uploads"),
			Bucket:          os.Getenv("OBJECT_STORAGE_BUCKET"),
			Region:          os.Getenv("OBJECT_STORAGE_REGION"),
			Endpoint:        os.Getenv("OBJECT_STORAGE_ENDPOINT"),
			AccessKeyID:     os.Getenv("OBJECT_STORAGE_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("OBJECT_STORAGE_SECRET_ACCESS_KEY"),
		},
		DocumentUpload: DocumentUploadConfig{
			Enabled: getEnv("DOCUMENT_UPLOAD_ENABLED", "true") == "true",
		},
	}

	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return cfg, nil
}

func (c *Config) Validate() error {
	switch c.AppEnv {
	case "local", "development", "staging", "production":
	default:
		return fmt.Errorf("APP_ENV must be one of local, development, staging, production")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("DATABASE_URL is required")
	}
	if strings.TrimSpace(c.Redis.Addr) == "" {
		return fmt.Errorf("REDIS_ADDR is required")
	}
	if strings.TrimSpace(c.JWTSecret) == "" {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.IsProduction() || c.AppEnv == "staging" {
		if strings.TrimSpace(os.Getenv("APP_ENV")) == "" {
			return fmt.Errorf("APP_ENV is required in staging/production")
		}
		if strings.TrimSpace(c.AppPort) == "" {
			return fmt.Errorf("APP_PORT is required in staging/production")
		}
		if len(c.CORSAllowedOrigins) == 0 {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS is required in staging/production")
		}
		if strings.TrimSpace(c.FrontendURL) == "" {
			return fmt.Errorf("FRONTEND_URL is required in staging/production")
		}
		if strings.TrimSpace(c.WhatsAppVerifyToken) == "" {
			return fmt.Errorf("WHATSAPP_VERIFY_TOKEN is required in staging/production")
		}
		if strings.TrimSpace(c.Redis.Password) == "" {
			return fmt.Errorf("REDIS_PASSWORD is required in staging/production")
		}
		if len(c.JWTSecret) < 32 {
			return fmt.Errorf("JWT_SECRET must be at least 32 characters in staging/production")
		}
		if isWeakSecret(c.JWTSecret) {
			return fmt.Errorf("JWT_SECRET is too weak for staging/production")
		}
		if len(strings.TrimSpace(c.AdminToken)) < 32 {
			return fmt.Errorf("ADMIN_TOKEN must be at least 32 characters in staging/production")
		}
		if c.DocumentUpload.Enabled && strings.TrimSpace(c.ObjectStorage.Bucket) == "" {
			return fmt.Errorf("OBJECT_STORAGE_BUCKET is required in staging/production when document upload is enabled")
		}
	}
	return nil
}

func (c *Config) IsProduction() bool {
	return c.AppEnv == "production"
}

func (c *Config) IsDevelopment() bool {
	return c.AppEnv == "local" || c.AppEnv == "development"
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func parseCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		cleaned := strings.TrimSpace(part)
		if cleaned != "" {
			result = append(result, cleaned)
		}
	}
	return result
}

func isWeakSecret(secret string) bool {
	lowered := strings.ToLower(strings.TrimSpace(secret))
	switch lowered {
	case "secret", "jwt_secret", "replace-with-a-long-random-secret", "local-dev-secret", "changeme":
		return true
	default:
		return false
	}
}
