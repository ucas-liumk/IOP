package iam

import (
	"context"
	"errors"
	"strings"
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
	GetUserByPhone(ctx context.Context, phone string) (*PlatformUser, error)
	GetUserByUsername(ctx context.Context, username string) (*PlatformUser, error)
	GetUserByLogin(ctx context.Context, login string) (*PlatformUser, error)
	GetUserByID(ctx context.Context, id kernel.ID) (*PlatformUser, error)
	ListUsers(ctx context.Context, search string, limit int) ([]*PlatformUser, error)
	UpdateUserStatus(ctx context.Context, id kernel.ID, status string) error
	UpdateUserPassword(ctx context.Context, id kernel.ID, hash string) error
	SetPasswordMustChange(ctx context.Context, id kernel.ID, must bool) error
	SetPlatformAdmin(ctx context.Context, id kernel.ID, isAdmin bool) error
	UpdateLastLogin(ctx context.Context, id kernel.ID, at time.Time) error

	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id kernel.ID) (*Session, error)
	RevokeSession(ctx context.Context, id kernel.ID) error
	// RevokeUserSessions marks all of a user's active sessions revoked and returns
	// their IDs (so the caller can also blacklist them in Redis).
	RevokeUserSessions(ctx context.Context, userID kernel.ID) ([]kernel.ID, error)

	GetRoleByCode(ctx context.Context, code string, tenantID *kernel.ID) (*Role, error)
	ListMemberRoles(ctx context.Context, memberID kernel.ID, tenantID kernel.ID) ([]*Role, error)
	GrantRole(ctx context.Context, g *RoleGrant) error
	ListPolicyForRoles(ctx context.Context, roleIDs []kernel.ID) ([]*PolicyRule, error)
}

type pgRepo struct{ pool *pgxpool.Pool }

func NewPGRepo(pool *pgxpool.Pool) Repository { return &pgRepo{pool: pool} }

func (r *pgRepo) CreateUser(ctx context.Context, u *PlatformUser) error {
	var username, email, phone any
	if u.Username != "" {
		username = u.Username
	}
	if u.Email != "" {
		email = u.Email
	}
	if u.Phone != "" {
		phone = u.Phone
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_user (id, username, phone, email, password_hash, mfa_secret, status, password_must_change, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		u.ID, username, phone, email, u.PasswordHash, u.MFASecret, u.Status, u.PasswordMustChange, u.CreatedAt)
	return err
}

func (r *pgRepo) SetPasswordMustChange(ctx context.Context, id kernel.ID, must bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.platform_user SET password_must_change = $1 WHERE id = $2`, must, id)
	return err
}

func (r *pgRepo) SetPlatformAdmin(ctx context.Context, id kernel.ID, isAdmin bool) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.platform_user SET is_platform_admin = $1 WHERE id = $2`, isAdmin, id)
	return err
}

const userSelectCols = `id, COALESCE(username,''), COALESCE(phone,''), COALESCE(email,''), password_hash, COALESCE(mfa_secret,''), status, COALESCE(password_must_change,false), COALESCE(is_platform_admin,false), last_login_at, created_at`

func scanUser(row pgx.Row) (*PlatformUser, error) {
	var u PlatformUser
	err := row.Scan(&u.ID, &u.Username, &u.Phone, &u.Email, &u.PasswordHash, &u.MFASecret, &u.Status, &u.PasswordMustChange, &u.IsPlatformAdmin, &u.LastLoginAt, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *pgRepo) GetUserByEmail(ctx context.Context, email string) (*PlatformUser, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM public.platform_user WHERE email = $1`, email))
}

func (r *pgRepo) GetUserByPhone(ctx context.Context, phone string) (*PlatformUser, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM public.platform_user WHERE phone = $1`, phone))
}

func (r *pgRepo) GetUserByUsername(ctx context.Context, username string) (*PlatformUser, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM public.platform_user WHERE username = $1`, username))
}

// GetUserByLogin tries username first, then email — lets users sign in with either.
// If the login string contains "@" we go straight to email lookup.
func (r *pgRepo) GetUserByLogin(ctx context.Context, login string) (*PlatformUser, error) {
	if login == "" {
		return nil, nil
	}
	if !strings.Contains(login, "@") {
		if u, err := r.GetUserByUsername(ctx, login); err != nil || u != nil {
			return u, err
		}
	}
	return r.GetUserByEmail(ctx, login)
}

func (r *pgRepo) GetUserByID(ctx context.Context, id kernel.ID) (*PlatformUser, error) {
	return scanUser(r.pool.QueryRow(ctx,
		`SELECT `+userSelectCols+` FROM public.platform_user WHERE id = $1`, id))
}

func (r *pgRepo) ListUsers(ctx context.Context, search string, limit int) ([]*PlatformUser, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	var rows pgx.Rows
	var err error
	if search = strings.TrimSpace(search); search == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT `+userSelectCols+` FROM public.platform_user
			 ORDER BY created_at DESC LIMIT $1`, limit)
	} else {
		pattern := "%" + strings.ToLower(search) + "%"
		rows, err = r.pool.Query(ctx,
			`SELECT `+userSelectCols+` FROM public.platform_user
			 WHERE LOWER(COALESCE(username,'')) LIKE $1
			    OR LOWER(COALESCE(email,'')) LIKE $1
			    OR COALESCE(phone,'') LIKE $1
			 ORDER BY created_at DESC LIMIT $2`, pattern, limit)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*PlatformUser{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, u)
	}
	return out, rows.Err()
}

func (r *pgRepo) UpdateUserStatus(ctx context.Context, id kernel.ID, status string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.platform_user SET status = $1 WHERE id = $2`, status, id)
	return err
}

func (r *pgRepo) UpdateUserPassword(ctx context.Context, id kernel.ID, hash string) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.platform_user SET password_hash = $1 WHERE id = $2`, hash, id)
	return err
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

func (r *pgRepo) RevokeUserSessions(ctx context.Context, userID kernel.ID) ([]kernel.ID, error) {
	rows, err := r.pool.Query(ctx,
		`UPDATE public.session SET revoked = TRUE
		 WHERE platform_user_id = $1 AND revoked = FALSE
		 RETURNING id`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []kernel.ID
	for rows.Next() {
		var id kernel.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
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
