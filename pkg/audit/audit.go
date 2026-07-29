package audit

import (
	"context"
	"time"

	"modalin-be/internal/model"

	"github.com/google/uuid"
)

const requestIDContextKey = "modalin.request_id"

func WithRequestID(ctx context.Context, requestID string) context.Context {
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestID(ctx context.Context) string {
	value, _ := ctx.Value(requestIDContextKey).(string)
	return value
}

// New creates an immutable audit event with a unique event ID. When a caller
// retries using the same X-Request-ID, the database contract prevents a second
// identical request/action/entity audit record. Callers pass redacted JSON.
func New(ctx context.Context, userID *uuid.UUID, action, entityType string, entityID uuid.UUID, oldValue, newValue *string) *model.AuditLog {
	entry := &model.AuditLog{
		EventID:    uuid.New(),
		UserID:     userID,
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		OldValue:   oldValue,
		NewValue:   newValue,
		CreatedAt:  time.Now().UTC(),
	}
	if requestID := RequestID(ctx); requestID != "" {
		entry.RequestID = &requestID
	}
	return entry
}
