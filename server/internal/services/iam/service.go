package iam

import (
	"context"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
)

const (
	accessTTL  = 30 * time.Minute
	refreshTTL = 24 * time.Hour
)

// Service is the public API of IAM.
type Service struct {
	repo    Repository
	appRepo ApplicationRepository
	signer  TokenSigner
	tenants *tenancy.Service
	rdb     *redis.Client // session blacklist; nil = no Redis (degraded mode)
	bus     eventbus.Bus
	clock   kernel.Clock
}

func NewService(repo Repository, appRepo ApplicationRepository, signer TokenSigner, tenants *tenancy.Service, rdb *redis.Client, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{repo: repo, appRepo: appRepo, signer: signer, tenants: tenants, rdb: rdb, bus: bus, clock: clk}
}

// RegisterUser creates a new platform user.
type RegisterCmd struct {
	Username string // required for self-signup (used as login + tenant slug)
	Phone    string // optional; CN mobile format (11 digits, starts 1[3-9])
	Email    string // optional
	Password string
}

// Username constraint mirrors the tenant slug regex so we can reuse a username as a slug.
var usernameRe = regexp.MustCompile(`^[a-z][a-z0-9_-]{1,30}[a-z0-9]$`)

// CN mobile pattern — basic shape check. Real verification will happen via SMS later.
var phoneRe = regexp.MustCompile(`^1[3-9]\d{9}$`)

func (s *Service) RegisterUser(ctx context.Context, cmd RegisterCmd) (*PlatformUser, error) {
	username := strings.TrimSpace(cmd.Username)
	if username != "" {
		if existing, _ := s.repo.GetUserByUsername(ctx, username); existing != nil {
			return nil, errors.New(errors.KindConflict, "iam.username_taken", "用户名已占用")
		}
	}
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if email != "" {
		if !strings.Contains(email, "@") {
			return nil, errors.New(errors.KindParam, "iam.invalid_email", "邮箱格式错误")
		}
		if existing, _ := s.repo.GetUserByEmail(ctx, email); existing != nil {
			return nil, errors.New(errors.KindConflict, "iam.email_taken", "邮箱已注册")
		}
	}
	phone := strings.TrimSpace(cmd.Phone)
	if phone != "" {
		if !phoneRe.MatchString(phone) {
			return nil, errors.New(errors.KindParam, "iam.invalid_phone", "手机号格式错误")
		}
		if existing, _ := s.repo.GetUserByPhone(ctx, phone); existing != nil {
			return nil, errors.New(errors.KindConflict, "iam.phone_taken", "手机号已注册")
		}
	}
	hash, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, err
	}
	u := &PlatformUser{
		ID:           kernel.NewID(),
		Username:     username,
		Phone:        phone,
		Email:        email,
		PasswordHash: hash,
		Status:       "active",
		CreatedAt:    s.clock.Now(),
	}
	if err := s.repo.CreateUser(ctx, u); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.create_user_failed", "create user failed", err)
	}
	return u, nil
}

type LoginCmd struct {
	Login    string // username OR email
	Password string
	IP       string
	UA       string
}

