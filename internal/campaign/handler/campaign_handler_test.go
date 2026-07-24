package handler

import (
	"bytes"
	"context"
	"errors"
	"mime/multipart"
	"net/http/httptest"
	"testing"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/campaign/service"
	"modalin-be/internal/model"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type proofAccessRepo struct {
	repository.Repository
	business *model.Business
	campaign *model.LoanCampaign
	proof    *model.FundUsageProof
	funded   bool
}

func (r *proofAccessRepo) GetBusinessByUserID(_ context.Context, id uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.UserID == id {
		return r.business, nil
	}
	return nil, errors.New("not found")
}
func (r *proofAccessRepo) GetCampaignForBusiness(_ context.Context, id, businessID uuid.UUID) (*model.LoanCampaign, error) {
	if id == r.campaign.ID && businessID == r.business.ID {
		return r.campaign, nil
	}
	return nil, errors.New("not found")
}
func (r *proofAccessRepo) GetFundUsageProof(_ context.Context, id uuid.UUID) (*model.FundUsageProof, error) {
	if id == r.proof.ID {
		return r.proof, nil
	}
	return nil, errors.New("not found")
}
func (r *proofAccessRepo) HasPaidFunding(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return r.funded, nil
}

type memoryProofStorage struct{}

func (memoryProofStorage) Store(*multipart.FileHeader) (string, error) { return "", nil }
func (memoryProofStorage) Delete(string) error                         { return nil }
func (memoryProofStorage) Read(string) ([]byte, error)                 { return []byte("proof"), nil }

type trackingProofStorage struct{ stored, deleted []string }

func (s *trackingProofStorage) Store(*multipart.FileHeader) (string, error) {
	url := "/uploads/proof-" + uuid.NewString() + ".png"
	s.stored = append(s.stored, url)
	return url, nil
}
func (s *trackingProofStorage) Delete(url string) error {
	s.deleted = append(s.deleted, url)
	return nil
}
func (s *trackingProofStorage) Read(string) ([]byte, error) {
	return nil, errors.New("not implemented")
}

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
	request := httptest.NewRequest("GET", "/campaigns?search=warung&category=kuliner&risk_level=low&min_amount=100000&max_amount=500000&min_risk_score=70&max_risk_score=90", nil)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusOK {
		t.Fatalf("expected 200, got %d", response.StatusCode)
	}
	if repo.received.Query != "warung" || repo.received.Category != "kuliner" || repo.received.RiskLevel != "low" || repo.received.MinAmount != 100000 || repo.received.MaxAmount != 500000 || repo.received.MinRiskScore != 70 || repo.received.MaxRiskScore != 90 {
		t.Fatalf("unexpected catalog filter: %#v", repo.received)
	}
}

func TestCatalogRejectsInvalidRiskScoreRange(t *testing.T) {
	app := fiber.New()
	app.Get("/campaigns", NewCampaignHandler(service.NewCampaignService(&catalogRepo{})).Catalog)
	response, err := app.Test(httptest.NewRequest("GET", "/campaigns?min_risk_score=90&max_risk_score=70", nil))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusBadRequest {
		t.Fatalf("expected 400, got %d", response.StatusCode)
	}
}

func TestProofDownloadEnforcesCampaignAccess(t *testing.T) {
	owner := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: owner}
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID}
	proof := &model.FundUsageProof{ID: uuid.New(), CampaignID: campaign.ID, FileURL: "/uploads/fund-usage-proofs/a.png"}
	repo := &proofAccessRepo{business: business, campaign: campaign, proof: proof}
	call := func(user uuid.UUID, roles []string) int {
		app := fiber.New()
		app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", user); c.Locals("roles", roles); return c.Next() })
		app.Get("/campaigns/:id/fund-usage-proofs/:proofID/download", NewCampaignHandler(service.NewCampaignService(repo), memoryProofStorage{}).DownloadFundUsageProof)
		req := httptest.NewRequest("GET", "/campaigns/"+campaign.ID.String()+"/fund-usage-proofs/"+proof.ID.String()+"/download", nil)
		res, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		return res.StatusCode
	}
	if status := call(owner, []string{"borrower"}); status != fiber.StatusOK {
		t.Fatalf("owner expected 200, got %d", status)
	}
	if status := call(uuid.New(), []string{"lender"}); status != fiber.StatusForbidden {
		t.Fatalf("unfunded lender expected 403, got %d", status)
	}
	repo.funded = true
	if status := call(uuid.New(), []string{"lender"}); status != fiber.StatusOK {
		t.Fatalf("funded lender expected 200, got %d", status)
	}
	if status := call(uuid.New(), []string{"admin"}); status != fiber.StatusOK {
		t.Fatalf("admin expected 200, got %d", status)
	}
}

func TestRevenueReportUploadIsCleanedUpWhenServiceRejectsIt(t *testing.T) {
	storage := &trackingProofStorage{}
	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", uuid.New()); return c.Next() })
	app.Post("/campaigns/:id/revenue-reports", NewCampaignHandler(service.NewCampaignService(&proofAccessRepo{}), storage).CreateRevenueReport)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	_ = writer.WriteField("period_month", "7")
	_ = writer.WriteField("period_year", "2026")
	_ = writer.WriteField("gross_revenue", "100000")
	_ = writer.WriteField("transaction_count", "1")
	_ = writer.WriteField("business_status", "running")
	file, err := writer.CreateFormFile("file", "receipt.png")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = file.Write([]byte("proof"))
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest("POST", "/campaigns/"+uuid.NewString()+"/revenue-reports", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode < 400 {
		t.Fatalf("expected rejection, got %d", response.StatusCode)
	}
	if len(storage.stored) != 1 || len(storage.deleted) != 1 || storage.stored[0] != storage.deleted[0] {
		t.Fatalf("uploaded report proof was not cleaned up: stored=%v deleted=%v", storage.stored, storage.deleted)
	}
}
