package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"modalin-be/internal/model"
	verificationRepo "modalin-be/internal/verification/repository"
	verificationSvc "modalin-be/internal/verification/service"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type mockHandlerRepo struct {
	verificationRepo.Repository
	business     *model.Business
	campaign     *model.LoanCampaign
	request      *model.VerificationRequest
	report       *model.VerificationReport
	hasRole      bool
	isAssigned   bool
	hasFunding   bool
	voteUpserted bool
}

func (r *mockHandlerRepo) HasActiveVerificationRequest(context.Context, uuid.UUID) (bool, error) {
	return false, nil
}

func (m *mockHandlerRepo) WithTransaction(_ context.Context, fn func(verificationRepo.Repository) error) error {
	return fn(m)
}
func (m *mockHandlerRepo) GetCampaignForUpdate(_ context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	if m.campaign != nil && m.campaign.ID == id {
		return m.campaign, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockHandlerRepo) GetBusiness(_ context.Context, id uuid.UUID) (*model.Business, error) {
	if m.business != nil && m.business.ID == id {
		return m.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockHandlerRepo) CreateRequest(_ context.Context, req *model.VerificationRequest) error {
	m.request = req
	return nil
}
func (m *mockHandlerRepo) GetRequestForUpdate(_ context.Context, id uuid.UUID) (*model.VerificationRequest, error) {
	if m.request != nil && m.request.ID == id {
		return m.request, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockHandlerRepo) GetRequest(_ context.Context, id uuid.UUID) (*model.VerificationRequest, error) {
	if m.request != nil && m.request.ID == id {
		return m.request, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockHandlerRepo) UpdateRequest(_ context.Context, req *model.VerificationRequest) error {
	m.request = req
	return nil
}
func (m *mockHandlerRepo) UpdateBusiness(_ context.Context, b *model.Business) error {
	m.business = b
	return nil
}
func (m *mockHandlerRepo) HasActiveRole(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return m.hasRole, nil
}
func (m *mockHandlerRepo) HasAnyActiveRole(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.hasRole, nil
}
func (m *mockHandlerRepo) IsAssignedVerifier(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.isAssigned, nil
}
func (m *mockHandlerRepo) HasFunding(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return m.hasFunding, nil
}
func (m *mockHandlerRepo) CreateReport(_ context.Context, r *model.VerificationReport) error {
	m.report = r
	return nil
}
func (m *mockHandlerRepo) HasReport(_ context.Context, _ uuid.UUID) (bool, error) {
	return m.report != nil, nil
}
func (m *mockHandlerRepo) LatestReport(_ context.Context, _ uuid.UUID) (*model.VerificationReport, error) {
	if m.report != nil {
		return m.report, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (m *mockHandlerRepo) UpsertVote(_ context.Context, _ *model.CommunityVote) error {
	m.voteUpserted = true
	return nil
}
func (m *mockHandlerRepo) CreateAuditLog(_ context.Context, _ *model.AuditLog) error {
	return nil
}
func (m *mockHandlerRepo) LatestCampaignID(_ context.Context, bID uuid.UUID) (uuid.UUID, error) {
	if m.campaign != nil && m.campaign.BusinessID == bID {
		return m.campaign.ID, nil
	}
	return uuid.Nil, gorm.ErrRecordNotFound
}
func (m *mockHandlerRepo) ListRequestsPage(_ context.Context, f verificationRepo.RequestFilter) ([]model.VerificationRequest, int64, error) {
	if m.request != nil {
		return []model.VerificationRequest{*m.request}, 1, nil
	}
	return nil, 0, nil
}

type mockProofStorage struct{}

func (mockProofStorage) StoreCategory(cat string, file *multipart.FileHeader) (string, error) {
	return "/uploads/" + cat + "/" + file.Filename, nil
}
func (mockProofStorage) Read(path string) ([]byte, error) {
	return []byte("photo-data"), nil
}

func TestHandlerCreateRequestAndAssign(t *testing.T) {
	borrowerID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID, CurrentBorrowingLimit: 5000000}
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID, Status: "admin_review"}
	repo := &mockHandlerRepo{business: business, campaign: campaign}

	svc := verificationSvc.New(repo)
	h := New(svc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", borrowerID); return c.Next() })
	app.Post("/campaigns/:id/verification-requests", h.CreateRequest)

	req := httptest.NewRequest("POST", "/campaigns/"+campaign.ID.String()+"/verification-requests", nil)
	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created, got %d", res.StatusCode)
	}
}

func TestHandlerVote(t *testing.T) {
	voterID := uuid.New()
	borrowerID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID}
	repo := &mockHandlerRepo{business: business, hasRole: true}

	svc := verificationSvc.New(repo)
	h := New(svc)

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error { c.Locals("user_id", voterID); return c.Next() })
	app.Post("/businesses/:id/community-vote", h.Vote)

	body, _ := json.Marshal(map[string]string{"vote_type": "up"})
	req := httptest.NewRequest("POST", "/businesses/"+business.ID.String()+"/community-vote", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", res.StatusCode)
	}
}

func TestHandlerUploadAndDownloadPhoto(t *testing.T) {
	verifierID := uuid.New()
	borrowerID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: borrowerID}
	campaign := &model.LoanCampaign{ID: uuid.New(), BusinessID: business.ID}
	vReq := &model.VerificationRequest{ID: uuid.New(), CampaignID: &campaign.ID, BusinessID: business.ID, AssignedVerifierID: &verifierID, Status: "assigned"}
	vReport := &model.VerificationReport{ID: uuid.New(), VerificationRequestID: vReq.ID, VerifierUserID: verifierID, PhotoURL: "/uploads/verification-reports/photo.jpg"}

	repo := &mockHandlerRepo{business: business, campaign: campaign, request: vReq, report: vReport}
	svc := verificationSvc.New(repo)
	h := New(svc, mockProofStorage{})

	app := fiber.New()
	app.Use(func(c *fiber.Ctx) error {
		c.Locals("user_id", verifierID)
		c.Locals("roles", []string{"verifier"})
		return c.Next()
	})
	app.Post("/verification-requests/:id/photo", h.UploadPhoto)
	app.Get("/verification-requests/:id/photo/download", h.DownloadPhoto)

	// Test Upload
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	file, _ := writer.CreateFormFile("photo", "test.jpg")
	_, _ = file.Write([]byte("image-bytes"))
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/verification-requests/"+vReq.ID.String()+"/photo", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	res, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201 Created for upload, got %d", res.StatusCode)
	}

	// Test Download
	reqDl := httptest.NewRequest("GET", "/verification-requests/"+vReq.ID.String()+"/photo/download", nil)
	resDl, err := app.Test(reqDl)
	if err != nil {
		t.Fatal(err)
	}
	if resDl.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK for download, got %d", resDl.StatusCode)
	}
	dlBytes, _ := io.ReadAll(resDl.Body)
	if string(dlBytes) != "photo-data" {
		t.Fatalf("unexpected downloaded content: %s", string(dlBytes))
	}
}
