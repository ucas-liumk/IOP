package iam

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// Repository combines user/session/role queries (M1-M2 simplicity).
// M5 could split if the file gets unwieldy.
type Repository interface {
	CreateUser(ctx context.Context, u *PlatformUser) error
	GetUserByEmail(ctx context.Context, email string) (*PlatformUser, error)
	GetUserByID(ctx context.Context, id kernel.ID) (*PlatformUser, error)
	UpdateLastLogin(ctx context.Context, id kernel.ID, at time.Time) error

	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id kernel.ID) (*Session, error)
	RevokeSession(ctx context.Context, id kernel.ID) error

	GetRoleByCode(ctx context.Context, code string, tenantID *kernel.ID) (*Role, error)
	ListMemberRoles(ctx context.Context, memberID kernel.ID, tenantID kernel.ID) ([]*Role, error)
	GrantRole(ctx context.Context, g *RoleGrant) error
	ListPolicyForRoles(ctx context.Context, roleIDs []kernel.ID) ([]*PolicyRule, error)
}

type pgRepo struct{ pool *pgxpool.Pool }

func NewPGRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) CreateUser(ctx context.Context, u *PlatformUser) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_user (id, email, password_hash, mfa_secret, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		u.ID, u.Email, u.PasswordHash, u.MFASecret, u.Status, u.CreatedAt)
	return err
}

func (r *pgRepo) GetUserByEmail(ctx context.Context, email string) (*PlatformUser, error) {
	var u PlatformUser
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, COALESCE(mfa_secret,''), status, last_login_at, created_at
		 FROM public.platform_user WHERE email = $1`, email).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.MFASecret, &u.Status, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *pgRepo) GetUserByID(ctx context.Context, id kernel.ID) (*PlatformUser, error) {
	var u PlatformUser
	err := r.pool.QueryRow(ctx,
		`SELECT id, email, password_hash, COALESCE(mfa_secret,''), status, last_login_at, created_at
		 FROM public.platform_user WHERE id = $1`, id).
		Scan(&u.ID, &u.Email, &u.PasswordHash, &u.MFASecret, &u.Status, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *pgRepo) UpdateLastLogin(ctx context.Context, id kernel.ID, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.platform_user SET last_login_at = $1 WHERE id = $2`, at, id)
	return err
}

func (r *pgRepo) CreateSession(ctx context.Context, s *Session) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.session (id, platform_user_id, tenant_id, member_id, issued_at, expires_at, revoked, ip_address, user_agent)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, NULLIF($8,'')::inet, $9)`,
		s.ID, s.PlatformUserID, idOrNil(s.TenantID), idOrNil(s.MemberID),
		s.IssuedAt, s.ExpiresAt, s.Revoked, s.IPAddress, s.UserAgent)
	return err
}

func (r *pgRepo) GetSession(ctx context.Context, id kernel.ID) (*Session, error) {
	var s Session
	var tenantID, memberID *kernel.ID
	var ipAddr *string
	err := r.pool.QueryRow(ctx,
		`SELECT id, platform_user_id, tenant_id, member_id, issued_at, expires_at, revoked, host(ip_address), COALESCE(user_agent,'')
		 FROM public.session WHERE id = $1`, id).
		Scan(&s.ID, &s.PlatformUserID, &tenantID, &memberID, &s.IssuedAt, &s.ExpiresAt, &s.Revoked, &ipAddr, &s.UserAgent)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	s.TenantID = tenantID
	s.MemberID = memberID
	if ipAddr != nil {
		s.IPAddress = *ipAddr
	}
	return &s, nil
}

func (r *pgRepo) RevokeSession(ctx context.Context, id kernel.ID) error {
	_, err := r.pool.Exec(ctx, `UPDATE public.session SET revoked = TRUE WHERE id = $1`, id)
	return err
}

func (r *pgRepo) GetRoleByCode(ctx context.Context, code string, tenantID *kernel.ID) (*Role, error) {
	var role Role
	var err error
	if tenantID == nil {
		err = r.pool.QueryRow(ctx,
			`SELECT id, tenant_id, code, name, created_at FROM public.role WHERE code = $1 AND tenant_id IS NULL`,
			code).Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt)
	} else {
		err = r.pool.QueryRow(ctx,
			`SELECT id, tenant_id, code, name, created_at FROM public.role WHERE code = $1 AND tenant_id = $2`,
			code, *tenantID).Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *pgRepo) ListMemberRoles(ctx context.Context, memberID kernel.ID, tenantID kernel.ID) ([]*Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.tenant_id, r.code, r.name, r.created_at
		 FROM public.role_grant g JOIN public.role r ON r.id = g.role_id
		 WHERE g.member_id = $1 AND g.tenant_id = $2`,
		memberID, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

func (r *pgRepo) GrantRole(ctx context.Context, g *RoleGrant) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.role_grant (role_id, member_id, tenant_id, granted_at)
		 VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
		g.RoleID, g.MemberID, g.TenantID, g.GrantedAt)
	return err
}

func (r *pgRepo) ListPolicyForRoles(ctx context.Context, roleIDs []kernel.ID) ([]*PolicyRule, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}
	ids := make([]string, len(roleIDs))
	for i, id := range roleIDs {
		ids[i] = string(id)
	}
	rows, err := r.pool.Query(ctx,
		`SELECT role_id, resource, action, effect FROM public.role_policy WHERE role_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*PolicyRule{}
	for rows.Next() {
		var p PolicyRule
		if err := rows.Scan(&p.RoleID, &p.Resource, &p.Action, &p.Effect); err != nil {
			return nil, err
		}
		out = append(out, &p)
	}
	return out, rows.Err()
}

func idOrNil(id *kernel.ID) any {
	if id == nil {
		return nil
	}
	return *id
}
