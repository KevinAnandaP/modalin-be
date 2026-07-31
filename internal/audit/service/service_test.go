package service

import (
	"context"
	"testing"

	"modalin-be/internal/audit/repository"
	"modalin-be/internal/model"
)

type fakeRepository struct{ filter repository.Filter }

func (r *fakeRepository) List(_ context.Context, filter repository.Filter) ([]model.AuditLog, int64, error) {
	r.filter = filter
	return []model.AuditLog{{Action: "campaign.updated"}}, 1, nil
}

func TestListCapsPageSizeAndNormalizesOffset(t *testing.T) {
	repo := &fakeRepository{}
	page, err := New(repo).List(context.Background(), Filter{Limit: 999, Offset: -1})
	if err != nil {
		t.Fatal(err)
	}
	if page.Limit != 100 || page.Offset != 0 || repo.filter.Limit != 100 || repo.filter.Offset != 0 {
		t.Fatalf("page/filter = %#v / %#v", page, repo.filter)
	}
}
