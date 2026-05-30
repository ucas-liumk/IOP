package iam

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"

	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// DemoSeedForced reports whether IOP_SEED_DEMO explicitly enables the demo seed.
// (In dev the seed runs regardless; this lets a non-dev env opt in.) Recognized
// truthy values: 1, true, yes, on.
func DemoSeedForced() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv("IOP_SEED_DEMO"))) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// demoSeedNamespace makes every generated UUID (departments, etc.) stable across
// re-runs, so the seed is idempotent: the same logical entity always maps to the
// same id and INSERT ... ON CONFLICT (id) DO NOTHING is a no-op on later boots.
var demoSeedNamespace = uuid.MustParse("a1b2c3d4-0000-4000-8000-0000000b1900")

const (
	demoTenantSlug  = "bjgov"
	demoTenantName  = "北京市人民政府"
	demoAdminUser   = "bjadmin"
	demoUserPwd     = "Bjgov12345!" // dev demo password for every seeded公务员 (incl. bjadmin)
	demoUserPrefix  = "bjgov"
)

// demoDept is a node in the seed department tree. Children are realized recursively;
// posts (处室) hang under their委办局 parent.
type demoDept struct {
	name     string
	children []demoDept
}

// demoOrgTree is a realistic subset of the 北京市人民政府 org structure: root →
// 委办局 → (for a few) 处室.
var demoOrgTree = demoDept{
	name: demoTenantName,
	children: []demoDept{
		{name: "市政府办公厅"},
		{name: "市发展和改革委员会", children: []demoDept{
			{name: "办公室"}, {name: "综合处"}, {name: "规划处"}, {name: "财金处"},
		}},
		{name: "市教育委员会"},
		{name: "市科学技术委员会"},
		{name: "市公安局", children: []demoDept{
			{name: "办公室"}, {name: "治安总队"}, {name: "交通管理局"}, {name: "出入境管理总队"},
		}},
		{name: "市民政局"},
		{name: "市财政局", children: []demoDept{
			{name: "预算处"}, {name: "国库处"}, {name: "税政处"},
		}},
		{name: "市人力资源和社会保障局"},
		{name: "市规划和自然资源委员会"},
		{name: "市住房和城乡建设委员会"},
		{name: "市城市管理委员会"},
		{name: "市交通委员会"},
		{name: "市水务局"},
		{name: "市农业农村局"},
		{name: "市商务局"},
		{name: "市文化和旅游局"},
		{name: "市卫生健康委员会"},
		{name: "市市场监督管理局"},
		{name: "市生态环境局"},
	},
}

// demoFirstNames / demoSurnames build plausible Chinese 公务员 names.
var demoSurnames = []string{"王", "李", "张", "刘", "陈", "杨", "赵", "黄", "周", "吴", "徐", "孙", "马", "朱", "胡", "郭", "何", "高", "林", "罗"}
var demoGiven = []string{"伟", "芳", "娜", "敏", "静", "强", "磊", "军", "洋", "勇", "艳", "杰", "娟", "涛", "明", "超", "霞", "平", "刚", "桂英"}
var demoTitles = []string{"主任", "副主任", "处长", "副处长", "科长", "主任科员", "副主任科员", "科员", "调研员"}