// Login authenticates the user. Returns a TokenPair with no tenant bound;
// caller is expected to call SwitchTenant once they know which tenant.
func (s *Service) Login(ctx context.Context, cmd LoginCmd) (*TokenPair, *PlatformUser, error) {
	login := strings.TrimSpace(cmd.Login)
	// Emails are case-insensitive; usernames are stored exactly as registered.
	if strings.Contains(login, "@") {
		login = strings.ToLower(login)
	}
	// Brute-force lockout: after maxLoginFails wrong attempts on this login id,
	// reject for loginLockTTL regardless of credentials. Keyed by the typed login
	// (covers both unknown-user and wrong-password) so it also blunts enumeration.
	failKey := "login_fail:" + login
	if s.loginLocked(ctx, failKey) {
		return nil, nil, errors.New(errors.KindRateLimit, "iam.too_many_attempts",
			"登录失败次数过多，请 15 分钟后再试")
	}
	u, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return nil, nil, err
	}
	if u == nil {
		s.recordLoginFail(ctx, failKey)
		return nil, nil, errors.New(errors.KindAuth, "iam.invalid_password", "用户名或密码错误")
	}
	if err := CheckPassword(cmd.Password, u.PasswordHash); err != nil {
		s.recordLoginFail(ctx, failKey)
		_ = s.bus.Publish(ctx, "iam.login_failed", map[string]any{"login": login})
		return nil, nil, err
	}
	// Status checked AFTER password so a disabled account is indistinguishable from
	// a wrong password to an unauthenticated probe (no user-enumeration oracle).
	if u.Status != "active" {
		return nil, nil, errors.New(errors.KindForbidden, "iam.user_disabled", "账号已停用")
	}
	s.clearLoginFail(ctx, failKey)
	now := s.clock.Now()
	sess := &Session{
		ID:             kernel.NewID(),
		PlatformUserID: u.ID,
		IssuedAt:       now,
		ExpiresAt:      now.Add(refreshTTL),
		IPAddress:      cmd.IP,
		UserAgent:      cmd.UA,
	}
	if err := s.repo.CreateSession(ctx, sess); err != nil {
		return nil, nil, err
	}
	_ = s.repo.UpdateLastLogin(ctx, u.ID, now)
	tok, err := s.issueTokens(sess, nil, nil)
	if err != nil {
		return nil, nil, err
	}
	_ = s.bus.Publish(ctx, "iam.user_logged_in", map[string]any{
		"platform_user_id": u.ID, "session_id": sess.ID, "ip": cmd.IP,
	})
	return tok, u, nil
}

// SwitchTenant validates membership and reissues tokens with tenant/member bound.
func (s *Service) SwitchTenant(ctx context.Context, sessionID, tenantID kernel.ID) (*TokenPair, error) {
	sess, err := s.repo.GetSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil || sess.Revoked || s.clock.Now().After(sess.ExpiresAt) {
		return nil, errors.New(errors.KindAuth, "iam.session_invalid", "会话已失效")
	}
	mem, err := s.tenants.GetMembership(ctx, sess.PlatformUserID, tenantID)
	if err != nil {
		return nil, err
	}
	if mem == nil {
		return nil, errors.New(errors.KindForbidden, "iam.not_a_member", "不是该租户成员")
	}
	t, err := s.tenants.GetTenant(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Status != tenancy.StatusActive {
		return nil, errors.New(errors.KindForbidden, "iam.tenant_inactive", "租户不可用")
	}
	tid := tenantID
	mid := mem.MemberID
	return s.issueTokens(sess, &tid, &mid)
}

// Refresh issues a fresh access token from a refresh token.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*TokenPair, error) {
	claims, err := s.signer.Verify(refreshToken)
	if err != nil {
		return nil, err
	}
	if claims.Type != "refresh" {
		return nil, errors.New(errors.KindAuth, "iam.wrong_token_type", "token 类型错误")
	}
	if s.isBlacklisted(ctx, claims.SessionID) {
		return nil, errors.New(errors.KindAuth, "iam.session_revoked", "会话已注销")
	}
	sess, err := s.repo.GetSession(ctx, claims.SessionID)
	if err != nil {
		return nil, err
	}
	if sess == nil || sess.Revoked {
		return nil, errors.New(errors.KindAuth, "iam.session_invalid", "会话无效")
	}
	tid, mid := claims.TenantID, claims.MemberID
	var tidP, midP *kernel.ID
	if tid != "" {
		tidP = &tid
	}
	if mid != "" {
		midP = &mid
	}
	return s.issueTokens(sess, tidP, midP)
}

// Logout revokes the session both in DB and Redis blacklist.
func (s *Service) Logout(ctx context.Context, sessionID kernel.ID) error {
	if err := s.repo.RevokeSession(ctx, sessionID); err != nil {
		return err
	}
	if s.rdb != nil {
		_ = s.rdb.Set(ctx, "session:revoked:"+string(sessionID), "1", refreshTTL).Err()
	}
	_ = s.bus.Publish(ctx, "iam.user_logged_out", map[string]any{"session_id": sessionID})
	return nil
}

