package iam

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
	// ListUsersPage returns one page of platform_users matching search plus the
	// total count of matching rows (for server-side pagination).
	ListUsersPage(ctx context.Context, search string, limit, offset int) ([]*PlatformUser, int, error)
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

	ListPlatformRolesForUser(ctx context.Context, platformUserID kernel.ID) ([]*Role, error)
	ListPlatformRoles(ctx context.Context) ([]*Role, error)
	GetPlatformRoleByCode(ctx context.Context, code string) (*Role, error)
	GetPlatformRoleByID(ctx context.Context, id kernel.ID) (*Role, error)
	CreatePlatformRole(ctx context.Context, id kernel.ID, code, name string) error
	DeletePlatformRole(ctx context.Context, id kernel.ID) error
	AddPlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error
	RemovePlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error
	GrantPlatformRole(ctx context.Context, roleID, platformUserID, grantedBy kernel.ID) error
	RevokePlatformRole(ctx context.Context, roleID, platformUserID kernel.ID) error
	ListPlatformRoleMembers(ctx context.Context, roleID kernel.ID) ([]kernel.ID, error)
	// EnsureSuperAdminGrants grants the super_admin platform role to every platform_user
	// whose is_platform_admin flag is set but who lacks the grant. Idempotent.
	EnsureSuperAdminGrants(ctx context.Context) error
	GetPlatformSetting(ctx context.Context, key string) (string, error)
	SetPlatformSetting(ctx context.Context, key, value string, by kernel.ID) error
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

// GetUserByLogin resolves a login string to a user. Username and phone are the
// primary identities; email is a fallback so legacy email accounts keep working.
//   - "@" present            → email lookup (the only form that can contain "@").
//   - all digits (10-15 len) → phone lookup, then username as a fallback.
//   - otherwise              → username, then email as a fallback.
//
// Email lookups stay safe when a user's email is NULL: the WHERE email = $1 simply
// matches no rows rather than erroring.
func (r *pgRepo) GetUserByLogin(ctx context.Context, login string) (*PlatformUser, error) {
	if login == "" {
		return nil, nil
	}
	if strings.Contains(login, "@") {
		return r.GetUserByEmail(ctx, login)
	}
	if isAllDigits(login) {
		if u, err := r.GetUserByPhone(ctx, login); err != nil || u != nil {
			return u, err
		}
		return r.GetUserByUsername(ctx, login)
	}
	if u, err := r.GetUserByUsername(ctx, login); err != nil || u != nil {
		return u, err
	}
	return r.GetUserByEmail(ctx, login)
}

