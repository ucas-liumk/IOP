# M1 Foundation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Deliver `iop/server` and `iop/web` minimum runnable skeleton + `deployments/docker-compose.yml` + `Makefile` + GitHub Actions CI, completing M1 milestone of v3.1 spec (骨架 + 基础设施).

**Architecture:** Go modular monolith (single `cmd/server` binary, single-tenant DB pool with per-tenant schema isolation), Vue 3 SPA (single Vite project, Pinia stores, axios client), in-process eventbus, docker-compose orchestration. No business logic in M1 — only the foundation that M2-M5 will build on.

**Tech Stack:**
- **Backend**: Go 1.22+, Gin v1.10, pgx/v5, jackc/tern (migrations), go-redis/v9, minio-go/v7, zap, Prometheus client_golang, google/uuid (v7), robfig/cron/v3
- **Frontend**: Vue 3.5, Vite 6, TypeScript 5.7, Pinia 2.3, vue-router 4.5, axios 1.7, dayjs, vitest
- **Infra**: PostgreSQL 16, Redis 7-alpine, MinIO latest
- **CI**: GitHub Actions

**Team Plan (3 fullstack, 4 weeks):**
- Week 1: Project skeleton + shared libs (Tasks 1-9)
- Week 2: Infrastructure + HTTP server (Tasks 10-17)
- Week 3: Migrations + services skeleton + frontend (Tasks 18-23)
- Week 4: CI + smoke + docs (Tasks 24-27)

**Out of scope for M1** (per v3.1 §0):
- Tenancy/IAM business logic (M2)
- Rate limiting / idempotency / slow query hook (M2)
- OKR business module (M4)
- Backups, monitoring dashboards (M2/M5)

---

## File Structure

After M1, the repo looks like:

```
iop/
├── server/
│   ├── cmd/{server,migrate,tenantctl}/main.go
│   ├── internal/
│   │   ├── shared/{kernel,errors,eventbus,tenantdb}/
│   │   ├── services/{dictionary,localization}/   ← skeleton only
│   │   ├── infrastructure/{pg,redis,minio,logger,metrics,health}/
│   │   ├── interface/{middleware,apiresp,server.go}
│   │   ├── config/config.go
│   │   └── app/app.go
│   ├── migrations/public/000001_init.{up,down}.sql
│   ├── api/openapi.yaml
│   ├── configs/dev.yaml
│   ├── go.mod / go.sum / Makefile
│   └── test/integration/smoke_test.go
├── web/
│   ├── src/
│   │   ├── shell/{layout,auth,tenant,notify,workspace,components,stores}/
│   │   ├── api/client.ts
│   │   ├── router/index.ts
│   │   ├── styles/{tokens.css,global.css}
│   │   ├── main.ts / App.vue / env.d.ts
│   ├── package.json / vite.config.ts / tsconfig.json / index.html
│   └── tests/setup.ts
├── deployments/
│   ├── docker-compose.yml
│   └── nginx/nginx.conf.dev
├── scripts/{dev.sh,openapi-gen.sh}
├── .github/workflows/ci.yml
├── .gitignore
├── .editorconfig
└── README.md
```

Each directory has clear responsibility. shared/ libs have no external dependencies on each other except kernel/errors. infrastructure/ wraps third-party clients. services/ in M1 is just empty skeletons that register HTTP routes for future expansion.

---

## Phase 1: Project Skeleton (Week 1)

### Task 1: Initialize Go module and directory structure

**Files:**
- Create: `iop/server/go.mod`
- Create: `iop/server/.gitignore`
- Create: `iop/server/Makefile`
- Create: `iop/server/internal/.keep`

- [ ] **Step 1: Initialize Go module**

```bash
cd /Users/leo/Documents/iop
mkdir -p server && cd server
go mod init github.com/leo/iop/server
```

- [ ] **Step 2: Create directory skeleton**

```bash
cd /Users/leo/Documents/iop/server
mkdir -p cmd/{server,migrate,tenantctl}
mkdir -p internal/{shared/{kernel,errors,eventbus,tenantdb},services/{dictionary,localization},infrastructure/{pg,redis,minio,logger,metrics,health},interface/{middleware,apiresp},config,app}
mkdir -p migrations/{public,tenant_template}
mkdir -p api configs test/integration
touch internal/{shared/kernel,shared/errors,shared/eventbus,shared/tenantdb,services/dictionary,services/localization}/.keep
```

- [ ] **Step 2.5: Create stub main.go for cmd/server, cmd/migrate, cmd/tenantctl**

So `make build` succeeds before T15/T16 fill these in:

```bash
cat > cmd/server/main.go <<'EOF'
package main

func main() { /* TODO Task 15 */ }
EOF

cat > cmd/migrate/main.go <<'EOF'
package main

func main() { /* TODO Task 16 */ }
EOF

cat > cmd/tenantctl/main.go <<'EOF'
package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Println("tenantctl: M2 will implement create/suspend/resume/close/migrate-all")
	os.Exit(0)
}
EOF
```

- [ ] **Step 3: Create `.gitignore`**

```
# Binaries
/bin/
*.test
*.out
coverage.html

# IDE
.idea/
.vscode/
*.swp

# Env / secrets
.env
.env.local
configs/local.yaml

# OS
.DS_Store
```

- [ ] **Step 4: Create minimal Makefile**

```makefile
.PHONY: dev build test lint migrate openapi-gen clean

GO         := go
GOFLAGS    := -trimpath
LDFLAGS    := -s -w
BIN_DIR    := bin

build:
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/server ./cmd/server
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/migrate ./cmd/migrate
	$(GO) build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/tenantctl ./cmd/tenantctl

test:
	$(GO) test -race -coverprofile=coverage.out ./...

lint:
	golangci-lint run ./...

dev:
	$(GO) run ./cmd/server

migrate:
	$(GO) run ./cmd/migrate up

openapi-gen:
	@echo "TODO M3: regenerate frontend SDK from api/openapi.yaml"

clean:
	rm -rf $(BIN_DIR) coverage.out
```

- [ ] **Step 5: Verify go.mod and build**

```bash
cd /Users/leo/Documents/iop/server
go mod tidy
go build ./...
```

Expected: build succeeds with empty packages (no errors).

- [ ] **Step 6: Commit**

```bash
cd /Users/leo/Documents/iop
git init  # if not yet a repo
git add server/.gitignore server/Makefile server/go.mod server/internal/ server/cmd/ server/migrations/ server/api/ server/configs/ server/test/
git commit -m "chore(server): scaffold Go module + directory structure"
```

---

### Task 2: docker-compose dev environment

**Files:**
- Create: `iop/deployments/docker-compose.yml`
- Create: `iop/deployments/nginx/nginx.conf.dev`

- [ ] **Step 1: Create `deployments/docker-compose.yml`**

```yaml
version: "3.9"

services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: iop
      POSTGRES_PASSWORD: iop_dev
      POSTGRES_DB: iop
    ports: ["5432:5432"]
    volumes:
      - pg-data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U iop -d iop"]
      interval: 5s
      timeout: 3s
      retries: 10

  redis:
    image: redis:7-alpine
    ports: ["6379:6379"]
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 10

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: iop_dev
      MINIO_ROOT_PASSWORD: iop_dev_password
    ports: ["9000:9000", "9001:9001"]
    volumes:
      - minio-data:/data
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 5s
      timeout: 3s
      retries: 10

  server:
    build:
      context: ../server
      dockerfile: ../deployments/Dockerfile.server
    depends_on:
      db: { condition: service_healthy }
      redis: { condition: service_healthy }
      minio: { condition: service_healthy }
    environment:
      IOP_ENV: dev
      IOP_DB_DSN: postgres://iop:iop_dev@db:5432/iop?sslmode=disable
      IOP_REDIS_ADDR: redis:6379
      IOP_MINIO_ENDPOINT: minio:9000
      IOP_MINIO_ACCESS_KEY: iop_dev
      IOP_MINIO_SECRET_KEY: iop_dev_password
    ports: ["8080:8080"]
    profiles: ["full"]   # ← M1 dev path: run `go run ./cmd/server` locally instead

  web:
    build:
      context: ../web
      dockerfile: ../deployments/Dockerfile.web
    ports: ["5173:80"]
    profiles: ["full"]   # ← M1 dev path: run `npm run dev` locally

volumes:
  pg-data:
  minio-data:
```

Note: `server` and `web` services use `profiles: full` so by default `docker compose up` only starts db/redis/minio. Developers run server / web locally with `go run` / `npm run dev` for faster iteration. M5 will remove the profile and switch to full container runs in prod.

- [ ] **Step 2: Create dev nginx config (stub for M5 prod)**

```nginx
# deployments/nginx/nginx.conf.dev
events {}
http {
    server {
        listen 80;
        location /api/ {
            proxy_pass http://server:8080/;
            proxy_set_header Host $host;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        }
        location / {
            root /usr/share/nginx/html;
            try_files $uri /index.html;
        }
    }
}
```

- [ ] **Step 3: Verify db / redis / minio start**

```bash
cd /Users/leo/Documents/iop/deployments
docker compose up -d db redis minio
docker compose ps
docker compose logs --tail 20
```

Expected: 3 services Up, healthy after ~10s.

- [ ] **Step 4: Verify PG connectivity**

```bash
docker compose exec db psql -U iop -d iop -c '\dt'
```

Expected: "Did not find any relations." (empty database, no error).

- [ ] **Step 5: Commit**

```bash
git add deployments/docker-compose.yml deployments/nginx/
git commit -m "chore(deploy): docker-compose dev with PG/Redis/MinIO (profile gated)"
```

---

## Phase 2: shared/ libraries (Week 1-2)

### Task 3: shared/kernel — IDs (UUID v7)

**Files:**
- Create: `server/internal/shared/kernel/ids.go`
- Create: `server/internal/shared/kernel/ids_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/shared/kernel/ids_test.go
package kernel

import (
	"testing"
	"time"
)

func TestNewID_Unique(t *testing.T) {
	a := NewID()
	b := NewID()
	if a == b {
		t.Fatalf("expected different IDs, got %s and %s", a, b)
	}
}

func TestNewID_TimeOrdered(t *testing.T) {
	a := NewID()
	time.Sleep(2 * time.Millisecond)
	b := NewID()
	if a >= b {
		t.Fatalf("expected UUID v7 to be time-ordered, got a=%s >= b=%s", a, b)
	}
}

func TestParseID_Valid(t *testing.T) {
	id := NewID()
	parsed, err := ParseID(string(id))
	if err != nil {
		t.Fatalf("expected valid parse, got %v", err)
	}
	if parsed != id {
		t.Fatalf("round-trip mismatch")
	}
}

func TestParseID_Invalid(t *testing.T) {
	if _, err := ParseID("not-a-uuid"); err == nil {
		t.Fatalf("expected error on invalid uuid")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
cd /Users/leo/Documents/iop/server
go test ./internal/shared/kernel/...
```

Expected: FAIL with "undefined: NewID" etc.

- [ ] **Step 3: Add google/uuid dependency**

```bash
go get github.com/google/uuid@v1.6.0
```

- [ ] **Step 4: Implement IDs**

```go
// internal/shared/kernel/ids.go
package kernel

import (
	"fmt"

	"github.com/google/uuid"
)

// ID is the canonical 128-bit identifier used across all aggregates and entities.
// Format: UUID v7 (time-ordered, sortable). String form: canonical 8-4-4-4-12 hex.
type ID string

// NewID returns a fresh UUID v7. Time-ordered: lexicographic sort == chronological.
func NewID() ID {
	u, err := uuid.NewV7()
	if err != nil {
		// uuid.NewV7 only fails if crypto/rand fails, which is unrecoverable.
		panic(fmt.Sprintf("kernel: cannot generate UUID v7: %v", err))
	}
	return ID(u.String())
}

// ParseID validates and normalizes an external ID string.
func ParseID(s string) (ID, error) {
	u, err := uuid.Parse(s)
	if err != nil {
		return "", fmt.Errorf("invalid id %q: %w", s, err)
	}
	return ID(u.String()), nil
}

// String for fmt.Stringer / logging compatibility.
func (id ID) String() string { return string(id) }
```

- [ ] **Step 5: Run test to verify pass**

```bash
go test ./internal/shared/kernel/... -v
```

Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add server/internal/shared/kernel/ids.go server/internal/shared/kernel/ids_test.go server/go.mod server/go.sum
git commit -m "feat(kernel): UUID v7 ID type with parse/validate"
```

---

### Task 4: shared/kernel — Context utilities (trace, tenant, member)

**Files:**
- Create: `server/internal/shared/kernel/context.go`
- Create: `server/internal/shared/kernel/context_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/shared/kernel/context_test.go
package kernel

import (
	"context"
	"testing"
)

func TestContext_TraceID(t *testing.T) {
	ctx := WithTraceID(context.Background(), "trace-abc")
	if got := TraceIDFromContext(ctx); got != "trace-abc" {
		t.Fatalf("expected trace-abc, got %q", got)
	}
}

func TestContext_TraceID_Missing(t *testing.T) {
	if got := TraceIDFromContext(context.Background()); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestContext_TenantID(t *testing.T) {
	tid := NewID()
	ctx := WithTenantID(context.Background(), tid)
	got, ok := TenantIDFromContext(ctx)
	if !ok || got != tid {
		t.Fatalf("expected %s, got %s ok=%v", tid, got, ok)
	}
}

func TestContext_MemberID(t *testing.T) {
	mid := NewID()
	ctx := WithMemberID(context.Background(), mid)
	got, ok := MemberIDFromContext(ctx)
	if !ok || got != mid {
		t.Fatalf("expected %s, got %s ok=%v", mid, got, ok)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/shared/kernel/...
```

Expected: FAIL with "undefined: WithTraceID" etc.

- [ ] **Step 3: Implement context helpers**

```go
// internal/shared/kernel/context.go
package kernel

import "context"

type ctxKey int

const (
	keyTraceID ctxKey = iota + 1
	keyTenantID
	keyMemberID
)

// WithTraceID attaches a request-scoped trace id (set by middleware/request_id).
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, keyTraceID, traceID)
}

// TraceIDFromContext returns the trace id or "" if not set.
func TraceIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(keyTraceID).(string); ok {
		return v
	}
	return ""
}

