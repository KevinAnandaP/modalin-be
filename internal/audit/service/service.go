package service

import (
	"context"
	"time"

	"modalin-be/internal/audit/repository"
	"modalin-be/internal/model"

	"github.com/google/uuid"
)

type Filter struct {
	UserID, EntityID   *uuid.UUID
	Action, EntityType string
	From, To           *time.Time
	Limit, Offset      int
}

type Page struct {
	Data   []model.AuditLog `json:"data"`
	Limit  int              `json:"limit"`
	Offset int              `json:"offset"`
	Total  int64            `json:"total"`
}

type Service struct{ repo repository.Repository }

func New(repo repository.Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, filter Filter) (*Page, error) {
	if filter.Limit <= 0 {
		filter.Limit = 20
	}
	if filter.Limit > 100 {
		filter.Limit = 100
	}
	if filter.Offset < 0 {
		filter.Offset = 0
	}
	rows, total, err := s.repo.List(ctx, repository.Filter(filter))
	if err != nil {
		return nil, err
	}
	return &Page{Data: rows, Limit: filter.Limit, Offset: filter.Offset, Total: total}, nil
}
