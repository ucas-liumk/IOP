package tenancy

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

var slugRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,30}[a-z0-9]$`)

// Service is the public API of the tenancy service.
type Service struct {
	tenantRepo TenantRepository
	memberRepo MembershipRepository
	prov       *SchemaProvisioner
	bus        eventbus.Bus
	clock      kernel.Clock
}

func NewService(t TenantRepository, m MembershipRepository, p *SchemaProvisioner, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{tenantRepo: t, memberRepo: m, prov: p, bus: bus, clock: clk}
}

// CreateTenantCmd is the input to CreateTenant.
type CreateTenantCmd struct {
	Slug string `json:"slug"`
	Name string `json:"name"`
}

// CreateTenant validates slug, inserts public.tenant, runs SchemaProvisioner.
func (s *Service) CreateTenant(ctx context.Context, cmd CreateTenantCmd) (*Tenant, error) {
	slug := strings.ToLower(strings.TrimSpace(cmd.Slug))
	if !slugRe.MatchString(slug) {
		return nil, errors.New(errors.KindParam, "tenancy.invalid_slug",
			"slug must be 3-32 chars, lowercase letters/digits/_/- and start with letter")
	}
	if strings.TrimSpace(cmd.Name) == "" {
		return nil, errors.New(errors.KindParam, "tenancy.invalid_name", "name required")
	}
	existing, err := s.tenantRepo.GetBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errors.New(errors.KindConflict, "tenancy.slug_taken", fmt.Sprintf("slug %q already in use", slug))
	}
	t := &Tenant{
		ID:         kernel.NewID(),
		Slug:       slug,
		Name:       cmd.Name,
		SchemaName: "tenant_" + strings.ReplaceAll(slug, "-", "_"),
		Status:     StatusActive,
		CreatedAt:  s.clock.Now(),
	}
	if err := s.tenantRepo.Create(ctx, t); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.create_failed", "insert tenant failed", err)
	}
	if err := s.prov.Provision(ctx, t.SchemaName); err != nil {
		// Best-effort rollback: mark tenant suspended so it's not visible.
		_ = s.tenantRepo.UpdateStatus(ctx, t.ID, StatusSuspended, s.clock.Now())
		return nil, err
	}
	_ = s.bus.Publish(ctx, "tenancy.tenant_created", map[string]any{
		"tenant_id": t.ID, "slug": t.Slug, "name": t.Name,
	})
	return t, nil
}

func (s *Service) SuspendTenant(ctx context.Context, id kernel.ID, reason string) error {
	if err := s.tenantRepo.UpdateStatus(ctx, id, StatusSuspended, s.clock.Now()); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.tenant_suspended", map[string]any{
		"tenant_id": id, "reason": reason,
	})
	return nil
}

func (s *Service) ResumeTenant(ctx context.Context, id kernel.ID) error {
	if err := s.tenantRepo.UpdateStatus(ctx, id, StatusActive, s.clock.Now()); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.tenant_resumed", map[string]any{"tenant_id": id})
	return nil
}

func (s *Service) CloseTenant(ctx context.Context, id kernel.ID) error {
	t, err := s.tenantRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if t == nil {
		return errors.New(errors.KindNotFound, "tenancy.not_found", "tenant not found")
	}
	if err := s.tenantRepo.UpdateStatus(ctx, id, StatusClosed, s.clock.Now()); err != nil {
		return err
	}
	// Schema drop is gated to 30-day retention in real deployments; tests/dev drop immediately.
	if err := s.prov.Drop(ctx, t.SchemaName); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "tenancy.tenant_closed", map[string]any{"tenant_id": id})
	return nil
}

// JoinMember creates a TenantMembership and inserts a `member` row in the tenant schema.
type JoinMemberCmd struct {
	PlatformUserID kernel.ID
	TenantID       kernel.ID
	DisplayName    string
	Email          string
	Department     string
	Title          string
}

func (s *Service) JoinMember(ctx context.Context, pool *pgxpool.Pool, cmd JoinMemberCmd) (*TenantMembership, error) {
	t, err := s.tenantRepo.GetByID(ctx, cmd.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errors.New(errors.KindNotFound, "tenancy.not_found", "tenant not found")
	}
	if existing, _ := s.memberRepo.Get(ctx, cmd.PlatformUserID, cmd.TenantID); existing != nil {
		return existing, nil
	}
	m := &TenantMembership{
		PlatformUserID: cmd.PlatformUserID,
		TenantID:       cmd.TenantID,
		MemberID:       kernel.NewID(),
		JoinedAt:       s.clock.Now(),
		Status:         "active",
	}
	if err := s.memberRepo.Create(ctx, m); err != nil {
		return nil, err
	}
	// Insert into per-tenant `member` projection.
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO member (id, platform_user_id, display_name, email, department, title, joined_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (platform_user_id) DO NOTHING`,
		m.MemberID, m.PlatformUserID, cmd.DisplayName, cmd.Email, cmd.Department, cmd.Title, m.JoinedAt); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, "tenancy.member_joined", map[string]any{
		"tenant_id": cmd.TenantID, "member_id": m.MemberID, "platform_user_id": cmd.PlatformUserID,
	})
	return m, nil
}

func (s *Service) GetTenant(ctx context.Context, id kernel.ID) (*Tenant, error) {
	return s.tenantRepo.GetByID(ctx, id)
}

func (s *Service) ListActiveTenants(ctx context.Context, p kernel.Pagination) ([]*Tenant, error) {
	return s.tenantRepo.ListActive(ctx, p)
}

func (s *Service) ListMyTenants(ctx context.Context, userID kernel.ID) ([]*Tenant, error) {
	mems, err := s.memberRepo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := []*Tenant{}
	for _, m := range mems {
		t, err := s.tenantRepo.GetByID(ctx, m.TenantID)
		if err != nil {
			return nil, err
		}
		if t != nil && t.Status == StatusActive {
			out = append(out, t)
		}
	}
	_ = time.Time{}
	return out, nil
}

// GetMembership for the (userID, tenantID) pair, returns nil if not joined.
func (s *Service) GetMembership(ctx context.Context, userID, tenantID kernel.ID) (*TenantMembership, error) {
	return s.memberRepo.Get(ctx, userID, tenantID)
}
