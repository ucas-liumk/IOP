package tenancy

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// memberCSVHeader is the column order for member export/import/template.
var memberCSVHeader = []string{"username", "display_name", "phone", "email", "gender", "org_code", "post_code", "role_code", "status", "initial_password", "remark"}

var memberExportHeader = []string{"username", "display_name", "phone", "email", "gender", "org_code", "department", "org_path", "title", "posts", "roles", "status", "created_at", "remark"}

// memberTemplateRows are illustrative sample rows for the member import template.
var memberTemplateRows = [][]string{
	{"zhangsan", "张三", "13800000001", "zhangsan@example.com", "male", "RD", "engineer", "tenant_member", "active", "", "研发部成员"},
	{"lisi", "李四", "13800000002", "", "female", "ALG", "leader", "tenant_member", "active", "", "算法组成员"},
}

// MemberCSVHeader returns a copy of the member CSV column order.
func MemberCSVHeader() []string { return append([]string(nil), memberCSVHeader...) }

// MemberTemplateRows returns the header + sample rows for the member import template.
func MemberTemplateRows() [][]string {
	out := make([][]string, 0, len(memberTemplateRows)+1)
	out = append(out, MemberCSVHeader())
	out = append(out, memberTemplateRows...)
	return out
}

// ExportMembers returns members matching the current tenant-scoped filters as
// spreadsheet rows (header first). Phone/email are masked because exports leave
// the online permission boundary.
func (s *Service) ExportMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd ListMembersCmd) ([][]string, error) {
	out := [][]string{append([]string(nil), memberExportHeader...)}
	cmd.Page = kernel.Pagination{Page: 1, PageSize: 200}
	for {
		page, err := s.ListMembers(ctx, pool, t, cmd)
		if err != nil {
			return nil, err
		}
		for _, m := range page.Data {
			out = append(out, []string{
				m.Username,
				m.DisplayName,
				maskPhone(m.Phone),
				maskEmail(m.Email),
				m.Gender,
				m.DeptCode,
				m.Department,
				m.DeptPath,
				m.Title,
				memberPostsText(m.Posts),
				memberRolesText(m.Roles),
				m.Status,
				m.JoinedAt,
				m.Remark,
			})
		}
		if len(page.Data) == 0 || cmd.Page.Page*cmd.Page.PageSize >= page.Total {
			break
		}
		cmd.Page.Page++
	}
	return out, nil
}

// DeptNameToID returns a department-name → id map for the current tenant, used to
// resolve a member's department column during import. Names are unique enough for
// import purposes; on a duplicate name the last one wins.
func (s *Service) DeptNameToID(ctx context.Context, pool *pgxpool.Pool, t *Tenant) (map[string]kernel.ID, error) {
	flat, err := s.ListDepts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	m := make(map[string]kernel.ID, len(flat))
	for _, d := range flat {
		if d.Status != "active" {
			continue
		}
		m[d.Name] = d.ID
	}
	return m, nil
}

// DeptCodeToID returns active org_code -> id mappings for the current tenant.
func (s *Service) DeptCodeToID(ctx context.Context, pool *pgxpool.Pool, t *Tenant) (map[string]kernel.ID, error) {
	flat, err := s.ListDepts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	m := make(map[string]kernel.ID, len(flat))
	for _, d := range flat {
		if d.Status != "active" {
			continue
		}
		m[strings.ToLower(strings.TrimSpace(d.OrgCode))] = d.ID
	}
	return m, nil
}

// PostCodeToID returns active post code -> id mappings for the current tenant.
func (s *Service) PostCodeToID(ctx context.Context, pool *pgxpool.Pool, t *Tenant) (map[string]kernel.ID, error) {
	posts, err := s.ListPosts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	m := make(map[string]kernel.ID, len(posts))
	for _, p := range posts {
		if p.Status != "active" {
			continue
		}
		m[strings.ToLower(strings.TrimSpace(p.Code))] = p.ID
	}
	return m, nil
}

// SetMemberDept assigns dept_id on a member row in the current tenant schema (used
// by member import to attach a resolved department). A nil deptID clears it.
func (s *Service) SetMemberDept(ctx context.Context, pool *pgxpool.Pool, t *Tenant, memberID kernel.ID, deptID *kernel.ID) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	if deptID != nil {
		var count int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM department
			 WHERE id = $1 AND tenant_id = $2 AND deleted_at IS NULL AND status = 'active'`,
			*deptID, t.ID).Scan(&count); err != nil {
			return err
		}
		if count == 0 {
			return errors.New(errors.KindParam, "tenancy.dept_not_assignable", "部门不存在或已停用")
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE member SET dept_id = $1 WHERE id = $2`, idPtrOrNil(deptID), memberID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func memberPostsText(posts []MemberPost) string {
	codes := make([]string, 0, len(posts))
	for _, p := range posts {
		if p.Code != "" {
			codes = append(codes, p.Code)
		}
	}
	return strings.Join(codes, ",")
}

func memberRolesText(roles []MemberRole) string {
	codes := make([]string, 0, len(roles))
	for _, r := range roles {
		if r.Code != "" {
			codes = append(codes, r.Code)
		}
	}
	return strings.Join(codes, ",")
}

func maskPhone(phone string) string {
	phone = strings.TrimSpace(phone)
	if len(phone) < 7 {
		return phone
	}
	return phone[:3] + "****" + phone[len(phone)-4:]
}

func maskEmail(email string) string {
	email = strings.TrimSpace(email)
	at := strings.Index(email, "@")
	if at <= 1 {
		return email
	}
	prefix := email[:at]
	if len(prefix) <= 2 {
		return prefix[:1] + "*" + email[at:]
	}
	return prefix[:1] + strings.Repeat("*", len(prefix)-2) + prefix[len(prefix)-1:] + email[at:]
}
