package service

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"modalin-be/internal/model"
	"modalin-be/internal/restructuring/repository"
	"modalin-be/pkg/audit"
	"strings"
	"time"
)

var (
	ErrNotFound    = errors.New("restructuring request not found")
	ErrForbidden   = errors.New("not allowed")
	ErrInvalid     = errors.New("invalid restructuring request")
	ErrUnavailable = errors.New("restructuring unavailable")
)

type Service struct{ r *repository.Repo }

func New(r *repository.Repo) *Service { return &Service{r} }

type Input struct {
	Reason              string  `json:"reason"`
	ProposedTenorMonths int     `json:"proposed_tenor_months"`
	ProofURL            *string `json:"proof_url"`
}

func (s *Service) Create(c context.Context, u, cid uuid.UUID, in Input) (*model.RepaymentRestructuringRequest, error) {
	if strings.TrimSpace(in.Reason) == "" || in.ProposedTenorMonths < 1 || in.ProposedTenorMonths > 12 {
		return nil, ErrInvalid
	}
	var out *model.RepaymentRestructuringRequest
	e := s.r.Tx(c, func(r *repository.Repo) error {
		camp, e := r.Campaign(c, cid)
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if camp.Business.UserID != u || camp.Status != "active" {
			return ErrForbidden
		}
		ss, e := r.Schedules(c, cid)
		if e != nil {
			return e
		}
		n := 0
		p, m := int64(0), int64(0)
		for _, x := range ss {
			if x.Status != "paid" {
				n++
				p += x.PrincipalDue
				if x.MarginDue != nil {
					m += *x.MarginDue
				}
			}
		}
		if n == 0 || in.ProposedTenorMonths <= n {
			return ErrInvalid
		}
		out = &model.RepaymentRestructuringRequest{CampaignID: cid, RequestedBy: u, Reason: strings.TrimSpace(in.Reason), ProposedTenorMonths: in.ProposedTenorMonths, PreviousTenorMonths: n, RemainingPrincipal: p, RemainingMargin: m, ProofURL: in.ProofURL, Status: "pending"}
		if e := r.Create(c, out); e != nil {
			return e
		}
		return r.Audit(c, audit.New(c, &u, "repayment_restructuring_request.created", "repayment_restructuring_request", out.ID, nil, nil))
	})
	return out, e
}
func (s *Service) Decide(c context.Context, a, id uuid.UUID, d, note string) error {
	d = strings.ToLower(strings.TrimSpace(d))
	if (d != "approve" && d != "reject") || strings.TrimSpace(note) == "" {
		return ErrInvalid
	}
	return s.r.Tx(c, func(r *repository.Repo) error {
		q, e := r.Get(c, id)
		if errors.Is(e, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if e != nil {
			return e
		}
		if q.Status != "pending" {
			return ErrUnavailable
		}
		now := time.Now().UTC()
		q.ApprovedBy = &a
		q.ApprovedAt = &now
		q.AdminNote = &note
		if d == "reject" {
			q.Status = "rejected"
			if e := r.SaveRequest(c, q); e != nil {
				return e
			}
			return r.Audit(c, audit.New(c, &a, "repayment_restructuring_request.rejected", "repayment_restructuring_request", q.ID, nil, nil))
		}
		active, e := r.ActiveDispute(c, q.CampaignID)
		if e != nil {
			return e
		}
		if active {
			return ErrUnavailable
		}
		pending, e := r.PendingRepayment(c, q.CampaignID)
		if e != nil {
			return e
		}
		if pending {
			return ErrUnavailable
		}
		camp, e := r.Campaign(c, q.CampaignID)
		if e != nil {
			return e
		}
		ss, e := r.Schedules(c, q.CampaignID)
		if e != nil {
			return e
		}
		for i := range ss {
			if ss[i].Status != "paid" {
				ss[i].Status = "restructured"
				if e = r.SaveSchedule(c, &ss[i]); e != nil {
					return e
				}
			}
		}
		camp.LoanTenorMonths = q.ProposedTenorMonths
		if e = r.SaveCampaign(c, camp); e != nil {
			return e
		}
		q.Status = "approved"
		if e = r.SaveRequest(c, q); e != nil {
			return e
		}
		if e := r.AddSchedules(c, build(q.CampaignID, q.ProposedTenorMonths, q.RemainingPrincipal, q.RemainingMargin, now)); e != nil {
			return e
		}
		return r.Audit(c, audit.New(c, &a, "repayment_restructuring_request.approved", "repayment_restructuring_request", q.ID, nil, nil))
	})
}
func (s *Service) List(c context.Context, u uuid.UUID, campaignID *uuid.UUID, admin bool, status string, limit, offset int) ([]model.RepaymentRestructuringRequest, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return s.r.List(c, u, campaignID, admin, status, limit, offset)
}
func build(cid uuid.UUID, n int, p, m int64, now time.Time) []model.RepaymentSchedule {
	out := make([]model.RepaymentSchedule, n)
	for i := range out {
		a, b := p/int64(n), m/int64(n)
		if int64(i) < p%int64(n) {
			a++
		}
		if int64(i) < m%int64(n) {
			b++
		}
		out[i] = model.RepaymentSchedule{CampaignID: cid, DueDate: now.AddDate(0, i+1, 0), PrincipalDue: a, MarginDue: &b, TotalDue: a + b, Status: "upcoming"}
	}
	return out
}