// VerifyAccessToken parses + checks blacklist. Returns Claims on success.
func (s *Service) VerifyAccessToken(ctx context.Context, token string) (*Claims, error) {
	c, err := s.signer.Verify(token)
	if err != nil {
		return nil, err
	}
	if c.Type != "access" {
		return nil, errors.New(errors.KindAuth, "iam.wrong_token_type", "token 类型错误")
	}
	if s.isBlacklisted(ctx, c.SessionID) {
		return nil, errors.New(errors.KindAuth, "iam.session_revoked", "会话已注销")
	}
	return c, nil
}

func (s *Service) isBlacklisted(ctx context.Context, sessionID kernel.ID) bool {
	if s.rdb == nil {
		return false
	}
	v, _ := s.rdb.Get(ctx, "session:revoked:"+string(sessionID)).Result()
	return v == "1"
}

// Brute-force lockout tunables.
const (
	maxLoginFails = 5
	loginLockTTL  = 15 * time.Minute
)

func (s *Service) loginLocked(ctx context.Context, failKey string) bool {
	if s.rdb == nil {
		return false
	}
	n, _ := s.rdb.Get(ctx, failKey).Int()
	return n >= maxLoginFails
}

func (s *Service) recordLoginFail(ctx context.Context, failKey string) {
	if s.rdb == nil {
		return
	}
	n, _ := s.rdb.Incr(ctx, failKey).Result()
	if n == 1 {
		_ = s.rdb.Expire(ctx, failKey, loginLockTTL).Err()
	}
}

func (s *Service) clearLoginFail(ctx context.Context, failKey string) {
	if s.rdb == nil {
		return
	}
	_ = s.rdb.Del(ctx, failKey).Err()
}

// RevokeAllSessions revokes every active session for a platform user (DB flag +
// Redis blacklist) so a disabled/role-changed account loses access immediately
// instead of staying valid until token expiry.
func (s *Service) RevokeAllSessions(ctx context.Context, userID kernel.ID) error {
	ids, err := s.repo.RevokeUserSessions(ctx, userID)
	if err != nil {
		return err
	}
	if s.rdb != nil {
		for _, id := range ids {
			_ = s.rdb.Set(ctx, "session:revoked:"+string(id), "1", refreshTTL).Err()
		}
	}
	return nil
}

func (s *Service) issueTokens(sess *Session, tid, mid *kernel.ID) (*TokenPair, error) {
	now := s.clock.Now()
	access := Claims{
		PlatformUserID: sess.PlatformUserID,
		SessionID:      sess.ID,
		Type:           "access",
		IssuedAt:       now.Unix(),
		ExpiresAt:      now.Add(accessTTL).Unix(),
	}
	refresh := access
	refresh.Type = "refresh"
	refresh.ExpiresAt = now.Add(refreshTTL).Unix()
	if tid != nil {
		access.TenantID = *tid
		refresh.TenantID = *tid
	}
	if mid != nil {
		access.MemberID = *mid
		refresh.MemberID = *mid
	}
	at, err := s.signer.Sign(access)
	if err != nil {
		return nil, err
	}
	rt, err := s.signer.Sign(refresh)
	if err != nil {
		return nil, err
	}
	return &TokenPair{
		AccessToken:      at,
		RefreshToken:     rt,
		AccessExpiresAt:  time.Unix(access.ExpiresAt, 0),
		RefreshExpiresAt: time.Unix(refresh.ExpiresAt, 0),
	}, nil
}

// ============================================================================
// RBAC
// ============================================================================

