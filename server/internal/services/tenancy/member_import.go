package tenancy

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// memberCSVHeader is the column order for member export/import/template.
var memberCSVHeader = []string{"username", "display_name", "phone", "email", "department", "title", "status"}

// memberTemplateRows are illustrative sample rows for the member import template.
var memberTemplateRows = [][]string{
	{"zhangsan", "张三", "13800000001", "zhangsan@example.com", "技术部", "工程师", "active"},
	{"lisi", "李四", "13800000002", "", "后端组", "组长", "active"},
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

// ExportMembers returns every member of the current tenant as CSV rows (header
// first), matching memberCSVHeader. department is the member's stored department
// name (the per-member display string), preferring the resolved dept tree name
// when a dept_id is set.
func (s *Service) ExportMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([][]string, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx,
		`SELECT COALESCE(u.username,''), m.display_name, COALESCE(m.phone,''), COALESCE(m.email,''),
		        COALESCE(d.name, m.department, ''), COALESCE(m.title,''), m.status
		 FROM member m
		 LEFT JOIN public.platform_user u ON u.id = m.platform_user_id
		 LEFT JOIN department d ON d.id = m.dept_id
		 ORDER BY m.joined_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := [][]string{MemberCSVHeader()}
	for rows.Next() {
		var username, displayName, phone, email, dept, title, status string
		if err := rows.Scan(&username, &displayName, &phone, &email, &dept, &title, &status); err != nil {
			return nil, err
		}
		out = append(out, []string{username, displayName, phone, email, dept, title, status})
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
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
		m[d.Name] = d.ID
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
	if _, err := tx.Exec(ctx, `UPDATE member SET dept_id = $1 WHERE id = $2`, idPtrOrNil(deptID), memberID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
