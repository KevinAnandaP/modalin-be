package service

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"modalin-be/internal/model"
	"modalin-be/internal/verification/repository"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound           = errors.New("verification request not found")
	ErrUnavailable        = errors.New("verification operation is unavailable")
	ErrIneligibleCampaign = errors.New("field verification is only available for tier 3 or tier 4 campaigns")
	ErrConflictOfInterest = errors.New("verifier must not be the borrower or a lender of this campaign")
	ErrInvalidReport      = errors.New("verification report is incomplete or invalid")
	ErrInvalidDecision    = errors.New("verification decision must be approve or reject")
	ErrInvalidVote        = errors.New("vote type must be up or down")
	ErrSelfVote           = errors.New("business owner cannot vote on their own business")
)

type Service struct{ repo repository.Repository }

func New(repo repository.Repository) *Service { return &Service{repo: repo} }

type ReportInput struct {
	IsBusinessExists bool   `json:"is_business_exists"`
	IsBusinessActive bool   `json:"is_business_active"`
	LocationMatch    bool   `json:"location_match"`
	PhotoURL         string `json:"photo_url"`
	Note             string `json:"note"`
	Recommendation   string `json:"recommendation"`
}
type RiskInput struct{ FinancialScore, VerificationScore, RepaymentScore, CommunityScore int }
type RiskResult struct {
	FinancialScore    int
	VerificationScore int
	RepaymentScore    int
	CommunityScore    int
	FinalScore        int
	RiskLevel         string
	DataLimited       bool
	MissingComponents []string
}
type RequestPage struct {
	Data       []model.VerificationRequest `json:"data"`
	Pagination Pagination                  `json:"pagination"`
}
type Pagination struct {
	Limit  int   `json:"limit"`
	Offset int   `json:"offset"`
	Total  int64 `json:"total"`
}

func CalculateRisk(in RiskInput) RiskResult {
	normalize := func(v int) int {
		if v == 0 {
			return 50
		}
		if v < 0 {
			return 0
		}
		if v > 100 {
			return 100
		}
		return v
	}
	f, v, r, c := normalize(in.FinancialScore), normalize(in.VerificationScore), normalize(in.RepaymentScore), normalize(in.CommunityScore)
	final := int(math.Round(float64(f)*.30 + float64(v)*.30 + float64(r)*.25 + float64(c)*.15))
	level := "high"
	if final >= 80 {
		level = "low"
	} else if final >= 60 {
		level = "medium"
	}
	missing := make([]string, 0, 4)
	if in.FinancialScore == 0 {
		missing = append(missing, "financial")
	}
	if in.VerificationScore == 0 {
		missing = append(missing, "verification")
	}
	if in.RepaymentScore == 0 {
		missing = append(missing, "repayment")
	}
	if in.CommunityScore == 0 {
		missing = append(missing, "community")
	}
	return RiskResult{FinancialScore: f, VerificationScore: v, RepaymentScore: r, CommunityScore: c, FinalScore: final, RiskLevel: level, DataLimited: len(missing) > 0, MissingComponents: missing}
}