// WithTenantID attaches the active tenant id (set by tenant-loader middleware in M2).
func WithTenantID(ctx context.Context, tenantID ID) context.Context {
	return context.WithValue(ctx, keyTenantID, tenantID)
}

// TenantIDFromContext returns (tenantID, true) if set, else ("", false).
func TenantIDFromContext(ctx context.Context) (ID, bool) {
	v, ok := ctx.Value(keyTenantID).(ID)
	return v, ok
}

// WithMemberID attaches the calling member id (set by IAM middleware in M2).
func WithMemberID(ctx context.Context, memberID ID) context.Context {
	return context.WithValue(ctx, keyMemberID, memberID)
}

// MemberIDFromContext returns (memberID, true) if set, else ("", false).
func MemberIDFromContext(ctx context.Context) (ID, bool) {
	v, ok := ctx.Value(keyMemberID).(ID)
	return v, ok
}
```

- [ ] **Step 4: Run test to verify pass**

```bash
go test ./internal/shared/kernel/... -v
```

Expected: PASS (8 tests now).

- [ ] **Step 5: Commit**

```bash
git add server/internal/shared/kernel/context.go server/internal/shared/kernel/context_test.go
git commit -m "feat(kernel): ctx helpers for trace/tenant/member id"
```

---

### Task 5: shared/kernel — Clock + Pagination

**Files:**
- Create: `server/internal/shared/kernel/time.go`
- Create: `server/internal/shared/kernel/pagination.go`
- Create: `server/internal/shared/kernel/time_test.go`
- Create: `server/internal/shared/kernel/pagination_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/shared/kernel/time_test.go
package kernel

import (
	"testing"
	"time"
)

func TestRealClock(t *testing.T) {
	c := RealClock{}
	now := c.Now()
	if now.IsZero() {
		t.Fatalf("expected non-zero time")
	}
	if time.Since(now) > time.Second {
		t.Fatalf("clock drift too large")
	}
}

func TestFakeClock(t *testing.T) {
	fixed := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	c := NewFakeClock(fixed)
	if !c.Now().Equal(fixed) {
		t.Fatalf("expected %v, got %v", fixed, c.Now())
	}
	c.Advance(2 * time.Hour)
	expected := fixed.Add(2 * time.Hour)
	if !c.Now().Equal(expected) {
		t.Fatalf("expected %v after advance, got %v", expected, c.Now())
	}
}
```

```go
// internal/shared/kernel/pagination_test.go
package kernel

import "testing"

func TestPagination_Defaults(t *testing.T) {
	p := Pagination{}.Normalize()
	if p.Page != 1 || p.PageSize != 20 {
		t.Fatalf("expected defaults page=1 pageSize=20, got %+v", p)
	}
}

func TestPagination_Clamp(t *testing.T) {
	p := Pagination{Page: -5, PageSize: 9999}.Normalize()
	if p.Page != 1 {
		t.Fatalf("expected page clamped to 1, got %d", p.Page)
	}
	if p.PageSize != 200 {
		t.Fatalf("expected pageSize clamped to max 200, got %d", p.PageSize)
	}
}

