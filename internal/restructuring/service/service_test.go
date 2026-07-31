package service

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildPreservesRemainingObligationAndStartsNextMonth(t *testing.T) {
	now := time.Date(2026, time.July, 25, 10, 0, 0, 0, time.UTC)
	schedules := build(uuid.New(), 3, 100, 10, now)
	if len(schedules) != 3 {
		t.Fatalf("schedule count = %d", len(schedules))
	}
	var principal, margin, total int64
	for _, schedule := range schedules {
		principal += schedule.PrincipalDue
		if schedule.MarginDue != nil {
			margin += *schedule.MarginDue
		}
		total += schedule.TotalDue
		if schedule.Status != "upcoming" {
			t.Fatalf("status = %q", schedule.Status)
		}
	}
	if principal != 100 || margin != 10 || total != 110 {
		t.Fatalf("obligation = principal:%d margin:%d total:%d", principal, margin, total)
	}
	if got := schedules[0].DueDate; !got.Equal(now.AddDate(0, 1, 0)) {
		t.Fatalf("first due date = %s", got)
	}
}

func TestBuildDistributesRemainderWithoutLoss(t *testing.T) {
	schedules := build(uuid.New(), 4, 5, 3, time.Now().UTC())
	var principal, margin int64
	for _, schedule := range schedules {
		principal += schedule.PrincipalDue
		margin += *schedule.MarginDue
	}
	if principal != 5 || margin != 3 {
		t.Fatalf("remainder lost: principal=%d margin=%d", principal, margin)
	}
}
