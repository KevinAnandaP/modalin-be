package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"mime/multipart"
	"net/http/httptest"
	"testing"
	"time"

	"modalin-be/internal/business/repository"
	"modalin-be/internal/business/service"
	"modalin-be/internal/business/storage"
	"modalin-be/internal/model"
	"modalin-be/pkg/middleware"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

const testJWTSecret = "test-secret"

type handlerRepo struct {
	business *model.Business
	record   *model.FinancialRecord
	proofs   []*model.FinancialRecordProof
}

func (r *handlerRepo) CreateBusiness(_ context.Context, business *model.Business, _ *model.StarterBusinessDetail) error {
	r.business = business
	business.ID = uuid.New()
	return nil
}
func (r *handlerRepo) GetBusinessByUserID(_ context.Context, userID uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.UserID == userID && r.business.Status == "active" {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerRepo) GetBusinessByID(_ context.Context, id uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.ID == id {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerRepo) UpdateBusiness(_ context.Context, business *model.Business, _ *model.StarterBusinessDetail) error {
	r.business = business
	return nil
}
func (r *handlerRepo) CreateFinancialRecord(_ context.Context, record *model.FinancialRecord, _ []model.FinancialRecordProof) error {
	r.record = record
	record.ID = uuid.New()
	return nil
}
func (r *handlerRepo) GetFinancialRecords(_ context.Context, _ uuid.UUID, _, _ int, _ string) ([]model.FinancialRecord, error) {
	if r.record == nil {
		return nil, nil
	}
	return []model.FinancialRecord{*r.record}, nil
}
func (r *handlerRepo) GetFinancialRecordByID(_ context.Context, recordID, businessID uuid.UUID) (*model.FinancialRecord, error) {
	if r.record != nil && r.record.ID == recordID && r.record.BusinessID == businessID {
		return r.record, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerRepo) UpdateFinancialRecord(_ context.Context, record *model.FinancialRecord) error {
	r.record = record
	return nil
}
func (r *handlerRepo) CreateFinancialRecordProof(_ context.Context, proof *model.FinancialRecordProof) error {
	proof.ID = uuid.New()
	r.proofs = append(r.proofs, proof)
	return nil
}
func (r *handlerRepo) GetFinancialRecordProof(_ context.Context, proofID, _ uuid.UUID) (*model.FinancialRecordProof, error) {
	for _, proof := range r.proofs {
		if proof.ID == proofID {
			return proof, nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *handlerRepo) GetFinancialSummary(_ context.Context, _ uuid.UUID, _, _ int) (*repository.FinancialSummary, error) {
	return &repository.FinancialSummary{}, nil
}
func (r *handlerRepo) CountDistinctFinancialRecordMonths(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *handlerRepo) DeleteFinancialRecord(_ context.Context, _, _ uuid.UUID) error { return nil }

func TestBusinessRoutesRequireBorrowerRole(t *testing.T) {
	repo := &handlerRepo{}
	h := NewBusinessHandler(service.NewBusinessService(repo), storage.NewLocalProofStorage(t.TempDir(), "/uploads"))
	app := fiber.New()
	app.Use(middleware.JWTProtected(testJWTSecret), middleware.RequireRole("borrower"))
	app.Post("/businesses", h.CreateBusiness)

	request := httptest.NewRequest("POST", "/businesses", bytes.NewBufferString(`{"business_name":"Toko","category_id":1,"description":"Test","business_type":"running","location_address":"Jakarta"}`))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+testToken(t, uuid.New(), []string{"lender"}))
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusForbidden {
		t.Fatalf("expected 403 for non-borrower, got %d", response.StatusCode)
	}
}

func TestBorrowerUploadProofStoresAmountAndOwnership(t *testing.T) {
	userID := uuid.New()
	business := &model.Business{ID: uuid.New(), UserID: userID, Status: "active"}
	record := &model.FinancialRecord{ID: uuid.New(), BusinessID: business.ID, RecordDate: time.Now()}
	repo := &handlerRepo{business: business, record: record}
	dir := t.TempDir()
	h := NewBusinessHandler(service.NewBusinessService(repo), storage.NewLocalProofStorage(dir, "/uploads"))
	app := fiber.New()
	app.Use(middleware.JWTProtected(testJWTSecret), middleware.RequireRole("borrower"))
	app.Post("/financial-records/:id/proofs", h.UploadFinancialRecordProof)

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", "receipt.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte("\x89PNG\r\n\x1a\nproof")); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("proof_type", "receipt"); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteField("amount", "125000"); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest("POST", "/financial-records/"+record.ID.String()+"/proofs", &body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("Authorization", "Bearer "+testToken(t, userID, []string{"borrower"}))
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != fiber.StatusCreated {
		t.Fatalf("expected 201, got %d", response.StatusCode)
	}
	if len(repo.proofs) != 1 || repo.proofs[0].Amount == nil || *repo.proofs[0].Amount != 125000 {
		t.Fatalf("expected stored proof amount 125000, got %#v", repo.proofs)
	}
	var payload map[string]any
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
}

func TestBorrowerCanManageOwnBusinessAndFinancialRecords(t *testing.T) {
	userID := uuid.New()
	repo := &handlerRepo{}
	h := NewBusinessHandler(service.NewBusinessService(repo), storage.NewLocalProofStorage(t.TempDir(), "/uploads"))
	app := fiber.New()
	app.Use(middleware.JWTProtected(testJWTSecret), middleware.RequireRole("borrower"))
	app.Post("/businesses", h.CreateBusiness)
	app.Get("/businesses/me", h.GetMyBusiness)
	app.Put("/businesses/me", h.UpdateBusiness)
	app.Delete("/businesses/me", h.DeactivateBusiness)
	app.Post("/financial-records", h.CreateFinancialRecord)
	app.Get("/financial-records", h.GetFinancialRecords)
	app.Get("/financial-records/summary", h.GetFinancialSummary)
	app.Get("/financial-records/:id", h.GetFinancialRecord)
	app.Put("/financial-records/:id", h.UpdateFinancialRecord)
	app.Delete("/financial-records/:id", h.DeleteFinancialRecord)
	auth := "Bearer " + testToken(t, userID, []string{"borrower"})

	assertHandlerStatus(t, app, "POST", "/businesses", `{"business_name":"Toko","category_id":1,"description":"Test","business_type":"running","location_address":"Jakarta"}`, "application/json", auth, fiber.StatusCreated)
	assertHandlerStatus(t, app, "GET", "/businesses/me", "", "", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "PUT", "/businesses/me", `{"business_name":"Toko Baru","category_id":1,"description":"Update","location_address":"Bogor"}`, "application/json", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "POST", "/financial-records", `{"income_amount":200000,"expense_amount":50000}`, "application/json", auth, fiber.StatusCreated)
	assertHandlerStatus(t, app, "GET", "/financial-records", "", "", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "GET", "/financial-records/summary", "", "", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "GET", "/financial-records/"+repo.record.ID.String(), "", "", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "PUT", "/financial-records/"+repo.record.ID.String(), `{"income_amount":300000,"expense_amount":100000}`, "application/json", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "DELETE", "/financial-records/"+repo.record.ID.String(), "", "", auth, fiber.StatusOK)
	assertHandlerStatus(t, app, "DELETE", "/businesses/me", "", "", auth, fiber.StatusOK)
}

func assertHandlerStatus(t *testing.T, app *fiber.App, method, path, body, contentType, authorization string, want int) {
	t.Helper()
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	if contentType != "" {
		request.Header.Set("Content-Type", contentType)
	}
	request.Header.Set("Authorization", authorization)
	response, err := app.Test(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != want {
		t.Fatalf("%s %s: expected %d, got %d", method, path, want, response.StatusCode)
	}
}

func testToken(t *testing.T, userID uuid.UUID, roles []string) string {
	t.Helper()
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": userID.String(), "roles": roles, "iss": "modalin-be", "exp": time.Now().Add(time.Hour).Unix()}).SignedString([]byte(testJWTSecret))
	if err != nil {
		t.Fatal(err)
	}
	return token
}
