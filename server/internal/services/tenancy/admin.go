package tenancy

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// MemberPost is a post assigned to a member (id + code + name).
type MemberPost struct {
	PostID kernel.ID `json:"post_id"`
	Code   string    `json:"code"`
	Name   string    `json:"name"`
}

// MemberRow joins tenant_<slug>.member with tenant_membership for admin listing.
type MemberRow struct {
	MemberID       kernel.ID    `json:"member_id"`
	PlatformUserID kernel.ID    `json:"platform_user_id"`
	DisplayName    string       `json:"display_name"`
	Email          string       `json:"email"`
	Department     string       `json:"department"`
	DeptID         *kernel.ID   `json:"dept_id,omitempty"`
	Title          string       `json:"title"`
	Phone          string       `json:"phone"`
	Status         string       `json:"status"`
	JoinedAt       string       `json:"joined_at"`
	Posts          []MemberPost `json:"posts"`
}

// ListMembersCmd parameters the member listing: pagination + free-text search +
// optional department filter (with optional subtree expansion).
type ListMembersCmd struct {
	Page    kernel.Pagination
	Search  string
	DeptID  *kernel.ID // nil = no dept filter
	Subtree bool       // when DeptID set, also include descendant departments
}

// ListMembers returns members of the current tenant (via TenantContext) + their
// posts. Supports free-text search and a department filter (optionally subtree).
func (s *Service) ListMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd ListMembersCmd) ([]MemberRow, error) {
	p := cmd.Page.Normalize()

	// Resolve dept filter ids (subtree expansion is computed from the in-Go tree).
	var deptFilter []kernel.ID
	if cmd.DeptID != nil {
		deptFilter = []kernel.ID{*cmd.DeptID}
		if cmd.Subtree {
			desc, err := s.DescendantDeptIDs(ctx, pool, t, *cmd.DeptID)
			if err != nil {
				return nil, err
			}
			deptFilter = append(deptFilter, desc...)
		}
	}

	conn, err := pool.Acquire(ctx)
	if err != nil {
		return nil, err
	}
	defer conn.Release()

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}

	args := []any{p.PageSize, p.Offset()}
	idx := 3
	where := ""
	if cmd.Search != "" {
		where += fmt.Sprintf(" AND (m.display_name ILIKE $%d OR m.email ILIKE $%d OR m.department ILIKE $%d)", idx, idx, idx)
		args = append(args, "%"+cmd.Search+"%")
		idx++
	}
	if cmd.DeptID != nil {
		where += fmt.Sprintf(" AND m.dept_id = ANY($%d)", idx)
		args = append(args, deptFilter)
		idx++
	}
	rows, err := tx.Query(ctx,
		`SELECT m.id, m.platform_user_id, m.display_name, m.email,
		        COALESCE(m.department,''), m.dept_id, COALESCE(m.title,''), COALESCE(m.phone,''),
		        m.status, to_char(m.joined_at, 'YYYY-MM-DD HH24:MI:SS')
		 FROM member m
		 WHERE 1=1`+where+`
		 ORDER BY m.joined_at DESC
		 LIMIT $1 OFFSET $2`, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []MemberRow{}
	idsList := []kernel.ID{}
	for rows.Next() {
		var r MemberRow
		if err := rows.Scan(&r.MemberID, &r.PlatformUserID, &r.DisplayName, &r.Email,
			&r.Department, &r.DeptID, &r.Title, &r.Phone, &r.Status, &r.JoinedAt); err != nil {
			return nil, err
		}
		r.Posts = []MemberPost{}
		out = append(out, r)
		idsList = append(idsList, r.MemberID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Attach posts per member in one query.
	if len(idsList) > 0 {
		prows, err := tx.Query(ctx,
			`SELECT mp.member_id, p.id, p.code, p.name
			 FROM member_post mp JOIN post p ON p.id = mp.post_id
			 WHERE mp.member_id = ANY($1)
			 ORDER BY p.order_num, p.code`, idsList)
		if err != nil {
			return nil, err
		}
		defer prows.Close()
		byMember := map[kernel.ID][]MemberPost{}
		for prows.Next() {
			var mid kernel.ID
			var mp MemberPost
			if err := prows.Scan(&mid, &mp.PostID, &mp.Code, &mp.Name); err != nil {
				return nil, err
			}
			byMember[mid] = append(byMember[mid], mp)
		}
		if err := prows.Err(); err != nil {
			return nil, err
		}
		for i := range out {
			if ps := byMember[out[i].MemberID]; ps != nil {
				out[i].Posts = ps
			}
		}
	}
	return out, tx.Commit(ctx)
}

// UpdateMember updates editable fields on a member row in the tenant schema.
// DeptID is a tri-state: nil = leave unchanged; *DeptID = set/clear (set ClearDept
// to clear to NULL, otherwise the pointed-to value is validated + applied).
type UpdateMemberCmd struct {
	MemberID    kernel.ID
	DisplayName *string
	Department  *string
	Title       *string
	Phone       *string
	DeptID      *kernel.ID // when SetDept is true: nil clears, non-nil assigns
	SetDept     bool       // whether dept_id was present in the request
}

func (s *Service) UpdateMember(ctx context.Context, pool *pgxpool.Pool, t *Tenant, cmd UpdateMemberCmd) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}

	// Build dynamic update — only set fields that were provided
	sets := []string{}
	args := []any{cmd.MemberID}
	idx := 2
	if cmd.DisplayName != nil {
		sets = append(sets, fmt.Sprintf("display_name = $%d", idx))
		args = append(args, *cmd.DisplayName)
		idx++
	}
	if cmd.Department != nil {
		sets = append(sets, fmt.Sprintf("department = $%d", idx))
		args = append(args, *cmd.Department)
		idx++
	}
	if cmd.Title != nil {
		sets = append(sets, fmt.Sprintf("title = $%d", idx))
		args = append(args, *cmd.Title)
		idx++
	}
	if cmd.Phone != nil {
		sets = append(sets, fmt.Sprintf("phone = $%d", idx))
		args = append(args, *cmd.Phone)
		idx++
	}
	if cmd.SetDept {
		if cmd.DeptID != nil {
			// Validate the target department exists in this tenant schema.
			var count int
			if err := tx.QueryRow(ctx, `SELECT count(*) FROM department WHERE id = $1`, *cmd.DeptID).Scan(&count); err != nil {
				return err
			}
			if count == 0 {
				return errors.New(errors.KindParam, "tenancy.dept_not_found", "部门不存在")
			}
		}
		sets = append(sets, fmt.Sprintf("dept_id = $%d", idx))
		args = append(args, idPtrOrNil(cmd.DeptID))
		idx++
	}
	if len(sets) == 0 {
		return nil
	}
	sql := "UPDATE member SET "
	for i, ss := range sets {
		if i > 0 {
			sql += ", "
		}
		sql += ss
	}
	sql += " WHERE id = $1"
	if _, err := tx.Exec(ctx, sql, args...); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.update_member_failed", "更新成员失败", err)
	}
	return tx.Commit(ctx)
}

