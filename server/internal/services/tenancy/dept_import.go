package tenancy

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// deptCSVHeader is the column order for department export/import/template.
var deptCSVHeader = []string{"name", "parent_name", "order_num", "leader", "phone", "email", "status"}

// deptTemplateRows are the two illustrative sample rows in the import template.
var deptTemplateRows = [][]string{
	{"技术部", "", "1", "张三", "13800000001", "tech@example.com", "active"},
	{"后端组", "技术部", "1", "李四", "13800000002", "backend@example.com", "active"},
}

// ExportDepts returns every department of the current tenant as CSV rows (header
// first). parent_name is resolved to the parent's display name (blank for roots).
func (s *Service) ExportDepts(ctx context.Context, pool *pgxpool.Pool, t *Tenant) ([][]string, error) {
	flat, err := s.ListDepts(ctx, pool, t)
	if err != nil {
		return nil, err
	}
	nameByID := make(map[kernel.ID]string, len(flat))
	for _, d := range flat {
		nameByID[d.ID] = d.Name
	}
	out := make([][]string, 0, len(flat)+1)
	out = append(out, append([]string(nil), deptCSVHeader...))
	for _, d := range flat {
		parentName := ""
		if d.ParentID != nil {
			parentName = nameByID[*d.ParentID]
		}
		out = append(out, []string{
			d.Name, parentName, strconv.Itoa(d.OrderNum),
			d.Leader, d.Phone, d.Email, d.Status,
		})
	}
	return out, nil
}

// DeptTemplateRows returns the header + sample rows for the import template.
func DeptTemplateRows() [][]string {
	out := make([][]string, 0, len(deptTemplateRows)+1)
	out = append(out, append([]string(nil), deptCSVHeader...))
	out = append(out, deptTemplateRows...)
	return out
}

// deptImportRow is a parsed CSV department row plus its source line number.
type deptImportRow struct {
	row        int // 1-based data line (header excluded)
	name       string
	parentName string
	orderNum   int
	leader     string
	phone      string
	email      string
	status     string
}

// ImportDepts bulk-creates departments from parsed CSV records (the first record
// is the header). Parents are resolved by name across the existing tree AND the
// rows being imported, using a multi-pass (topological) insert so a child may
// reference a parent defined later in the file. Validation failures (missing name,
// unknown parent, would-be cycle) are recorded per-row; the whole import runs in
// one transaction and never 500s on bad data — partial success is the norm.
func (s *Service) ImportDepts(ctx context.Context, pool *pgxpool.Pool, t *Tenant, records [][]string) (*kernel.BulkResult, error) {
	res := kernel.NewBulkResult()
	if len(records) == 0 {
		return res, nil
	}
	// Skip the header row if it looks like one (first column "name").
	dataRecords := records
	if len(records) > 0 && strings.EqualFold(strings.TrimSpace(firstCol(records[0])), "name") {
		dataRecords = records[1:]
	}

	parsed := make([]deptImportRow, 0, len(dataRecords))
	for i, rec := range dataRecords {
		lineNo := i + 1
		res.Total++
		name := strings.TrimSpace(col(rec, 0))
		if name == "" {
			res.Fail(lineNo, "", "部门名称不能为空")
			continue
		}
		order := 0
		if v := strings.TrimSpace(col(rec, 2)); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				res.Fail(lineNo, name, "order_num 必须是整数")
				continue
			}
			order = n
		}
		status := strings.TrimSpace(col(rec, 6))
		if status == "" {
			status = "active"
		}
		if status != "active" && status != "disabled" {
			res.Fail(lineNo, name, "status 只能是 active 或 disabled")
			continue
		}
		parsed = append(parsed, deptImportRow{
			row: lineNo, name: name,
			parentName: strings.TrimSpace(col(rec, 1)),
			orderNum:   order,
			leader:     strings.TrimSpace(col(rec, 3)),
			phone:      strings.TrimSpace(col(rec, 4)),
			email:      strings.TrimSpace(col(rec, 5)),
			status:     status,
		})
	}
	if len(parsed) == 0 {
		return res, nil
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
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return nil, err
	}
	rootID, err := rootDeptID(ctx, tx, t)
	if err != nil {
		return nil, err
	}

	// Seed the known-name → id map from existing departments so children can attach
	// to an already-persisted parent.
	rows, err := tx.Query(ctx,
		`SELECT id, name FROM department
		 WHERE tenant_id = $1 AND deleted_at IS NULL`, t.ID)
	if err != nil {
		return nil, err
	}
	nameToID := map[string]kernel.ID{}
	for rows.Next() {
		var id kernel.ID
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			rows.Close()
			return nil, err
		}
		nameToID[name] = id
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Multi-pass insert: a row whose parent_name is blank or already resolvable is
	// inserted; rows whose parent is not yet known are deferred to a later pass.
	// We stop when a full pass makes no progress — any rows still pending then have
	// an unresolvable (missing or cyclic) parent.
	pending := parsed
	for len(pending) > 0 {
		progressed := false
		var next []deptImportRow
		for _, r := range pending {
			var parentID *kernel.ID
			if r.parentName != "" {
				if r.parentName == r.name {
					res.Fail(r.row, r.name, "部门不能以自身为父部门")
					progressed = true
					continue
				}
				pid, ok := nameToID[r.parentName]
				if !ok {
					next = append(next, r) // parent maybe defined later
					continue
				}
				parentID = &pid
			} else {
				parentID = &rootID
			}
			id := kernel.NewID()
			if _, err := tx.Exec(ctx,
				`INSERT INTO department (id, tenant_id, name, parent_id, order_num, leader, phone, email, status, is_root)
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,false)`,
				id, t.ID, r.name, idPtrOrNil(parentID), r.orderNum,
				nullStr(r.leader), nullStr(r.phone), nullStr(r.email), r.status); err != nil {
				res.Fail(r.row, r.name, "写入失败: "+err.Error())
				progressed = true
				continue
			}
			nameToID[r.name] = id
			res.Ok()
			progressed = true
		}
		pending = next
		if !progressed {
			break
		}
	}
	// Anything still pending could not resolve its parent (missing or cyclic).
	for _, r := range pending {
		res.Fail(r.row, r.name, fmt.Sprintf("父部门 %q 不存在或存在循环引用", r.parentName))
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.import_depts_failed", "导入部门失败", err)
	}
	return res, nil
}

// col returns the i-th field of rec, or "" if out of range.
func col(rec []string, i int) string {
	if i < len(rec) {
		return rec[i]
	}
	return ""
}

func firstCol(rec []string) string { return col(rec, 0) }