func (s *Service) CreateRequest(ctx context.Context, borrowerID, campaignID uuid.UUID) (*model.VerificationRequest, error) {
	var request *model.VerificationRequest
	err := s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		campaign, err := repo.GetCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		business, err := repo.GetBusiness(ctx, campaign.BusinessID)
		if err != nil {
			return err
		}
		if business.UserID != borrowerID {
			return ErrNotFound
		}
		if campaign.Status != "admin_review" {
			return ErrUnavailable
		}
		if business.CurrentBorrowingLimit < 5000000 {
			return ErrIneligibleCampaign
		}
		active, err := repo.HasActiveVerificationRequest(ctx, campaignID)
		if err != nil {
			return err
		}
		if active {
			return ErrUnavailable
		}
		request = &model.VerificationRequest{CampaignID: &campaignID, BusinessID: campaign.BusinessID, RequestedBy: borrowerID, Status: "pending"}
		if err := repo.CreateRequest(ctx, request); err != nil {
			return err
		}
		return audit(ctx, repo, &borrowerID, "verification_request.created", "verification_request", request.ID, nil, map[string]any{"campaign_id": campaignID, "status": "pending"})
	})
	return request, err
}
func (s *Service) Assign(ctx context.Context, adminID, requestID, verifierID uuid.UUID) error {
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		req, err := repo.GetRequestForUpdate(ctx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if req.Status != "pending" || req.CampaignID == nil {
			return ErrUnavailable
		}
		if err := s.ensureIndependent(ctx, repo, *req.CampaignID, req.BusinessID, verifierID); err != nil {
			return err
		}
		req.AssignedVerifierID = &verifierID
		req.Status = "assigned"
		if err := repo.UpdateRequest(ctx, req); err != nil {
			return err
		}
		return audit(ctx, repo, &adminID, "verification_request.assigned", "verification_request", req.ID, map[string]any{"status": "pending"}, map[string]any{"status": "assigned", "verifier_id": verifierID})
	})
}
func (s *Service) Reassign(ctx context.Context, adminID, requestID, verifierID uuid.UUID) error {
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		req, err := repo.GetRequestForUpdate(ctx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if req.Status != "assigned" || req.CampaignID == nil {
			return ErrUnavailable
		}
		if err := s.ensureIndependent(ctx, repo, *req.CampaignID, req.BusinessID, verifierID); err != nil {
			return err
		}
		old := req.AssignedVerifierID
		req.AssignedVerifierID = &verifierID
		if err := repo.UpdateRequest(ctx, req); err != nil {
			return err
		}
		return audit(ctx, repo, &adminID, "verification_request.reassigned", "verification_request", req.ID, map[string]any{"verifier_id": old}, map[string]any{"verifier_id": verifierID})
	})
}
func (s *Service) Cancel(ctx context.Context, adminID, requestID uuid.UUID) error {
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		req, err := repo.GetRequestForUpdate(ctx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if req.Status != "pending" && req.Status != "assigned" {
			return ErrUnavailable
		}
		old := req.Status
		req.Status = "cancelled"
		if err := repo.UpdateRequest(ctx, req); err != nil {
			return err
		}
		return audit(ctx, repo, &adminID, "verification_request.cancelled", "verification_request", req.ID, map[string]any{"status": old}, map[string]any{"status": "cancelled"})
	})
}
func (s *Service) SubmitReport(ctx context.Context, verifierID, requestID uuid.UUID, in ReportInput) (*model.VerificationReport, error) {
	if !validReport(in) {
		return nil, ErrInvalidReport
	}
	var report *model.VerificationReport
	err := s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		req, err := repo.GetRequestForUpdate(ctx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if req.Status != "assigned" || req.AssignedVerifierID == nil || *req.AssignedVerifierID != verifierID || req.CampaignID == nil {
			return ErrUnavailable
		}
		if err := s.ensureIndependent(ctx, repo, *req.CampaignID, req.BusinessID, verifierID); err != nil {
			return err
		}
		exists, err := repo.HasReport(ctx, req.ID)
		if err != nil {
			return err
		}
		if exists {
			return ErrUnavailable
		}
		report = &model.VerificationReport{VerificationRequestID: req.ID, VerifierUserID: verifierID, IsBusinessExists: in.IsBusinessExists, IsBusinessActive: in.IsBusinessActive, LocationMatch: in.LocationMatch, PhotoURL: strings.TrimSpace(in.PhotoURL), Note: strings.TrimSpace(in.Note), Recommendation: strings.ToLower(strings.TrimSpace(in.Recommendation))}
		if err := repo.CreateReport(ctx, report); err != nil {
			return err
		}
		req.Status = "reviewed"
		if err := repo.UpdateRequest(ctx, req); err != nil {
			return err
		}
		return audit(ctx, repo, &verifierID, "verification_report.submitted", "verification_report", report.ID, nil, map[string]any{"request_id": req.ID, "recommendation": report.Recommendation})
	})
	return report, err
}
func (s *Service) CanUploadPhoto(ctx context.Context, verifierID, requestID uuid.UUID) error {
	req, err := s.repo.GetRequest(ctx, requestID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if req.Status != "assigned" || req.AssignedVerifierID == nil || *req.AssignedVerifierID != verifierID {
		return ErrUnavailable
	}
	return nil
}
func (s *Service) PhotoURL(ctx context.Context, userID, requestID uuid.UUID, roles []string) (string, error) {
	req, err := s.repo.GetRequest(ctx, requestID)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	admin := false
	for _, role := range roles {
		if role == "admin" {
			admin = true
			break
		}
	}
	business, err := s.repo.GetBusiness(ctx, req.BusinessID)
	if err != nil {
		return "", err
	}
	if !admin && business.UserID != userID && (req.AssignedVerifierID == nil || *req.AssignedVerifierID != userID) {
		return "", ErrConflictOfInterest
	}
	report, err := s.repo.LatestReport(ctx, req.ID)
	if err != nil {
		return "", ErrNotFound
	}
	return report.PhotoURL, nil
}
func (s *Service) Decide(ctx context.Context, adminID, requestID uuid.UUID, decision string) error {
	decision = strings.ToLower(strings.TrimSpace(decision))
	if decision != "approve" && decision != "reject" {
		return ErrInvalidDecision
	}
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		req, err := repo.GetRequestForUpdate(ctx, requestID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if req.Status != "reviewed" {
			return ErrUnavailable
		}
		report, err := repo.LatestReport(ctx, req.ID)
		if err != nil {
			return err
		}
		business, err := repo.GetBusiness(ctx, req.BusinessID)
		if err != nil {
			return err
		}
		req.Status = map[bool]string{true: "approved", false: "rejected"}[decision == "approve"]
		if err := repo.UpdateRequest(ctx, req); err != nil {
			return err
		}
		if decision == "approve" {
			business.VerificationStatus = "verified"
		} else {
			business.VerificationStatus = "rejected"
			if req.CampaignID != nil {
				campaign, err := repo.GetCampaignForUpdate(ctx, *req.CampaignID)
				if err != nil {
					return err
				}
				if campaign.Status == "admin_review" {
					campaign.Status = "rejected"
					if err := repo.UpdateCampaign(ctx, campaign); err != nil {
						return err
					}
				}
			}
		}
		if err := repo.UpdateBusiness(ctx, business); err != nil {
			return err
		}
		if req.CampaignID != nil {
			if _, err := s.assess(ctx, repo, *req.CampaignID, req.BusinessID, report); err != nil {
				return err
			}
		}
		return audit(ctx, repo, &adminID, "verification_request.decided", "verification_request", req.ID, map[string]any{"status": "reviewed"}, map[string]any{"status": req.Status})
	})
}
func (s *Service) Vote(ctx context.Context, userID, businessID uuid.UUID, voteType string, reason *string) error {
	voteType = strings.ToLower(strings.TrimSpace(voteType))
	if voteType != "up" && voteType != "down" {
		return ErrInvalidVote
	}
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		business, err := repo.GetBusiness(ctx, businessID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if business.UserID == userID {
			return ErrSelfVote
		}
		active, err := repo.HasAnyActiveRole(ctx, userID)
		if err != nil {
			return err
		}
		assigned, err := repo.IsAssignedVerifier(ctx, businessID, userID)
		if err != nil {
			return err
		}
		if !active || assigned {
			return ErrConflictOfInterest
		}
		vote := &model.CommunityVote{BusinessID: businessID, UserID: userID, VoteType: voteType, Reason: reason}
		if err := repo.UpsertVote(ctx, vote); err != nil {
			return err
		}
		campaigns, err := s.latestCampaign(ctx, repo, businessID)
		if err == nil && campaigns != uuid.Nil {
			_, err = s.assess(ctx, repo, campaigns, businessID, nil)
			if err != nil {
				return err
			}
		}
		return audit(ctx, repo, &userID, "community_vote.upserted", "community_vote", vote.ID, nil, map[string]any{"business_id": businessID, "vote_type": voteType})
	})
}
func (s *Service) List(ctx context.Context, status string, verifier *uuid.UUID) ([]model.VerificationRequest, error) {
	return s.repo.ListRequests(ctx, status, verifier)
}
func (s *Service) ListPage(ctx context.Context, status string, verifier, campaignID *uuid.UUID, limit, offset int) (*RequestPage, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	rows, total, err := s.repo.ListRequestsPage(ctx, repository.RequestFilter{Status: status, VerifierID: verifier, CampaignID: campaignID, Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	return &RequestPage{Data: rows, Pagination: Pagination{Limit: limit, Offset: offset, Total: total}}, nil
}
func (s *Service) RefreshCampaignRisk(ctx context.Context, campaignID uuid.UUID) error {
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		campaign, err := repo.GetCampaignForUpdate(ctx, campaignID)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		_, err = s.assess(ctx, repo, campaignID, campaign.BusinessID, nil)
		return err
	})
}
func (s *Service) RefreshBusinessRisk(ctx context.Context, businessID uuid.UUID) error {
	return s.repo.WithTransaction(ctx, func(repo repository.Repository) error {
		business, err := repo.GetBusiness(ctx, businessID)
		if err != nil {
			return err
		}
		campaigns, err := repo.ListActiveCampaignsForBusiness(ctx, businessID)
		if err != nil {
			return err
		}
		if len(campaigns) == 0 {
			monthsRecords, countErr := repo.CountFinancialMonths(ctx, businessID)
			if countErr != nil {
				return countErr
			}
			business.TrustScore = 0
			business.CurrentBorrowingLimit = tierLimit(business, monthsRecords, 0)
			return repo.UpdateBusiness(ctx, business)
		}
		for _, campaign := range campaigns {
			if _, err := s.assess(ctx, repo, campaign.ID, businessID, nil); err != nil {
				return err
			}
		}
		paid, late, total, err := repo.RepaymentStats(ctx, campaigns[0].ID)
		if err != nil {
			return err
		}
		repaymentScore := 0
		if total > 0 {
			repaymentScore = int((paid*100)/total) - int((late*25)/total)
			if repaymentScore < 0 {
				repaymentScore = 0
			}
		}
		monthsRecords, err := repo.CountFinancialMonths(ctx, businessID)
		if err != nil {
			return err
		}
		business.TrustScore = repaymentScore
		business.CurrentBorrowingLimit = tierLimit(business, monthsRecords, repaymentScore)
		return repo.UpdateBusiness(ctx, business)
	})
}
func (s *Service) BackfillActiveRisks(ctx context.Context) (int, error) {
	campaigns, err := s.repo.ListActiveCampaigns(ctx)
	if err != nil {
		return 0, err
	}
	for _, campaign := range campaigns {
		if err := s.RefreshCampaignRisk(ctx, campaign.ID); err != nil {
			return 0, err
		}
	}
	return len(campaigns), nil
}
func tierLimit(b *model.Business, monthsRecords, repaymentScore int) int64 {
	if b.BusinessType == "starter" {
		if monthsRecords >= 3 {
			return 1500000
		}
		if monthsRecords >= 2 {
			return 1000000
		}
		return 300000
	}
	months := 0
	if b.StartedAt != nil {
		now := time.Now().UTC()
		months = (now.Year()-b.StartedAt.Year())*12 + int(now.Month()-b.StartedAt.Month())
		if months < 0 {
			months = 0
		}
	}
	if months >= 10 && monthsRecords >= 10 && repaymentScore >= 80 {
		return 15000000
	}
	if months >= 5 && monthsRecords >= 5 {
		return 5000000
	}
	if months >= 3 && monthsRecords >= 2 {
		return 3000000
	}
	return 1000000
}
func (s *Service) ensureIndependent(ctx context.Context, repo repository.Repository, campaignID, businessID, verifierID uuid.UUID) error {
	business, err := repo.GetBusiness(ctx, businessID)
	if err != nil {
		return err
	}
	active, err := repo.HasActiveRole(ctx, verifierID, "verifier")
	if err != nil {
		return err
	}
	funded, err := repo.HasFunding(ctx, campaignID, verifierID)
	if err != nil {
		return err
	}
	if !active || business.UserID == verifierID || funded {
		return ErrConflictOfInterest
	}
	return nil
}
func validReport(in ReportInput) bool {
	if strings.TrimSpace(in.PhotoURL) == "" || strings.TrimSpace(in.Note) == "" {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(in.Recommendation)) {
	case "approve", "review", "reject":
		return true
	}
	return false
}
func (s *Service) latestCampaign(ctx context.Context, repo repository.Repository, businessID uuid.UUID) (uuid.UUID, error) {
	return repo.LatestCampaignID(ctx, businessID)
}
func (s *Service) assess(ctx context.Context, repo repository.Repository, campaignID, businessID uuid.UUID, report *model.VerificationReport) (*model.RiskAssessment, error) {
	if report == nil {
		latest, err := repo.LatestReportForCampaign(ctx, campaignID)
		if err == nil {
			report = latest
		} else if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
	}
	financialCount, err := repo.CountFinancialRecords(ctx, businessID)
	if err != nil {
		return nil, err
	}
	paid, late, total, err := repo.RepaymentStats(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	up, down, err := repo.VoteStats(ctx, businessID)
	if err != nil {
		return nil, err
	}
	verification := 0
	if report != nil {
		verification = boolScore(report.IsBusinessExists, report.IsBusinessActive, report.LocationMatch, report.Recommendation)
	}
	financial := 0
	if financialCount > 0 {
		financial = int(financialCount * 10)
		if financial > 100 {
			financial = 100
		}
	}
	repayment := 0
	if total > 0 {
		repayment = int((paid*100)/total) - int((late*25)/total)
		if repayment < 0 {
			repayment = 0
		}
	}
	community := 0
	if up+down > 0 {
		community = int((up * 100) / (up + down))
	}
	risk := CalculateRisk(RiskInput{financial, verification, repayment, community})
	note := ""
	if risk.DataLimited {
		note = "data_limited"
	}
	fingerprint := riskFingerprint(campaignID, businessID, financialCount, paid, late, total, up, down, report)
	latest, latestErr := repo.LatestRiskAssessment(ctx, campaignID)
	if latestErr == nil && latest.SourceFingerprint == fingerprint {
		return latest, nil
	}
	if latestErr != nil && !errors.Is(latestErr, gorm.ErrRecordNotFound) {
		return nil, latestErr
	}
	a := &model.RiskAssessment{CampaignID: campaignID, BusinessID: businessID, FinancialScore: risk.FinancialScore, VerificationScore: risk.VerificationScore, RepaymentScore: risk.RepaymentScore, CommunityScore: risk.CommunityScore, FinalScore: risk.FinalScore, RiskLevel: risk.RiskLevel, RiskFormulaVersion: "v1", SourceFingerprint: fingerprint, DataLimited: risk.DataLimited, MissingComponentsRaw: strings.Join(risk.MissingComponents, ","), MissingComponents: risk.MissingComponents, Note: &note}
	if err := repo.CreateRiskAssessment(ctx, a); err != nil {
		return nil, err
	}
	campaign, err := repo.GetCampaignForUpdate(ctx, campaignID)
	if err != nil {
		return nil, err
	}
	campaign.RiskLevel = risk.RiskLevel
	if err := repo.UpdateCampaign(ctx, campaign); err != nil {
		return nil, err
	}
	return a, nil
}
func riskFingerprint(campaignID, businessID uuid.UUID, financial, paid, late, total, up, down int64, report *model.VerificationReport) string {
	reportID := ""
	if report != nil {
		reportID = report.ID.String()
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("v1|%s|%s|%d|%d|%d|%d|%d|%d|%s", campaignID, businessID, financial, paid, late, total, up, down, reportID)))
	return fmt.Sprintf("%x", sum)
}
func boolScore(a, b, c bool, recommendation string) int {
	score := 0
	if a {
		score += 25
	}
	if b {
		score += 25
	}
	if c {
		score += 20
	}
	switch recommendation {
	case "approve":
		score += 30
	case "review":
		score += 15
	}
	return score
}
func audit(ctx context.Context, repo repository.Repository, userID *uuid.UUID, action, entity string, id uuid.UUID, oldValue, newValue any) error {
	encode := func(v any) *string {
		if v == nil {
			return nil
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil
		}
		s := string(b)
		return &s
	}
	return repo.CreateAuditLog(ctx, &model.AuditLog{UserID: userID, Action: action, EntityType: entity, EntityID: id, OldValue: encode(oldValue), NewValue: encode(newValue), CreatedAt: time.Now().UTC()})
}
