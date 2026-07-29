package router

import (
	"net/http/httptest"
	"testing"

	"modalin-be/pkg/config"

	"github.com/gofiber/fiber/v2"
)

func TestCoreRoutesExposePingAndProtectPrivateEndpoints(t *testing.T) {
	config.AppConfig = &config.Config{JWTSecret: "router-test", UploadDir: t.TempDir()}
	app := fiber.New()
	SetupRoutes(app)
	ping, err := app.Test(httptest.NewRequest("GET", "/api/v1/ping", nil))
	if err != nil {
		t.Fatal(err)
	}
	if ping.StatusCode != fiber.StatusOK {
		t.Fatalf("ping got %d", ping.StatusCode)
	}
	protected, err := app.Test(httptest.NewRequest("POST", "/api/v1/campaigns/11111111-1111-1111-1111-111111111111/fundings", nil))
	if err != nil {
		t.Fatal(err)
	}
	if protected.StatusCode != fiber.StatusUnauthorized {
		t.Fatalf("protected route got %d", protected.StatusCode)
	}
}
