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

var deptImportHeader = []string{
	"org_name", "org_code", "parent_org_code", "org_type",
	"leader_account", "phone", "order_num", "status", "remark",
}

var deptExportHeader = []string{
	"org_name", "org_code", "parent_org_code", "org_type",
	"leader", "leader_account", "phone", "order_num", "status", "remark", "path",
}

var deptTemplateRows = [][]string{
	{"技术部", "TECH", "ROOT", "department", "zhangsan", "13800000001", "1", "active", "一级部门"},
	{"后端组", "TECH-BE", "TECH", "team", "lisi", "13800000002", "1", "active", "二级小组"},
}

// ExportDepts returns organizations for the current tenant as spreadsheet rows.
// The tenant is server-resolved; callers cannot widen scope by passing tenant_id.
func (s *Service) ExportDepts(ctx context.Context, pool *pgxpool.Pool, t *Tenant, filter DeptListFilter) ([][]string, error) {
	flat, err := s.ListDeptsFiltered(ctx, pool, t, filter)
	if err != nil {
		return nil, err
	}
	codeByID := make(map[kernel.ID]string, len(flat))
	for _, d := range flat {
		codeByID[d.ID] = d.OrgCode
	}
	out := make([][]string, 0, len(flat)+1)
	out = append(out, append([]string(nil), deptExportHeader...))
	for _, d := range flat {
		parentCode := ""
		if d.ParentID != nil {
			parentCode = codeByID[*d.ParentID]
		}
		out = append(out, []string{
			d.Name, d.OrgCode, parentCode, d.OrgType,
			d.Leader, d.LeaderAccount, d.Phone, strconv.Itoa(d.OrderNum),
			d.Status, d.Remark, d.Path,
		})
	}
	return out, nil
}

// DeptTemplateRows returns the header + sample rows for the import template.
func DeptTemplateRows() [][]string {
	out := make([][]string, 0, len(deptTemplateRows)+1)
	out = append(out, append([]string(nil), deptImportHeader...))
	out = append(out, deptTemplateRows...)
	return out
}

type deptImportRow struct {
	row           int
	name          string
	orgCode       string
	parentOrgCode string
	orgType       string
	leaderAccount string
	phone         string
	orderNum      int
	status        string
	remark        string
}

