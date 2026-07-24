package job

import (
	"testing"
	"time"
)

func TestRepaymentScheduleIsOverdueInWIBAfterDueDate(t *testing.T) {
	jakarta, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatal(err)
	}
	dueDate := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	if RepaymentScheduleIsOverdue("upcoming", dueDate, time.Date(2026, 7, 20, 23, 0, 0, 0, jakarta)) {
		t.Fatal("schedule must not be late on its WIB due date")
	}
	if !RepaymentScheduleIsOverdue("due", dueDate, time.Date(2026, 7, 21, 0, 5, 0, 0, jakarta)) {
		t.Fatal("unpaid schedule must be late after its WIB due date")
	}
	if RepaymentScheduleIsOverdue("paid", dueDate, time.Date(2026, 7, 21, 0, 5, 0, 0, jakarta)) {
		t.Fatal("paid schedule must never become late")
	}
}
