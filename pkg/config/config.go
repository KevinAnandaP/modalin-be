package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	AppEnv                 string
	Port                   string
	DBHost                 string
	DBPort                 string
	DBUser                 string
	DBPassword             string
	DBName                 string
	DBSSLMode              string
	DBTimeZone             string
	JWTSecret              string
	UploadDir              string
	CORSAllowedOrigins     []string
	GoogleOAuthClientIDs   []string
	BootstrapAdminEmail    string
	BootstrapAdminPassword string
	XenditSecretKey        string
	XenditWebhookToken     string
	XenditMode             string
}

var AppConfig *Config

func LoadConfig() {
	// Load env file from configs folder
	err := godotenv.Load("configs/.env")
	if err != nil {
		log.Println("Warning: No configs/.env file found, reading from system environment variables.")
	}

	AppConfig = &Config{
		AppEnv:                 getEnv("APP_ENV", "development"),
		Port:                   getEnv("PORT", "8080"),
		DBHost:                 getEnv("DB_HOST", "127.0.0.1"),
		DBPort:                 getEnv("DB_PORT", "5432"),
		DBUser:                 getEnv("DB_USER", "postgres"),
		DBPassword:             getEnv("DB_PASSWORD", ""),
		DBName:                 getEnv("DB_NAME", "modalin-db"),
		DBSSLMode:              getEnv("DB_SSLMODE", "require"),
		DBTimeZone:             getEnv("DB_TIMEZONE", "Asia/Jakarta"),
		JWTSecret:              getEnv("JWT_SECRET", ""),
		UploadDir:              getEnv("UPLOAD_DIR", "uploads"),
		CORSAllowedOrigins:     splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173")),
		GoogleOAuthClientIDs:   splitCSV(getEnv("GOOGLE_OAUTH_CLIENT_IDS", "")),
		BootstrapAdminEmail:    strings.TrimSpace(getEnv("BOOTSTRAP_ADMIN_EMAIL", "")),
		BootstrapAdminPassword: getEnv("BOOTSTRAP_ADMIN_PASSWORD", ""),
		XenditSecretKey:        getEnv("XENDIT_SECRET_KEY", ""),
		XenditWebhookToken:     getEnv("XENDIT_WEBHOOK_VERIFICATION_TOKEN", ""),
		XenditMode:             getEnv("XENDIT_MODE", "sandbox"),
	}
}

// Validate rejects configurations that are unsafe for an internet-facing deployment.
func (c Config) Validate() error {
	if c.JWTSecret == "" {
		return errors.New("JWT_SECRET must be configured")
	}
	if (c.BootstrapAdminEmail == "") != (c.BootstrapAdminPassword == "") {
		return errors.New("BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD must both be set or both be empty")
	}
	if strings.EqualFold(strings.TrimSpace(c.AppEnv), "production") {
		if len(c.JWTSecret) < 32 {
			return errors.New("JWT_SECRET must be at least 32 characters in production")
		}
		if len(c.CORSAllowedOrigins) == 0 {
			return errors.New("CORS_ALLOWED_ORIGINS must be configured in production")
		}
		for _, origin := range c.CORSAllowedOrigins {
			if origin == "*" {
				return errors.New("CORS_ALLOWED_ORIGINS must not contain wildcard in production")
			}
			if !strings.HasPrefix(origin, "https://") {
				return fmt.Errorf("CORS_ALLOWED_ORIGINS entry %q must use https in production", origin)
			}
		}
		if len(c.GoogleOAuthClientIDs) == 0 {
			return errors.New("GOOGLE_OAUTH_CLIENT_IDS must be configured in production")
		}
	}
	return nil
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		if origin := strings.TrimSpace(part); origin != "" {
			origins = append(origins, origin)
		}
	}
	return origins
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
