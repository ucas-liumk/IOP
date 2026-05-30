# CLAUDE.md — IOP 项目向导（给 Claude Code 读）

This file is auto-loaded by Claude Code. It orients an AI session so you can pick up
development on any machine after `git clone`. Keep it short and current.

## 这是什么

IOP（一站通办）= 多租户 B 端办公平台**基座/框架**。Go 后端 + Vue3 前端。目标：像
ruoyi/jeecg 那样供开发者直接拿来接业务模块，但天生多租户、可插拔。

## 核心模型（务必先理解，否则容易做错权限/隔离）

- **组织 ≡ 租户（1:1）**：每个组织 = 一个隔离的 PG schema `tenant_<slug>`。业务叫「组织」，技术叫「租户」。
- **platform_admin 是全局身份**：`public.platform_user.is_platform_admin` 布尔列，**不依赖任何租户成员关系**。判定用 `iam.Service.IsPlatformAdminUser(platformUserID)`。
- **tenant_admin 是租户内角色**：`role_grant`（member×tenant）里的 `tenant_admin`。判定用 `IsTenantAdmin(memberID, tenantID)`。平台管理员**不**自动是租户管理员。
- **两套控制台**：
  - 平台控制台 `/platform/*`（前端）↔ `/api/platform/*` + `/api/tenants`（后端，`PlatformAdminRequired`，**无租户上下文**）。治理：组织/全局用户/全局注册申请/概览。
  - 组织控制台 `/admin/*` ↔ `/api/admin/*`（后端，`TenantLoader` + `TenantAdminRequired`，**需租户上下文**）。本组织：成员/角色/部门/设置/注册申请/应用/字典/审计。
- 平台管理员只治理，不进组织内部业务数据。

## 关键路径

- 路由分组 / 装配：`server/internal/app/app.go`（看 `Engine()` 里的 `authT` / `admin` / `platform` 三组）。
- 鉴权中间件：`server/internal/services/iam/middleware.go`（`JWTAuth` `TenantLoader` `RBAC` `PasswordChangeGate`）、`admin_http.go`（`TenantAdminRequired` `PlatformAdminRequired`、`/platform/*`、`/admin/*`）。
- 模块契约：`server/internal/shared/module/{module.go,registry.go}`（`Module`、`Manifest`、`Deps.Authz`、`Deps.AppEnabled`、`InstallHook`）。
- 参考业务模块：`server/internal/contexts/{okr,tasks}/`（DDD 四层：domain/infrastructure/application/interface + module.go）。
- 多租户 DB：`server/internal/shared/tenantdb/`（`TenantDB.Transaction` 自动 `SET LOCAL search_path`）。
- 迁移：`server/migrations/public/`（平台表，按序号）、`server/migrations/tenant_template/`（每个租户 schema 的表；新模块表加这里，启动时 `SyncAllSchemas` 自动同步到已有租户）。
- 前端外壳：`web/src/shell/`（auth.store、guard、layout/{AppLayout,TopBar,LeftRail}、notify/confirm、components）。
- 前端模块：`web/src/modules/{admin,platform,me,okr,tasks}/`；`router/index.ts` 用 `import.meta.glob` 自动发现 `modules/*/manifest.ts`。

## 常用命令

```bash
./scripts/dev.sh                       # 一键：infra + 迁移 + 后端:8080 + 前端:5174
cd server && go build ./... && make build && go vet ./...
cd server && go run ./cmd/migrate up   # 跑平台迁移
cd web && npx vue-tsc --noEmit && npm run build
# 集成测试（需 docker 起 db；含 5 个隔离命门测试）：
IOP_INTEGRATION=1 IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  IOP_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" IOP_REDIS_ADDR="localhost:6380" \
  go test ./test/integration/...
```

环境/端口：PG `localhost:5433`、Redis `6380`、MinIO `9100`、后端 `:8080`、前端 `:5174`（见 `deployments/docker-compose.yml` + `server/configs/dev.yaml`）。**从 server 目录启动后端**，否则找不到 `configs/dev.yaml`。

默认账号：`admin / Admin12345!`（首登强制改密；`IOP_SEED_ADMIN_PASSWORD` 可覆盖）。

## 约定 & 注意

- 新增业务模块：`./scripts/new-module.sh <code> "<名>" "<分类>"` → 在 `app.go` `registry.Register(...)` 加一行。模块路由用 `deps.Authz(resource, action)` 按 Manifest 权限网关；表加到 `tenant_template`。
- RBAC：每条业务路由都要 `Authz` 网关；权限 resource/action 必须出现在 `Manifest.Permissions`（否则角色编辑器看不到、也无法授予）。boot 时会给 `tenant_member` 角色默认授予各模块 read/write。
- 错误：用 `shared/errors`（`KindParam/Auth/Forbidden/NotFound/Conflict/...`），HTTP 层 `apiresp.Fail/OK`。
- 提交信息：Conventional Commits（中文 OK），见 git log。
- 改完务必：`go build ./...` + `go vet` + `vue-tsc --noEmit`，并尽量跑命门测试。
- 不要把 `platform_admin` 当成租户角色去判定；不要在 `CreateTenant` 里 DROP 已存在的租户 schema。

## 当前状态（2026-05）

可发布基座：双控制台、全局 platform_admin、注册-审批、应用中心、RBAC 强制、首登改密、限流/审计/指标齐全。两个参考模块 OKR + 任务清单（仿滴答清单）已完成。`docs/` 有架构 spec 与运维手册。
