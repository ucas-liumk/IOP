package tenantdb

import (
	"context"
	"regexp"

	"github.com/leo/iop/server/internal/shared/errors"
)

// TenantContext is the loaded tenant metadata held in request ctx.
// Populated by tenant-loader middleware (M2). M1 tests fake this.
type TenantContext struct {
	ID         string // public.tenant.id (UUID string)
	Slug       string // 'acme'
	SchemaName string // 'tenant_acme'
	Status     string // active / suspended / closed
}

type tenantCtxKeyT struct{}

var tenantCtxKey = tenantCtxKeyT{}

func WithTenant(ctx context.Context, tc *TenantContext) context.Context {
	return context.WithValue(ctx, tenantCtxKey, tc)
}

func FromContext(ctx context.Context) (*TenantContext, bool) {
	v, ok := ctx.Value(tenantCtxKey).(*TenantContext)
	return v, ok
}

// schemaIdentRe enforces safe identifier: lowercase letters, digits, underscore.
// Prevents SQL injection in SET LOCAL search_path.
var schemaIdentRe = regexp.MustCompile(`^[a-z][a-z0-9_]{0,62}$`)

func validateSchemaIdent(name string) error {
	if !schemaIdentRe.MatchString(name) {
		return errors.New(errors.KindInternal, "tenantdb.invalid_schema",
			"schema name failed identifier validation: "+name)
	}
	return nil
}
