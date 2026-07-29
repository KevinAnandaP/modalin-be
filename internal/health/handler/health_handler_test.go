package handler

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestLiveDoesNotRequireDatabaseButReadyDoes(t *testing.T) {
	app := fiber.New()
	health := NewHealthHandler()
	app.Get("/health", health.Live)
	app.Get("/ready", health.Ready)
	live, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	if live.StatusCode != fiber.StatusOK {
		t.Fatalf("liveness got %d", live.StatusCode)
	}
	ready, err := app.Test(httptest.NewRequest("GET", "/ready", nil))
	if err != nil {
		t.Fatal(err)
	}
	if ready.StatusCode != fiber.StatusServiceUnavailable {
		t.Fatalf("readiness got %d", ready.StatusCode)
	}
}
