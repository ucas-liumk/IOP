package eventbus

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Event is the generic envelope. Data carries the topic-specific payload.
type Event struct {
	ID         kernel.ID
	Topic      string    // e.g. "okr.plan_created"
	OccurredAt time.Time // UTC
	TenantID   kernel.ID // empty if cross-tenant
	Actor      string    // member id or "system"
	TraceID    string
	Data       any
}
