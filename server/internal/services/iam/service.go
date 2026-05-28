package iam

import (
	"context"
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
	signer  TokenSigner
	tenants *tenancy.Service
	rdb     *redis.Client // session blacklist; nil = no Redis (degraded mode)
	bus     eventbus.Bus
	clock   kernel.Clock
}

func NewService(repo Repository, signer TokenSigner, tenants *tenancy.Service, rdb *redis.Client, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{repo: repo, signer: signer, tenants: tenants, rdb: rdb, bus: bus, clock: clk}
}

// RegisterUser creates a new platform user.
type RegisterCmd struct {
	Email    string
	Password string
}

func (s *Service) RegisterUser(ctx context.Context, cmd RegisterCmd) (*PlatformUser, error) {
	email := strings.ToLower(strings.TrimSpace(cmd.Email))
	if !strings.Contains(email, "@") {
		return nil, errors.New(errors.KindParam, "iam.invalid_email", "邮箱格式错误")
	}
	if existing, _ := s.repo.GetUserByEmail(ctx, email); existing != nil {
		return nil, errors.New(errors.KindConflict, "iam.email_taken", "邮箱已注册")
	}
	hash, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, err
	}
	u := &PlatformUser{
		ID:           kernel.NewID(),
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
	Email    string
	Password string
	IP       string
	UA       string
}

// Login authenticates the user. Returns a TokenPair with no tenant bound;
// caller is expected to call SwitchTenant once they know which tenant.
func (s *Service) Login(ctx context.Context, cmd LoginCmd) (*TokenPair, *PlatformUser, error) {
	u, err := s.repo.GetUserByEmail(ctx, strings.ToLower(strings.TrimSpace(cmd.Email)))
	if err != nil {
		return nil, nil, err
	}
	if u == nil {
		return nil, nil, errors.New(errors.KindAuth, "iam.invalid_password", "用户名或密码错误")
	}
	if u.Status != "active" {
		return nil, nil, errors.New(errors.KindForbidden, "iam.user_disabled", "账号已停用")
	}
	if err := CheckPassword(cmd.Password, u.PasswordHash); err != nil {
		_ = s.bus.Publish(ctx, "iam.login_failed", map[string]any{"email": u.Email})
		return nil, nil, err
	}
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
	// Built-in shortcut: tenant_admin / platform_admin bypass.
	for _, r := range roles {
		if r.Code == "platform_admin" || r.Code == "tenant_admin" {
			return nil
		}
	}
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
