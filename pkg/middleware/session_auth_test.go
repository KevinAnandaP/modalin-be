package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeAuthorizationStateReader struct {
	status       string
	tokenVersion uint64
	roles        []string
	err          error
}

func (r fakeAuthorizationStateReader) GetAuthorizationState(_ context.Context, _ uuid.UUID) (string, uint64, []string, error) {
	return r.status, r.tokenVersion, r.roles, r.err
}

func TestJWTProtectedRejectsRevokedTokenAndUsesCurrentRoles(t *testing.T) {
	request := func(reader fakeAuthorizationStateReader, claims map[string]any) int {
		app := fiber.New()
		app.Get("/lender", JWTProtected(testJWTSecret, reader), RequireRole("lender"), func(c *fiber.Ctx) error { return c.SendStatus(fiber.StatusNoContent) })
		jwtClaims := jwt.MapClaims(claims)
		req := httptest.NewRequest("GET", "/lender", nil)
		req.Header.Set(fiber.HeaderAuthorization, "Bearer "+token(t, jwtClaims))
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return response.StatusCode
	}

	if got := request(fakeAuthorizationStateReader{status: "active", tokenVersion: 2, roles: []string{"lender"}}, map[string]any{"token_version": 1, "roles": []string{"lender"}}); got != fiber.StatusUnauthorized {
		t.Fatalf("revoked token: got %d", got)
	}
	if got := request(fakeAuthorizationStateReader{status: "blocked", tokenVersion: 1, roles: []string{"lender"}}, map[string]any{"token_version": 1, "roles": []string{"lender"}}); got != fiber.StatusForbidden {
		t.Fatalf("blocked user: got %d", got)
	}
	if got := request(fakeAuthorizationStateReader{status: "active", tokenVersion: 1, roles: []string{"borrower"}}, map[string]any{"token_version": 1, "roles": []string{"lender"}}); got != fiber.StatusForbidden {
		t.Fatalf("revoked role: got %d", got)
	}
	if got := request(fakeAuthorizationStateReader{status: "active", tokenVersion: 1, roles: []string{"lender"}}, map[string]any{"token_version": 1, "roles": []string{"borrower"}}); got != fiber.StatusNoContent {
		t.Fatalf("newly granted role: got %d", got)
	}
	if got := request(fakeAuthorizationStateReader{err: gorm.ErrRecordNotFound}, map[string]any{"token_version": 1}); got != fiber.StatusUnauthorized {
		t.Fatalf("missing user: got %d", got)
	}
}
