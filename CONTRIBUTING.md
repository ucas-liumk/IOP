# 贡献指南 / Contributing to IOP

感谢你对 IOP（企业多租户 B 端办公平台基座，Go 后端 + Vue 3 前端）的关注！本指南介绍如何搭建开发环境、运行测试、遵循代码规范，以及提交 PR 的流程。

Thanks for your interest in IOP (a multi-tenant B2B office-platform framework: Go backend + Vue 3 frontend). This guide covers dev setup, testing, code style, the module contract, and the PR flow.

> 行为准则见 [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md)；安全问题请勿走公开 issue，见 [SECURITY.md](./SECURITY.md)。
>
> See [CODE_OF_CONDUCT.md](./CODE_OF_CONDUCT.md) for expected behavior. For security issues, do **not** open a public issue — see [SECURITY.md](./SECURITY.md).

## 1. 开发环境 / Dev Setup

需要 / Requirements: **Go 1.22+**, **Node 20+**, **Docker**。

### 一键启动 / One command (recommended)

```bash
./scripts/dev.sh
```

该脚本会启动依赖容器、跑迁移，并同时拉起后端（:8080）与前端（:5174）。

This starts the dependency containers, runs migrations, and launches both the backend (:8080) and frontend (:5174).

### 手动启动 / Manual

```bash
# 1. 起依赖 (重映射端口避免本地 SSH 隧道冲突)
#    Start dependencies (remapped ports to avoid local SSH-tunnel conflicts)
#    PostgreSQL :5433 · Redis :6380 · MinIO API :9100 · MinIO UI :9101
cd deployments && docker compose up -d db redis minio

# 2. 跑迁移 / Run migrations
cd ../server && go run ./cmd/migrate up

# 3. 起后端 (终端 A) / Backend (terminal A) → http://localhost:8080
go run ./cmd/server          # 或 make build && ./bin/server

# 4. 起前端 (终端 B) / Frontend (terminal B) → http://localhost:5174
cd ../web && npm install && npm run dev
```

常用命令 / Common commands:

```bash
# Backend
make -C server build         # 构建 server / migrate / tenantctl
make -C server test          # 单测 (要求 PG 起着 / requires PG up)
make -C server lint          # golangci-lint
make -C server migrate       # go run ./cmd/migrate up

# Frontend
cd web && npm run build      # vue-tsc --noEmit && vite build
cd web && npm run dev
```

## 2. 运行测试 / Running Tests

提交前请确保测试通过。Please make sure tests pass before submitting.

### 后端 / Backend

```bash
# 单元测试 (需 PG 起着) / Unit tests (requires PG running)
make -C server test          # go test -race -coverprofile=coverage.out ./...

# 集成 / 命门隔离测试 / Integration + keystone isolation tests
cd server && IOP_INTEGRATION=1 go test ./... -v
cd server && IOP_INTEGRATION=1 go test ./test/integration/... -v
```

- **命门（keystone）多租户隔离**是本项目的核心约束。任何涉及租户级数据访问的改动，都必须保证 `server/test/integration/tenant_isolation_test.go` 通过，且不得引入跨 schema / 跨租户的数据可达性。
- 集成测试通过环境变量 `IOP_INTEGRATION=1` 开启；未设置时这些测试会被跳过。

Tenant isolation (the keystone) is a core invariant. Any change touching tenant-scoped data access must keep `server/test/integration/tenant_isolation_test.go` green and must not introduce cross-schema / cross-tenant reachability. Integration tests are gated behind `IOP_INTEGRATION=1` and skipped otherwise.

### 前端 / Frontend

```bash
cd web && npm run build      # 包含类型检查 / includes vue-tsc --noEmit type check
cd web && npm run test       # vitest run
```

`vue-tsc` 类型检查必须无错误。`vue-tsc` type checking must pass with no errors.

## 3. 代码规范 / Code Style

### Go

- 提交前运行 `gofmt`（或 `go fmt ./...`）与 `go vet ./...`。
- `make -C server lint`（`golangci-lint`）必须无告警。
- 遵循 README「开发约定」：
  - 跨服务通信走 `internal/shared/eventbus`（Publish-Subscribe），**不直接 import** 其他服务/上下文。
  - 错误统一 `internal/shared/errors`，code = `<source>.<resource>.<reason>`。
  - 日志携带 `trace_id`，**不打** password / token / secret（用 `logger.Sanitize`）。
  - 多租户 DB 访问走 `tenantdb.TenantDB.Transaction`（自动 `SET LOCAL search_path`）。

Run `gofmt` and `go vet ./...` before committing; `golangci-lint` must be clean. Follow the conventions above: cross-service comms via the event bus (never direct imports), unified errors, `trace_id` logging with no secrets, and tenant DB access via `tenantdb.TenantDB.Transaction`.

