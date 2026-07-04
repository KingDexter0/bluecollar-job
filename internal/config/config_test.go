package config

import "testing"

func TestProductionValidationRequiresStrongSecretAndOrigins(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	cfg := &Config{
		AppEnv:      "production",
		DatabaseURL: "postgres://example",
		Redis: RedisConfig{
			Addr: "redis:6379",
		},
		JWTSecret: "short",
	}

	if err := cfg.Validate(); err == nil {
		t.Fatal("expected production validation to reject missing CORS origins and weak JWT secret")
	}
}

func TestProductionValidationAcceptsRequiredSettings(t *testing.T) {
	t.Setenv("APP_ENV", "production")

	cfg := &Config{
		AppEnv:             "production",
		CORSAllowedOrigins: []string{"https://app.example.com"},
		DatabaseURL:        "postgres://example",
		Redis: RedisConfig{
			Addr: "redis:6379",
		},
		JWTSecret: "this-is-a-long-random-production-secret",
	}

	if err := cfg.Validate(); err != nil {
		t.Fatalf("expected production config to validate: %v", err)
	}
}

func TestDevelopmentModeIncludesLocalAndDevelopment(t *testing.T) {
	for _, env := range []string{"local", "development"} {
		cfg := &Config{AppEnv: env}
		if !cfg.IsDevelopment() {
			t.Fatalf("expected %s to be treated as development mode", env)
		}
	}

	cfg := &Config{AppEnv: "production"}
	if cfg.IsDevelopment() {
		t.Fatal("expected production to disable development mode")
	}
}