// ImportDepts validates then bulk-creates organizations. It is all-or-nothing:
// any validation or persistence error prevents partial tree writes. dryRun runs
// every validation without inserting.
func (s *Service) ImportDepts(ctx context.Context, pool *pgxpool.Pool, t *Tenant, records [][]string, dryRun bool) (*kernel.BulkResult, error) {
	res := kernel.NewBulkResult()
	parsed := parseDeptImportRows(records, res)
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
	if _, err := rootDeptID(ctx, tx, t); err != nil {
		return nil, err
	}

	codeToID := map[string]kernel.ID{}
	rows, err := tx.Query(ctx,
		`SELECT id, org_code
		 FROM department
		 WHERE tenant_id = $1 AND deleted_at IS NULL`, t.ID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id kernel.ID
		var code string
		if err := rows.Scan(&id, &code); err != nil {
			rows.Close()
			return nil, err
		}
		codeToID[lowerCode(code)] = id
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	importByCode := map[string]deptImportRow{}
	for _, r := range parsed {
		key := lowerCode(r.orgCode)
		if _, ok := importByCode[key]; ok {
			res.Fail(r.row, r.orgCode, "文件内组织编码重复")
			continue
		}
		importByCode[key] = r
	}
	for _, r := range parsed {
		key := lowerCode(r.orgCode)
		if _, exists := codeToID[key]; exists {
			res.Fail(r.row, r.orgCode, "同一租户内组织编码已存在")
		}
		if strings.TrimSpace(r.parentOrgCode) == "" {
			res.Fail(r.row, r.orgCode, "根组织已由租户创建，不能导入第二个根组织；请填写父级组织编码")
			continue
		}
		parentKey := lowerCode(r.parentOrgCode)
		if parentKey == key {
			res.Fail(r.row, r.orgCode, "组织不能以自身为父级")
			continue
		}
		if _, ok := codeToID[parentKey]; !ok {
			if _, ok := importByCode[parentKey]; !ok {
				res.Fail(r.row, r.orgCode, fmt.Sprintf("父级组织编码 %q 不存在", r.parentOrgCode))
			}
		}
	}
	failImportCycles(parsed, importByCode, res)
	if res.Failed > 0 {
		if dryRun {
			res.Succeeded = res.Total - res.Failed
		}
		return res, nil
	}
	if dryRun {
		res.Succeeded = res.Total
		return res, nil
	}

	pending := append([]deptImportRow(nil), parsed...)
	for len(pending) > 0 {
		progressed := false
		next := []deptImportRow{}
		for _, r := range pending {
			parentID, ok := codeToID[lowerCode(r.parentOrgCode)]
			if !ok {
				next = append(next, r)
				continue
			}
			id := kernel.NewID()
			if _, err := tx.Exec(ctx,
				`INSERT INTO department (
				    id, tenant_id, name, org_code, parent_id, org_type, order_num,
				    leader_account, phone, status, remark, is_root
				 )
				 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,false)`,
				id, t.ID, r.name, r.orgCode, parentID, r.orgType, r.orderNum,
				nullStr(r.leaderAccount), nullStr(r.phone), r.status, nullStr(r.remark)); err != nil {
				res.Failed = 0
				res.Errors = nil
				res.Fail(r.row, r.orgCode, "写入失败: "+err.Error())
				res.Succeeded = 0
				return res, nil
			}
			codeToID[lowerCode(r.orgCode)] = id
			res.Ok()
			progressed = true
		}
		pending = next
		if !progressed {
			for _, r := range pending {
				res.Fail(r.row, r.orgCode, fmt.Sprintf("父级组织编码 %q 不存在或存在循环引用", r.parentOrgCode))
			}
			res.Succeeded = 0
			return res, nil
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, errors.Wrap(errors.KindDatabase, "tenancy.import_depts_failed", "导入组织失败", err)
	}
	_ = s.bus.Publish(ctx, "tenancy.dept_imported", map[string]any{
		"tenant_id": t.ID, "succeeded": res.Succeeded,
	})
	return res, nil
}

func parseDeptImportRows(records [][]string, res *kernel.BulkResult) []deptImportRow {
	if len(records) == 0 {
		return nil
	}
	start := 0
	if isDeptHeader(records[0]) {
		start = 1
	}
	parsed := make([]deptImportRow, 0, len(records)-start)
	for i := start; i < len(records); i++ {
		rec := records[i]
		lineNo := i + 1
		if emptyRecord(rec) {
			continue
		}
		res.Total++
		name := strings.TrimSpace(col(rec, 0))
		code := strings.TrimSpace(col(rec, 1))
		if name == "" {
			res.Fail(lineNo, code, "组织名称不能为空")
			continue
		}
		if code == "" {
			res.Fail(lineNo, name, "组织编码不能为空")
			continue
		}
		order := 0
		if v := strings.TrimSpace(col(rec, 6)); v != "" {
			n, err := strconv.Atoi(v)
			if err != nil {
				res.Fail(lineNo, code, "order_num 必须是整数")
				continue
			}
			order = n
		}
		status := normalizeDeptStatus(col(rec, 7))
		if err := validateDeptStatus(status); err != nil {
			res.Fail(lineNo, code, "status 只能是 active 或 disabled")
			continue
		}
		parsed = append(parsed, deptImportRow{
			row: lineNo, name: name, orgCode: code,
			parentOrgCode: strings.TrimSpace(col(rec, 2)),
			orgType:       normalizeOrgType(col(rec, 3)),
			leaderAccount: strings.TrimSpace(col(rec, 4)),
			phone:         strings.TrimSpace(col(rec, 5)),
			orderNum:      order,
			status:        status,
			remark:        strings.TrimSpace(col(rec, 8)),
		})
	}
	return parsed
}

func failImportCycles(rows []deptImportRow, byCode map[string]deptImportRow, res *kernel.BulkResult) {
	for _, r := range rows {
		seen := map[string]bool{}
		cur := lowerCode(r.orgCode)
		parent := lowerCode(r.parentOrgCode)
		for parent != "" {
			if parent == cur || seen[parent] {
				res.Fail(r.row, r.orgCode, "导入数据存在循环父子关系")
				break
			}
			seen[parent] = true
			pr, ok := byCode[parent]
			if !ok {
				break
			}
			parent = lowerCode(pr.parentOrgCode)
		}
	}
}

func isDeptHeader(rec []string) bool {
	c := strings.ToLower(strings.TrimSpace(firstCol(rec)))
	return c == "org_name" || c == "name" || c == "组织名称"
}

func lowerCode(v string) string { return strings.ToLower(strings.TrimSpace(v)) }

func emptyRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

func col(rec []string, i int) string {
	if i < len(rec) {
		return rec[i]
	}
	return ""
}

func firstCol(rec []string) string { return col(rec, 0) }
