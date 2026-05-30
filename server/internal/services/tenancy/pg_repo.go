package tenancy

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/shared/kernel"
)

type pgTenantRepo struct{ pool *pgxpool.Pool }

func NewPGTenantRepo(pool *pgxpool.Pool) TenantRepository { return &pgTenantRepo{pool: pool} }

func (r *pgTenantRepo) Create(ctx context.Context, t *Tenant) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.tenant (id, slug, name, schema_name, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		t.ID, t.Slug, t.Name, t.SchemaName, t.Status, t.CreatedAt)
	return err
}

func (r *pgTenantRepo) GetByID(ctx context.Context, id kernel.ID) (*Tenant, error) {
	var t Tenant
	err := r.pool.QueryRow(ctx,
		`SELECT id, slug, name, schema_name, status, created_at, suspended_at, closed_at
		 FROM public.tenant WHERE id = $1`, id).
		Scan(&t.ID, &t.Slug, &t.Name, &t.SchemaName, &t.Status, &t.CreatedAt, &t.SuspendedAt, &t.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgTenantRepo) GetBySlug(ctx context.Context, slug string) (*Tenant, error) {
	var t Tenant
	err := r.pool.QueryRow(ctx,
		`SELECT id, slug, name, schema_name, status, created_at, suspended_at, closed_at
		 FROM public.tenant WHERE slug = $1`, slug).
		Scan(&t.ID, &t.Slug, &t.Name, &t.SchemaName, &t.Status, &t.CreatedAt, &t.SuspendedAt, &t.ClosedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *pgTenantRepo) ListActive(ctx context.Context, p kernel.Pagination) ([]*Tenant, error) {
	p = p.Normalize()
	rows, err := r.pool.Query(ctx,
		`SELECT id, slug, name, schema_name, status, created_at, suspended_at, closed_at
		 FROM public.tenant WHERE status = 'active' ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		p.PageSize, p.Offset())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Tenant{}
	for rows.Next() {
		var t Tenant
		if err := rows.Scan(&t.ID, &t.Slug, &t.Name, &t.SchemaName, &t.Status, &t.CreatedAt, &t.SuspendedAt, &t.ClosedAt); err != nil {
			return nil, err
		}
		out = append(out, &t)
	}
	return out, rows.Err()
}

func (r *pgTenantRepo) UpdateStatus(ctx context.Context, id kernel.ID, status string, at time.Time) error {
	var col string
	switch status {
	case StatusSuspended:
		col = "suspended_at"
	case StatusClosed:
		col = "closed_at"
	case StatusActive:
		// resume: clear suspended_at
		_, err := r.pool.Exec(ctx,
			`UPDATE public.tenant SET status = 'active', suspended_at = NULL WHERE id = $1`, id)
		return err
	default:
		col = "suspended_at"
	}
	_, err := r.pool.Exec(ctx,
		`UPDATE public.tenant SET status = $1, `+col+` = $2 WHERE id = $3`,
		status, at, id)
	return err
}

func (r *pgTenantRepo) Delete(ctx context.Context, id kernel.ID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM public.tenant WHERE id = $1`, id)
	return err
}

type pgMembershipRepo struct{ pool *pgxpool.Pool }

func NewPGMembershipRepo(pool *pgxpool.Pool) MembershipRepository {
	return &pgMembershipRepo{pool: pool}
}

func (r *pgMembershipRepo) Create(ctx context.Context, m *TenantMembership) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.tenant_membership (platform_user_id, tenant_id, member_id, joined_at, status)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (platform_user_id, tenant_id) DO NOTHING`,
		m.PlatformUserID, m.TenantID, m.MemberID, m.JoinedAt, m.Status)
	return err
}

func (r *pgMembershipRepo) ListByUser(ctx context.Context, userID kernel.ID) ([]*TenantMembership, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT platform_user_id, tenant_id, member_id, joined_at, status
		 FROM public.tenant_membership WHERE platform_user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*TenantMembership{}
	for rows.Next() {
		var m TenantMembership
		if err := rows.Scan(&m.PlatformUserID, &m.TenantID, &m.MemberID, &m.JoinedAt, &m.Status); err != nil {
			return nil, err
		}
		out = append(out, &m)
	}
	return out, rows.Err()
}

func (r *pgMembershipRepo) Get(ctx context.Context, userID, tenantID kernel.ID) (*TenantMembership, error) {
	var m TenantMembership
	err := r.pool.QueryRow(ctx,
		`SELECT platform_user_id, tenant_id, member_id, joined_at, status
		 FROM public.tenant_membership WHERE platform_user_id = $1 AND tenant_id = $2`,
		userID, tenantID).
		Scan(&m.PlatformUserID, &m.TenantID, &m.MemberID, &m.JoinedAt, &m.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &m, nil
}