// Enforce returns nil if the (memberID, tenantID) has at least one role with a
// policy matching (resource, action) effect=allow.
func (s *Service) Enforce(ctx context.Context, memberID, tenantID kernel.ID, resource, action string) error {
	roles, err := s.repo.ListMemberRoles(ctx, memberID, tenantID)
	if err != nil {
		return err
	}
	if len(roles) == 0 {
		return errors.New(errors.KindForbidden, "iam.no_role", "无权限")
	}
	// No code-level admin bypass: built-in admin roles (tenant_admin/platform_admin)
	// carry an all-access '*'/'*' role_policy (seeded in migration 000010), so they
	// pass the generic policy match below like any other role.
	ids := make([]kernel.ID, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}
	policies, err := s.repo.ListPolicyForRoles(ctx, ids)
	if err != nil {
		return err
	}
	for _, p := range policies {
		if matchPolicy(p.Resource, resource) && matchPolicy(p.Action, action) && p.Effect == "allow" {
			return nil
		}
	}
	return errors.New(errors.KindForbidden, "iam.access_denied", "无权限访问 "+resource+":"+action)
}

func matchPolicy(pattern, value string) bool {
	if pattern == "*" || pattern == value {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		return strings.HasPrefix(value, strings.TrimSuffix(pattern, "*"))
	}
	return false
}

// PermitsRule reports whether the given set of effective policy rules permits
// (resource, action). It mirrors Enforce's match (wildcard-aware, effect=allow)
// but operates IN MEMORY on rules already fetched — so callers can pull a user's
// rules ONCE and test many resource:action pairs (e.g. filtering a menu tree)
// without per-check DB round trips.
func PermitsRule(rules []PolicyRule, resource, action string) bool {
	for _, p := range rules {
		if p.Effect == "allow" && matchPolicy(p.Resource, resource) && matchPolicy(p.Action, action) {
			return true
		}
	}
	return false
}

// MemberPerms returns the member's effective policy rules for the tenant
// (ListMemberRoles → ListPolicyForRoles), de-referenced to values. Fetch once,
// then test many resource:action pairs with PermitsRule. Returns an empty slice
// (no error) when the member has no roles.
func (s *Service) MemberPerms(ctx context.Context, memberID, tenantID kernel.ID) ([]PolicyRule, error) {
	roles, err := s.repo.ListMemberRoles(ctx, memberID, tenantID)
	if err != nil {
		return nil, err
	}
	if len(roles) == 0 {
		return []PolicyRule{}, nil
	}
	ids := make([]kernel.ID, len(roles))
	for i, r := range roles {
		ids[i] = r.ID
	}
	pols, err := s.repo.ListPolicyForRoles(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make([]PolicyRule, 0, len(pols))
	for _, p := range pols {
		if p != nil {
			out = append(out, *p)
		}
	}
	return out, nil
}

// GrantRoleByCode grants a role (by code) to a member.
func (s *Service) GrantRoleByCode(ctx context.Context, memberID, tenantID kernel.ID, roleCode string) error {
	role, err := s.repo.GetRoleByCode(ctx, roleCode, nil)
	if err != nil {
		return err
	}
	if role == nil {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "角色不存在: "+roleCode)
	}
	return s.repo.GrantRole(ctx, &RoleGrant{
		RoleID: role.ID, MemberID: memberID, TenantID: tenantID, GrantedAt: s.clock.Now(),
	})
}

// GrantRoleByID grants a role (by id) to a member of the given tenant. The role
// must be visible in that tenant — either a platform-wide built-in (tenant_id IS
// NULL) or one owned by this tenant — so a platform admin acting on org :tid can
// only grant roles that legitimately belong to it. The (role,member,tenant) grant
// is idempotent.
func (s *Service) GrantRoleByID(ctx context.Context, memberID, tenantID, roleID kernel.ID) error {
	pool := s.repo.(*pgRepo).pool
	var exists bool
	if err := pool.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM public.role
		 WHERE id = $1 AND (tenant_id IS NULL OR tenant_id = $2))`,
		roleID, tenantID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return errors.New(errors.KindNotFound, "iam.role_not_found", "角色不存在")
	}
	return s.repo.GrantRole(ctx, &RoleGrant{
		RoleID: roleID, MemberID: memberID, TenantID: tenantID, GrantedAt: s.clock.Now(),
	})
}