func TestPagination_Offset(t *testing.T) {
	p := Pagination{Page: 3, PageSize: 50}
	if p.Offset() != 100 {
		t.Fatalf("expected offset 100, got %d", p.Offset())
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/shared/kernel/...
```

Expected: FAIL with undefined symbols.

- [ ] **Step 3: Implement Clock**

```go
// internal/shared/kernel/time.go
package kernel

import (
	"sync"
	"time"
)

// Clock abstracts time. Injected into services so tests can use FakeClock.
type Clock interface {
	Now() time.Time
}

// RealClock is the production implementation. Use in app.go DI.
type RealClock struct{}

func (RealClock) Now() time.Time { return time.Now().UTC() }

// FakeClock is for tests. Time advances only when Advance() is called.
type FakeClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFakeClock(t time.Time) *FakeClock { return &FakeClock{now: t} }

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}
```

- [ ] **Step 4: Implement Pagination**

```go
// internal/shared/kernel/pagination.go
package kernel

const (
	defaultPage     = 1
	defaultPageSize = 20
	maxPageSize     = 200
)

// Pagination is a value object passed to Query application services.
type Pagination struct {
	Page     int `form:"page"      json:"page"`
	PageSize int `form:"page_size" json:"page_size"`
}

// Normalize clamps invalid input to safe defaults; returns a new Pagination.
// Always call Normalize() before using Pagination in a repository.
func (p Pagination) Normalize() Pagination {
	if p.Page < 1 {
		p.Page = defaultPage
	}
	if p.PageSize < 1 {
		p.PageSize = defaultPageSize
	}
	if p.PageSize > maxPageSize {
		p.PageSize = maxPageSize
	}
	return p
}

// Offset returns the SQL OFFSET equivalent.
func (p Pagination) Offset() int {
	return (p.Page - 1) * p.PageSize
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
go test ./internal/shared/kernel/... -v
```

Expected: PASS (all tests).

- [ ] **Step 6: Commit**

```bash
git add server/internal/shared/kernel/time.go server/internal/shared/kernel/time_test.go server/internal/shared/kernel/pagination.go server/internal/shared/kernel/pagination_test.go
git commit -m "feat(kernel): Clock + Pagination value objects"
```

---

### Task 6: shared/errors — Error model

**Files:**
- Create: `server/internal/shared/errors/kind.go`
- Create: `server/internal/shared/errors/error.go`
- Create: `server/internal/shared/errors/error_test.go`

- [ ] **Step 1: Write failing tests**

```go
// internal/shared/errors/error_test.go
package errors

import (
	"errors"
	"testing"
)

func TestNew(t *testing.T) {
	e := New(KindBusiness, "okr.plan.invalid_period", "period must be non-empty")
	if e.Kind != KindBusiness {
		t.Fatalf("kind mismatch")
	}
	if e.Code != "okr.plan.invalid_period" {
		t.Fatalf("code mismatch")
	}
	if e.Error() != "okr.plan.invalid_period: period must be non-empty" {
		t.Fatalf("error string mismatch: %q", e.Error())
	}
}

func TestWrap(t *testing.T) {
	cause := errors.New("connection refused")
	e := Wrap(KindDatabase, "db.connection_failed", "cannot connect to PG", cause)
	if !errors.Is(e, cause) {
		t.Fatalf("expected errors.Is to find cause")
	}
}

func TestAs(t *testing.T) {
	e := New(KindAuth, "iam.invalid_password", "wrong password")
	var target *Error
	if !errors.As(e, &target) {
		t.Fatalf("expected errors.As to match")
	}
	if target.Code != "iam.invalid_password" {
		t.Fatalf("As did not set target")
	}
}

func TestKind_String(t *testing.T) {
	if KindBusiness.String() != "business" {
		t.Fatalf("kind name mismatch")
	}
	if KindUnknown.String() != "unknown" {
		t.Fatalf("kind name mismatch")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/shared/errors/...
```

Expected: FAIL.

- [ ] **Step 3: Implement Kind enum**

```go
// internal/shared/errors/kind.go
package errors

// Kind categorizes errors for HTTP status mapping and observability.
type Kind int

const (
	KindUnknown    Kind = iota
	KindBusiness         // 400 — business rule violation, safe to show user
	KindParam            // 400 — invalid request parameter
	KindAuth             // 401 — unauthenticated
	KindForbidden        // 403 — authenticated but not authorized
	KindNotFound         // 404 — resource missing or hidden by tenant scope
	KindConflict         // 409 — idempotency / version conflict
	KindRateLimit        // 429 — rate limited
	KindDatabase         // 500 — DB / persistence failure
	KindExternal         // 502 — external service failure
	KindInternal         // 500 — programming error / panic recovered
)

func (k Kind) String() string {
	switch k {
	case KindBusiness:
		return "business"
	case KindParam:
		return "param"
	case KindAuth:
		return "auth"
	case KindForbidden:
		return "forbidden"
	case KindNotFound:
		return "not_found"
	case KindConflict:
		return "conflict"
	case KindRateLimit:
		return "rate_limit"
	case KindDatabase:
		return "database"
	case KindExternal:
		return "external"
	case KindInternal:
		return "internal"
	default:
		return "unknown"
	}
}

// HTTPStatus maps Kind to an HTTP status code (used by interface/apiresp).
func (k Kind) HTTPStatus() int {
	switch k {
	case KindBusiness, KindParam:
		return 400
	case KindAuth:
		return 401
	case KindForbidden:
		return 403
	case KindNotFound:
		return 404
	case KindConflict:
		return 409
	case KindRateLimit:
		return 429
	case KindExternal:
		return 502
	case KindDatabase, KindInternal:
		return 500
	default:
		return 500
	}
}
```

- [ ] **Step 4: Implement Error**

```go
// internal/shared/errors/error.go
package errors

import "fmt"

// Error is the canonical error type. Always construct via New or Wrap.
// Code is a stable machine-readable string (e.g. "okr.plan.invalid_period")
// that maps 1:1 to an i18n message key.
type Error struct {
	Kind    Kind   // category for HTTP status + logging
	Code    string // stable code, dot-separated: <source>.<resource>.<reason>
	Message string // human-readable Chinese message for fallback (i18n key takes priority)
	Cause   error  // underlying cause (may be nil)
}

func New(kind Kind, code, message string) *Error {
	return &Error{Kind: kind, Code: code, Message: message}
}

func Wrap(kind Kind, code, message string, cause error) *Error {
	return &Error{Kind: kind, Code: code, Message: message, Cause: cause}
}

func (e *Error) Error() string {
	if e.Message == "" {
		return e.Code
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap supports errors.Is / errors.As.
func (e *Error) Unwrap() error { return e.Cause }
```

- [ ] **Step 5: Run tests to verify pass**

```bash
go test ./internal/shared/errors/... -v
```

Expected: PASS (4 tests).

- [ ] **Step 6: Commit**

```bash
git add server/internal/shared/errors/
git commit -m "feat(errors): Kind enum + Error type with Wrap/Unwrap"
```

---

### Task 7: shared/eventbus — Bus interface + in-process impl

**Files:**
- Create: `server/internal/shared/eventbus/event.go`
- Create: `server/internal/shared/eventbus/bus.go`
- Create: `server/internal/shared/eventbus/inproc_bus.go`
- Create: `server/internal/shared/eventbus/inproc_bus_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/shared/eventbus/inproc_bus_test.go
package eventbus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestInprocBus_PublishSubscribe(t *testing.T) {
	bus := NewInprocBus(4)
	defer bus.Close()

	var mu sync.Mutex
	var received []Event
	bus.Subscribe("test.foo", func(ctx context.Context, e Event) error {
		mu.Lock()
		received = append(received, e)
		mu.Unlock()
		return nil
	})

	bus.Start()
	if err := bus.Publish(context.Background(), "test.foo", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("publish failed: %v", err)
	}

	// Wait up to 1s for the async handler.
	deadline := time.Now().Add(1 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(received)
		mu.Unlock()
		if n == 1 {
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(received) != 1 {
		t.Fatalf("expected 1 event received, got %d", len(received))
	}
	if received[0].Topic != "test.foo" {
		t.Fatalf("topic mismatch")
	}
}

func TestInprocBus_MultipleSubscribers(t *testing.T) {
	bus := NewInprocBus(4)
	defer bus.Close()

	var count1, count2 int
	var mu sync.Mutex
	bus.Subscribe("test.bar", func(ctx context.Context, e Event) error {
		mu.Lock(); count1++; mu.Unlock(); return nil
	})
	bus.Subscribe("test.bar", func(ctx context.Context, e Event) error {
		mu.Lock(); count2++; mu.Unlock(); return nil
	})
	bus.Start()
	_ = bus.Publish(context.Background(), "test.bar", nil)

	time.Sleep(100 * time.Millisecond)
	mu.Lock()
	defer mu.Unlock()
	if count1 != 1 || count2 != 1 {
		t.Fatalf("expected each subscriber to receive 1, got %d / %d", count1, count2)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/shared/eventbus/...
```

Expected: FAIL.

- [ ] **Step 3: Implement Event**

```go
// internal/shared/eventbus/event.go
package eventbus

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Event is the generic envelope. Data carries the topic-specific payload.
type Event struct {
	ID         kernel.ID
	Topic      string    // e.g. "okr.plan_created"
	OccurredAt time.Time // UTC
	TenantID   kernel.ID // empty if cross-tenant
	Actor      string    // member id or "system"
	TraceID    string
	Data       any
}
```

- [ ] **Step 4: Implement Bus interface**

```go
// internal/shared/eventbus/bus.go
package eventbus

import "context"

// Handler is called once per matching event. Errors are logged but do not propagate
// to the publisher (fire-and-forget semantics; if you need synchronous error
// handling, use a direct service call instead of the bus).
type Handler func(ctx context.Context, e Event) error

// Bus is the publish-subscribe abstraction. v1 implementation is in-process channels.
// Future swap target: NATS (when真实削峰信号 appears).
type Bus interface {
	// Publish enqueues an event for asynchronous fan-out. Returns immediately.
	Publish(ctx context.Context, topic string, data any) error

	// Subscribe registers a handler for a topic. Must be called before Start().
	// Calling after Start is allowed but not recommended; new subscriptions
	// only see events published after the call.
	Subscribe(topic string, h Handler)

	// Start launches worker goroutines. Call once during app boot.
	Start()

	// Close drains pending events with a max timeout, then stops workers.
	Close() error
}
```

- [ ] **Step 5: Implement in-process bus**

```go
// internal/shared/eventbus/inproc_bus.go
package eventbus

import (
	"context"
	"sync"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
	"go.uber.org/zap"
)

// InprocBus is a goroutine-pool backed bus. M1 default.
type InprocBus struct {
	workers     int
	queue       chan Event
	subscribers map[string][]Handler
	mu          sync.RWMutex
	wg          sync.WaitGroup
	closed      bool
	logger      *zap.Logger
}

// NewInprocBus creates a bus with `workers` parallel handler goroutines.
// Queue size is workers*16; tune later if backpressure shows up.
func NewInprocBus(workers int) *InprocBus {
	if workers <= 0 {
		workers = 4
	}
	return &InprocBus{
		workers:     workers,
		queue:       make(chan Event, workers*16),
		subscribers: make(map[string][]Handler),
		logger:      zap.NewNop(), // app DI will swap to real logger
	}
}

// WithLogger swaps in a real logger (used during app wiring).
func (b *InprocBus) WithLogger(l *zap.Logger) *InprocBus {
	b.logger = l
	return b
}

func (b *InprocBus) Publish(ctx context.Context, topic string, data any) error {
	b.mu.RLock()
	closed := b.closed
	b.mu.RUnlock()
	if closed {
		return nil // silent no-op after Close (publish-from-deferred during shutdown)
	}
	tenantID, _ := kernel.TenantIDFromContext(ctx)
	actor := ""
	if mid, ok := kernel.MemberIDFromContext(ctx); ok {
		actor = string(mid)
	}
	e := Event{
		ID:         kernel.NewID(),
		Topic:      topic,
		OccurredAt: time.Now().UTC(),
		TenantID:   tenantID,
		Actor:      actor,
		TraceID:    kernel.TraceIDFromContext(ctx),
		Data:       data,
	}
	select {
	case b.queue <- e:
		return nil
	default:
		// Queue full — log and drop. M2 will add metric + alert.
		b.logger.Warn("eventbus queue full, dropping event",
			zap.String("topic", topic), zap.String("event_id", string(e.ID)))
		return nil
	}
}

func (b *InprocBus) Subscribe(topic string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.subscribers[topic] = append(b.subscribers[topic], h)
}

func (b *InprocBus) Start() {
	for i := 0; i < b.workers; i++ {
		b.wg.Add(1)
		go b.worker()
	}
}

func (b *InprocBus) worker() {
	defer b.wg.Done()
	for e := range b.queue {
		b.dispatch(e)
	}
}

func (b *InprocBus) dispatch(e Event) {
	b.mu.RLock()
	handlers := b.subscribers[e.Topic]
	b.mu.RUnlock()
	for _, h := range handlers {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		// Re-attach event context so handlers can use trace/tenant/member.
		ctx = kernel.WithTraceID(ctx, e.TraceID)
		if e.TenantID != "" {
			ctx = kernel.WithTenantID(ctx, e.TenantID)
		}
		if err := h(ctx, e); err != nil {
			b.logger.Error("eventbus handler error",
				zap.String("topic", e.Topic),
				zap.String("event_id", string(e.ID)),
				zap.Error(err))
		}
		cancel()
	}
}

func (b *InprocBus) Close() error {
	b.mu.Lock()
	if b.closed {
		b.mu.Unlock()
		return nil
	}
	b.closed = true
	b.mu.Unlock()
	close(b.queue)
	// Drain with timeout.
	done := make(chan struct{})
	go func() { b.wg.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		b.logger.Warn("eventbus drain timeout, some events may be lost")
	}
	return nil
}
```

- [ ] **Step 6: Add zap dependency and run tests**

```bash
go get go.uber.org/zap@v1.27.0
go test ./internal/shared/eventbus/... -v -race
```

Expected: PASS, no race detected.

- [ ] **Step 7: Commit**

```bash
git add server/internal/shared/eventbus/ server/go.mod server/go.sum
git commit -m "feat(eventbus): in-process Bus with worker pool + topic subscribe"
```

---

### Task 8: infrastructure/pg — pgx pool

**Files:**
- Create: `server/internal/infrastructure/pg/pool.go`
- Create: `server/internal/infrastructure/pg/pool_test.go`

- [ ] **Step 1: Add pgx dependency**

```bash
cd /Users/leo/Documents/iop/server
go get github.com/jackc/pgx/v5@v5.7.1
```

- [ ] **Step 2: Write failing test (integration, uses real PG via docker-compose)**

```go
// internal/infrastructure/pg/pool_test.go
package pg

import (
	"context"
	"os"
	"testing"
	"time"
)

func dsnFromEnv(t *testing.T) string {
	t.Helper()
	dsn := os.Getenv("IOP_TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://iop:iop_dev@localhost:5432/iop?sslmode=disable"
	}
	return dsn
}

func TestNewPool_Connects(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	pool, err := NewPool(ctx, Config{DSN: dsnFromEnv(t), MaxConns: 4})
	if err != nil {
		t.Skipf("PG not reachable (start docker-compose first): %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

func TestNewPool_InvalidDSN(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	_, err := NewPool(ctx, Config{DSN: "postgres://nope:nope@127.0.0.1:1/nope", MaxConns: 1})
	if err == nil {
		t.Fatalf("expected connection error")
	}
}
```

- [ ] **Step 3: Run tests to verify they fail (NewPool undefined)**

```bash
go test ./internal/infrastructure/pg/...
```

Expected: FAIL.

- [ ] **Step 4: Implement pool**

```go
// internal/infrastructure/pg/pool.go
package pg

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config is what app wiring passes in from configs/dev.yaml.
type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
	MaxConnIdleTime time.Duration
}

// NewPool returns a connected pgx pool. Caller owns Close.
func NewPool(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.MaxConns == 0 {
		cfg.MaxConns = 20
	}
	if cfg.MinConns == 0 {
		cfg.MinConns = 2
	}
	if cfg.MaxConnLifetime == 0 {
		cfg.MaxConnLifetime = time.Hour
	}
	if cfg.MaxConnIdleTime == 0 {
		cfg.MaxConnIdleTime = 30 * time.Minute
	}

	pcfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("pg: parse dsn: %w", err)
	}
	pcfg.MaxConns = cfg.MaxConns
	pcfg.MinConns = cfg.MinConns
	pcfg.MaxConnLifetime = cfg.MaxConnLifetime
	pcfg.MaxConnIdleTime = cfg.MaxConnIdleTime

	// M2 will add an AfterRelease hook that RESETs search_path (tenantdb double-protect).
	// Slow query hook (>200ms) added in M2 via interceptor.

	pool, err := pgxpool.NewWithConfig(ctx, pcfg)
	if err != nil {
		return nil, fmt.Errorf("pg: connect: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pg: ping: %w", err)
	}
	return pool, nil
}
```

- [ ] **Step 5: Run tests to verify pass (with PG running)**

```bash
cd /Users/leo/Documents/iop/deployments && docker compose up -d db
cd /Users/leo/Documents/iop/server && go test ./internal/infrastructure/pg/... -v
```

Expected: TestNewPool_Connects PASS, TestNewPool_InvalidDSN PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/infrastructure/pg/ server/go.mod server/go.sum
git commit -m "feat(infra-pg): pgxpool with configurable pool size + Ping check"
```

---

### Task 9: shared/tenantdb — PlatformDB + TenantDB

**Files:**
- Create: `server/internal/shared/tenantdb/tenant_context.go`
- Create: `server/internal/shared/tenantdb/platform_db.go`
- Create: `server/internal/shared/tenantdb/tenant_db.go`
- Create: `server/internal/shared/tenantdb/tenant_db_test.go`

- [ ] **Step 1: Write failing test (integration)**

```go
// internal/shared/tenantdb/tenant_db_test.go
package tenantdb

import (
	"context"
	"os"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/leo/iop/server/internal/infrastructure/pg"
)

func setupPool(t *testing.T) interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Close()
} {
	t.Helper()
	dsn := os.Getenv("IOP_TEST_DB_DSN")
	if dsn == "" {
		dsn = "postgres://iop:iop_dev@localhost:5432/iop?sslmode=disable"
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

	// Make sure schema exists for the test.
	_, _ = pool.(interface {
		Exec(ctx context.Context, sql string, args ...any) (pgx.CommandTag, error)
	}).Exec(context.Background(), "CREATE SCHEMA IF NOT EXISTS tenant_smoketest")
	defer pool.(interface {
		Exec(ctx context.Context, sql string, args ...any) (pgx.CommandTag, error)
	}).Exec(context.Background(), "DROP SCHEMA IF EXISTS tenant_smoketest CASCADE")

	tdb := NewTenantDB(pool.(interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	}))
	ctx := WithTenant(context.Background(), &TenantContext{
		ID: "id-smoke", Slug: "smoketest", SchemaName: "tenant_smoketest", Status: "active",
	})

	err := tdb.Transaction(ctx, func(tx pgx.Tx) error {
		var sp string
		if err := tx.QueryRow(ctx, "SHOW search_path").Scan(&sp); err != nil {
			return err
		}
		if sp != `"tenant_smoketest", public` && sp != "tenant_smoketest, public" {
			t.Fatalf("expected search_path tenant_smoketest, got %q", sp)
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
	tdb := NewTenantDB(pool.(interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	}))
	err := tdb.Transaction(context.Background(), func(tx pgx.Tx) error { return nil })
	if err == nil {
		t.Fatalf("expected error when ctx has no tenant")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/shared/tenantdb/...
```

Expected: FAIL (types undefined).

- [ ] **Step 3: Implement TenantContext**

```go
// internal/shared/tenantdb/tenant_context.go
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
```

- [ ] **Step 4: Implement PlatformDB**

```go
// internal/shared/tenantdb/platform_db.go
package tenantdb

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PlatformDB wraps the pgx pool for public-schema access.
// All queries run on the `public` schema (no SET LOCAL).
type PlatformDB struct {
	pool *pgxpool.Pool
}

func NewPlatformDB(pool *pgxpool.Pool) *PlatformDB {
	return &PlatformDB{pool: pool}
}

func (p *PlatformDB) Pool() *pgxpool.Pool { return p.pool }

// Transaction runs fn in a transaction on the public schema.
func (p *PlatformDB) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	return pgx.BeginFunc(ctx, p.pool, fn)
}

// Query / Exec passthroughs for read-only or single-statement use.
func (p *PlatformDB) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return p.pool.Query(ctx, sql, args...)
}

func (p *PlatformDB) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return p.pool.QueryRow(ctx, sql, args...)
}

func (p *PlatformDB) Exec(ctx context.Context, sql string, args ...any) error {
	_, err := p.pool.Exec(ctx, sql, args...)
	return err
}
```

- [ ] **Step 5: Implement TenantDB**

```go
// internal/shared/tenantdb/tenant_db.go
package tenantdb

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/shared/errors"
)

// TenantDB wraps the same pool as PlatformDB but enforces SET LOCAL search_path
// inside every transaction it opens.
type TenantDB struct {
	pool *pgxpool.Pool
}

func NewTenantDB(pool *pgxpool.Pool) *TenantDB {
	return &TenantDB{pool: pool}
}

// Transaction starts a tx, sets search_path to the tenant's schema, then runs fn.
// SET LOCAL is automatically rolled back on COMMIT or ROLLBACK — no stale state
// can leak to the next user of this connection.
func (t *TenantDB) Transaction(ctx context.Context, fn func(tx pgx.Tx) error) error {
	tc, ok := FromContext(ctx)
	if !ok {
		return errors.New(errors.KindInternal, "tenantdb.context_missing",
			"TenantDB.Transaction called without tenant context")
	}
	if err := validateSchemaIdent(tc.SchemaName); err != nil {
		return err
	}
	return pgx.BeginFunc(ctx, t.pool, func(tx pgx.Tx) error {
		// SET LOCAL is scoped to this transaction only.
		sql := fmt.Sprintf("SET LOCAL search_path TO %q, public", tc.SchemaName)
		if _, err := tx.Exec(ctx, sql); err != nil {
			return errors.Wrap(errors.KindDatabase, "tenantdb.set_search_path",
				"failed to set search_path", err)
		}
		return fn(tx)
	})
}
```

- [ ] **Step 6: Run tests to verify pass**

```bash
go test ./internal/shared/tenantdb/... -v -race
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add server/internal/shared/tenantdb/
git commit -m "feat(tenantdb): PlatformDB + TenantDB with SET LOCAL search_path"
```

---

## Phase 3: HTTP server skeleton (Week 2)

### Task 10: infrastructure/logger — zap with sanitization

**Files:**
- Create: `server/internal/infrastructure/logger/logger.go`
- Create: `server/internal/infrastructure/logger/logger_test.go`

- [ ] **Step 1: Write failing test**

```go
// internal/infrastructure/logger/logger_test.go
package logger

import (
	"bytes"
	"testing"

	"go.uber.org/zap/zapcore"
)

func TestNew_WritesJSON(t *testing.T) {
	var buf bytes.Buffer
	l := newTestable(&buf, zapcore.DebugLevel)
	l.Info("hello", )
	if !bytes.Contains(buf.Bytes(), []byte(`"msg":"hello"`)) {
		t.Fatalf("expected JSON output, got: %s", buf.String())
	}
}

func TestSanitize_RedactsPasswords(t *testing.T) {
	in := map[string]string{
		"username": "alice",
		"password": "supersecret",
		"token":    "abc.def.ghi",
	}
	out := Sanitize(in)
	m, _ := out.(map[string]string)
	if m["password"] != "[REDACTED]" {
		t.Fatalf("password not redacted: %v", m)
	}
	if m["token"] != "[REDACTED]" {
		t.Fatalf("token not redacted: %v", m)
	}
	if m["username"] != "alice" {
		t.Fatalf("username changed: %v", m)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/infrastructure/logger/...
```

Expected: FAIL.

- [ ] **Step 3: Implement logger**

```go
// internal/infrastructure/logger/logger.go
package logger

import (
	"io"
	"strings"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config controls log level + output format.
type Config struct {
	Level  string // "debug" | "info" | "warn" | "error"
	Format string // "json" (prod) | "console" (dev)
}

// New returns a configured *zap.Logger writing to stderr.
func New(cfg Config) (*zap.Logger, error) {
	lvl, err := zapcore.ParseLevel(cfg.Level)
	if err != nil {
		lvl = zapcore.InfoLevel
	}
	zcfg := zap.NewProductionConfig()
	if cfg.Format == "console" {
		zcfg = zap.NewDevelopmentConfig()
	}
	zcfg.Level = zap.NewAtomicLevelAt(lvl)
	zcfg.DisableStacktrace = false
	return zcfg.Build()
}

// newTestable lets tests inject a buffer writer.
func newTestable(w io.Writer, lvl zapcore.Level) *zap.Logger {
	enc := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewCore(enc, zapcore.AddSync(w), lvl)
	return zap.New(core)
}

// sensitiveKeys are redacted from any structured field map before logging.
var sensitiveKeys = []string{"password", "passwd", "token", "secret", "authorization", "cookie", "api_key"}

// Sanitize walks a map[string]string and replaces values for sensitive keys.
// Extend to map[string]any in M2 when handler logging touches richer payloads.
func Sanitize(v any) any {
	m, ok := v.(map[string]string)
	if !ok {
		return v
	}
	out := make(map[string]string, len(m))
	for k, val := range m {
		lk := strings.ToLower(k)
		redacted := false
		for _, s := range sensitiveKeys {
			if strings.Contains(lk, s) {
				out[k] = "[REDACTED]"
				redacted = true
				break
			}
		}
		if !redacted {
			out[k] = val
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests to verify pass**

```bash
go test ./internal/infrastructure/logger/... -v
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add server/internal/infrastructure/logger/
git commit -m "feat(logger): zap config + sensitive-key sanitization"
```

---

### Task 11: infrastructure/metrics + health

**Files:**
- Create: `server/internal/infrastructure/metrics/metrics.go`
- Create: `server/internal/infrastructure/health/registry.go`
- Create: `server/internal/infrastructure/health/registry_test.go`

- [ ] **Step 1: Add Prometheus dependency**

```bash
go get github.com/prometheus/client_golang@v1.20.5
```

- [ ] **Step 2: Write failing test for health registry**

```go
// internal/infrastructure/health/registry_test.go
package health

import (
	"context"
	"errors"
	"testing"
)

func TestRegistry_AllHealthy(t *testing.T) {
	r := NewRegistry()
	r.Register(Check{Name: "pg", Critical: true, Check: func(ctx context.Context) error { return nil }})
	r.Register(Check{Name: "redis", Critical: false, Check: func(ctx context.Context) error { return nil }})

	report := r.Report(context.Background())
	if !report.Ready {
		t.Fatalf("expected ready=true")
	}
	if !report.Live {
		t.Fatalf("expected live=true")
	}
}

func TestRegistry_CriticalDown(t *testing.T) {
	r := NewRegistry()
	r.Register(Check{Name: "pg", Critical: true, Check: func(ctx context.Context) error { return errors.New("down") }})
	report := r.Report(context.Background())
	if report.Ready {
		t.Fatalf("expected ready=false when critical down")
	}
	if !report.Live {
		t.Fatalf("expected live=true regardless")
	}
}

func TestRegistry_NoncriticalDown(t *testing.T) {
	r := NewRegistry()
	r.Register(Check{Name: "minio", Critical: false, Check: func(ctx context.Context) error { return errors.New("down") }})
	report := r.Report(context.Background())
	if !report.Ready {
		t.Fatalf("expected ready=true when noncritical down")
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

```bash
go test ./internal/infrastructure/health/...
```

Expected: FAIL.

- [ ] **Step 4: Implement health registry**

```go
// internal/infrastructure/health/registry.go
package health

import (
	"context"
	"sync"
	"time"
)

// Check is one dependency probe.
type Check struct {
	Name     string
	Critical bool // true → if fails, readyz returns 503
	Check    func(ctx context.Context) error
}

// Report is the aggregated state returned by /healthz handler.
type Report struct {
	Live    bool                  `json:"live"`
	Ready   bool                  `json:"ready"`
	Details map[string]CheckState `json:"details"`
}

type CheckState struct {
	OK    bool   `json:"ok"`
	Err   string `json:"error,omitempty"`
	MS    int64  `json:"latency_ms"`
}

type Registry struct {
	mu     sync.RWMutex
	checks []Check
}

func NewRegistry() *Registry { return &Registry{} }

func (r *Registry) Register(c Check) {
	r.mu.Lock()
	r.checks = append(r.checks, c)
	r.mu.Unlock()
}

// Report runs every registered check sequentially (M1 simplicity).
// M2 may parallelize if cumulative latency exceeds 100ms.
func (r *Registry) Report(ctx context.Context) Report {
	r.mu.RLock()
	checks := append([]Check(nil), r.checks...)
	r.mu.RUnlock()

	rep := Report{Live: true, Ready: true, Details: make(map[string]CheckState)}
	for _, c := range checks {
		start := time.Now()
		err := c.Check(ctx)
		state := CheckState{OK: err == nil, MS: time.Since(start).Milliseconds()}
		if err != nil {
			state.Err = err.Error()
			if c.Critical {
				rep.Ready = false
			}
		}
		rep.Details[c.Name] = state
	}
	return rep
}
```

- [ ] **Step 5: Implement metrics**

```go
// internal/infrastructure/metrics/metrics.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Registry returns a *prometheus.Registry seeded with default collectors.
// app.go wires this into the Gin /metrics handler.
func Registry() *prometheus.Registry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(prometheus.NewGoCollector())
	reg.MustRegister(prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}))
	return reg
}

// HTTPDuration is the histogram for request latency by route + status.
var HTTPDuration = promauto.NewHistogramVec(
	prometheus.HistogramOpts{
		Name:    "iop_http_request_duration_seconds",
		Help:    "HTTP request latency.",
		Buckets: prometheus.DefBuckets,
	},
	[]string{"route", "method", "status"},
)
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/infrastructure/... -v
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add server/internal/infrastructure/health/ server/internal/infrastructure/metrics/ server/go.mod server/go.sum
git commit -m "feat(infra): health registry (livez/readyz semantics) + Prometheus base"
```

---

### Task 12: middleware — RequestID + Recover

**Files:**
- Create: `server/internal/interface/middleware/request_id.go`
- Create: `server/internal/interface/middleware/recover.go`
- Create: `server/internal/interface/middleware/request_id_test.go`
- Create: `server/internal/interface/middleware/recover_test.go`

- [ ] **Step 1: Add Gin dependency**

```bash
go get github.com/gin-gonic/gin@v1.10.0
```

- [ ] **Step 2: Write failing test for RequestID**

```go
// internal/interface/middleware/request_id_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
)

func TestRequestID_GeneratesIfMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) {
		tid := kernel.TraceIDFromContext(c.Request.Context())
		if tid == "" {
			t.Errorf("expected non-empty trace id in ctx")
		}
		c.String(200, tid)
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") == "" {
		t.Fatalf("expected X-Request-Id response header")
	}
	if w.Body.String() == "" {
		t.Fatalf("expected handler to see trace id")
	}
}

func TestRequestID_PreservesIncoming(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RequestID())
	r.GET("/", func(c *gin.Context) { c.Status(200) })
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Request-Id", "client-trace-xyz")
	r.ServeHTTP(w, req)
	if w.Header().Get("X-Request-Id") != "client-trace-xyz" {
		t.Fatalf("expected echo of client trace id")
	}
}
```

- [ ] **Step 3: Write failing test for Recover**

```go
// internal/interface/middleware/recover_test.go
package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestRecover_CatchesPanic(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(Recover(zap.NewNop()))
	r.GET("/boom", func(c *gin.Context) {
		panic("kaboom")
	})
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}
```

- [ ] **Step 4: Run tests to verify they fail**

```bash
go test ./internal/interface/middleware/...
```

Expected: FAIL.

- [ ] **Step 5: Implement RequestID**

```go
// internal/interface/middleware/request_id.go
package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
)

const headerRequestID = "X-Request-Id"

// RequestID echoes a client-provided X-Request-Id or generates a UUID v7 if missing.
// The id is attached to the request context (kernel.TraceIDFromContext) and the
// response headers. Downstream services and logs should propagate it verbatim.
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		tid := c.GetHeader(headerRequestID)
		if tid == "" {
			tid = string(kernel.NewID())
		}
		c.Header(headerRequestID, tid)
		ctx := kernel.WithTraceID(c.Request.Context(), tid)
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}
```

- [ ] **Step 6: Implement Recover**

```go
// internal/interface/middleware/recover.go
package middleware

import (
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
	"go.uber.org/zap"
)

// Recover catches panics, logs with trace id + stack, returns 500 JSON.
// Always installed BEFORE any business middleware so panics in those are caught too.
func Recover(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if r := recover(); r != nil {
				traceID := kernel.TraceIDFromContext(c.Request.Context())
				logger.Error("panic recovered",
					zap.String("trace_id", traceID),
					zap.String("path", c.Request.URL.Path),
					zap.Any("panic", r),
					zap.ByteString("stack", debug.Stack()),
				)
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":     "internal.panic",
						"message":  "服务器内部错误",
						"trace_id": traceID,
					},
				})
			}
		}()
		c.Next()
	}
}
```

- [ ] **Step 7: Run tests to verify pass**

```bash
go test ./internal/interface/middleware/... -v
```

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add server/internal/interface/middleware/ server/go.mod server/go.sum
git commit -m "feat(middleware): RequestID + Recover with trace id propagation"
```

---

### Task 13: middleware — Logger + CORS

**Files:**
- Create: `server/internal/interface/middleware/logger.go`
- Create: `server/internal/interface/middleware/cors.go`

- [ ] **Step 1: Implement Logger middleware**

```go
// internal/interface/middleware/logger.go
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/kernel"
	"go.uber.org/zap"
)

// Logger logs each request with trace + tenant + member + duration.
// Skips /livez and /healthz to keep log volume low.
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if path == "/livez" || path == "/healthz" || path == "/metrics" {
			c.Next()
			return
		}
		start := time.Now()
		c.Next()
		dur := time.Since(start)

		ctx := c.Request.Context()
		tid := kernel.TraceIDFromContext(ctx)
		fields := []zap.Field{
			zap.String("trace_id", tid),
			zap.String("method", c.Request.Method),
			zap.String("path", path),
			zap.Int("status", c.Writer.Status()),
			zap.Int64("duration_ms", dur.Milliseconds()),
			zap.String("remote", c.ClientIP()),
		}
		if tenantID, ok := kernel.TenantIDFromContext(ctx); ok {
			fields = append(fields, zap.String("tenant_id", string(tenantID)))
		}
		if memberID, ok := kernel.MemberIDFromContext(ctx); ok {
			fields = append(fields, zap.String("member_id", string(memberID)))
		}
		if len(c.Errors) > 0 {
			fields = append(fields, zap.Strings("errors", c.Errors.Errors()))
		}
		switch {
		case c.Writer.Status() >= 500:
			logger.Error("http", fields...)
		case c.Writer.Status() >= 400:
			logger.Warn("http", fields...)
		default:
			logger.Info("http", fields...)
		}
	}
}
```

- [ ] **Step 2: Implement CORS middleware**

```go
// internal/interface/middleware/cors.go
package middleware

import "github.com/gin-gonic/gin"

// CORS handles preflight + Access-Control-* for a configured set of origins.
// M1 default: allow any origin for dev. M2+ wires AllowedOrigins from configs.
func CORS(allowedOrigins []string) gin.HandlerFunc {
	allowAll := len(allowedOrigins) == 1 && allowedOrigins[0] == "*"
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		if allowAll {
			c.Header("Access-Control-Allow-Origin", "*")
		} else if origin != "" && allowed[origin] {
			c.Header("Access-Control-Allow-Origin", origin)
			c.Header("Vary", "Origin")
		}
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization,X-Request-Id,Idempotency-Key")
		c.Header("Access-Control-Expose-Headers", "X-Request-Id")
		c.Header("Access-Control-Max-Age", "86400")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}
```

- [ ] **Step 3: Smoke-test compilation**

```bash
go build ./internal/interface/middleware/...
```

Expected: build succeeds.

- [ ] **Step 4: Commit**

```bash
git add server/internal/interface/middleware/logger.go server/internal/interface/middleware/cors.go
git commit -m "feat(middleware): Logger (skip-healthz) + CORS (dev: allow-all)"
```

---

### Task 14: apiresp + Gin server.go

**Files:**
- Create: `server/internal/interface/apiresp/response.go`
- Create: `server/internal/interface/apiresp/response_test.go`
- Create: `server/internal/interface/server.go`

- [ ] **Step 1: Write failing test for apiresp**

```go
// internal/interface/apiresp/response_test.go
package apiresp

import (
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/errors"
)

func TestOK_WrapsData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	OK(c, map[string]string{"hello": "world"})

	var body struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body.Code != 0 {
		t.Fatalf("expected code=0, got %d", body.Code)
	}
	if body.Data["hello"] != "world" {
		t.Fatalf("data not wrapped")
	}
}

func TestFail_MapsKindToStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	Fail(c, errors.New(errors.KindNotFound, "iam.user.not_found", "用户不存在"))
	if w.Code != 404 {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/interface/apiresp/...
```

Expected: FAIL.

- [ ] **Step 3: Implement apiresp**

```go
// internal/interface/apiresp/response.go
package apiresp

import (
	stderrors "errors"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/shared/errors"
	"github.com/leo/iop/server/internal/shared/kernel"
)

// envelope is the uniform response shape:
// success:  {"code":0, "data": ..., "trace_id": "..."}
// failure:  {"code":-1, "error": {"code","message","kind"}, "trace_id":"..."}
type envelope struct {
	Code    int    `json:"code"`
	Data    any    `json:"data,omitempty"`
	Error   *errOut `json:"error,omitempty"`
	TraceID string `json:"trace_id"`
}

type errOut struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Kind    string `json:"kind"`
}

// OK writes 200 with the wrapped data.
func OK(c *gin.Context, data any) {
	c.JSON(200, envelope{
		Code:    0,
		Data:    data,
		TraceID: kernel.TraceIDFromContext(c.Request.Context()),
	})
}

// Created writes 201 with the wrapped data.
func Created(c *gin.Context, data any) {
	c.JSON(201, envelope{
		Code:    0,
		Data:    data,
		TraceID: kernel.TraceIDFromContext(c.Request.Context()),
	})
}

// Fail writes the error using Kind → HTTP status mapping.
// If err is not *errors.Error, treats as KindInternal.
func Fail(c *gin.Context, err error) {
	var e *errors.Error
	if !stderrors.As(err, &e) {
		e = errors.Wrap(errors.KindInternal, "internal.unwrapped", "未分类错误", err)
	}
	c.AbortWithStatusJSON(e.Kind.HTTPStatus(), envelope{
		Code: -1,
		Error: &errOut{
			Code:    e.Code,
			Message: e.Message,
			Kind:    e.Kind.String(),
		},
		TraceID: kernel.TraceIDFromContext(c.Request.Context()),
	})
}
```

- [ ] **Step 4: Implement server.go (Gin wiring + livez/readyz/version/metrics)**

```go
// internal/interface/server.go
package iface

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/infrastructure/health"
	"github.com/leo/iop/server/internal/infrastructure/metrics"
	"github.com/leo/iop/server/internal/interface/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Version is set at build time via -ldflags "-X main.Version=...".
var Version = "dev"

// Config carries runtime knobs from configs/dev.yaml via app DI.
type Config struct {
	AllowedOrigins []string
}

// New wires the Gin engine with M1 middleware chain + system endpoints.
// services/* will register their own routes via the *gin.Engine later.
func New(cfg Config, logger *zap.Logger, healthReg *health.Registry, metricReg interface {
	Register(prometheus.Collector) error
}) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	// IMPORTANT: Recover must be first so panics inside other middleware are caught.
	r.Use(middleware.Recover(logger))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	// System endpoints (no auth, no tenant scope).
	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		rep := healthReg.Report(c.Request.Context())
		status := http.StatusOK
		if !rep.Ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"ready": rep.Ready, "live": rep.Live})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, healthReg.Report(c.Request.Context()))
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})))

	return r
}

// Run wraps http.Server with graceful shutdown.
func Run(ctx context.Context, addr string, h http.Handler, logger *zap.Logger) error {
	srv := &http.Server{Addr: addr, Handler: h}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}
```

NOTE: The import of `prometheus` in the signature is awkward. Let me simplify — remove the metricReg parameter, since metrics.Registry() returns the registry directly:

Actually the simpler clean version of `New`:

```go
// internal/interface/server.go (revised final)
package iface

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/infrastructure/health"
	"github.com/leo/iop/server/internal/infrastructure/metrics"
	"github.com/leo/iop/server/internal/interface/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

var Version = "dev"

type Config struct {
	AllowedOrigins []string
}

func New(cfg Config, logger *zap.Logger, healthReg *health.Registry) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()

	r.Use(middleware.Recover(logger))
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))

	r.GET("/livez", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "live"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		rep := healthReg.Report(c.Request.Context())
		status := http.StatusOK
		if !rep.Ready {
			status = http.StatusServiceUnavailable
		}
		c.JSON(status, gin.H{"ready": rep.Ready, "live": rep.Live})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, healthReg.Report(c.Request.Context()))
	})
	r.GET("/version", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"version": Version})
	})
	r.GET("/metrics", gin.WrapH(promhttp.HandlerFor(metrics.Registry(), promhttp.HandlerOpts{})))

	return r
}

