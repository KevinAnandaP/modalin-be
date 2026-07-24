package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestAuthHandlerRejectsIncompleteCredentialsBeforeServiceCall(t *testing.T) {
	app := fiber.New()
	h := &AuthHandler{}
	app.Post("/register", h.Register)
	app.Post("/login", h.Login)
	register, err := app.Test(httptest.NewRequest("POST", "/register", strings.NewReader(`{"email":"a@example.com"}`)))
	if err != nil {
		t.Fatal(err)
	}
	if register.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("register got %d", register.StatusCode)
	}
	login, err := app.Test(httptest.NewRequest("POST", "/login", strings.NewReader(`{"email":"a@example.com"}`)))
	if err != nil {
		t.Fatal(err)
	}
	if login.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("login got %d", login.StatusCode)
	}
}
