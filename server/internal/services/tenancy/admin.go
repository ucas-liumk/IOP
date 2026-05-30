package tenancy

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// MemberRow joins tenant_<slug>.member with tenant_membership for admin listing.
type MemberRow struct {
	MemberID       kernel.ID `json:"member_id"`
	PlatformUserID kernel.ID `json:"platform_user_id"`
	DisplayName    string    `json:"display_name"`
	Email          string    `json:"email"`
	Department     string    `json:"department"`
	Title          string    `json:"title"`
	Phone          string    `json:"phone"`
	Status         string    `json:"status"`
	JoinedAt       string    `json:"joined_at"`
}

// ListMembers returns members of the current tenant (via TenantContext) + their platform_user emails.
func (s *Service) ListMembers(ctx context.Context, pool *pgxpool.Pool, t *Tenant, p kernel.Pagination, search string) ([]MemberRow, error) {
	p = p.Normalize()
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
	where := ""
	if search != "" {
		where = " AND (m.display_name ILIKE $3 OR m.email ILIKE $3 OR m.department ILIKE $3)"
		args = append(args, "%"+search+"%")
	}
	rows, err := tx.Query(ctx,
		`SELECT m.id, m.platform_user_id, m.display_name, m.email,
		        COALESCE(m.department,''), COALESCE(m.title,''), COALESCE(m.phone,''),
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
	for rows.Next() {
		var r MemberRow
		if err := rows.Scan(&r.MemberID, &r.PlatformUserID, &r.DisplayName, &r.Email,
			&r.Department, &r.Title, &r.Phone, &r.Status, &r.JoinedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, tx.Commit(ctx)
}

// UpdateMember updates editable fields on a member row in the tenant schema.
type UpdateMemberCmd struct {
	MemberID    kernel.ID
	DisplayName *string
	Department  *string
	Title       *string
	Phone       *string
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