### Web

- `vue-tsc --noEmit`（`npm run build` 会执行）必须通过。
- 若仓库配置了 ESLint，请保持 `eslint` 无告警；遵循现有 TypeScript + Vue 3 `<script setup>` 风格。
- 复用 `@/shell/components` 提供的基础组件（`PageHeader` / `StatCard` / `DataTable` / `EmptyState`），不要在模块间互相 import。

`vue-tsc --noEmit` (run by `npm run build`) must pass. If ESLint is configured, keep it clean. Reuse the shared shell components instead of cross-module imports.

## 4. 添加业务模块 / Adding a Business Module

平台是「基座 + 可插拔模块」架构。任意业务模块通过统一的 `Module` 契约接入，**框架内的修改 ≈ 一行注册代码**。

The platform is "core + pluggable modules". Any business module plugs in via the unified `Module` contract — the change inside the framework is roughly one line of registration.

```bash
# 脚手架生成后端 (DDD 四层) + 前端 (manifest/routes/api/views) + 迁移
./scripts/new-module.sh <code> "<中文名>" "<分类>"
# 例 / e.g.
./scripts/new-module.sh crm "客户管理 CRM" "业务管理"
```

然后在 `server/internal/app/app.go` 加一行 `registry.Register(<code>.New(deps))`，跑迁移与构建即可。完整步骤、`Module` 接口与 `Manifest` 字段、`Deps` 注入、事件总线通信约定，详见 **[`docs/developer/adding-new-module.md`](./docs/developer/adding-new-module.md)**。

Then add one `registry.Register(<code>.New(deps))` line in `server/internal/app/app.go`, run migrations, and build. The full contract — `Module` interface, `Manifest` fields, injected `Deps`, and event-bus communication rules — is documented in **[`docs/developer/adding-new-module.md`](./docs/developer/adding-new-module.md)**.

关键约束 / Key rules:

- 模块**不应**直接访问其他模块的代码，模块间通信走事件总线（`deps.Bus.Publish` / `Subscribe`）。
- 前端路由通过 `manifest.ts` + `routes.ts` **自动发现**，不要手改 `router/index.ts`。
- 不要共享其他模块的 schema 表；监听其事件维护自己的投影。

Modules must not import other modules' code (communicate via the event bus); frontend routes are auto-discovered via `manifest.ts` + `routes.ts`; never share another module's schema tables.

## 5. 提交信息约定 / Commit Convention

采用 Conventional-Commits 风格，描述可用中文，技术名词保留英文（与现有 git 历史保持一致）。

We use a Conventional-Commits style; descriptions may be in Chinese with English tech terms, matching the existing history.

```
<type>(<scope>): <简短描述>
```

`type` 常用 `feat` / `fix` / `chore` / `docs` / `refactor` / `test`。参考近期历史 / Examples from recent history:

```
feat(framework): 通用模块化框架 — Module 契约 + AppStore + 脚手架
feat(admin): 完整后台管理 — 8 页 + AppCenter + 双角色门控
fix(okr): JSON tags on domain types + List hydrates items + Vue field rename
chore: 清理 demo / 测试 / 备份 — 只保留框架本身
```

约定 / Conventions:
- 提交粒度小，TDD 优先（见 `docs/superpowers/plans/`）。
- 一个提交聚焦一件事，提交信息说明「为什么」。

Keep commits small, prefer TDD, and explain the "why".

### Co-Authored-By

若提交内容由协作工具或他人协助完成，请在提交信息末尾追加协作者标记：

If a commit was co-authored, add a trailer at the end of the commit message:

```
Co-Authored-By: Name <email@example.com>
```

## 6. 分支与 PR 流程 / Branch & PR Flow

1. **Fork** 仓库并从 `main` 切出特性分支：`feat/<topic>` / `fix/<topic>`。
2. 小步提交，遵循上面的提交约定。
3. 推送前确保本地通过：后端 `make -C server test` + `IOP_INTEGRATION=1 go test ./...`、`make -C server lint`；前端 `npm run build`。
4. 开 PR 指向 `main`，填写 [PR 模板](./.github/pull_request_template.md) 的核对清单（测试通过、lint 干净、文档更新、无密钥泄露、新端点已考虑 RBAC）。
5. 关联相关 issue（`Closes #123`），等待 review；按反馈迭代。
6. 通过 review 且 CI 绿后合并。

Fork, branch off `main` (`feat/<topic>` / `fix/<topic>`), commit in small steps, ensure tests + lint + frontend build pass locally, open a PR against `main` filling in the [PR template](./.github/pull_request_template.md) checklist, link related issues, and iterate on review until CI is green.

感谢你的贡献！/ Thank you for contributing!
