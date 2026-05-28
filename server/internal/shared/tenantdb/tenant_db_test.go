package tenantdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/infrastructure/pg"
)

func setupPool(t *testing.T) *pgxpool.Pool {
	t.Helper()
	dsn := os.Getenv("IOP_TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable"
	}
	pool, err := pg.NewPool(context.Background(), pg.Config{DSN: dsn, MaxConns: 4})
	if err != nil {
		t.Skipf("PG unavailable: %v", err)
	}
	return pool
}

func TestTenantDB_SetsSearchPath(t *testing.T) {
	pool := setupPool(t)
	defer pool.Close()

	ctx := context.Background()
	_, _ = pool.Exec(ctx, "CREATE SCHEMA IF NOT EXISTS tenant_smoketest")
	defer func() { _, _ = pool.Exec(ctx, "DROP SCHEMA IF EXISTS tenant_smoketest CASCADE") }()

	tdb := NewTenantDB(pool)
	ctxT := WithTenant(ctx, &TenantContext{
		ID: "id-smoke", Slug: "smoketest", SchemaName: "tenant_smoketest", Status: "active",
	})

	err := tdb.Transaction(ctxT, func(tx pgx.Tx) error {
		var sp string
		if err := tx.QueryRow(ctxT, "SHOW search_path").Scan(&sp); err != nil {
			return err
		}
		// PG returns the value with quotes around identifiers containing special chars,
		// or without if the identifier is simple. Both forms acceptable.
		if sp != `"tenant_smoketest", public` && sp != "tenant_smoketest, public" {
			t.Fatalf("expected search_path tenant_smoketest first, got %q", sp)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("transaction failed: %v", err)
	}
}

func TestTenantDB_MissingContext(t *testing.T) {
	pool := setupPool(t)
	defer pool.Close()
	tdb := NewTenantDB(pool)
	err := tdb.Transaction(context.Background(), func(tx pgx.Tx) error { return nil })
	if err == nil {
		t.Fatalf("expected error when ctx has no tenant")
	}
}

func TestTenantDB_InvalidSchemaName(t *testing.T) {
	pool := setupPool(t)
	defer pool.Close()
	tdb := NewTenantDB(pool)
	ctx := WithTenant(context.Background(), &TenantContext{
		ID: "x", Slug: "x", SchemaName: "tenant; DROP TABLE users;--", Status: "active",
	})
	err := tdb.Transaction(ctx, func(tx pgx.Tx) error { return nil })
	if err == nil {
		t.Fatalf("expected validation error on dangerous schema name")
	}
}