func Run(ctx context.Context, addr string, h http.Handler, logger *zap.Logger) error {
	srv := &http.Server{Addr: addr, Handler: h}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("http server listening", zap.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return srv.Shutdown(shutCtx)
	}
}
```

- [ ] **Step 5: Run tests to verify pass**

```bash
go test ./internal/interface/... -v
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add server/internal/interface/
git commit -m "feat(interface): apiresp envelope + Gin server with livez/readyz/version/metrics"
```

---

### Task 15: cmd/server main + config loader + app wiring

**Files:**
- Create: `server/internal/config/config.go`
- Create: `server/internal/app/app.go`
- Create: `server/cmd/server/main.go`
- Create: `server/configs/dev.yaml`

- [ ] **Step 1: Add viper dependency**

```bash
go get github.com/spf13/viper@v1.20.1
```

- [ ] **Step 2: Implement config loader**

```go
// internal/config/config.go
package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config is the root config object. Tagged for viper unmarshaling.
type Config struct {
	Env    string `mapstructure:"env"`
	Server struct {
		Addr           string   `mapstructure:"addr"`
		AllowedOrigins []string `mapstructure:"allowed_origins"`
	} `mapstructure:"server"`
	DB struct {
		DSN string `mapstructure:"dsn"`
	} `mapstructure:"db"`
	Redis struct {
		Addr string `mapstructure:"addr"`
	} `mapstructure:"redis"`
	MinIO struct {
		Endpoint  string `mapstructure:"endpoint"`
		AccessKey string `mapstructure:"access_key"`
		SecretKey string `mapstructure:"secret_key"`
	} `mapstructure:"minio"`
	Logger struct {
		Level  string `mapstructure:"level"`
		Format string `mapstructure:"format"`
	} `mapstructure:"logger"`
}