// AssignMemberPost adds a (member, post) mapping. Validates both rows exist; the
// mapping is idempotent (ON CONFLICT DO NOTHING).
func (s *Service) AssignMemberPost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, memberID, postID kernel.ID) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	var mc, pc int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM member WHERE id = $1`, memberID).Scan(&mc); err != nil {
		return err
	}
	if mc == 0 {
		return errors.New(errors.KindNotFound, "tenancy.member_not_found", "成员不存在")
	}
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM post WHERE id = $1`, postID).Scan(&pc); err != nil {
		return err
	}
	if pc == 0 {
		return errors.New(errors.KindNotFound, "tenancy.post_not_found", "岗位不存在")
	}
	if _, err := tx.Exec(ctx,
		`INSERT INTO member_post (member_id, post_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		memberID, postID); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.assign_post_failed", "分配岗位失败", err)
	}
	return tx.Commit(ctx)
}

// RemoveMemberPost removes a (member, post) mapping.
func (s *Service) RemoveMemberPost(ctx context.Context, pool *pgxpool.Pool, t *Tenant, memberID, postID kernel.ID) error {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM member_post WHERE member_id = $1 AND post_id = $2`, memberID, postID); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.remove_post_failed", "移除岗位失败", err)
	}
	return tx.Commit(ctx)
}

// SetMemberStatus toggles a member between active / disabled.
func (s *Service) SetMemberStatus(ctx context.Context, pool *pgxpool.Pool, t *Tenant, id kernel.ID, status string) error {
	if status != "active" && status != "disabled" {
		return errors.New(errors.KindParam, "tenancy.invalid_status", "状态非法")
	}
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE member SET status = $1 WHERE id = $2`, status, id); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// UpdateName updates tenant display name (admin only).
func (s *Service) UpdateName(ctx context.Context, id kernel.ID, name string) error {
	if name == "" {
		return errors.New(errors.KindParam, "tenancy.empty_name", "名称不能为空")
	}
	_, err := s.tenantRepo.(*pgTenantRepo).pool.Exec(ctx,
		`UPDATE public.tenant SET name = $1 WHERE id = $2`, name, id)
	return err
}

// CountMembers returns active member count for current tenant.
func (s *Service) CountMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant) (int, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %q, public", t.SchemaName)); err != nil {
		return 0, err
	}
	defer conn.Exec(ctx, "RESET search_path") //nolint:errcheck
	var n int
	if err := conn.QueryRow(ctx, "SELECT count(*) FROM member WHERE status = 'active'").Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

var _ pgx.Tx