// SeedDemoOrg ensures the 北京市人民政府 (slug "bjgov") demo organization exists and is
// populated with a realistic department tree + ~40 sample 公务员. It is:
//
//   - GUARDED: the caller passes enabled=false in production so it never pollutes a
//     real deployment (app.Build sets enabled = !cfg.IsProd() || IOP_SEED_DEMO set).
//   - IDEMPOTENT: if the tenant already exists AND already has members, it does
//     nothing; otherwise it self-heals (creates whatever is missing). Departments use
//     deterministic UUIDs + ON CONFLICT, members are keyed by stable usernames.
//
// It logs what it seeded (dept + member counts).
func SeedDemoOrg(ctx context.Context, s *Service, tenants *tenancy.Service, pool *pgxpool.Pool, enabled bool, logger *zap.Logger) error {
	if !enabled {
		return nil
	}

	// 1. Ensure tenant exists + provisioned.
	t, err := tenants.GetTenantBySlug(ctx, demoTenantSlug)
	if err != nil {
		return fmt.Errorf("demo seed: lookup tenant: %w", err)
	}
	created := false
	if t == nil {
		t, err = tenants.CreateTenant(ctx, tenancy.CreateTenantCmd{Slug: demoTenantSlug, Name: demoTenantName})
		if err != nil {
			return fmt.Errorf("demo seed: create tenant: %w", err)
		}
		created = true
	}

	// Idempotency short-circuit: if not freshly created and members already exist, skip.
	if !created {
		n, err := countTenantRows(ctx, pool, t.SchemaName, "member")
		if err != nil {
			return fmt.Errorf("demo seed: count members: %w", err)
		}
		if n > 0 {
			logger.Info("demo org already seeded, skipping",
				zap.String("slug", demoTenantSlug), zap.Int("members", n))
			return nil
		}
	}

	// 2. Seed department tree (deterministic UUIDs, ON CONFLICT DO NOTHING).
	deptCount, rootID, err := seedDemoDepts(ctx, pool, t)
	if err != nil {
		return fmt.Errorf("demo seed: depts: %w", err)
	}

	// Departments eligible to receive members: every node except the root org node.
	// Iterate the tree (deterministic order) rather than the map so assignment is stable.
	leafCandidates := collectAssignableDepts()

	// 3. Seed members. bjadmin → tenant_admin; the rest → tenant_member.
	memberCount := 0
	const total = 42
	for i := 1; i <= total; i++ {
		var username, realName, role string
		if i == 1 {
			username = demoAdminUser
			realName = "管理员"
			role = "tenant_admin"
		} else {
			username = fmt.Sprintf("%s%04d", demoUserPrefix, i)
			realName = demoName(i)
			role = "tenant_member"
		}
		phone := demoPhone(i)
		// Pick a department deterministically by index (admin → root org).
		var assignedDept kernel.ID
		if i == 1 {
			assignedDept = rootID
		} else if len(leafCandidates) > 0 {
			assignedDept = leafCandidates[i%len(leafCandidates)]
		}
		title := demoTitles[i%len(demoTitles)]

		ok, err := seedDemoMember(ctx, s, tenants, pool, t, seedMemberArgs{
			Username: username, RealName: realName, Phone: phone,
			Role: role, DeptID: assignedDept, Title: title,
		})
		if err != nil {
			return fmt.Errorf("demo seed: member %s: %w", username, err)
		}
		if ok {
			memberCount++
		}
	}

	logger.Info("seeded demo org",
		zap.String("slug", demoTenantSlug),
		zap.String("tenant_id", string(t.ID)),
		zap.Bool("tenant_created", created),
		zap.Int("departments", deptCount),
		zap.Int("members", memberCount),
		zap.String("admin_user", demoAdminUser),
	)
	return nil
}

// deptID derives a stable UUID for a department from its FULL path (e.g.
// "/北京市人民政府/市财政局/预算处"). Using the full path means 处室 with duplicate short
// names under different 委办局 (e.g. multiple "办公室") get distinct, stable ids.
func deptID(path string) kernel.ID {
	return kernel.ID(uuid.NewSHA1(demoSeedNamespace, []byte("dept:"+path)).String())
}

// collectAssignableDepts walks demoOrgTree in the same full-path order as
// seedDemoDepts and returns the ids of every node EXCEPT the root org, i.e. those a
// member can be assigned to.
func collectAssignableDepts() []kernel.ID {
	var out []kernel.ID
	var walk func(node demoDept, path string, isRoot bool)
	walk = func(node demoDept, path string, isRoot bool) {
		full := path + "/" + node.name
		if !isRoot {
			out = append(out, deptID(full))
		}
		for _, ch := range node.children {
			walk(ch, full, false)
		}
	}
	walk(demoOrgTree, "", true)
	return out
}