// Load reads configs/<env>.yaml + env overrides. Env vars: IOP_<SECTION>_<KEY>.
// Example: IOP_DB_DSN overrides db.dsn.
func Load() (*Config, error) {
	v := viper.New()
	v.SetConfigType("yaml")
	v.SetConfigName("dev")
	v.AddConfigPath("./configs")
	v.AddConfigPath("./server/configs")

	v.SetEnvPrefix("IOP")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	v.SetDefault("env", "dev")
	v.SetDefault("server.addr", ":8080")
	v.SetDefault("server.allowed_origins", []string{"*"})
	v.SetDefault("logger.level", "info")
	v.SetDefault("logger.format", "console")

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("config: %w", err)
		}
		// Missing file is fine if env vars cover required fields.
	}

	var c Config
	if err := v.Unmarshal(&c); err != nil {
		return nil, fmt.Errorf("config unmarshal: %w", err)
	}
	return &c, nil
}
```

- [ ] **Step 3: Implement app.go wiring**

```go
// internal/app/app.go
package app

import (
	"context"
	"fmt"

	"github.com/leo/iop/server/internal/config"
	"github.com/leo/iop/server/internal/infrastructure/health"
	pginfra "github.com/leo/iop/server/internal/infrastructure/pg"
	loggerinfra "github.com/leo/iop/server/internal/infrastructure/logger"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/tenantdb"
	iface "github.com/leo/iop/server/internal/interface"
	"go.uber.org/zap"
)

// App holds every wired component. Owned by main(), shut down on signal.
type App struct {
	Cfg      *config.Config
	Logger   *zap.Logger
	Pool     *pgxpool.Pool   // for direct close
	Platform *tenantdb.PlatformDB
	Tenant   *tenantdb.TenantDB
	Bus      *eventbus.InprocBus
	Health   *health.Registry
}

// Build wires components in dependency order. Returns (*App, cleanup, error).
func Build(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	logger, err := loggerinfra.New(loggerinfra.Config{Level: cfg.Logger.Level, Format: cfg.Logger.Format})
	if err != nil {
		return nil, nil, fmt.Errorf("logger: %w", err)
	}

	pool, err := pginfra.NewPool(ctx, pginfra.Config{DSN: cfg.DB.DSN})
	if err != nil {
		return nil, nil, fmt.Errorf("pg: %w", err)
	}

	bus := eventbus.NewInprocBus(4).WithLogger(logger)
	bus.Start()

	healthReg := health.NewRegistry()
	healthReg.Register(health.Check{
		Name: "pg", Critical: true,
		Check: func(c context.Context) error { return pool.Ping(c) },
	})

	app := &App{
		Cfg:      cfg,
		Logger:   logger,
		Pool:     pool,
		Platform: tenantdb.NewPlatformDB(pool),
		Tenant:   tenantdb.NewTenantDB(pool),
		Bus:      bus,
		Health:   healthReg,
	}

	cleanup := func() {
		_ = bus.Close()
		pool.Close()
		_ = logger.Sync()
	}
	return app, cleanup, nil
}

// Engine returns the wired Gin engine.
func (a *App) Engine() *gin.Engine {
	return iface.New(iface.Config{AllowedOrigins: a.Cfg.Server.AllowedOrigins}, a.Logger, a.Health)
}
```

NOTE: imports above include `pgxpool` and `gin` — add those imports. Final clean file:

```go
// internal/app/app.go (final)
package app

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leo/iop/server/internal/config"
	"github.com/leo/iop/server/internal/infrastructure/health"
	loggerinfra "github.com/leo/iop/server/internal/infrastructure/logger"
	pginfra "github.com/leo/iop/server/internal/infrastructure/pg"
	iface "github.com/leo/iop/server/internal/interface"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/tenantdb"
	"go.uber.org/zap"
)

type App struct {
	Cfg      *config.Config
	Logger   *zap.Logger
	Pool     *pgxpool.Pool
	Platform *tenantdb.PlatformDB
	Tenant   *tenantdb.TenantDB
	Bus      *eventbus.InprocBus
	Health   *health.Registry
}

func Build(ctx context.Context, cfg *config.Config) (*App, func(), error) {
	logger, err := loggerinfra.New(loggerinfra.Config{Level: cfg.Logger.Level, Format: cfg.Logger.Format})
	if err != nil {
		return nil, nil, fmt.Errorf("logger: %w", err)
	}
	pool, err := pginfra.NewPool(ctx, pginfra.Config{DSN: cfg.DB.DSN})
	if err != nil {
		return nil, nil, fmt.Errorf("pg: %w", err)
	}
	bus := eventbus.NewInprocBus(4).WithLogger(logger)
	bus.Start()
	healthReg := health.NewRegistry()
	healthReg.Register(health.Check{
		Name: "pg", Critical: true,
		Check: func(c context.Context) error { return pool.Ping(c) },
	})
	app := &App{
		Cfg: cfg, Logger: logger, Pool: pool,
		Platform: tenantdb.NewPlatformDB(pool),
		Tenant:   tenantdb.NewTenantDB(pool),
		Bus:      bus, Health: healthReg,
	}
	cleanup := func() {
		_ = bus.Close()
		pool.Close()
		_ = logger.Sync()
	}
	return app, cleanup, nil
}

func (a *App) Engine() *gin.Engine {
	return iface.New(iface.Config{AllowedOrigins: a.Cfg.Server.AllowedOrigins}, a.Logger, a.Health)
}
```

- [ ] **Step 4: Implement cmd/server/main.go**

```go
// cmd/server/main.go
package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/leo/iop/server/internal/app"
	"github.com/leo/iop/server/internal/config"
	iface "github.com/leo/iop/server/internal/interface"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config load: %v", err)
	}

	a, cleanup, err := app.Build(ctx, cfg)
	if err != nil {
		log.Fatalf("app build: %v", err)
	}
	defer cleanup()

	if err := iface.Run(ctx, cfg.Server.Addr, a.Engine(), a.Logger); err != nil {
		log.Fatalf("server: %v", err)
	}
}
```

- [ ] **Step 5: Create configs/dev.yaml**

```yaml
env: dev

server:
  addr: ":8080"
  allowed_origins: ["*"]

db:
  dsn: "postgres://iop:iop_dev@localhost:5432/iop?sslmode=disable"

redis:
  addr: "localhost:6379"

minio:
  endpoint: "localhost:9000"
  access_key: "iop_dev"
  secret_key: "iop_dev_password"

logger:
  level: "debug"
  format: "console"
```

- [ ] **Step 6: Build and run**

```bash
cd /Users/leo/Documents/iop/server
make build
./bin/server
```

Expected: server starts, logs "http server listening addr=:8080".

- [ ] **Step 7: Smoke test endpoints**

In another terminal:

```bash
curl http://localhost:8080/livez
# {"status":"live"}
curl http://localhost:8080/readyz
# {"ready":true,"live":true}
curl http://localhost:8080/version
# {"version":"dev"}
curl http://localhost:8080/healthz
# {"live":true,"ready":true,"details":{"pg":{"ok":true,"latency_ms":N}}}
curl http://localhost:8080/metrics | head -20
# Prometheus output
```

Expected: all return 200 with sensible JSON.

- [ ] **Step 8: Test panic recovery**

Temporarily add to `internal/interface/server.go` after `/version` route:

```go
r.GET("/boom", func(c *gin.Context) { panic("test") })
```

Rebuild, hit `/boom`, expect 500 + log line with stack trace. Then **remove the line and recommit**.

```bash
curl -i http://localhost:8080/boom
# HTTP/1.1 500 Internal Server Error
# {"error":{"code":"internal.panic","message":"服务器内部错误","trace_id":"..."}}
```

- [ ] **Step 9: Commit**

```bash
git add server/internal/config/ server/internal/app/ server/cmd/server/ server/configs/
git commit -m "feat(app): wire config + DI + cmd/server with livez/readyz/version/metrics"
```

---

## Phase 4: Migrations + services skeleton (Week 3)

### Task 16: migrations/public/000001 + cmd/migrate

**Files:**
- Create: `server/migrations/public/000001_init.up.sql`
- Create: `server/migrations/public/000001_init.down.sql`
- Create: `server/cmd/migrate/main.go`

- [ ] **Step 1: Add tern dependency**

```bash
go get github.com/jackc/tern/v2/migrate@v2.2.4
```

- [ ] **Step 2: Write initial migration**

```sql
-- migrations/public/000001_init.up.sql
-- v3.1 base tables. M2 will add tenant / platform_user / membership.

