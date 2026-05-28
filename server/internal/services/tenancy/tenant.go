package tenancy

import (
	"context"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Status values for Tenant.Status.
const (
	StatusActive    = "active"
	StatusSuspended = "suspended"
	StatusClosed    = "closed"
)

// Tenant is the top-level tenant record (lives in public.tenant).
type Tenant struct {
	ID          kernel.ID  `json:"id"`
	Slug        string     `json:"slug"`
	Name        string     `json:"name"`
	SchemaName  string     `json:"schema_name"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	SuspendedAt *time.Time `json:"suspended_at,omitempty"`
	ClosedAt    *time.Time `json:"closed_at,omitempty"`
}

// TenantMembership links a platform user to a tenant.
type TenantMembership struct {
	PlatformUserID kernel.ID `json:"platform_user_id"`
	TenantID       kernel.ID `json:"tenant_id"`
	MemberID       kernel.ID `json:"member_id"`
	JoinedAt       time.Time `json:"joined_at"`
	Status         string    `json:"status"`
}

// TenantRepository abstracts persistence for Tenant.
type TenantRepository interface {
	Create(ctx context.Context, t *Tenant) error
	GetByID(ctx context.Context, id kernel.ID) (*Tenant, error)
	GetBySlug(ctx context.Context, slug string) (*Tenant, error)
	ListActive(ctx context.Context, p kernel.Pagination) ([]*Tenant, error)
	UpdateStatus(ctx context.Context, id kernel.ID, status string, at time.Time) error
}

// MembershipRepository abstracts persistence for TenantMembership.
type MembershipRepository interface {
	Create(ctx context.Context, m *TenantMembership) error
	ListByUser(ctx context.Context, userID kernel.ID) ([]*TenantMembership, error)
	Get(ctx context.Context, userID, tenantID kernel.ID) (*TenantMembership, error)
}