// isAllDigits reports whether s is a non-empty run of ASCII digits of plausible
// phone length, so we only attempt a phone lookup for phone-shaped logins.
func isAllDigits(s string) bool {
	if len(s) < 6 || len(s) > 20 {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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

func (r *pgRepo) ListUsersPage(ctx context.Context, search string, limit, offset int) ([]*PlatformUser, int, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	search = strings.TrimSpace(search)
	var total int
	var rows pgx.Rows
	var err error
	if search == "" {
		if err := r.pool.QueryRow(ctx, `SELECT count(*) FROM public.platform_user`).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx,
			`SELECT `+userSelectCols+` FROM public.platform_user
			 ORDER BY created_at DESC LIMIT $1 OFFSET $2`, limit, offset)
	} else {
		pattern := "%" + strings.ToLower(search) + "%"
		if err := r.pool.QueryRow(ctx,
			`SELECT count(*) FROM public.platform_user
			 WHERE LOWER(COALESCE(username,'')) LIKE $1
			    OR LOWER(COALESCE(email,'')) LIKE $1
			    OR COALESCE(phone,'') LIKE $1`, pattern).Scan(&total); err != nil {
			return nil, 0, err
		}
		rows, err = r.pool.Query(ctx,
			`SELECT `+userSelectCols+` FROM public.platform_user
			 WHERE LOWER(COALESCE(username,'')) LIKE $1
			    OR LOWER(COALESCE(email,'')) LIKE $1
			    OR COALESCE(phone,'') LIKE $1
			 ORDER BY created_at DESC LIMIT $2 OFFSET $3`, pattern, limit, offset)
	}
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []*PlatformUser{}
	for rows.Next() {
		u, err := scanUser(rows)
		if err != nil {
			return nil, 0, err
		}
		out = append(out, u)
	}
	return out, total, rows.Err()
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

// idOrNilV returns nil for a zero kernel.ID so it inserts SQL NULL, else the id.
func idOrNilV(id kernel.ID) any {
	if id == "" {
		return nil
	}
	return id
}

// --- Platform RBAC ---

// ListPlatformRolesForUser returns the platform-level roles (tenant_id IS NULL)
// granted to a platform_user via platform_role_grant.
func (r *pgRepo) ListPlatformRolesForUser(ctx context.Context, platformUserID kernel.ID) ([]*Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.tenant_id, r.code, r.name, r.created_at
		 FROM public.platform_role_grant g JOIN public.role r ON r.id = g.role_id
		 WHERE g.platform_user_id = $1 AND r.tenant_id IS NULL`,
		platformUserID)
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

// ListPlatformRoles returns all platform-level roles with member counts.
func (r *pgRepo) ListPlatformRoles(ctx context.Context) ([]*Role, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT r.id, r.tenant_id, r.code, r.name, r.created_at,
		        (SELECT count(*) FROM public.platform_role_grant g WHERE g.role_id = r.id)
		 FROM public.role r WHERE r.tenant_id IS NULL ORDER BY r.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*Role{}
	for rows.Next() {
		var role Role
		if err := rows.Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt, &role.MemberCount); err != nil {
			return nil, err
		}
		out = append(out, &role)
	}
	return out, rows.Err()
}

// GetPlatformRoleByCode returns a platform role by code, or (nil, nil) if absent.
func (r *pgRepo) GetPlatformRoleByCode(ctx context.Context, code string) (*Role, error) {
	return r.GetRoleByCode(ctx, code, nil)
}

// GetPlatformRoleByID returns a platform role by ID, or (nil, nil) if absent.
func (r *pgRepo) GetPlatformRoleByID(ctx context.Context, id kernel.ID) (*Role, error) {
	var role Role
	err := r.pool.QueryRow(ctx,
		`SELECT id, tenant_id, code, name, created_at FROM public.role WHERE id = $1 AND tenant_id IS NULL`, id).
		Scan(&role.ID, &role.TenantID, &role.Code, &role.Name, &role.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &role, nil
}

// CreatePlatformRole inserts a custom platform role (tenant_id NULL).
func (r *pgRepo) CreatePlatformRole(ctx context.Context, id kernel.ID, code, name string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.role (id, tenant_id, code, name) VALUES ($1, NULL, $2, $3)`,
		id, code, name)
	return err
}

// DeletePlatformRole removes a platform role (cascades grants + policies via FK).
func (r *pgRepo) DeletePlatformRole(ctx context.Context, id kernel.ID) error {
	_, err := r.pool.Exec(ctx, `DELETE FROM public.role WHERE id = $1 AND tenant_id IS NULL`, id)
	return err
}

func (r *pgRepo) AddPlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.role_policy (role_id, resource, action, effect)
		 VALUES ($1, $2, $3, 'allow') ON CONFLICT (role_id, resource, action) DO NOTHING`,
		roleID, resource, action)
	return err
}

func (r *pgRepo) RemovePlatformPolicy(ctx context.Context, roleID kernel.ID, resource, action string) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM public.role_policy WHERE role_id = $1 AND resource = $2 AND action = $3`,
		roleID, resource, action)
	return err
}

func (r *pgRepo) GrantPlatformRole(ctx context.Context, roleID, platformUserID, grantedBy kernel.ID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_role_grant (role_id, platform_user_id, granted_by)
		 VALUES ($1, $2, $3) ON CONFLICT (role_id, platform_user_id) DO NOTHING`,
		roleID, platformUserID, idOrNilV(grantedBy))
	return err
}

func (r *pgRepo) RevokePlatformRole(ctx context.Context, roleID, platformUserID kernel.ID) error {
	_, err := r.pool.Exec(ctx,
		`DELETE FROM public.platform_role_grant WHERE role_id = $1 AND platform_user_id = $2`,
		roleID, platformUserID)
	return err
}

func (r *pgRepo) ListPlatformRoleMembers(ctx context.Context, roleID kernel.ID) ([]kernel.ID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT platform_user_id FROM public.platform_role_grant WHERE role_id = $1`, roleID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []kernel.ID{}
	for rows.Next() {
		var id kernel.ID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (r *pgRepo) EnsureSuperAdminGrants(ctx context.Context) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_role_grant (role_id, platform_user_id, granted_by)
		 SELECT r.id, u.id, NULL
		 FROM public.platform_user u
		 CROSS JOIN public.role r
		 WHERE u.is_platform_admin = TRUE AND r.code = 'super_admin' AND r.tenant_id IS NULL
		 ON CONFLICT (role_id, platform_user_id) DO NOTHING`)
	return err
}

func (r *pgRepo) GetPlatformSetting(ctx context.Context, key string) (string, error) {
	var raw []byte
	err := r.pool.QueryRow(ctx, `SELECT value FROM public.platform_setting WHERE key = $1`, key).Scan(&raw)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		return "", fmt.Errorf("platform_setting %q is not a JSON string: %w", key, err)
	}
	return s, nil
}

func (r *pgRepo) SetPlatformSetting(ctx context.Context, key, value string, by kernel.ID) error {
	v, _ := json.Marshal(value)
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.platform_setting (key, value, updated_by, updated_at)
		 VALUES ($1, $2::jsonb, $3, now())
		 ON CONFLICT (key) DO UPDATE SET value = $2::jsonb, updated_by = $3, updated_at = now()`,
		key, string(v), idOrNilV(by))
	return err
}
