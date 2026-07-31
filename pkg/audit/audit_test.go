package audit

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewUsesRequestIDFromContextAndGeneratesEventID(t *testing.T) {
	ctx := WithRequestID(context.Background(), "request-123")
	entry := New(ctx, nil, "campaign.updated", "campaign", uuid.New(), nil, nil)
	if entry.EventID == uuid.Nil {
		t.Fatal("expected event id")
	}
	if entry.RequestID == nil || *entry.RequestID != "request-123" {
		t.Fatalf("request id = %#v", entry.RequestID)
	}
}
