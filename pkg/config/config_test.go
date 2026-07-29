package config

import (
	"strings"
	"testing"
)

func TestLoadConfigReadsEnvironmentAndDefaults(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("UPLOAD_DIR", "test-uploads")
	t.Setenv("APP_ENV", "development")
	t.Setenv("CORS_ALLOWED_ORIGINS", "http://localhost:5173, https://modalin.my.id")
	t.Setenv("GOOGLE_OAUTH_CLIENT_IDS", "web-client-id,android-client-id")
	LoadConfig()
	if AppConfig.Port != "9090" || AppConfig.JWTSecret != "test-secret" || AppConfig.UploadDir != "test-uploads" {
		t.Fatalf("unexpected config: %#v", AppConfig)
	}
	if AppConfig.DBHost == "" || AppConfig.DBPort == "" || AppConfig.DBName == "" {
		t.Fatal("expected database defaults")
	}
	if len(AppConfig.CORSAllowedOrigins) != 2 || AppConfig.CORSAllowedOrigins[1] != "https://modalin.my.id" {
		t.Fatalf("unexpected cors origins: %#v", AppConfig.CORSAllowedOrigins)
	}
	if len(AppConfig.GoogleOAuthClientIDs) != 2 || AppConfig.GoogleOAuthClientIDs[0] != "web-client-id" {
		t.Fatalf("unexpected Google OAuth client IDs: %#v", AppConfig.GoogleOAuthClientIDs)
	}
}

func TestValidateForProductionRejectsUnsafeConfiguration(t *testing.T) {
	tests := []struct {
		name string
		cfg  Config
		want string
	}{
		{"missing cors allowlist", Config{AppEnv: "production", JWTSecret: strings.Repeat("s", 32)}, "CORS_ALLOWED_ORIGINS"},
		{"wildcard cors", Config{AppEnv: "production", JWTSecret: strings.Repeat("s", 32), CORSAllowedOrigins: []string{"*"}}, "must not contain wildcard"},
		{"short jwt secret", Config{AppEnv: "production", JWTSecret: "short", CORSAllowedOrigins: []string{"https://modalin.my.id"}}, "at least 32"},
		{"partial bootstrap credentials", Config{AppEnv: "production", JWTSecret: strings.Repeat("s", 32), CORSAllowedOrigins: []string{"https://modalin.my.id"}, BootstrapAdminEmail: "admin@modalin.my.id"}, "both be set"},
		{"missing Google OAuth client IDs", Config{AppEnv: "production", JWTSecret: strings.Repeat("s", 32), CORSAllowedOrigins: []string{"https://modalin.my.id"}}, "GOOGLE_OAUTH_CLIENT_IDS"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.cfg.Validate(); err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Validate() error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestValidateForProductionAcceptsExplicitConfiguration(t *testing.T) {
	cfg := Config{AppEnv: "production", JWTSecret: strings.Repeat("s", 32), CORSAllowedOrigins: []string{"https://modalin.my.id", "https://www.modalin.my.id"}, GoogleOAuthClientIDs: []string{"web-client-id"}}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}
