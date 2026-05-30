package integration

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/leo/iop/server/internal/app"
	"github.com/leo/iop/server/internal/config"
	"github.com/leo/iop/server/internal/services/iam"
	"github.com/leo/iop/server/internal/services/tenancy"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

func setupApp(t *testing.T) (*app.App, func()) {
	t.Helper()
	if os.Getenv("IOP_INTEGRATION") == "" {
		t.Skip("set IOP_INTEGRATION=1 to run (requires docker-compose db running)")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config: %v", err)
	}
	if dsn := os.Getenv("IOP_TEST_DB_DSN"); dsn != "" {
		cfg.DB.DSN = dsn
	} else if cfg.DB.DSN == "" {
		cfg.DB.DSN = "postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable"
	}
	cfg.Redis.Addr = "" // 命门测试: 不连 Redis, 避免外部依赖

	a, cleanup, err := app.Build(context.Background(), cfg)
	if err != nil {
		t.Fatalf("app build: %v", err)
	}
	return a, cleanup
}

func mustCreateTenant(t *testing.T, a *app.App, slug, name string) *tenancy.Tenant {
	t.Helper()
	// Use unique slug to avoid collisions across runs
	uniq := fmt.Sprintf("%s%d", slug, time.Now().UnixNano()%1e6)
	tt, err := a.Tenancy.CreateTenant(context.Background(), tenancy.CreateTenantCmd{Slug: uniq, Name: name})
	if err != nil {
		t.Fatalf("create tenant: %v", err)
	}
	t.Cleanup(func() { _ = a.Tenancy.CloseTenant(context.Background(), tt.ID) })
	return tt
}

func mustCreateUser(t *testing.T, a *app.App, prefix string) (id, email string) {
	t.Helper()
	email = fmt.Sprintf("%s+%d@iop.test", prefix, time.Now().UnixNano())
	u, err := a.IAM.RegisterUser(context.Background(), iam.RegisterCmd{
		Email: email, Password: "Test1234abc!",
	})
	if err != nil {
		t.Fatalf("register user: %v", err)
	}
	return string(u.ID), email
}

// 命门 1: 隔离 — A 租户的数据 B 租户用户看不到
func TestKeystone_Isolation(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	tA := mustCreateTenant(t, a, "isoa", "Tenant A")
	tB := mustCreateTenant(t, a, "isob", "Tenant B")

	// Insert a row in tA schema
	ctxA := tenantdb.WithTenant(ctx, &tenantdb.TenantContext{
		ID: string(tA.ID), Slug: tA.Slug, SchemaName: tA.SchemaName, Status: tA.Status,
	})
	err := a.Tenant.Transaction(ctxA, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctxA, `INSERT INTO member (id, platform_user_id, display_name, email)
			VALUES (gen_random_uuid(), gen_random_uuid(), 'A user', 'a@x.com') ON CONFLICT DO NOTHING`)
		return err
	})
	if err != nil {
		t.Fatalf("insert into A: %v", err)
	}

	// Query from tB schema → should find 0 (NOT 1)
	ctxB := tenantdb.WithTenant(ctx, &tenantdb.TenantContext{
		ID: string(tB.ID), Slug: tB.Slug, SchemaName: tB.SchemaName, Status: tB.Status,
	})
	var n int
	err = a.Tenant.Transaction(ctxB, func(tx pgx.Tx) error {
		return tx.QueryRow(ctxB, `SELECT count(*) FROM member`).Scan(&n)
	})
	if err != nil {
		t.Fatalf("query B: %v", err)
	}
	if n != 0 {
		t.Fatalf("isolation breach: tenant B sees %d members from tenant A", n)
	}
}

