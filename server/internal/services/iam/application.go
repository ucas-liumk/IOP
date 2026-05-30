package iam

import (
	"context"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// Statuses for registration_application.status.
const (
	AppStatusPending  = "pending"
	AppStatusApproved = "approved"
	AppStatusRejected = "rejected"
)

// RegistrationApplication mirrors public.registration_application.
type RegistrationApplication struct {
	ID             kernel.ID  `json:"id"`
	Username       string     `json:"username"`
	RealName       string     `json:"real_name"`
	Organization   string     `json:"organization"`
	Phone          string     `json:"phone,omitempty"`
	PasswordHash   string     `json:"-"`
	Status         string     `json:"status"`
	AppliedAt      time.Time  `json:"applied_at"`
	ReviewedBy     *kernel.ID `json:"reviewed_by,omitempty"`
	ReviewedAt     *time.Time `json:"reviewed_at,omitempty"`
	RejectReason   string     `json:"reject_reason,omitempty"`
	TargetTenantID *kernel.ID `json:"target_tenant_id,omitempty"`
	GrantedRole    string     `json:"granted_role,omitempty"`
}

// ApplicationRepository persists pending/approved/rejected applications.
type ApplicationRepository interface {
	Create(ctx context.Context, a *RegistrationApplication) error
	GetByID(ctx context.Context, id kernel.ID) (*RegistrationApplication, error)
	GetPendingByUsername(ctx context.Context, username string) (*RegistrationApplication, error)
	List(ctx context.Context, status string) ([]*RegistrationApplication, error)
	MarkApproved(ctx context.Context, id kernel.ID, reviewer kernel.ID, tenantID kernel.ID, role string, at time.Time) error
	MarkRejected(ctx context.Context, id kernel.ID, reviewer kernel.ID, reason string, at time.Time) error
}

type pgAppRepo struct{ pool *pgxpool.Pool }

func NewPGApplicationRepo(pool *pgxpool.Pool) ApplicationRepository { return &pgAppRepo{pool: pool} }

const appSelectCols = `id, username, real_name, organization, COALESCE(phone,''), password_hash, status, applied_at, reviewed_by, reviewed_at, COALESCE(reject_reason,''), target_tenant_id, COALESCE(granted_role,'')`

func scanApp(row pgx.Row) (*RegistrationApplication, error) {
	var a RegistrationApplication
	err := row.Scan(
		&a.ID, &a.Username, &a.RealName, &a.Organization, &a.Phone, &a.PasswordHash,
		&a.Status, &a.AppliedAt, &a.ReviewedBy, &a.ReviewedAt, &a.RejectReason,
		&a.TargetTenantID, &a.GrantedRole,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func (r *pgAppRepo) Create(ctx context.Context, a *RegistrationApplication) error {
	var phone any
	if a.Phone != "" {
		phone = a.Phone
	}
	_, err := r.pool.Exec(ctx,
		`INSERT INTO public.registration_application
		   (id, username, real_name, organization, phone, password_hash, status, applied_at, target_tenant_id)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		a.ID, a.Username, a.RealName, a.Organization, phone, a.PasswordHash, a.Status, a.AppliedAt, a.TargetTenantID)
	return err
}

func (r *pgAppRepo) GetByID(ctx context.Context, id kernel.ID) (*RegistrationApplication, error) {
	return scanApp(r.pool.QueryRow(ctx,
		`SELECT `+appSelectCols+` FROM public.registration_application WHERE id = $1`, id))
}

func (r *pgAppRepo) GetPendingByUsername(ctx context.Context, username string) (*RegistrationApplication, error) {
	return scanApp(r.pool.QueryRow(ctx,
		`SELECT `+appSelectCols+` FROM public.registration_application
		 WHERE username = $1 AND status = 'pending'`, username))
}

func (r *pgAppRepo) List(ctx context.Context, status string) ([]*RegistrationApplication, error) {
	var rows pgx.Rows
	var err error
	if status == "" {
		rows, err = r.pool.Query(ctx,
			`SELECT `+appSelectCols+` FROM public.registration_application
			 ORDER BY applied_at DESC LIMIT 200`)
	} else {
		rows, err = r.pool.Query(ctx,
			`SELECT `+appSelectCols+` FROM public.registration_application
			 WHERE status = $1 ORDER BY applied_at DESC LIMIT 200`, status)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []*RegistrationApplication{}
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *pgAppRepo) MarkApproved(ctx context.Context, id kernel.ID, reviewer kernel.ID, tenantID kernel.ID, role string, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.registration_application
		   SET status = 'approved', reviewed_by = $2, reviewed_at = $3,
		       target_tenant_id = $4, granted_role = $5
		 WHERE id = $1 AND status = 'pending'`,
		id, reviewer, at, tenantID, role)
	return err
}

func (r *pgAppRepo) MarkRejected(ctx context.Context, id kernel.ID, reviewer kernel.ID, reason string, at time.Time) error {
	_, err := r.pool.Exec(ctx,
		`UPDATE public.registration_application
		   SET status = 'rejected', reviewed_by = $2, reviewed_at = $3, reject_reason = $4
		 WHERE id = $1 AND status = 'pending'`,
		id, reviewer, at, reason)
	return err
}

// === Service layer ===

// SubmitApplicationCmd is the public-facing submit-an-application payload.
// OrganizationID points at an active tenant; the tenant becomes the application's
// target. The applicant cannot pick a tenant that is suspended or non-existent.
type SubmitApplicationCmd struct {
	Username       string
	RealName       string
	OrganizationID kernel.ID
	Phone          string
	Password       string
}

// SubmitApplication hashes the password and inserts a pending row.
// Validation: username regex; tenant must exist & be active; collision check
// across pending apps and existing users.
func (s *Service) SubmitApplication(ctx context.Context, cmd SubmitApplicationCmd) (*RegistrationApplication, error) {
	username := strings.TrimSpace(cmd.Username)
	if !usernameRe.MatchString(username) {
		return nil, errors.New(errors.KindParam, "iam.invalid_username",
			"用户名 3-32 位，仅含小写字母/数字/-/_，须以字母开头、字母或数字结尾")
	}
	if strings.TrimSpace(cmd.RealName) == "" {
		return nil, errors.New(errors.KindParam, "iam.real_name_required", "请填写真实姓名")
	}
	if cmd.OrganizationID == "" {
		return nil, errors.New(errors.KindParam, "iam.organization_required", "请选择所在单位")
	}
	t, err := s.tenants.GetTenant(ctx, cmd.OrganizationID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Status != tenancy.StatusActive {
		return nil, errors.New(errors.KindParam, "iam.organization_not_found", "所选单位不存在或已停用")
	}
	phone := strings.TrimSpace(cmd.Phone)
	if phone != "" && !phoneRe.MatchString(phone) {
		return nil, errors.New(errors.KindParam, "iam.invalid_phone", "手机号格式错误")
	}
	// Username must not collide with existing users or other pending apps.
	if existing, _ := s.repo.GetUserByUsername(ctx, username); existing != nil {
		return nil, errors.New(errors.KindConflict, "iam.username_taken", "用户名已占用")
	}
	if existing, _ := s.appRepo.GetPendingByUsername(ctx, username); existing != nil {
		return nil, errors.New(errors.KindConflict, "iam.application_pending",
			"该用户名已有正在审批的申请")
	}
	hash, err := HashPassword(cmd.Password)
	if err != nil {
		return nil, err
	}
	tid := t.ID
	app := &RegistrationApplication{
		ID:             kernel.NewID(),
		Username:       username,
		RealName:       strings.TrimSpace(cmd.RealName),
		Organization:   t.Name, // snapshot tenant name at submit time
		Phone:          phone,
		PasswordHash:   hash,
		Status:         AppStatusPending,
		AppliedAt:      s.clock.Now(),
		TargetTenantID: &tid,
	}
	if err := s.appRepo.Create(ctx, app); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "iam.application_create_failed", "提交失败", err)
	}
	_ = s.bus.Publish(ctx, "iam.application_submitted", map[string]any{
		"application_id": app.ID, "username": username,
		"tenant_id": tid, "organization": app.Organization,
	})
	return app, nil
}

// GetApplication fetches a single application by ID.
func (s *Service) GetApplication(ctx context.Context, id kernel.ID) (*RegistrationApplication, error) {
	return s.appRepo.GetByID(ctx, id)
}

// ListApplications returns up to 200 rows, optionally filtered by status.
// If tenantFilter is non-empty, only applications targeted at that tenant are returned —
// used to scope a tenant_admin's view to their own tenant.
func (s *Service) ListApplications(ctx context.Context, status string, tenantFilter *kernel.ID) ([]*RegistrationApplication, error) {
	rows, err := s.appRepo.List(ctx, status)
	if err != nil {
		return nil, err
	}
	if tenantFilter == nil {
		return rows, nil
	}
	out := rows[:0]
	for _, r := range rows {
		if r.TargetTenantID != nil && *r.TargetTenantID == *tenantFilter {
			out = append(out, r)
		}
	}
	return out, nil
}

// ApproveApplicationCmd is the reviewer's input.
// The tenant is the one the applicant chose at submission time; reviewer only sets the role.
type ApproveApplicationCmd struct {
	ApplicationID kernel.ID
	ReviewerID    kernel.ID // platform_user_id of reviewer
	Role          string    // tenant_member (default) or tenant_admin
}

// reusableUser decides how to obtain a platform_user when (re)provisioning the
// given username into targetTenant. The provisioning steps below are not wrapped
// in a single cross-schema transaction (JoinMember writes its own tenant schema),
// so a transient failure mid-flow can leave a created user with no/partial
// membership. To make a retry succeed instead of dead-ending on "username_taken",
// we reuse such an orphan: a user that exists but belongs to no OTHER tenant is
// either a fresh orphan (no memberships) or already half-provisioned into the
// target tenant — both safe to continue from (JoinMember + GrantRole are
// idempotent). A user that is a member of a DIFFERENT tenant is a genuine
// username collision and is rejected.
//
// Returns (user, reuse=true) to reuse, (nil, false) to create fresh, or an error.
func (s *Service) reusableUser(ctx context.Context, username string, targetTenant kernel.ID) (*PlatformUser, bool, error) {
	existing, err := s.repo.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, false, err
	}
	if existing == nil {
		return nil, false, nil
	}
	mine, err := s.tenants.ListMyTenants(ctx, existing.ID)
	if err != nil {
		return nil, false, err
	}
	for _, t := range mine {
		if t.ID != targetTenant {
			return nil, false, errors.New(errors.KindConflict, "iam.username_taken", "用户名已被占用")
		}
	}
	return existing, true, nil
}

// ApproveApplication runs the full provisioning: create user, join member, grant role,
// mark application approved. Safe to retry after a partial failure: an orphaned user
// from a prior attempt is reused rather than dead-ending (see reusableUser), and
// JoinMember / GrantRoleByCode / MarkApproved are each idempotent.
func (s *Service) ApproveApplication(ctx context.Context, pool *pgxpool.Pool, cmd ApproveApplicationCmd) (*PlatformUser, error) {
	app, err := s.appRepo.GetByID(ctx, cmd.ApplicationID)
	if err != nil {
		return nil, err
	}
	if app == nil {
		return nil, errors.New(errors.KindNotFound, "iam.application_not_found", "申请不存在")
	}
	if app.Status != AppStatusPending {
		return nil, errors.New(errors.KindConflict, "iam.application_not_pending", "该申请已处理")
	}
	role := strings.TrimSpace(cmd.Role)
	if role == "" {
		role = "tenant_member"
	}
	if role != "tenant_member" && role != "tenant_admin" {
		return nil, errors.New(errors.KindParam, "iam.invalid_role", "角色无效")
	}
	if app.TargetTenantID == nil {
		return nil, errors.New(errors.KindParam, "iam.application_no_target", "申请未指定目标单位")
	}
	t, err := s.tenants.GetTenant(ctx, *app.TargetTenantID)
	if err != nil {
		return nil, err
	}
	if t == nil || t.Status != tenancy.StatusActive {
		return nil, errors.New(errors.KindForbidden, "iam.tenant_inactive", "目标单位已停用，请先恢复后再审批")
	}

	reuse, ok, err := s.reusableUser(ctx, app.Username, t.ID)
	if err != nil {
		return nil, err
	}
	var u *PlatformUser
	if ok {
		u = reuse // orphan / half-provisioned from a prior attempt — continue idempotently
	} else {
		// Fresh creation: pre-check phone uniqueness for a friendly 409 instead of a
		// raw unique-violation 500 from the INSERT.
		if app.Phone != "" {
			if ex, _ := s.repo.GetUserByPhone(ctx, app.Phone); ex != nil {
				return nil, errors.New(errors.KindConflict, "iam.phone_taken", "手机号已被注册")
			}
		}
		u = &PlatformUser{
			ID:           kernel.NewID(),
			Username:     app.Username,
			Phone:        app.Phone,
			PasswordHash: app.PasswordHash,
			Status:       "active",
			CreatedAt:    s.clock.Now(),
		}
		if err := s.repo.CreateUser(ctx, u); err != nil {
			return nil, errors.Wrap(errors.KindDatabase, "iam.create_user_failed", "创建账号失败", err)
		}
	}
	mem, err := s.tenants.JoinMember(ctx, pool, tenancy.JoinMemberCmd{
		PlatformUserID: u.ID, TenantID: t.ID,
		DisplayName: app.RealName, Phone: app.Phone,
	})
	if err != nil {
		return nil, err
	}
	if err := s.GrantRoleByCode(ctx, mem.MemberID, t.ID, role); err != nil {
		return nil, err
	}
	if err := s.appRepo.MarkApproved(ctx, app.ID, cmd.ReviewerID, t.ID, role, s.clock.Now()); err != nil {
		return nil, err
	}
	_ = s.bus.Publish(ctx, "iam.application_approved", map[string]any{
		"application_id": app.ID, "platform_user_id": u.ID,
		"tenant_id": t.ID, "role": role, "reviewer_id": cmd.ReviewerID,
	})
	return u, nil
}

// RejectApplication marks the row rejected and records reason.
func (s *Service) RejectApplication(ctx context.Context, appID kernel.ID, reviewerID kernel.ID, reason string) error {
	app, err := s.appRepo.GetByID(ctx, appID)
	if err != nil {
		return err
	}
	if app == nil {
		return errors.New(errors.KindNotFound, "iam.application_not_found", "申请不存在")
	}
	if app.Status != AppStatusPending {
		return errors.New(errors.KindConflict, "iam.application_not_pending", "该申请已处理")
	}
	if err := s.appRepo.MarkRejected(ctx, appID, reviewerID, strings.TrimSpace(reason), s.clock.Now()); err != nil {
		return err
	}
	_ = s.bus.Publish(ctx, "iam.application_rejected", map[string]any{
		"application_id": appID, "reviewer_id": reviewerID, "reason": reason,
	})
	return nil
}
