package handler

import (
	"context"
	"net/http/httptest"
	"testing"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/campaign/service"
	"modalin-be/internal/model"

	"github.com/gofiber/fiber/v2"
)

type catalogRepo struct {
	repository.Repository
	received repository.CatalogFilter
}

func (r *catalogRepo) ListCatalog(_ context.Context, filter repository.CatalogFilter) ([]model.LoanCampaign, error) {
	r.received = filter
	return []model.LoanCampaign{{Title: "Modal Warung", Status: "published"}}, nil
}

func TestCatalogIsPublicAndPassesSearchFilters(t *testing.T) {
	repo := &catalogRepo{}
	app := fiber.New()
	app.Get("/campaigns", NewCampaignHandler(service.NewCampaignService(repo)).Catalog)
	request := httptest.NewRequest("GET", "/campaigns?search=warung&category=kuliner&risk_level=low&min_amount=100000&max_amount=500000", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if repo.received.Query != "warung" || repo.received.Category != "kuliner" || repo.received.RiskLevel != "low" || repo.received.MinAmount != 100000 || repo.received.MaxAmount != 500000 {
		t.Fatalf("unexpected catalog filter: %#v", repo.received)
	}
}
