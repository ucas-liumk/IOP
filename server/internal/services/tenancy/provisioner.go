package tenancy

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/shared/errors"
)

// SchemaProvisioner creates per-tenant schemas and applies the tenant_template migrations.
type SchemaProvisioner struct {
	pool        *pgxpool.Pool
	templateDir string
}

func NewSchemaProvisioner(pool *pgxpool.Pool, templateDir string) *SchemaProvisioner {
	return &SchemaProvisioner{pool: pool, templateDir: templateDir}
}

var schemaNameRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

// Provision creates schema <schemaName> and runs every tenant_template/*.up.sql in order.
// Also records each applied migration in public.migration_history with scope=<schemaName>.
func (p *SchemaProvisioner) Provision(ctx context.Context, schemaName string) error {
	if !schemaNameRe.MatchString(schemaName) {
		return errors.New(errors.KindParam, "tenancy.invalid_schema_name",
			"schema name failed validation: "+schemaName)
	}
	if _, err := p.pool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA IF NOT EXISTS %q", schemaName)); err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.create_schema_failed", "create schema failed", err)
	}

	migs, err := loadMigrations(p.templateDir)
	if err != nil {
		return err
	}

	for _, m := range migs {
		// Set search_path to the new schema for this connection only.
		conn, err := p.pool.Acquire(ctx)
		if err != nil {
			return err
		}
		tx, err := conn.Begin(ctx)
		if err != nil {
			conn.Release()
			return err
		}
		if _, err := tx.Exec(ctx, fmt.Sprintf("SET LOCAL search_path TO %q, public", schemaName)); err != nil {
			tx.Rollback(ctx)
			conn.Release()
			return errors.Wrap(errors.KindDatabase, "tenancy.set_search_path_failed", "set search_path failed", err)
		}
		if _, err := tx.Exec(ctx, m.sql); err != nil {
			tx.Rollback(ctx)
			conn.Release()
			return errors.Wrap(errors.KindDatabase, "tenancy.apply_template_failed",
				fmt.Sprintf("apply %s failed", m.id), err)
		}
		// Record in migration_history (scope = schemaName)
		if _, err := tx.Exec(ctx,
			"INSERT INTO public.migration_history (id, scope, migration_id, applied_at, checksum) VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING",
			uuid.New(), schemaName, m.id, time.Now().UTC(), m.sum); err != nil {
			tx.Rollback(ctx)
			conn.Release()
			return err
		}
		if err := tx.Commit(ctx); err != nil {
			conn.Release()
			return err
		}
		conn.Release()
	}
	return nil
}

// Drop is used for tenant close; CASCADE removes all per-tenant data.
func (p *SchemaProvisioner) Drop(ctx context.Context, schemaName string) error {
	if !schemaNameRe.MatchString(schemaName) {
		return errors.New(errors.KindParam, "tenancy.invalid_schema_name", "schema name invalid")
	}
	_, err := p.pool.Exec(ctx, fmt.Sprintf("DROP SCHEMA IF EXISTS %q CASCADE", schemaName))
	if err != nil {
		return errors.Wrap(errors.KindDatabase, "tenancy.drop_schema_failed", "drop schema failed", err)
	}
	return nil
}

type migration struct {
	id  string
	sql string
	sum string
}

func loadMigrations(dir string) ([]migration, error) {
	// Resolve dir from multiple candidate locations so the same code works whether
	// the server binary is launched from server/ or a test from server/test/integration/.
	candidates := []string{
		os.Getenv("IOP_MIGRATIONS_DIR"),
		dir,
		"../" + dir,
		"../../" + dir,
		"../../../" + dir,
	}
	var entries []os.DirEntry
	var resolved string
	var err error
	for _, c := range candidates {
		if c == "" {
			continue
		}
		if entries, err = os.ReadDir(c); err == nil {
			resolved = c
			break
		}
	}
	if err != nil {
		return nil, errors.Wrap(errors.KindInternal, "tenancy.read_template_dir",
			"cannot read tenant_template dir "+dir, err)
	}
	dir = resolved
	var out []migration
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".up.sql") {
			continue
		}
		id := strings.TrimSuffix(e.Name(), ".up.sql")
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		h := sha256.Sum256(raw)
		out = append(out, migration{id: id, sql: string(raw), sum: hex.EncodeToString(h[:])})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].id < out[j].id })
	return out, nil
}
