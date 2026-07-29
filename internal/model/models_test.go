package model

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestUserJSONNeverExposesPasswordHash(t *testing.T) {
	encoded, err := json.Marshal(User{PasswordHash: strings.Repeat("x", 12)})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "PasswordHash") || strings.Contains(string(encoded), "password_hash") {
		t.Fatalf("password hash exposed in JSON: %s", encoded)
	}
}

func TestAuditLogBeforeCreateAssignsEventID(t *testing.T) {
	entry := AuditLog{}
	if err := entry.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if entry.ID == uuid.Nil || entry.EventID == uuid.Nil {
		t.Fatalf("expected identifiers, got %#v", entry)
	}
}

func TestAuditLogCarriesRequestID(t *testing.T) {
	requestID := "request-123"
	entry := AuditLog{RequestID: &requestID}
	if got := entry.RequestID; got == nil || *got != requestID {
		t.Fatalf("request id = %v", got)
	}
}

func TestDisputeBeforeCreateAssignsID(t *testing.T) {
	note := "resolved with evidence"
	now := time.Now().UTC()
	dispute := Dispute{ResolutionNote: &note, ResolvedAt: &now}
	if err := dispute.BeforeCreate(nil); err != nil {
		t.Fatal(err)
	}
	if dispute.ID == uuid.Nil {
		t.Fatal("expected dispute ID")
	}
}
