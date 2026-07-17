package service_test

import (
	"context"
	"strings"
	"testing"

	"modalin-be/internal/auth/service"
)

func TestRegisterRejectsUnacceptedTerms(t *testing.T) {
	svc := service.NewAuthService(nil, "test-secret")

	_, err := svc.Register(context.Background(), service.RegisterInput{
		FullName:      "Budi Santoso",
		Email:         "budi@example.com",
		Phone:         "08123456789",
		Password:      strings.Repeat("x", 12),
		City:          "Jakarta",
		Address:       "Jl. Merdeka 1",
		TermsAccepted: false,
	})

	if err != service.ErrTermsNotAccepted {
		t.Fatalf("expected ErrTermsNotAccepted, got %v", err)
	}
}
