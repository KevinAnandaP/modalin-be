package service_test

import (
	"context"
	"errors"
	"testing"

	"modalin-be/internal/campaign/repository"
	"modalin-be/internal/campaign/service"
	"modalin-be/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type fakeCampaignRepo struct {
	business   *model.Business
	campaigns  map[uuid.UUID]*model.LoanCampaign
	budgets    map[uuid.UUID][]model.CampaignBudgetItem
	milestones map[uuid.UUID][]model.CampaignMilestone
}

func (r *fakeCampaignRepo) GetBusinessByUserID(_ context.Context, userID uuid.UUID) (*model.Business, error) {
	if r.business != nil && r.business.UserID == userID {
		return r.business, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) CreateCampaign(_ context.Context, c *model.LoanCampaign) error {
	c.ID = uuid.New()
	r.campaigns[c.ID] = c
	return nil
}
func (r *fakeCampaignRepo) GetCampaignByID(_ context.Context, id uuid.UUID) (*model.LoanCampaign, error) {
	if c := r.campaigns[id]; c != nil {
		return c, nil
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) GetCampaignForBusiness(_ context.Context, id, businessID uuid.UUID) (*model.LoanCampaign, error) {
	c, err := r.GetCampaignByID(context.Background(), id)
	if err != nil || c.BusinessID != businessID {
		return nil, gorm.ErrRecordNotFound
	}
	return c, nil
}
func (r *fakeCampaignRepo) UpdateCampaign(_ context.Context, c *model.LoanCampaign) error {
	r.campaigns[c.ID] = c
	return nil
}
func (r *fakeCampaignRepo) DeleteCampaign(_ context.Context, id uuid.UUID) error {
	delete(r.campaigns, id)
	return nil
}
func (r *fakeCampaignRepo) ListCampaignsForBusiness(_ context.Context, businessID uuid.UUID) ([]model.LoanCampaign, error) {
	out := []model.LoanCampaign{}
	for _, c := range r.campaigns {
		if c.BusinessID == businessID {
			out = append(out, *c)
		}
	}
	return out, nil
}
func (r *fakeCampaignRepo) CreateBudgetItem(_ context.Context, b *model.CampaignBudgetItem) error {
	b.ID = uuid.New()
	r.budgets[b.CampaignID] = append(r.budgets[b.CampaignID], *b)
	return nil
}
func (r *fakeCampaignRepo) GetBudgetItem(_ context.Context, id, campaignID uuid.UUID) (*model.CampaignBudgetItem, error) {
	for i := range r.budgets[campaignID] {
		if r.budgets[campaignID][i].ID == id {
			return &r.budgets[campaignID][i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateBudgetItem(_ context.Context, b *model.CampaignBudgetItem) error {
	for i := range r.budgets[b.CampaignID] {
		if r.budgets[b.CampaignID][i].ID == b.ID {
			r.budgets[b.CampaignID][i] = *b
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) DeleteBudgetItem(_ context.Context, id, campaignID uuid.UUID) error {
	for i := range r.budgets[campaignID] {
		if r.budgets[campaignID][i].ID == id {
			r.budgets[campaignID] = append(r.budgets[campaignID][:i], r.budgets[campaignID][i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) ListBudgetItems(_ context.Context, campaignID uuid.UUID) ([]model.CampaignBudgetItem, error) {
	return r.budgets[campaignID], nil
}
func (r *fakeCampaignRepo) CountBudgetItems(_ context.Context, campaignID uuid.UUID) (int, error) {
	return len(r.budgets[campaignID]), nil
}
func (r *fakeCampaignRepo) CreateMilestone(_ context.Context, m *model.CampaignMilestone) error {
	m.ID = uuid.New()
	r.milestones[m.CampaignID] = append(r.milestones[m.CampaignID], *m)
	return nil
}
func (r *fakeCampaignRepo) GetMilestone(_ context.Context, id, campaignID uuid.UUID) (*model.CampaignMilestone, error) {
	for i := range r.milestones[campaignID] {
		if r.milestones[campaignID][i].ID == id {
			return &r.milestones[campaignID][i], nil
		}
	}
	return nil, gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) UpdateMilestone(_ context.Context, m *model.CampaignMilestone) error {
	for i := range r.milestones[m.CampaignID] {
		if r.milestones[m.CampaignID][i].ID == m.ID {
			r.milestones[m.CampaignID][i] = *m
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) DeleteMilestone(_ context.Context, id, campaignID uuid.UUID) error {
	for i := range r.milestones[campaignID] {
		if r.milestones[campaignID][i].ID == id {
			r.milestones[campaignID] = append(r.milestones[campaignID][:i], r.milestones[campaignID][i+1:]...)
			return nil
		}
	}
	return gorm.ErrRecordNotFound
}
func (r *fakeCampaignRepo) ListMilestones(_ context.Context, campaignID uuid.UUID) ([]model.CampaignMilestone, error) {
	return r.milestones[campaignID], nil
}
func (r *fakeCampaignRepo) ReplaceMilestones(_ context.Context, campaignID uuid.UUID, milestones []model.CampaignMilestone) error {
	for i := range milestones {
		milestones[i].ID = uuid.New()
	}
	r.milestones[campaignID] = milestones
	return nil
}
func (r *fakeCampaignRepo) ListCatalog(_ context.Context, _ repository.CatalogFilter) ([]model.LoanCampaign, error) {
	return nil, nil
}

func newRepo(userID uuid.UUID, limit int64) *fakeCampaignRepo {
	return &fakeCampaignRepo{business: &model.Business{ID: uuid.New(), UserID: userID, CurrentBorrowingLimit: limit, Status: "active"}, campaigns: map[uuid.UUID]*model.LoanCampaign{}, budgets: map[uuid.UUID][]model.CampaignBudgetItem{}, milestones: map[uuid.UUID][]model.CampaignMilestone{}}
}

func TestCampaignSubmissionRequiresLimitAndCompletePlan(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	_, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Modal kios", Description: "Menambah stok", RequestedAmount: 1_500_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	if !errors.Is(err, service.ErrAmountExceedsLimit) {
		t.Fatalf("expected tier limit rejection, got %v", err)
	}
	campaign, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Modal kios", Description: "Menambah stok", RequestedAmount: 1_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); !errors.Is(err, service.ErrIncompleteCampaignPlan) {
		t.Fatalf("expected incomplete plan error, got %v", err)
	}
	_, _ = svc.CreateBudgetItem(context.Background(), userID, campaign.ID, service.BudgetItemInput{ItemName: "Stok", Category: "stock", Amount: 1_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"})
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	if campaign.Status != "admin_review" {
		t.Fatalf("expected admin_review, got %s", campaign.Status)
	}
}

func TestCampaignReviewPublishesOnlyCompleteReview(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	campaign, _ := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Modal kios", Description: "Menambah stok", RequestedAmount: 1_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	campaign.Status = "admin_review"
	adminID := uuid.New()
	if err := svc.ReviewCampaign(context.Background(), adminID, campaign.ID, "publish"); !errors.Is(err, service.ErrIncompleteCampaignPlan) {
		t.Fatalf("expected complete-plan review rejection, got %v", err)
	}
	repo.budgets[campaign.ID] = []model.CampaignBudgetItem{{CampaignID: campaign.ID, Amount: 1_000_000, PriorityLevel: "high"}}
	repo.milestones[campaign.ID] = []model.CampaignMilestone{{CampaignID: campaign.ID, Amount: 1_000_000, SequenceNo: 1}}
	if err := svc.ReviewCampaign(context.Background(), adminID, campaign.ID, "publish"); err != nil {
		t.Fatal(err)
	}
	if campaign.Status != "published" || campaign.ApprovedBy == nil || *campaign.ApprovedBy != adminID {
		t.Fatalf("campaign was not published by admin: %#v", campaign)
	}
}

func TestSubmissionBuildsMilestonesFromBudgetPriorityLevel(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 10_000_000)
	svc := service.NewCampaignService(repo)
	campaign, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Peralatan usaha", Description: "Tiga barang", RequestedAmount: 10_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []service.BudgetItemInput{
		{ItemName: "Mesin kopi", Category: "equipment", Amount: 7_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"},
		{ItemName: "Bahan baku", Category: "stock", Amount: 2_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "medium"},
		{ItemName: "Kemasan", Category: "packaging", Amount: 1_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "low"},
	} {
		if _, err := svc.CreateBudgetItem(context.Background(), userID, campaign.ID, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	milestones := repo.milestones[campaign.ID]
	if len(milestones) != 3 {
		t.Fatalf("expected one milestone per budget item, got %d", len(milestones))
	}
	if milestones[0].Title != "Mesin kopi" || milestones[0].SequenceNo != 1 || milestones[0].Status != "available" {
		t.Fatalf("highest-priority item should be the first available milestone: %#v", milestones[0])
	}
	if milestones[1].Title != "Bahan baku" || milestones[1].SequenceNo != 2 || milestones[1].Status != "locked" {
		t.Fatalf("second-priority item should remain locked: %#v", milestones[1])
	}
	if milestones[2].Title != "Kemasan" || milestones[2].SequenceNo != 3 || milestones[2].Status != "locked" {
		t.Fatalf("third-priority item should remain locked: %#v", milestones[2])
	}
}

func TestSamePriorityUsesBorrowerInputOrder(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	campaign, err := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Peralatan", Description: "Urutan sama", RequestedAmount: 300_000, LoanTenorMonths: 3, BenefitType: "principal_only"})
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range []service.BudgetItemInput{
		{ItemName: "Barang pertama", Category: "equipment", Amount: 100_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"},
		{ItemName: "Barang kedua", Category: "equipment", Amount: 100_000, PurchaseMethod: "direct_purchase", PriorityLevel: "high"},
		{ItemName: "Barang ketiga", Category: "equipment", Amount: 100_000, PurchaseMethod: "direct_purchase", PriorityLevel: "low"},
	} {
		if _, err := svc.CreateBudgetItem(context.Background(), userID, campaign.ID, item); err != nil {
			t.Fatal(err)
		}
	}
	if err := svc.SubmitCampaign(context.Background(), userID, campaign.ID); err != nil {
		t.Fatal(err)
	}
	if got := repo.milestones[campaign.ID]; got[0].Title != "Barang pertama" || got[1].Title != "Barang kedua" {
		t.Fatalf("same-priority milestones must preserve input order: %#v", got)
	}
}

func TestCreateBudgetItemRejectsInvalidPriorityLevel(t *testing.T) {
	userID := uuid.New()
	repo := newRepo(userID, 1_000_000)
	svc := service.NewCampaignService(repo)
	campaign, _ := svc.CreateCampaign(context.Background(), userID, service.CampaignInput{Title: "Stok", Description: "Stok usaha", RequestedAmount: 1_000_000, LoanTenorMonths: 6, BenefitType: "principal_only"})
	_, err := svc.CreateBudgetItem(context.Background(), userID, campaign.ID, service.BudgetItemInput{ItemName: "Stok", Category: "stock", Amount: 1_000_000, PurchaseMethod: "direct_purchase", PriorityLevel: "urgent"})
	if !errors.Is(err, service.ErrInvalidBudgetItem) {
		t.Fatalf("expected invalid priority level error, got %v", err)
	}
}
