# 上线前审计报告 / Pre-launch Audit — 2026-05-31

A full pre-launch test of the IOP platform: all build/test gates, a live end-to-end
smoke, and an exhaustive multi-agent code audit across 8 launch-critical dimensions
(RBAC, tenant isolation, the 9 business modules, frontend↔backend contract, accumulated
collateral, migrations, security, operational readiness). Every finding was
adversarially verified before being acted on.

## Gate results — all green

| Gate | Result |
|------|--------|
| `go build ./...` / `go vet ./...` / `make build` (clean cache) | ✅ |
| `go test ./...` (unit) | ✅ |
| Integration tests — 5 命门 tenant-isolation + RBAC keystone + E2E smoke | ✅ 12/12 |
| `vue-tsc --noEmit` / `npm run build` | ✅ |
| Fresh-binary boot smoke (migrate → seed → SyncAllSchemas → serve) | ✅ |
| Live login + both consoles + per-user apps probes | ✅ |

## Audit outcome

28 raw findings → 24 confirmed after adversarial verification: **3 blockers, 3 majors, 18 minor/nits**.

### 🔴 Blockers — all FIXED

1. **`appHomeRoute()` sent 7 of 9 apps to a 404.** `web/src/shell/appcenter/appstore.ts`
   used a hardcoded `ROUTE_BY_CODE` map (okr/tasks/approval/crm) and fell back to
   `/apps/<code>`, which has no route — so every launcher (LeftRail, WorkspaceHome,
   AppCenter) 404'd docs/project/mindmap/news/books/lcform/approval.
   **Fix:** drive `appHomeRoute` from each module's `manifest.ts` (`homeRoute`, falling
   back to `routePrefix`) via the same `import.meta.glob` the router uses; removed the
   dead `crm` entry.

2 & 3. **Production `docker-compose.prod.yml` could not boot.** It set `IOP_ENV=prod`
   but a DSN with `sslmode=disable` and inherited `allowed_origins=["*"]`; both are
   fatal in `config.Validate()` → crash-loop on the documented prod path.
   **Fix:** (a) the bundled `db` is on an internal-only network, so added an explicit,
   default-`false` escape hatch `db.allow_insecure` / `IOP_DB_ALLOW_INSECURE` that
   downgrades the prod `sslmode=disable` check to a warning only when consciously set;
   (b) `IOP_SERVER_ALLOWED_ORIGINS` is now applied reliably (comma-separated env) and
   the prod compose requires the operator to set `IOP_ALLOWED_ORIGINS`. The documented
   prod path now boots; the secure-by-default invariant is preserved (verified with a
   positive+negative boot test and `config_test.go`).

### 🟠 Majors

1. **tenant_member could not submit/approve/borrow — FIXED.** The boot seeder granted
   the built-in `tenant_member` role only `read`/`write`, so approval initiation/approval
   and book borrowing were admin-only out of the box. Changed the seeder to grant every
   module action except the admin-only `delete`/`manage` (so `submit`/`approve`/`borrow`
   are now member-default). Verified in the DB.

2. **`/metrics` did not expose the runbook's alert metrics — FIXED.**
   `iop_http_request_duration_seconds` was never observed (no middleware) and both it
   and `iop_pg_slow_query_total` were registered on the global default registry, not the
   served one. Added a shared registry (`metrics.Registerer()`), registered the
   slow-query counter on it, and added a latency-recording middleware. Confirmed
   `iop_http_request_duration_seconds` is now scrapeable.

3. **Session revocation not enforced on access tokens when Redis is down — DEFERRED
   (documented).** `VerifyAccessToken` only consults the Redis blacklist; with Redis
   absent, logout / user-disable does not cut an already-issued access token until it
   expires (≤30 min). The refresh path *is* DB-protected and production runs Redis, so
   exposure is bounded and degraded-mode-only.
   **Recommended hardening:** treat Redis as a hard prod dependency (fail readiness when
   it is down), or have `VerifyAccessToken` fall back to a DB check of `session.revoked`
   + `user.status` when `rdb == nil` (nearly free — `PasswordChangeGate` already loads
   the user). Not a launch blocker.

### Minor / nit findings (18)

Fixed in this pass:
- **IDOR:** `/me/sessions/:id/revoke` now verifies the session belongs to the caller.
- **okr & tasks menus:** both modules now contribute tenant-console menu nodes, making
  all 9 modules uniform in the nav catalog.
- Stale `crm` route entry removed (folded into the appHomeRoute blocker fix).

Deferred (tracked, non-blocking) — see the audit transcript for full reasoning:
- Stale comments: platform console gating described as `PlatformAdminRequired` (actually
  `PlatformAccess` + per-route `PlatformAuthz`); `TenantAdminRequired` does not admit
  platform_admin (correct, but the comment claims otherwise).
- Platform `:tid` org routes don't check tenant status (operate on suspended/closed orgs).
- `member_joined` welcome-notice handler writes with an empty schema (silently no-ops).
- A few ad-hoc `SET search_path` sites use Go `%q` instead of Postgres identifier
  quoting / bypass `validateSchemaIdent`; one uses non-`LOCAL` set on a pooled conn.
- `menu:write` is ungrantable to custom tenant roles (only the `tenant_admin` wildcard).
- tenant_template migrations 000001–000004 ship no `.down.sql`; the in-house migrator is
  forward-only (all `.down.sql` are reference-only).
- Tenant-console registration approvals are not written to the audit log.
- Operator-supplied seed admin password is not force-rotated.
- Prod compose does not auto-run migrations before the server (manual step); no redis
  health check registered when redis is absent at boot; runbook references a
  `tenantctl migrate-all` command that does not exist; root-dept rewrite runs for every
  tenant on every boot inside `SyncAllSchemas` (paginated at 1000).

## Dev-DB hygiene

Running the integration suite against the shared dev DB (`localhost:5433`) repopulates
it with `iso*`/`poll*`/`susp*` test tenants. The dev DB was cleaned back to the 3 real
tenants (`system`, `bjgov`, `demo541`) + 58 users after testing. (A follow-up could point
the integration harness at a throwaway DB to avoid this.)
