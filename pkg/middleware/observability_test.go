package middleware

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestJSONRequestLoggerRecordsSafeRequestMetadata(t *testing.T) {
	var output bytes.Buffer
	app := fiber.New()
	app.Use(JSONRequestLogger(&output))
	app.Get("/health", func(c *fiber.Ctx) error {
		c.Locals("requestid", "request-123")
		return c.SendStatus(fiber.StatusNoContent)
	})

	response, err := app.Test(httptest.NewRequest("GET", "/health", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d", response.StatusCode)
	}

	var entry map[string]any
	if err := json.Unmarshal(output.Bytes(), &entry); err != nil {
		t.Fatalf("log is not JSON: %q (%v)", output.String(), err)
	}
	if entry["request_id"] != "request-123" || entry["method"] != "GET" || entry["path"] != "/health" {
		t.Fatalf("unexpected log entry: %#v", entry)
	}
	if _, ok := entry["authorization"]; ok {
		t.Fatalf("log must not include authorization: %#v", entry)
	}
}

func TestRequestMetricsCountStatusAndLatencyWithoutRequestData(t *testing.T) {
	metrics := NewRequestMetrics()
	app := fiber.New()
	app.Use(metrics.Middleware())
	app.Get("/ok", func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	response, err := app.Test(httptest.NewRequest("GET", "/ok", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNoContent {
		t.Fatalf("status = %d", response.StatusCode)
	}
	snapshot := metrics.Snapshot()
	if snapshot.Requests != 1 || snapshot.Statuses["204"] != 1 || snapshot.TotalLatencyMS < 0 {
		t.Fatalf("unexpected metrics: %#v", snapshot)
	}
}

func TestRequestMetricsRecordReturnedErrorStatus(t *testing.T) {
	metrics := NewRequestMetrics()
	app := fiber.New()
	app.Use(metrics.Middleware())
	app.Get("/missing", func(*fiber.Ctx) error { return fiber.ErrNotFound })
	response, err := app.Test(httptest.NewRequest("GET", "/missing", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusNotFound || metrics.Snapshot().Statuses["404"] != 1 {
		t.Fatalf("unexpected error metrics: status=%d snapshot=%#v", response.StatusCode, metrics.Snapshot())
	}
}