CREATE TABLE IF NOT EXISTS public.migration_history (
    id            UUID PRIMARY KEY,
    scope         TEXT NOT NULL,          -- 'public' or tenant slug
    migration_id  TEXT NOT NULL,          -- '000001_init'
    applied_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    checksum      TEXT NOT NULL,
    UNIQUE (scope, migration_id)
);

CREATE INDEX IF NOT EXISTS migration_history_applied_at_idx
    ON public.migration_history (applied_at DESC);

-- Marker row so cmd/migrate can verify a fresh DB.
INSERT INTO public.migration_history (id, scope, migration_id, checksum)
VALUES (
    gen_random_uuid(),
    'public',
    '000001_init',
    'placeholder-m1'
)
ON CONFLICT DO NOTHING;
```

- [ ] **Step 3: Down migration**

```sql
-- migrations/public/000001_init.down.sql
DROP TABLE IF EXISTS public.migration_history;
```

- [ ] **Step 4: Implement cmd/migrate using tern**

```go
// cmd/migrate/main.go
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/tern/v2/migrate"
	"github.com/leo/iop/server/internal/config"
)

func main() {
	dir := flag.String("dir", "./migrations/public", "migrations directory")
	flag.Parse()

	cmd := flag.Arg(0)
	if cmd == "" {
		cmd = "up"
	}

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, cfg.DB.DSN)
	if err != nil {
		log.Fatalf("connect: %v", err)
	}
	defer conn.Close(ctx)

	m, err := migrate.NewMigrator(ctx, conn, "public.schema_version")
	if err != nil {
		log.Fatalf("migrator: %v", err)
	}
	if err := m.LoadMigrations(os.DirFS(*dir)); err != nil {
		log.Fatalf("load: %v", err)
	}

	switch cmd {
	case "up":
		if err := m.Migrate(ctx); err != nil {
			log.Fatalf("up: %v", err)
		}
		log.Printf("migrated to version %d", m.Migrations[len(m.Migrations)-1].Sequence)
	case "down":
		// tern.MigrateTo(ctx, version) — keep simple: one step down
		cur, _ := m.GetCurrentVersion(ctx)
		if cur > 0 {
			if err := m.MigrateTo(ctx, cur-1); err != nil {
				log.Fatalf("down: %v", err)
			}
		}
		log.Printf("rolled back one step")
	case "status":
		ver, _ := m.GetCurrentVersion(ctx)
		log.Printf("current version: %d", ver)
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}
```

- [ ] **Step 5: Build and run migration**

```bash
cd /Users/leo/Documents/iop/server
make build
./bin/migrate up
```

Expected: `migrated to version 1`. Verify in PG:

```bash
cd /Users/leo/Documents/iop/deployments
docker compose exec db psql -U iop -d iop -c "SELECT scope, migration_id FROM public.migration_history;"
```

Expected: 1 row with `public | 000001_init`.

- [ ] **Step 6: Commit**

```bash
git add server/migrations/ server/cmd/migrate/ server/go.mod server/go.sum
git commit -m "feat(migrate): cmd/migrate via tern + 000001_init (migration_history)"
```

---

### Task 17: services/dictionary skeleton

**Files:**
- Create: `server/internal/services/dictionary/dict.go`
- Create: `server/internal/services/dictionary/service.go`
- Create: `server/internal/services/dictionary/http.go`
- Create: `server/internal/services/dictionary/service_test.go`

- [ ] **Step 1: Write failing test (in-memory M1)**

```go
// internal/services/dictionary/service_test.go
package dictionary

import (
	"context"
	"testing"
)

func TestLookup_ReturnsSeededItems(t *testing.T) {
	svc := NewService(MemoryRepo(map[string][]Item{
		"plan_level": {
			{Code: "year", Name: "年度", SortOrder: 1, Active: true},
			{Code: "month", Name: "月度", SortOrder: 2, Active: true},
		},
	}))
	items, err := svc.Lookup(context.Background(), "plan_level")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}

func TestLookup_UnknownType(t *testing.T) {
	svc := NewService(MemoryRepo(map[string][]Item{}))
	_, err := svc.Lookup(context.Background(), "nope")
	if err == nil {
		t.Fatalf("expected error for unknown type")
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/services/dictionary/...
```

Expected: FAIL.

- [ ] **Step 3: Implement dict types**

```go
// internal/services/dictionary/dict.go
package dictionary

import "context"

// Item is one entry in a dictionary type.
type Item struct {
	TypeCode  string `json:"type_code"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
	Active    bool   `json:"active"`
}

// Repository is the persistence contract. M1: in-memory MemoryRepo.
// M3: replaced by pgRepo with Redis cache layer.
type Repository interface {
	List(ctx context.Context, typeCode string) ([]Item, error)
}

// MemoryRepo is an in-memory Repository used by M1 tests and dev seeding.
func MemoryRepo(seed map[string][]Item) Repository {
	return memRepo{data: seed}
}

type memRepo struct{ data map[string][]Item }

func (m memRepo) List(_ context.Context, typeCode string) ([]Item, error) {
	if items, ok := m.data[typeCode]; ok {
		return items, nil
	}
	return nil, ErrTypeNotFound
}
```

- [ ] **Step 4: Implement service.go**

```go
// internal/services/dictionary/service.go
package dictionary

import (
	"context"

	"github.com/leo/iop/server/internal/shared/errors"
)

// ErrTypeNotFound is returned when a type_code has no items.
var ErrTypeNotFound = errors.New(errors.KindNotFound, "dictionary.type.not_found", "字典类型不存在")

// Service is the public API of the dictionary service.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service { return &Service{repo: repo} }

// Lookup returns active items for a type_code.
// M3 will add Redis caching + tenant override merging.
func (s *Service) Lookup(ctx context.Context, typeCode string) ([]Item, error) {
	items, err := s.repo.List(ctx, typeCode)
	if err != nil {
		return nil, err
	}
	out := make([]Item, 0, len(items))
	for _, it := range items {
		if it.Active {
			out = append(out, it)
		}
	}
	return out, nil
}
```

- [ ] **Step 5: Implement http.go (route registration)**

```go
// internal/services/dictionary/http.go
package dictionary

import (
	"github.com/gin-gonic/gin"
	"github.com/leo/iop/server/internal/interface/apiresp"
)

// RegisterRoutes wires GET /dict/:typeCode to the engine.
// Called by app/wiring.go after engine creation.
func RegisterRoutes(r *gin.RouterGroup, svc *Service) {
	r.GET("/dict/:typeCode", func(c *gin.Context) {
		items, err := svc.Lookup(c.Request.Context(), c.Param("typeCode"))
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.OK(c, gin.H{"items": items})
	})
}
```

- [ ] **Step 6: Run tests**

```bash
go test ./internal/services/dictionary/... -v
```

Expected: PASS.

- [ ] **Step 7: Wire into app.Engine()**

Modify `internal/app/app.go` to seed dictionary memory data and register routes:

```go
// Add to internal/app/app.go imports:
import "github.com/leo/iop/server/internal/services/dictionary"

// Add to App struct:
type App struct {
	// ...existing...
	Dictionary *dictionary.Service
}

// Add to Build() after healthReg setup:
dictSvc := dictionary.NewService(dictionary.MemoryRepo(map[string][]dictionary.Item{
	"plan_level": {
		{Code: "year", Name: "年度", SortOrder: 1, Active: true},
		{Code: "half_year", Name: "半年", SortOrder: 2, Active: true},
		{Code: "month", Name: "月度", SortOrder: 3, Active: true},
		{Code: "week", Name: "周度", SortOrder: 4, Active: true},
	},
	"report_type": {
		{Code: "daily", Name: "日报", SortOrder: 1, Active: true},
		{Code: "weekly", Name: "周报", SortOrder: 2, Active: true},
	},
}))
app.Dictionary = dictSvc

// Modify Engine() to wire routes:
func (a *App) Engine() *gin.Engine {
	r := iface.New(iface.Config{AllowedOrigins: a.Cfg.Server.AllowedOrigins}, a.Logger, a.Health)
	api := r.Group("/api")
	dictionary.RegisterRoutes(api, a.Dictionary)
	return r
}
```

- [ ] **Step 8: Verify HTTP endpoint**

```bash
make build && ./bin/server &
sleep 1
curl http://localhost:8080/api/dict/plan_level
# {"code":0, "data":{"items":[...]}, "trace_id":"..."}
curl http://localhost:8080/api/dict/nope
# 404 with error envelope
kill %1
```

- [ ] **Step 9: Commit**

```bash
git add server/internal/services/dictionary/ server/internal/app/app.go
git commit -m "feat(dict): in-memory dictionary service + GET /api/dict/:type"
```

---

### Task 18: services/localization skeleton

**Files:**
- Create: `server/internal/services/localization/translate.go`
- Create: `server/internal/services/localization/yaml_bundle.go`
- Create: `server/internal/services/localization/translate_test.go`
- Create: `server/configs/i18n/zh-CN.yaml`

- [ ] **Step 1: Write failing test**

```go
// internal/services/localization/translate_test.go
package localization

import (
	"context"
	"testing"
)

func TestT_KnownKey(t *testing.T) {
	svc := NewService(MapBundle(map[string]map[string]string{
		"zh-CN": {"okr.plan.invalid_period": "时段不合法"},
	}), "zh-CN")
	if got := svc.T(context.Background(), "okr.plan.invalid_period"); got != "时段不合法" {
		t.Fatalf("got %q", got)
	}
}

func TestT_UnknownKey_ReturnsKey(t *testing.T) {
	svc := NewService(MapBundle(nil), "zh-CN")
	if got := svc.T(context.Background(), "missing.key"); got != "missing.key" {
		t.Fatalf("expected key fallback, got %q", got)
	}
}

func TestT_TemplateArgs(t *testing.T) {
	svc := NewService(MapBundle(map[string]map[string]string{
		"zh-CN": {"hello": "你好, {name}"},
	}), "zh-CN")
	got := svc.T(context.Background(), "hello", "name", "leo")
	if got != "你好, leo" {
		t.Fatalf("got %q", got)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

```bash
go test ./internal/services/localization/...
```

Expected: FAIL.

- [ ] **Step 3: Implement Bundle + Service**

```go
// internal/services/localization/translate.go
package localization

import (
	"context"
	"strings"
)

// Bundle is the i18n storage contract.
type Bundle interface {
	Lookup(locale, key string) (string, bool)
}

// MapBundle is an in-memory Bundle for tests.
func MapBundle(data map[string]map[string]string) Bundle { return mapBundle{data: data} }

type mapBundle struct{ data map[string]map[string]string }

func (m mapBundle) Lookup(locale, key string) (string, bool) {
	if locale := m.data[locale]; locale != nil {
		v, ok := locale[key]
		return v, ok
	}
	return "", false
}

// Service exposes T(ctx, key, args...). args is [k, v, k, v, ...].
type Service struct {
	bundle        Bundle
	defaultLocale string
}

func NewService(b Bundle, defaultLocale string) *Service {
	if defaultLocale == "" {
		defaultLocale = "zh-CN"
	}
	return &Service{bundle: b, defaultLocale: defaultLocale}
}

// T returns the translation for key, or key itself if missing.
// Args are [name, value, name, value...] pairs replacing {name} placeholders.
func (s *Service) T(ctx context.Context, key string, args ...string) string {
	locale := s.defaultLocale
	if v, ok := localeFromContext(ctx); ok {
		locale = v
	}
	tpl, ok := s.bundle.Lookup(locale, key)
	if !ok {
		return key
	}
	out := tpl
	for i := 0; i+1 < len(args); i += 2 {
		out = strings.ReplaceAll(out, "{"+args[i]+"}", args[i+1])
	}
	return out
}

// localeFromContext is populated by middleware in M2 (Accept-Language parsing).
// In M1 it always returns "".
type localeCtxKey struct{}

func WithLocale(ctx context.Context, locale string) context.Context {
	return context.WithValue(ctx, localeCtxKey{}, locale)
}

func localeFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(localeCtxKey{}).(string)
	return v, ok
}
```

- [ ] **Step 4: Implement YAML loader**

```go
// internal/services/localization/yaml_bundle.go
package localization

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// LoadYAMLBundle reads configs/i18n/*.yaml into a Bundle.
// Each filename = locale; values are flat key→string maps.
func LoadYAMLBundle(dir string) (Bundle, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("i18n: read dir %s: %w", dir, err)
	}
	data := make(map[string]map[string]string)
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		locale := e.Name()[:len(e.Name())-5]
		raw, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			return nil, err
		}
		m := map[string]string{}
		if err := yaml.Unmarshal(raw, &m); err != nil {
			return nil, fmt.Errorf("i18n: parse %s: %w", e.Name(), err)
		}
		data[locale] = m
	}
	return MapBundle(data), nil
}
```

- [ ] **Step 5: Add yaml dependency**

```bash
go get gopkg.in/yaml.v3@v3.0.1
```

- [ ] **Step 6: Seed configs/i18n/zh-CN.yaml**

```yaml
# Common
common.ok: "成功"
common.not_found: "未找到"
common.forbidden: "无权限"
common.unauthorized: "请先登录"

# Internal errors
internal.panic: "服务器内部错误"
internal.unwrapped: "未分类错误"

# Dictionary
dictionary.type.not_found: "字典类型不存在"

