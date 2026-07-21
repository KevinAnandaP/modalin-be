package job

import (
	"testing"
	"time"
)

func TestProofQualifiesForNextMilestoneUnlock(t *testing.T) {
	now := time.Date(2026, 7, 20, 10, 0, 0, 0, time.UTC)
	if !ProofQualifiesForUnlock("approved", now, now) {
		t.Fatal("approved proof must unlock immediately")
	}
	if !ProofQualifiesForUnlock("pending", now.Add(-5*time.Hour), now) {
		t.Fatal("pending proof at five hours must unlock")
	}
	if ProofQualifiesForUnlock("pending", now.Add(-4*time.Hour-59*time.Minute), now) {
		t.Fatal("recent pending proof must stay locked")
	}
	if ProofQualifiesForUnlock("rejected", now.Add(-10*time.Hour), now) {
		t.Fatal("rejected proof must not unlock")
	}
}