// seedDemoDepts inserts the department tree into the tenant schema and returns the
// number of departments newly inserted. Uses deterministic ids keyed by full path
// (so 处室 with duplicate short names under different 委办局 don't collide) and
// ON CONFLICT (id) DO NOTHING so re-runs are no-ops.
func seedDemoDepts(ctx context.Context, pool *pgxpool.Pool, t *tenancy.Tenant) (int, kernel.ID, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, "", err
	}
	defer conn.Release()
	tx, err := conn.Begin(ctx)
	if err != nil {
		return 0, "", err
	}
	defer tx.Rollback(ctx) //nolint:errcheck
	if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", t.SchemaName)); err != nil {
		return 0, "", err
	}
	var rootID kernel.ID
	if err := tx.QueryRow(ctx,
		`SELECT id
		 FROM department
		 WHERE tenant_id = $1 AND is_root = TRUE AND deleted_at IS NULL
		 ORDER BY created_at ASC
		 LIMIT 1`, t.ID).Scan(&rootID); err != nil {
		return 0, "", err
	}

	count := 0
	var insert func(node demoDept, parent *kernel.ID, path string, order int) error
	insert = func(node demoDept, parent *kernel.ID, path string, order int) error {
		fullPath := path + "/" + node.name
		id := deptID(fullPath)
		var parentArg any
		if parent != nil {
			parentArg = string(*parent)
		}
		code := "ORG-" + strings.ToUpper(strings.ReplaceAll(string(id), "-", "")[:12])
		ct, err := tx.Exec(ctx,
			`INSERT INTO department (id, tenant_id, name, org_code, parent_id, org_type, order_num, status, is_root)
			 VALUES ($1, $2, $3, $4, $5, 'department', $6, 'active', FALSE)
			 ON CONFLICT (id) DO NOTHING`,
			string(id), t.ID, node.name, code, parentArg, order)
		if err != nil {
			return err
		}
		if ct.RowsAffected() > 0 {
			count++
		}
		for i, ch := range node.children {
			if err := insert(ch, &id, fullPath, i+1); err != nil {
				return err
			}
		}
		return nil
	}
	for i, ch := range demoOrgTree.children {
		if err := insert(ch, &rootID, "/"+demoOrgTree.name, i+1); err != nil {
			return 0, "", err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, "", err
	}
	return count, rootID, nil
}

type seedMemberArgs struct {
	Username string
	RealName string
	Phone    string
	Role     string
	DeptID   kernel.ID
	Title    string
}

// seedDemoMember ensures one platform_user (username+phone, NO email) joined to the
// demo tenant with a member row (dept_id + title) and the given role. Returns whether
// a NEW member row was created. Idempotent: re-running matches the existing user by
// username and the existing membership, then just re-applies dept/role.
func seedDemoMember(ctx context.Context, s *Service, tenants *tenancy.Service, pool *pgxpool.Pool, t *tenancy.Tenant, a seedMemberArgs) (bool, error) {
	u, err := s.repo.GetUserByUsername(ctx, a.Username)
	if err != nil {
		return false, err
	}
	if u == nil {
		u, err = s.RegisterUser(ctx, RegisterCmd{
			Username: a.Username, Phone: a.Phone, Password: demoUserPwd,
		})
		if err != nil {
			return false, err
		}
	}
	mem, err := tenants.JoinMember(ctx, pool, tenancy.JoinMemberCmd{
		PlatformUserID: u.ID, TenantID: t.ID,
		DisplayName: a.RealName, Phone: a.Phone, Title: a.Title,
	})
	if err != nil {
		return false, err
	}
	// Assign department + title on the member projection (JoinMember does not set dept_id).
	if a.DeptID != "" {
		if err := tenants.UpdateMember(ctx, pool, t, tenancy.UpdateMemberCmd{
			MemberID: mem.MemberID, DeptID: &a.DeptID, SetDept: true,
		}); err != nil {
			return false, err
		}
	}
	if err := s.GrantRoleByCode(ctx, mem.MemberID, t.ID, a.Role); err != nil {
		return false, err
	}
	return true, nil
}

// countTenantRows returns the row count of a table in the given tenant schema.
func countTenantRows(ctx context.Context, pool *pgxpool.Pool, schema, table string) (int, error) {
	conn, err := pool.Acquire(ctx)
	if err != nil {
		return 0, err
	}
	defer conn.Release()
	if _, err := conn.Exec(ctx, fmt.Sprintf("SET search_path TO %q, public", schema)); err != nil {
		return 0, err
	}
	defer conn.Exec(ctx, "RESET search_path") //nolint:errcheck
	var n int
	// table is a fixed internal constant ("member"), never user input — safe to interpolate.
	if err := conn.QueryRow(ctx, fmt.Sprintf("SELECT count(*) FROM %s", table)).Scan(&n); err != nil {
		return 0, err
	}
	return n, nil
}

// demoName builds a deterministic Chinese display name from an index.
func demoName(i int) string {
	s := demoSurnames[i%len(demoSurnames)]
	g := demoGiven[(i*7)%len(demoGiven)]
	return s + g
}

// demoPhone builds a deterministic, valid-shaped CN mobile (1[3-9]xxxxxxxxx, 11 digits).
func demoPhone(i int) string {
	// 138 prefix + 8 digits derived from i, zero-padded → 11 digits total.
	return fmt.Sprintf("138%08d", 10000000+i)
}