# Future: OKR keys land here in M4
```

- [ ] **Step 7: Run tests**

```bash
go test ./internal/services/localization/... -v
```

Expected: PASS.

- [ ] **Step 8: Wire into app.Engine()**

In `internal/app/app.go`, after dictSvc:

```go
bundle, _ := localization.LoadYAMLBundle("./configs/i18n") // best-effort; missing dir → MapBundle(nil)
if bundle == nil {
	bundle = localization.MapBundle(nil)
}
i18n := localization.NewService(bundle, "zh-CN")
app.I18n = i18n
```

Add `I18n *localization.Service` to App struct and import `"github.com/leo/iop/server/internal/services/localization"`.

- [ ] **Step 9: Commit**

```bash
git add server/internal/services/localization/ server/configs/i18n/ server/internal/app/app.go server/go.mod server/go.sum
git commit -m "feat(i18n): YAML bundle + T(ctx, key, args...) translator"
```

---

### Task 19: Initial OpenAPI + system endpoints documented

**Files:**
- Create: `server/api/openapi.yaml`
- Create: `scripts/openapi-gen.sh`

- [ ] **Step 1: Write minimal openapi.yaml covering current endpoints**

```yaml
openapi: 3.0.3
info:
  title: IOP API
  version: "0.1.0"
  description: |
    IOP 多租户 B 端基座 API. M1: 仅基础设施端点。
    M2 起加 Tenancy/IAM, M4 加 OKR.

servers:
  - url: http://localhost:8080
    description: Local dev

paths:
  /livez:
    get:
      summary: Liveness probe
      tags: [system]
      responses:
        "200":
          description: Process is alive
          content:
            application/json:
              schema:
                type: object
                properties: { status: { type: string, example: live } }

  /readyz:
    get:
      summary: Readiness probe (主路径依赖)
      tags: [system]
      responses:
        "200":
          description: Ready
          content:
            application/json:
              schema:
                type: object
                properties:
                  ready: { type: boolean }
                  live: { type: boolean }
        "503":
          description: Critical dependency down

  /healthz:
    get:
      summary: Detailed dependency status (internal use)
      tags: [system]
      responses:
        "200":
          description: Status of all registered checks

  /version:
    get:
      summary: Build version
      tags: [system]
      responses:
        "200":
          description: Version string

  /metrics:
    get:
      summary: Prometheus metrics
      tags: [system]
      responses:
        "200":
          description: Prometheus exposition format

  /api/dict/{typeCode}:
    get:
      summary: Lookup dictionary items
      tags: [dictionary]
      parameters:
        - name: typeCode
          in: path
          required: true
          schema: { type: string }
      responses:
        "200":
          description: Items list
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/Envelope"
        "404":
          description: Type not found

components:
  schemas:
    Envelope:
      type: object
      required: [code, trace_id]
      properties:
        code: { type: integer, example: 0 }
        data: { type: object, nullable: true }
        error:
          type: object
          nullable: true
          properties:
            code: { type: string }
            message: { type: string }
            kind: { type: string }
        trace_id: { type: string }
```

- [ ] **Step 2: Write scripts/openapi-gen.sh stub**

```bash
#!/usr/bin/env bash
# scripts/openapi-gen.sh
# Generates frontend TypeScript SDK from server/api/openapi.yaml.
# M1: stub. M3+ wires openapi-typescript-codegen.

set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
ROOT="$SCRIPT_DIR/.."

echo "[openapi-gen] Source: $ROOT/server/api/openapi.yaml"
echo "[openapi-gen] M1 stub — TODO M3: integrate openapi-typescript-codegen"
# Placeholder so users can call `make openapi-gen` without errors.
```

Make executable:

```bash
chmod +x scripts/openapi-gen.sh
```

- [ ] **Step 3: Validate openapi.yaml is well-formed**

```bash
docker run --rm -v $(pwd)/server/api:/spec -w /spec redocly/cli lint openapi.yaml
# or:
npx @redocly/cli lint server/api/openapi.yaml
```

Expected: 0 errors. (Warnings about missing examples are OK.)

- [ ] **Step 4: Commit**

```bash
git add server/api/openapi.yaml scripts/openapi-gen.sh
git commit -m "docs(api): OpenAPI 0.1 covering system + dict endpoints"
```

---

## Phase 5: Frontend skeleton (Week 3-4)

### Task 20: web/ Vite project scaffold

**Files:**
- Create: `web/package.json`
- Create: `web/vite.config.ts`
- Create: `web/tsconfig.json`
- Create: `web/tsconfig.node.json`
- Create: `web/index.html`
- Create: `web/src/main.ts`
- Create: `web/src/App.vue`
- Create: `web/src/env.d.ts`

Note: existing `iop/web/` from earlier scaffolding likely has files. Replace with v3.1-aligned single Vite project.

- [ ] **Step 1: Inspect existing web/ and back up if needed**

```bash
cd /Users/leo/Documents/iop
ls web/
# If existing files conflict, rename:
mv web web.bak
mkdir web
```

- [ ] **Step 2: Create package.json**

```json
{
  "name": "iop-web",
  "version": "0.1.0",
  "private": true,
  "type": "module",
  "scripts": {
    "dev": "vite",
    "build": "vue-tsc --noEmit && vite build",
    "preview": "vite preview",
    "test": "vitest run",
    "test:watch": "vitest",
    "lint": "eslint src/ --ext .ts,.vue"
  },
  "dependencies": {
    "vue": "^3.5.13",
    "vue-router": "^4.5.0",
    "pinia": "^2.3.0",
    "axios": "^1.7.9",
    "dayjs": "^1.11.13"
  },
  "devDependencies": {
    "@vitejs/plugin-vue": "^5.2.1",
    "@vue/test-utils": "^2.4.6",
    "happy-dom": "^15.11.7",
    "typescript": "~5.7.2",
    "vite": "^6.0.5",
    "vitest": "^2.1.8",
    "vue-tsc": "^2.2.0"
  }
}
```

- [ ] **Step 3: Create vite.config.ts**

```ts
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import { fileURLToPath, URL } from "node:url";

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      "@": fileURLToPath(new URL("./src", import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      "/api": {
        target: "http://localhost:8080",
        changeOrigin: true,
      },
    },
  },
});
```

- [ ] **Step 4: Create tsconfig.json + tsconfig.node.json**

```json
// tsconfig.json
{
  "compilerOptions": {
    "target": "ESNext",
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "strict": true,
    "jsx": "preserve",
    "esModuleInterop": true,
    "skipLibCheck": true,
    "lib": ["ESNext", "DOM"],
    "baseUrl": ".",
    "paths": { "@/*": ["src/*"] },
    "types": ["vite/client"]
  },
  "include": ["src/**/*.ts", "src/**/*.vue", "src/**/*.d.ts"],
  "references": [{ "path": "./tsconfig.node.json" }]
}
```

```json
// tsconfig.node.json
{
  "compilerOptions": {
    "module": "ESNext",
    "moduleResolution": "Bundler",
    "skipLibCheck": true
  },
  "include": ["vite.config.ts"]
}
```

- [ ] **Step 5: Create index.html**

```html
<!DOCTYPE html>
<html lang="zh-CN">
  <head>
    <meta charset="UTF-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0" />
    <title>IOP</title>
  </head>
  <body>
    <div id="app"></div>
    <script type="module" src="/src/main.ts"></script>
  </body>
</html>
```

- [ ] **Step 6: Create src/main.ts and App.vue**

```ts
// src/main.ts
import { createApp } from "vue";
import { createPinia } from "pinia";
import App from "./App.vue";
import router from "./router";
import "./styles/global.css";

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.mount("#app");
```

```vue
<!-- src/App.vue -->
<template>
  <router-view />
</template>
```

- [ ] **Step 7: Create env.d.ts**

```ts
// src/env.d.ts
/// <reference types="vite/client" />

declare module "*.vue" {
  import type { DefineComponent } from "vue";
  const component: DefineComponent<{}, {}, any>;
  export default component;
}
```

- [ ] **Step 8: Install + verify build**

```bash
cd /Users/leo/Documents/iop/web
npm install
npm run build
```

Expected: build succeeds (will fail at router import — Task 21 fixes).

- [ ] **Step 9: Commit (so far)**

```bash
git add web/package.json web/package-lock.json web/vite.config.ts web/tsconfig.json web/tsconfig.node.json web/index.html web/src/main.ts web/src/App.vue web/src/env.d.ts
git commit -m "chore(web): Vite scaffold (Vue 3 + Pinia + vue-router + axios)"
```

---

### Task 21: web/shell layout + router

**Files:**
- Create: `web/src/router/index.ts`
- Create: `web/src/shell/layout/AppLayout.vue`
- Create: `web/src/shell/layout/NavBar.vue`
- Create: `web/src/shell/workspace/WorkspaceHome.vue`
- Create: `web/src/styles/global.css`
- Create: `web/src/styles/tokens.css`

- [ ] **Step 1: Create style tokens**

```css
/* src/styles/tokens.css */
:root {
  --color-primary: #1677ff;
  --color-text: #1f2329;
  --color-text-muted: #646a73;
  --color-bg: #f5f6f7;
  --color-surface: #ffffff;
  --color-border: #dee0e3;
  --space-1: 4px;
  --space-2: 8px;
  --space-3: 12px;
  --space-4: 16px;
  --space-6: 24px;
  --space-8: 32px;
  --radius: 6px;
  --shadow-sm: 0 1px 2px rgba(0, 0, 0, 0.05);
  --shadow-md: 0 4px 8px rgba(0, 0, 0, 0.08);
}
```

- [ ] **Step 2: Create global.css**

```css
/* src/styles/global.css */
@import "./tokens.css";

* {
  box-sizing: border-box;
  margin: 0;
  padding: 0;
}

html, body, #app {
  height: 100%;
}

body {
  font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "PingFang SC",
    "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
  color: var(--color-text);
  background: var(--color-bg);
  font-size: 14px;
  line-height: 1.6;
}

a {
  color: var(--color-primary);
  text-decoration: none;
}
a:hover {
  text-decoration: underline;
}
```

- [ ] **Step 3: Create AppLayout.vue**

```vue
<!-- src/shell/layout/AppLayout.vue -->
<template>
  <div class="app-layout">
    <NavBar />
    <main class="app-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import NavBar from "./NavBar.vue";
</script>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  min-height: 100vh;
}
.app-main {
  flex: 1;
  padding: var(--space-6);
}
</style>
```

- [ ] **Step 4: Create NavBar.vue**

```vue
<!-- src/shell/layout/NavBar.vue -->
<template>
  <header class="navbar">
    <div class="brand">IOP</div>
    <nav class="links">
      <router-link to="/">工作台</router-link>
    </nav>
  </header>
</template>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  padding: var(--space-3) var(--space-6);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
}
.brand {
  font-weight: 600;
  font-size: 18px;
  color: var(--color-primary);
}
.links {
  display: flex;
  gap: var(--space-4);
}
</style>
```

- [ ] **Step 5: Create WorkspaceHome.vue**

```vue
<!-- src/shell/workspace/WorkspaceHome.vue -->
<template>
  <section class="workspace">
    <h1>工作台</h1>
    <p>欢迎使用 IOP. M1 仅基座可用; M2 起加租户与登录.</p>
    <div class="card" v-if="version">
      <h2>系统</h2>
      <ul>
        <li>版本: {{ version }}</li>
        <li>API 基址: <code>{{ apiBase }}</code></li>
      </ul>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { client } from "@/api/client";

const version = ref<string>("");
const apiBase = ref<string>(import.meta.env.MODE === "development" ? "/api (proxy → :8080)" : "/api");

onMounted(async () => {
  try {
    const res = await client.get("/version");
    version.value = (res.data?.version as string) ?? "unknown";
  } catch (e) {
    version.value = "(unable to reach backend)";
  }
});
</script>

<style scoped>
.workspace {
  max-width: 800px;
  margin: 0 auto;
}
.card {
  margin-top: var(--space-6);
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius);
}
code {
  background: var(--color-bg);
  padding: 2px 6px;
  border-radius: 3px;
}
</style>
```

- [ ] **Step 6: Create router**

```ts
// src/router/index.ts
import { createRouter, createWebHistory } from "vue-router";
import AppLayout from "@/shell/layout/AppLayout.vue";

const router = createRouter({
  history: createWebHistory(),
  routes: [
    {
      path: "/",
      component: AppLayout,
      children: [
        {
          path: "",
          name: "workspace",
          component: () => import("@/shell/workspace/WorkspaceHome.vue"),
        },
      ],
    },
  ],
});

export default router;
```

- [ ] **Step 7: Smoke-build**

```bash
cd /Users/leo/Documents/iop/web
npm run build
```

Expected: build fails on missing `@/api/client` — Task 22 fixes.

- [ ] **Step 8: Commit**

```bash
git add web/src/styles/ web/src/shell/ web/src/router/
git commit -m "feat(shell): layout + router + workspace home (calls /version)"
```

---

### Task 22: web/api client skeleton

**Files:**
- Create: `web/src/api/client.ts`
- Create: `web/src/api/types.ts`
- Create: `web/src/shell/stores/.keep`

- [ ] **Step 1: Implement axios client**

```ts
// src/api/client.ts
import axios, { type AxiosInstance, type InternalAxiosRequestConfig } from "axios";

function newId(): string {
  // UUID v4 — sufficient for Idempotency-Key (M2 enforces).
  return crypto.randomUUID();
}

function createClient(): AxiosInstance {
  const baseURL =
    import.meta.env.VITE_API_BASE_URL ?? (import.meta.env.MODE === "development" ? "/api" : "/api");

  const instance = axios.create({
    baseURL,
    timeout: 30_000,
    withCredentials: false,
  });

  instance.interceptors.request.use((cfg: InternalAxiosRequestConfig) => {
    const method = (cfg.method ?? "get").toUpperCase();
    if (method !== "GET" && method !== "HEAD") {
      // Idempotency-Key for all mutations. M2 server-side middleware will enforce.
      cfg.headers.set("Idempotency-Key", newId());
    }
    // X-Request-Id for trace propagation. Server echoes via RequestID middleware.
    cfg.headers.set("X-Request-Id", newId());
    return cfg;
  });

  instance.interceptors.response.use(
    (res) => res,
    (err) => {
      if (err.response?.status === 401) {
        // M2: redirect to login. M1: just propagate.
      }
      return Promise.reject(err);
    },
  );

  return instance;
}

