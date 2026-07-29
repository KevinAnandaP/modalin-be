package job

import (
	"context"
	"log"
	"strings"
	"time"

	"gorm.io/gorm"
)

var jakartaLocation, _ = time.LoadLocation("Asia/Jakarta")

func RepaymentScheduleIsOverdue(status string, dueDate, now time.Time) bool {
	if strings.ToLower(strings.TrimSpace(status)) != "upcoming" && strings.ToLower(strings.TrimSpace(status)) != "due" {
		return false
	}
	if jakartaLocation == nil {
		jakartaLocation = time.FixedZone("WIB", 7*60*60)
	}
	today := dateOnly(now.In(jakartaLocation))
	return dateOnly(dueDate.In(jakartaLocation)).Before(today)
}

func MarkOverdueRepaymentSchedules(ctx context.Context, db *gorm.DB, now time.Time) (int64, error) {
	if jakartaLocation == nil {
		jakartaLocation = time.FixedZone("WIB", 7*60*60)
	}
	today := dateOnly(now.In(jakartaLocation))
	result := db.WithContext(ctx).Exec(`
		UPDATE repayment_schedules
		SET status = 'late', updated_at = ?
		WHERE deleted_at IS NULL
		  AND status IN ('upcoming', 'due')
		  AND due_date < ?`, now.UTC(), today)
	return result.RowsAffected, result.Error
}

func StartRepaymentOverdueScheduler(ctx context.Context, db *gorm.DB) {
	go func() {
		for {
			now := time.Now()
			if _, err := MarkOverdueRepaymentSchedules(ctx, db, now); err != nil && ctx.Err() == nil {
				log.Printf("repayment overdue job failed: %v", err)
			}
			if !waitForNextRun(ctx, durationUntilNextWIBRun(now)) {
				return
			}
		}
	}()
}

func waitForNextRun(ctx context.Context, duration time.Duration) bool {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}

func durationUntilNextWIBRun(now time.Time) time.Duration {
	if jakartaLocation == nil {
		jakartaLocation = time.FixedZone("WIB", 7*60*60)
	}
	local := now.In(jakartaLocation)
	next := time.Date(local.Year(), local.Month(), local.Day(), 0, 5, 0, 0, jakartaLocation)
	if !local.Before(next) {
		next = next.AddDate(0, 0, 1)
	}
	return time.Until(next)
}

func dateOnly(value time.Time) time.Time {
	return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, value.Location())
}
