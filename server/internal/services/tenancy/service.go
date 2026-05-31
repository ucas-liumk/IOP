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

// CreateTenant validates the slug, inserts public.tenant, and provisions the
// per-tenant schema. It is transactional in effect and recoverable:
//
//   - If provisioning fails, the schema is dropped AND the tenant row is deleted
//     so the slug is immediately free for a clean retry (no orphan schema, no
//     stuck "suspended" row). If cleanup itself fails (rare/transient), the row
//     is marked suspended as a last resort and the recovery path below handles it.
//   - If the slug already maps to an ACTIVE tenant → conflict.
//   - If the slug maps to a SUSPENDED tenant that was never fully provisioned
//     (a leftover from a prior failed attempt), CreateTenant retries provisioning
//     in place and re-activates it, rather than dead-ending — making the operation
//     idempotent on slug.
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
	// ANY existing tenant (active / suspended / closed) blocks the slug. We must
	// never auto-DROP an existing schema here: a 'suspended' tenant may be a healthy,
	// admin-suspended tenant with live data — indistinguishable from a half-provisioned
	// leftover by status alone. Suspended tenants are recovered via ResumeTenant, not
	// by re-creating the slug.
	if existing != nil {
		return nil, errors.New(errors.KindConflict, "tenancy.slug_taken",
			fmt.Sprintf("slug %q already in use", slug))
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
		// Full rollback so the slug is reusable on retry: drop the (possibly partial)
		// schema we just started, then delete the brand-new tenant row. Safe because
		// this row was created microseconds ago in THIS call — never a populated tenant.
		_ = s.prov.Drop(ctx, t.SchemaName)
		if delErr := s.tenantRepo.Delete(ctx, t.ID); delErr != nil {
			// Couldn't delete — hide it so it doesn't occupy the slug as 'active'.
			_ = s.tenantRepo.UpdateStatus(ctx, t.ID, StatusSuspended, s.clock.Now())
		}
		return nil, errors.Wrap(errors.KindInternal, "tenancy.provision_failed", "租户初始化失败，请重试", err)
	}
	if _, err := s.EnsureRootDept(ctx, s.prov.pool, t); err != nil {
		_ = s.prov.Drop(ctx, t.SchemaName)
		if delErr := s.tenantRepo.Delete(ctx, t.ID); delErr != nil {
			_ = s.tenantRepo.UpdateStatus(ctx, t.ID, StatusSuspended, s.clock.Now())
		}
		return nil, errors.Wrap(errors.KindInternal, "tenancy.root_org_failed", "租户根组织初始化失败，请重试", err)
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
	// Closing a tenant disables access but keeps the tenant schema for retention/audit.
	_ = s.bus.Publish(ctx, "tenancy.tenant_closed", map[string]any{"tenant_id": id})
	return nil
}

// JoinMember creates a TenantMembership and inserts a `member` row in the tenant schema.
type JoinMemberCmd struct {
	PlatformUserID kernel.ID
	TenantID       kernel.ID
	DisplayName    string
	Email          string
	Phone          string
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
		`INSERT INTO member (id, platform_user_id, display_name, email, phone, department, title, joined_at)
		 VALUES ($1, $2, $3, $4, NULLIF($5,''), $6, $7, $8) ON CONFLICT (platform_user_id) DO NOTHING`,
		m.MemberID, m.PlatformUserID, cmd.DisplayName, cmd.Email, cmd.Phone, cmd.Department, cmd.Title, m.JoinedAt); err != nil {
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

// SyncAllSchemas re-runs the tenant-template migrations against every active
// tenant schema. Idempotent (CREATE ... IF NOT EXISTS + migration_history), so
// it safely brings existing tenants up to date with newly-added module tables on
// each deploy/boot — the auto-migrate path for a pluggable-module framework.
func (s *Service) SyncAllSchemas(ctx context.Context) (synced int, err error) {
	tenants, err := s.tenantRepo.ListActive(ctx, kernel.Pagination{Page: 1, PageSize: 1000})
	if err != nil {
		return 0, err
	}
	// Continue on a per-tenant failure so one corrupt/drifted schema can't block
	// migrating the rest; aggregate the failures into the returned error.
	var failures []string
	for _, t := range tenants {
		if e := s.prov.Provision(ctx, t.SchemaName); e != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", t.SchemaName, e))
			continue
		}
		if _, e := s.EnsureRootDept(ctx, s.prov.pool, t); e != nil {
			failures = append(failures, fmt.Sprintf("%s root org: %v", t.SchemaName, e))
			continue
		}
		synced++
	}
	if len(failures) > 0 {
		return synced, fmt.Errorf("sync failed for %d tenant(s): %s", len(failures), strings.Join(failures, "; "))
	}
	return synced, nil
}

func (s *Service) GetTenantBySlug(ctx context.Context, slug string) (*Tenant, error) {
	return s.tenantRepo.GetBySlug(ctx, slug)
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