export const client = createClient();
```

- [ ] **Step 2: Implement shared types**

```ts
// src/api/types.ts

// Envelope must match server/internal/interface/apiresp/response.go.
export interface Envelope<T = unknown> {
  code: number;
  data?: T;
  error?: {
    code: string;
    message: string;
    kind: string;
  };
  trace_id: string;
}
```

- [ ] **Step 3: Run smoke build**

```bash
cd /Users/leo/Documents/iop/web
npm run build
```

Expected: build PASSES.

- [ ] **Step 4: Run frontend dev server end-to-end**

In one terminal:

```bash
cd /Users/leo/Documents/iop/server
make build && ./bin/server
```

In another:

```bash
cd /Users/leo/Documents/iop/web
npm run dev
# open http://localhost:5173
```

Expected:
- Workspace loads showing "版本: dev"
- Browser devtools network panel: GET /api/version → 200 with `{"code":0,"data":{"version":"dev"},"trace_id":"..."}`
- Request headers include `X-Request-Id` and `Idempotency-Key`

- [ ] **Step 5: Commit**

```bash
git add web/src/api/ web/src/shell/stores/
git commit -m "feat(api-client): axios with Idempotency-Key + X-Request-Id auto-inject"
```

---

## Phase 6: CI + smoke + docs (Week 4)

### Task 23: GitHub Actions CI

**Files:**
- Create: `.github/workflows/ci.yml`

- [ ] **Step 1: Create workflow**

```yaml
# .github/workflows/ci.yml
name: CI

on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  server:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:16-alpine
        env:
          POSTGRES_USER: iop
          POSTGRES_PASSWORD: iop_dev
          POSTGRES_DB: iop
        ports: ["5432:5432"]
        options: >-
          --health-cmd "pg_isready -U iop -d iop"
          --health-interval 5s
          --health-timeout 3s
          --health-retries 10
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
          cache: true
          cache-dependency-path: server/go.sum
      - name: Go vet
        working-directory: ./server
        run: go vet ./...
      - name: golangci-lint
        uses: golangci/golangci-lint-action@v6
        with:
          working-directory: ./server
          args: --timeout=5m
      - name: Unit + integration tests
        working-directory: ./server
        env:
          IOP_TEST_DB_DSN: "postgres://iop:iop_dev@localhost:5432/iop?sslmode=disable"
        run: go test -race -coverprofile=coverage.out ./...
      - name: Build all binaries
        working-directory: ./server
        run: make build

  web:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with:
          node-version: "20"
          cache: "npm"
          cache-dependency-path: web/package-lock.json
      - name: Install deps
        working-directory: ./web
        run: npm ci
      - name: Type check + build
        working-directory: ./web
        run: npm run build
      - name: Unit tests
        working-directory: ./web
        run: npm test --if-present

  openapi-lint:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-node@v4
        with: { node-version: "20" }
      - name: Lint OpenAPI
        run: npx --yes @redocly/cli lint server/api/openapi.yaml
```

- [ ] **Step 2: Create root .golangci.yml**

```yaml
# server/.golangci.yml
run:
  timeout: 5m
  go: "1.22"

linters:
  enable:
    - errcheck
    - gosimple
    - govet
    - ineffassign
    - staticcheck
    - typecheck
    - unused
    - gofmt
    - goimports

linters-settings:
  goimports:
    local-prefixes: github.com/leo/iop/server

issues:
  exclude-rules:
    - path: _test\.go
      linters: [errcheck]
```

- [ ] **Step 3: Verify CI config locally**

```bash
cd /Users/leo/Documents/iop/server
go vet ./...
# install golangci-lint if needed:
# brew install golangci-lint
golangci-lint run ./... --timeout=5m
```

Expected: 0 errors.

- [ ] **Step 4: Commit and verify push triggers CI**

```bash
git add .github/ server/.golangci.yml
git commit -m "ci: GitHub Actions for go test + lint + web build + openapi lint"
```

(If repo is pushed to GitHub, verify the workflow runs green on the PR or push.)

---

### Task 24: README + dev docs + integration smoke

**Files:**
- Create: `iop/README.md`
- Create: `server/test/integration/smoke_test.go`
- Create: `scripts/dev.sh`

- [ ] **Step 1: Write integration smoke test**

```go
// server/test/integration/smoke_test.go
package integration

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/leo/iop/server/internal/app"
	"github.com/leo/iop/server/internal/config"
)

func TestSmoke_EndToEnd(t *testing.T) {
	if os.Getenv("IOP_INTEGRATION") == "" {
		t.Skip("set IOP_INTEGRATION=1 to run (requires docker-compose db running)")
	}

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("config load: %v", err)
	}
	cfg.DB.DSN = "postgres://iop:iop_dev@localhost:5432/iop?sslmode=disable"

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	a, cleanup, err := app.Build(ctx, cfg)
	if err != nil {
		t.Fatalf("app build: %v", err)
	}
	defer cleanup()

	srv := httptest.NewServer(a.Engine())
	defer srv.Close()

	for _, tc := range []struct {
		name  string
		path  string
		want  int
	}{
		{"livez", "/livez", 200},
		{"readyz", "/readyz", 200},
		{"version", "/version", 200},
		{"healthz", "/healthz", 200},
		{"metrics", "/metrics", 200},
		{"dict-known", "/api/dict/plan_level", 200},
		{"dict-unknown", "/api/dict/nope", 404},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resp, err := http.Get(srv.URL + tc.path)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, resp.StatusCode)
			}
		})
	}

	// Verify X-Request-Id echoes when sent
	t.Run("trace propagation", func(t *testing.T) {
		req, _ := http.NewRequest("GET", srv.URL+"/version", nil)
		req.Header.Set("X-Request-Id", "trace-smoke-123")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("get: %v", err)
		}
		defer resp.Body.Close()
		if got := resp.Header.Get("X-Request-Id"); got != "trace-smoke-123" {
			t.Fatalf("expected echo of trace id, got %q", got)
		}
		var body map[string]any
		_ = json.NewDecoder(resp.Body).Decode(&body)
		if body["trace_id"] != "trace-smoke-123" {
			t.Fatalf("envelope missing trace id")
		}
	})
}
```

- [ ] **Step 2: Run smoke test**

```bash
cd /Users/leo/Documents/iop/deployments && docker compose up -d db
cd /Users/leo/Documents/iop/server
./bin/migrate up
IOP_INTEGRATION=1 go test ./test/integration/... -v
```

Expected: all subtests PASS.

- [ ] **Step 3: Write dev.sh helper**

```bash
#!/usr/bin/env bash
# scripts/dev.sh — one-command local dev: starts infra, migrates, runs server + web.

set -euo pipefail
ROOT=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)

echo "[1/4] Starting infra (db/redis/minio)..."
(cd "$ROOT/deployments" && docker compose up -d db redis minio)

echo "[2/4] Waiting for PG healthy..."
until (cd "$ROOT/deployments" && docker compose exec -T db pg_isready -U iop -d iop > /dev/null 2>&1); do
  sleep 1
done

echo "[3/4] Running migrations..."
(cd "$ROOT/server" && go run ./cmd/migrate up)

echo "[4/4] Starting server (port 8080) + web (port 5173) in background..."
echo "  Server:  http://localhost:8080"
echo "  Web:     http://localhost:5173"
echo "  Stop:    kill $(jobs -p) && (cd $ROOT/deployments && docker compose down)"
(cd "$ROOT/server" && go run ./cmd/server) &
(cd "$ROOT/web" && npm run dev) &
wait
```

```bash
chmod +x scripts/dev.sh
```

- [ ] **Step 4: Write iop/README.md**

```markdown
# IOP

企业内部多租户 B 端办公平台基座. v3.1 设计 spec 在 `docs/superpowers/specs/`.

## 目录

- `server/` — Go 后端 (单 `cmd/server` 二进制)
- `web/` — Vue 3 单 Vite 前端
- `deployments/` — docker-compose 编排
- `docs/` — 架构 spec / 实施 plan / 学习材料
- `scripts/` — dev.sh / openapi-gen.sh
- `legacy/` — v1 资产 (Spring Boot + Element Plus), 仅作迁移参考

## 快速开始

需要 Go 1.22+, Node 20+, Docker, jq (可选).

```bash
# 一键起 (推荐):
./scripts/dev.sh
```

手动:

```bash
# 1. 起依赖
cd deployments && docker compose up -d db redis minio

# 2. 跑迁移
cd ../server && go run ./cmd/migrate up

# 3. 起后端 (终端 A)
go run ./cmd/server
# → http://localhost:8080

# 4. 起前端 (终端 B)
cd ../web && npm install && npm run dev
# → http://localhost:5173
```

## 主要命令

```bash
# Server
make -C server build        # build all binaries (server / migrate / tenantctl)
make -C server test         # unit + integration (要求 PG 起着)
make -C server lint
make -C server dev          # go run ./cmd/server

# Web
cd web && npm run build
cd web && npm test

# OpenAPI
make -C server openapi-gen  # M1 stub, M3+ 生成前端 SDK
```

## 阶段路线 (v3.1 §7)

- **M1** (当前): 骨架 + 基础设施 + livez/readyz/version/metrics
- M2: Tenancy + IAM + 命门测试 + B2B SaaS 基本盘 (限流/幂等/慢查询/备份)
- M3: Audit + Notification + FileStorage + Dictionary 完整
- M4: OKR 完整闭环 (核心学 DDD 里程碑)
- M5: 生产部署 + KingbaseV8

## 开发约定

- 提交粒度小, TDD 优先 (见 `docs/superpowers/plans/`)
- 跨服务通信走 `internal/shared/eventbus`, 不直接 import
- 错误统一 `internal/shared/errors`, code = `<source>.<resource>.<reason>`
- 日志 trace_id 贯穿; 不打 password / token / secret
```

- [ ] **Step 5: Commit final M1 deliverables**

```bash
cd /Users/leo/Documents/iop
git add README.md server/test/integration/ scripts/dev.sh
git commit -m "docs+test: README + dev.sh + integration smoke covering M1 endpoints"
```

---

## M1 Acceptance Verification (run before declaring M1 done)

- [ ] **AC1: `./scripts/dev.sh` boots end-to-end**

```bash
./scripts/dev.sh
# Wait ~15s; visit http://localhost:5173 in browser.
# Workspace shows "版本: dev" (proves frontend → backend reachability).
```

- [ ] **AC2: livez/readyz/version/metrics all work**

```bash
for ep in livez readyz version metrics; do
  echo "=== /$ep ==="
  curl -s http://localhost:8080/$ep | head -5
done
```

- [ ] **AC3: Panic does not kill process**

Add `r.GET("/boom", ...)` route temporarily, hit it, expect 500 + process still running. Remove the route.

- [ ] **AC4: trace_id propagates**

```bash
curl -s -H "X-Request-Id: trace-ac4" http://localhost:8080/api/dict/plan_level | jq .trace_id
# "trace-ac4"
```

- [ ] **AC5: CI green**

Push to a branch and verify GitHub Actions all three jobs pass.

- [ ] **AC6: Test coverage acceptable**

```bash
cd server && go test -coverprofile=coverage.out ./internal/shared/... ./internal/services/... ./internal/interface/...
go tool cover -func=coverage.out | tail -5
```

Expected: `shared/*` and `services/*` ≥ 80%; `interface/*` lower acceptable (middleware tested via integration).

- [ ] **AC7: All commits push cleanly**

```bash
git status
# nothing to commit
git log --oneline | head -25
# should show ~25 commits spanning all tasks
```

---

## Self-Review Notes

**Spec coverage** (v3.1 §7 M1 checklist):

| Deliverable | Task |
|---|---|
| cmd/server + cmd/migrate + cmd/tenantctl | T1 (skel), T15 (server), T16 (migrate); **note: cmd/tenantctl is created as empty in T1; full impl is M2** |
| shared/{kernel,errors,eventbus,tenantdb} | T3-T9 |
| middleware: RequestID/Recover/Logger/CORS/Error | T12-T14 |
| services/{dictionary,localization} skeleton | T17, T18 |
| OpenAPI for system endpoints | T19 |
| migrations/public/000001 | T16 |
| web/ Vite skeleton | T20-T22 |
| docker-compose.yml | T2 |
| Makefile | T1 |
| CI green | T23 |
| Smoke acceptance | T24 + AC1-AC7 |

**Gap noted**: `cmd/tenantctl` per spec §7 M1 is listed but full functionality lands in M2 (CreateTenant requires Tenancy service). Resolution: T1 creates `cmd/tenantctl/main.go` as a 3-line stub (`fmt.Println("TODO M2: tenantctl"); os.Exit(0)`). Add this in the T1 directory step so `make build` succeeds.

Also confirm: M1 §7 does NOT include the v3.1-specific B2B SaaS basics (rate_limit, idempotency, slow_query, backup) — those are M2 per §7. ✅ This plan correctly defers them.

**Placeholder scan**: none — every step has actual code or commands.

**Type consistency**: 
- `kernel.ID` used throughout
- `eventbus.Event` fields consistent in producer and test
- `apiresp.envelope` matches `web/src/api/types.ts:Envelope`
- `health.Check` / `health.Report` consistent in registry + handler

---

Plan complete and saved to `docs/superpowers/plans/2026-05-29-m1-foundation.md`.

**Two execution options:**

**1. Subagent-Driven (recommended)** — I dispatch a fresh subagent per task, review between tasks, fast iteration.

**2. Inline Execution** — Execute tasks in this session using executing-plans, batch execution with checkpoints.

**Which approach?**
