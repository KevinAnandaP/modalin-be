package service

import (
	"context"
	"testing"

	"modalin-be/internal/model"
	verificationRepo "modalin-be/internal/verification/repository"

	"github.com/google/uuid"
)

type lifecycleRepo struct {
	verificationRepo.Repository
	request  *model.VerificationRequest
	business *model.Business
	rows     []model.VerificationRequest
	total    int64
}

func (r *lifecycleRepo) WithTransaction(_ context.Context, fn func(verificationRepo.Repository) error) error {
	return fn(r)
}
func (r *lifecycleRepo) GetRequestForUpdate(_ context.Context, _ uuid.UUID) (*model.VerificationRequest, error) {
	return r.request, nil
}
func (r *lifecycleRepo) GetRequest(_ context.Context, _ uuid.UUID) (*model.VerificationRequest, error) {
	return r.request, nil
}
func (r *lifecycleRepo) UpdateRequest(_ context.Context, v *model.VerificationRequest) error {
	r.request = v
	return nil
}
func (r *lifecycleRepo) GetBusiness(_ context.Context, _ uuid.UUID) (*model.Business, error) {
	return r.business, nil
}
func (r *lifecycleRepo) HasActiveRole(_ context.Context, _ uuid.UUID, _ string) (bool, error) {
	return true, nil
}
func (r *lifecycleRepo) HasFunding(_ context.Context, _ uuid.UUID, _ uuid.UUID) (bool, error) {
	return false, nil
}
func (r *lifecycleRepo) CreateAuditLog(_ context.Context, _ *model.AuditLog) error { return nil }
func (r *lifecycleRepo) ListRequestsPage(_ context.Context, _ verificationRepo.RequestFilter) ([]model.VerificationRequest, int64, error) {
	return r.rows, r.total, nil
}

func TestReassignRevokesPriorAssignee(t *testing.T) {
	old, next, businessID, campaignID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	r := &lifecycleRepo{request: &model.VerificationRequest{ID: uuid.New(), BusinessID: businessID, CampaignID: &campaignID, AssignedVerifierID: &old, Status: "assigned"}, business: &model.Business{ID: businessID, UserID: uuid.New()}}
	if err := New(r).Reassign(context.Background(), uuid.New(), r.request.ID, next); err != nil {
		t.Fatal(err)
	}
	if *r.request.AssignedVerifierID != next || r.request.Status != "assigned" {
		t.Fatalf("unexpected reassignment %#v", r.request)
	}
	if err := New(r).CanUploadPhoto(context.Background(), old, r.request.ID); err == nil {
		t.Fatal("old verifier must lose upload access")
	}
}
func TestCancelOnlyPendingOrAssigned(t *testing.T) {
	r := &lifecycleRepo{request: &model.VerificationRequest{ID: uuid.New(), Status: "assigned"}}
	if err := New(r).Cancel(context.Background(), uuid.New(), r.request.ID); err != nil {
		t.Fatal(err)
	}
	if r.request.Status != "cancelled" {
		t.Fatalf("got %s", r.request.Status)
	}
	if err := New(r).Cancel(context.Background(), uuid.New(), r.request.ID); err == nil {
		t.Fatal("cancelled request must not be cancelled again")
	}
}
func TestListPageClampsPagination(t *testing.T) {
	r := &lifecycleRepo{rows: []model.VerificationRequest{{ID: uuid.New()}}, total: 1}
	page, err := New(r).ListPage(context.Background(), "", nil, nil, 500, -1)
	if err != nil {
		t.Fatal(err)
	}
	if page.Pagination.Limit != 100 || page.Pagination.Offset != 0 || page.Pagination.Total != 1 {
		t.Fatalf("unexpected page %#v", page.Pagination)
	}
}
