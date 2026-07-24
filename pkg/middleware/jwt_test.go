package middleware

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
)

const testJWTSecret = "test-secret"

func token(t *testing.T, claims jwt.MapClaims) string {
	t.Helper()
	claims["iss"] = "modalin-be"
	claims["user_id"] = "11111111-1111-1111-1111-111111111111"
	claims["exp"] = time.Now().Add(time.Hour).Unix()
	signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	return signed
}

func TestJWTProtectedAndRequireRole(t *testing.T) {
	app := fiber.New()
	app.Get("/lender", JWTProtected(testJWTSecret), RequireRole("lender"), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	request := func(auth string) int {
		req := httptest.NewRequest("GET", "/lender", nil)
		if auth != "" {
			req.Header.Set(fiber.HeaderAuthorization, auth)
		}
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode
	}
	if got := request(""); got != fiber.StatusUnauthorized {
		t.Fatalf("missing token: got %d", got)
	}
	if got := request("Bearer " + token(t, jwt.MapClaims{"roles": []string{"borrower"}})); got != fiber.StatusForbidden {
		t.Fatalf("wrong role: got %d", got)
	}
	if got := request("Bearer " + token(t, jwt.MapClaims{"roles": []string{"lender"}})); got != fiber.StatusNoContent {
		t.Fatalf("valid role: got %d", got)
	}
}

func TestJWTProtectedRejectsWrongIssuerAndExpiredToken(t *testing.T) {
	app := fiber.New()
	app.Get("/private", JWTProtected(testJWTSecret), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
	request := func(claims jwt.MapClaims) int {
		signed, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testJWTSecret))
		if err != nil {
			t.Fatal(err)
		}
		req := httptest.NewRequest("GET", "/private", nil)
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+signed)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode
	}
	if got := request(jwt.MapClaims{"iss": "other", "user_id": "11111111-1111-1111-1111-111111111111", "exp": time.Now().Add(time.Hour).Unix()}); got != fiber.StatusUnauthorized {
		t.Fatalf("wrong issuer: got %d", got)
	}
	if got := request(jwt.MapClaims{"iss": "modalin-be", "user_id": "11111111-1111-1111-1111-111111111111", "exp": time.Now().Add(-time.Hour).Unix()}); got != fiber.StatusUnauthorized {
		t.Fatalf("expired: got %d", got)
	}
}