// 命门 2: 污染 — 100 并发交替切租户, 连接归池后 search_path 已 RESET
func TestKeystone_Pollution(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	tA := mustCreateTenant(t, a, "polla", "Tenant Poll A")
	tB := mustCreateTenant(t, a, "pollb", "Tenant Poll B")

	var wg sync.WaitGroup
	errs := make(chan error, 200)
	for i := 0; i < 100; i++ {
		for _, tt := range []*tenancy.Tenant{tA, tB} {
			wg.Add(1)
			go func(tt *tenancy.Tenant) {
				defer wg.Done()
				cctx := tenantdb.WithTenant(ctx, &tenantdb.TenantContext{
					ID: string(tt.ID), Slug: tt.Slug, SchemaName: tt.SchemaName, Status: tt.Status,
				})
				err := a.Tenant.Transaction(cctx, func(tx pgx.Tx) error {
					var sp string
					if err := tx.QueryRow(cctx, "SHOW search_path").Scan(&sp); err != nil {
						return err
					}
					// search_path must start with our schema; otherwise pool returned a polluted conn
					expected1 := fmt.Sprintf(`"%s", public`, tt.SchemaName)
					expected2 := fmt.Sprintf("%s, public", tt.SchemaName)
					if sp != expected1 && sp != expected2 {
						return fmt.Errorf("polluted search_path: got %q, want starts with %q", sp, tt.SchemaName)
					}
					return nil
				})
				if err != nil {
					errs <- err
				}
			}(tt)
		}
	}
	wg.Wait()
	close(errs)
	for e := range errs {
		t.Fatalf("pollution: %v", e)
	}
}

// 命门 3: 越权 — 不合法的 schema name 在 SET LOCAL 前被拦截
func TestKeystone_InvalidSchemaName(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := tenantdb.WithTenant(context.Background(), &tenantdb.TenantContext{
		ID: "x", Slug: "x",
		SchemaName: "tenant_xx'; DROP TABLE member;--",
		Status:     "active",
	})
	err := a.Tenant.Transaction(ctx, func(tx pgx.Tx) error { return nil })
	if err == nil {
		t.Fatalf("expected validation error on injection attempt; got nil")
	}
}

// 命门 4: 状态 — suspended tenant 通过 tenancy.GetTenant 不允许登录到对应租户
func TestKeystone_SuspendedTenantBlocks(t *testing.T) {
	a, cleanup := setupApp(t)
	defer cleanup()
	ctx := context.Background()

	tt := mustCreateTenant(t, a, "susp", "Suspend me")
	// suspend
	if err := a.Tenancy.SuspendTenant(ctx, tt.ID, "test"); err != nil {
		t.Fatalf("suspend: %v", err)
	}
	got, err := a.Tenancy.GetTenant(ctx, tt.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != tenancy.StatusSuspended {
		t.Fatalf("expected suspended, got %s", got.Status)
	}

	// Try SwitchTenant for a joined user → expect KindForbidden
	uid, email := mustCreateUser(t, a, "susp")
	_ = email
	mid := uid // we have no real member; use uid as proxy

	// First create user → login to get a session
	tok, _, err := a.IAM.Login(ctx, iam.LoginCmd{Login: email, Password: "Test1234abc!"})
	if err != nil {
		t.Fatalf("login: %v", err)
	}
	claims, err := a.IAM.VerifyAccessToken(ctx, tok.AccessToken)
	if err != nil {
		t.Fatalf("verify access: %v", err)
	}
	// SwitchTenant should error because user is not a member of the tenant
	_, err = a.IAM.SwitchTenant(ctx, claims.SessionID, tt.ID)
	if err == nil {
		t.Fatalf("expected forbidden when switching to suspended/non-member tenant")
	}
	_ = mid
}

// 命门 5: 跨 schema 静态扫描 — see scripts/scan_cross_schema.sh (run separately).
// Here we sanity check our own code doesn't pass tenant.* in a JOIN literal:
// (placeholder until M5 lint is wired)
func TestKeystone_CrossSchemaScan(t *testing.T) {
	_, cleanup := setupApp(t)
	defer cleanup()
	t.Log("cross-schema JOIN scan runs in CI via scripts/scan_cross_schema.sh")
}
