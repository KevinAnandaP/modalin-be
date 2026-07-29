package config

import "testing"

func TestLoadConfigReadsEnvironmentAndDefaults(t *testing.T) {
	t.Setenv("PORT", "9090")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("UPLOAD_DIR", "test-uploads")
	LoadConfig()
	if AppConfig.Port != "9090" || AppConfig.JWTSecret != "test-secret" || AppConfig.UploadDir != "test-uploads" {
		t.Fatalf("unexpected config: %#v", AppConfig)
	}
	if AppConfig.DBHost == "" || AppConfig.DBPort == "" || AppConfig.DBName == "" {
		t.Fatal("expected database defaults")
	}
}
