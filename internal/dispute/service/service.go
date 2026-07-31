package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"modalin-be/internal/model"
	"modalin-be/pkg/audit"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrNotFound    = errors.New("dispute not found")
	ErrForbidden   = errors.New("not allowed to access this campaign dispute")
	ErrInvalid     = errors.New("invalid dispute")
	ErrUnavailable = errors.New("dispute transition unavailable")
)

type Repository interface {
	WithTransaction(context.Context, func(Repository) error) error
	Campaign(context.Context, uuid.UUID) (*model.LoanCampaign, error)
	HasPaidFunding(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	Create(context.Context, *model.Dispute) error
	GetForUpdate(context.Context, uuid.UUID) (*model.Dispute, error)
	Update(context.Context, *model.Dispute) error
	List(context.Context, uuid.UUID, bool, string, int, int) ([]model.Dispute, int64, error)
	CreateAuditLog(context.Context, *model.AuditLog) error
}
type Service struct{ repo Repository }

func New(repo Repository) *Service { return &Service{repo: repo} }

type CreateInput struct {
	TargetUserID *uuid.UUID `json:"target_user_id"`
	Type         string     `json:"type"`
	Description  string     `json:"description"`
}

func (s *Service) Create(ctx context.Context, userID, campaignID uuid.UUID, in CreateInput) (*model.Dispute, error) {
	if !validType(in.Type) || strings.TrimSpace(in.Description) == "" || len(strings.TrimSpace(in.Description)) > 4000 || (in.TargetUserID != nil && *in.TargetUserID == userID) {
		return nil, ErrInvalid
	}
	if err := s.participant(ctx, userID, campaignID); err != nil {
		return nil, err
	}
	if in.TargetUserID != nil {
		if err := s.participant(ctx, *in.TargetUserID, campaignID); err != nil {
			return nil, ErrInvalid
		}
	}
	d := &model.Dispute{CampaignID: campaignID, ReportedBy: userID, TargetUserID: in.TargetUserID, Type: strings.ToLower(strings.TrimSpace(in.Type)), Description: strings.TrimSpace(in.Description), Status: "open"}
	err := s.repo.WithTransaction(ctx, func(r Repository) error {
		if err := r.Create(ctx, d); err != nil {
			return err
		}
		return auditLog(ctx, r, &userID, "dispute.created", d.ID, nil, map[string]any{"campaign_id": campaignID, "type": d.Type, "status": d.Status})
	})
	if err != nil {
		return nil, err
	}
	return d, nil
}
func (s *Service) Review(ctx context.Context, adminID, id uuid.UUID) error {
	return s.transition(ctx, adminID, id, "under_review", "")
}
func (s *Service) Decide(ctx context.Context, adminID, id uuid.UUID, decision, note string) error {
	decision = strings.ToLower(strings.TrimSpace(decision))
	if (decision != "resolved" && decision != "rejected") || strings.TrimSpace(note) == "" {
		return ErrInvalid
	}
	return s.transition(ctx, adminID, id, decision, note)
}
func (s *Service) transition(ctx context.Context, adminID, id uuid.UUID, status, note string) error {
	return s.repo.WithTransaction(ctx, func(r Repository) error {
		d, err := r.GetForUpdate(ctx, id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if status == "under_review" {
			if d.Status != "open" {
				return ErrUnavailable
			}
		} else if d.Status != "open" && d.Status != "under_review" {
			return ErrUnavailable
		}
		old := d.Status
		d.Status = status
		if status == "resolved" || status == "rejected" {
			now := time.Now().UTC()
			d.ResolvedBy = &adminID
			d.ResolvedAt = &now
			n := strings.TrimSpace(note)
			d.ResolutionNote = &n
		}
		if err = r.Update(ctx, d); err != nil {
			return err
		}
		return auditLog(ctx, r, &adminID, "dispute."+status, d.ID, map[string]any{"status": old}, map[string]any{"status": status})
	})
}
func (s *Service) List(ctx context.Context, userID uuid.UUID, admin bool, status string, limit, offset int) ([]model.Dispute, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.repo.List(ctx, userID, admin, status, limit, offset)
}
func (s *Service) participant(ctx context.Context, user, campaign uuid.UUID) error {
	c, err := s.repo.Campaign(ctx, campaign)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if c.Business.UserID == user {
		return nil
	}
	ok, err := s.repo.HasPaidFunding(ctx, campaign, user)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}
func validType(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "fraud_suspected", "late_payment", "invalid_proof", "misuse_of_funds", "other":
		return true
	}
	return false
}
func auditLog(ctx context.Context, r Repository, user *uuid.UUID, action string, id uuid.UUID, old, new any) error {
	encode := func(value any) *string {
		if value == nil {
			return nil
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil
		}
		encoded := string(data)
		return &encoded
	}
	return r.CreateAuditLog(ctx, audit.New(ctx, user, action, "dispute", id, encode(old), encode(new)))
}
