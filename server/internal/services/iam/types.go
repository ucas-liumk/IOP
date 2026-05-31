package iam

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// PlatformUser is the global cross-tenant identity.
type PlatformUser struct {
	ID                 kernel.ID  `json:"id"`
	Username           string     `json:"username,omitempty"`
	Phone              string     `json:"phone,omitempty"`
	Email              string     `json:"email,omitempty"`
	PasswordHash       string     `json:"-"`
	MFASecret          string     `json:"-"`
	Status             string     `json:"status"`
	PasswordMustChange bool       `json:"password_must_change"`
	IsPlatformAdmin    bool       `json:"is_platform_admin"`
	LastLoginAt        *time.Time `json:"last_login_at,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

// Session represents one login.
type Session struct {
	ID             kernel.ID  `json:"id"`
	PlatformUserID kernel.ID  `json:"platform_user_id"`
	TenantID       *kernel.ID `json:"tenant_id,omitempty"`
	MemberID       *kernel.ID `json:"member_id,omitempty"`
	IssuedAt       time.Time  `json:"issued_at"`
	ExpiresAt      time.Time  `json:"expires_at"`
	Revoked        bool       `json:"revoked"`
	IPAddress      string     `json:"ip_address,omitempty"`
	UserAgent      string     `json:"user_agent,omitempty"`
}

// Role + policy + grant.
type Role struct {
	ID          kernel.ID  `json:"id"`
	TenantID    *kernel.ID `json:"tenant_id,omitempty"`
	Code        string     `json:"code"`
	Name        string     `json:"name"`
	Status      string     `json:"status"`
	OrderNum    int        `json:"order_num"`
	Remark      string     `json:"remark,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	MemberCount int        `json:"member_count"`
}

type PolicyRule struct {
	RoleID   kernel.ID `json:"role_id"`
	Resource string    `json:"resource"`
	Action   string    `json:"action"`
	Effect   string    `json:"effect"`
}

type RoleGrant struct {
	RoleID    kernel.ID `json:"role_id"`
	MemberID  kernel.ID `json:"member_id"`
	TenantID  kernel.ID `json:"tenant_id"`
	GrantedAt time.Time `json:"granted_at"`
}

// TokenPair returned by Login / Refresh.
type TokenPair struct {
	AccessToken      string    `json:"access_token"`
	RefreshToken     string    `json:"refresh_token"`
	AccessExpiresAt  time.Time `json:"access_expires_at"`
	RefreshExpiresAt time.Time `json:"refresh_expires_at"`
}

// Claims that go inside JWT.
type Claims struct {
	PlatformUserID kernel.ID `json:"sub"`
	SessionID      kernel.ID `json:"sid"`
	TenantID       kernel.ID `json:"tid,omitempty"`
	MemberID       kernel.ID `json:"mid,omitempty"`
	Type           string    `json:"typ"` // "access" | "refresh"
	IssuedAt       int64     `json:"iat"`
	ExpiresAt      int64     `json:"exp"`
}
