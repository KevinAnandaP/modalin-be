package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"modalin-be/pkg/config"

	"github.com/gofiber/fiber/v2"
)

func TestNewAppDoesNotServeUploadedFilesPublicly(t *testing.T) {
	config.AppConfig = &config.Config{CORSAllowedOrigins: []string{"https://modalin.my.id"}}
	app := newApp()

	request := httptest.NewRequest(http.MethodGet, "/uploads/financial-proofs/known-file.pdf", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusNotFound {
		t.Fatalf("status = %d, want %d so uploads cannot be fetched without authorization", response.StatusCode, http.StatusNotFound)
	}
}

func TestShutdownAppStopsFiberWithoutDatabase(t *testing.T) {
	app := newApp()
	if err := shutdownApp(context.Background(), app); err != nil {
		t.Fatalf("shutdown failed: %v", err)
	}
}

func TestNewAppUsesSafeLimitsAndReturnsRedactedJSONErrors(t *testing.T) {
	config.AppConfig = &config.Config{CORSAllowedOrigins: []string{"https://modalin.my.id"}}
	app := newApp()
	if app.Config().BodyLimit != 6*1024*1024 || app.Config().ReadTimeout == 0 || app.Config().WriteTimeout == 0 || app.Config().IdleTimeout == 0 {
		t.Fatalf("unsafe server limits: %#v", app.Config())
	}
	app.Get("/boom", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusInternalServerError, "database password leaked")
	})
	response, err := app.Test(httptest.NewRequest(http.MethodGet, "/boom", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusInternalServerError || response.Header.Get(fiber.HeaderXRequestID) == "" {
		t.Fatalf("unexpected error response: status=%d request_id=%q", response.StatusCode, response.Header.Get(fiber.HeaderXRequestID))
	}
	var body map[string]string
	if err := json.NewDecoder(response.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "internal server error" || body["request_id"] == "" {
		t.Fatalf("unexpected error body: %#v", body)
	}
}
