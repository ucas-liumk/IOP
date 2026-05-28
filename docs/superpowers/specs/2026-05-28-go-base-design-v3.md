# Go 多租户 B 端基座 · 极简设计文档 v3.0

| 项 | 内容 |
|---|---|
| 版本 | v3.1.0 (在 v3.0 基础上补 6 项 B2B SaaS 基本盘) |
| 起草日期 | 2026-05-28 |
| 状态 | Draft, 取代 v2.1.1; 待 spec 复核后转入 writing-plans |
| 范围 | `iop/{server,web,deployments,docs,scripts,legacy}` 顶层直挂 |
| 首期落地业务 | **OKR (工作安排) 单模块走闭环**, M4 集中学透 DDD |
| 多租户隔离 | PG 每租户独立 schema + 事务级 `SET LOCAL search_path` (保留 v2.1.1 命门设计) |
| v3.1 相对 v3.0 增量 | 补齐 6 项 B2B SaaS 标配: 租户级限流 / 幂等性 / 慢查询防护 / 健康检查依赖矩阵 / 数据备份策略 / 后台任务多实例协调路径 |

---

## §0 目的与范围

### 目的

把 v2.1.1 (15 个上下文 / 多产品 / Module 接口 / Tauri 抽象 / pnpm workspace / 33 ADR) 收敛为**学习者第一次落地能成的最小架构**, 遵循三条原则:

1. **DDD 只用在真正需要的地方**: OKR 一个有界上下文走真 DDD; 其他 7 个全部降级为"服务包" (无 domain 层, Active Record + Service)
2. **YAGNI**: 删除所有"未来可能 / 预留 / v1.5+ / 不在范围但已建目录"的预案; 真有第二个产品 / 桌面端 / 模块开关时再设计
3. **范围克制**: 7 业务模块 → 1 个 (OKR); 后续业务模块在 v1.5+ 按真实需求按月解锁

### 范围

**包含**:
- Go 后端: 单 `cmd/server` 二进制
- Vue 前端: 单一 Vite 工程, 不上 monorepo
- 7 个**服务包** (services/): Tenancy / IAM / Audit / Notification / FileStorage / Dictionary / Localization — 仅 service + repository + http, 无 domain/application 层
- 1 个**有界上下文** (contexts/): OKR (工作安排) — 完整四层 DDD
- 多租户 PG schema 隔离 + 命门测试
- 错误模型 / 结构化日志 / 字典驱动枚举 / 事件总线
- **B2B SaaS 基本盘** (v3.1 补): 租户级限流 / 幂等性 / 慢查询防护 / 健康检查依赖矩阵 / 备份策略 (M2 起就位)
- 部署: docker-compose (dev + prod) + Nginx + KingbaseV8 生产兼容

**明确推迟到 v1.5+** (按真实业务需求触发, 不预先建目录):
- Problem (问题协同) — 状态机复杂, 等 OKR 上线后团队熟悉再做
- Contacts (通讯录) / Documents (资源平台) / Assets (资产管理) / Announcements (时政热点) — 业务真要再加
- Party (党务管理) — 国情化模块, 单独评估法律 / 数据合规风险后再立项
- AppCenter (应用中心) — 等有第二个业务模块再考虑模块开关
- 桌面端 (Tauri) — 等有客户付钱要桌面版再做
- 多产品 (`cmd/<product>`) — 等真有第二个产品再拆
- `Module` 接口 + Registry — 等 ≥ 3 个业务模块再抽
- pnpm workspace — 等 ≥ 3 个前端模块包再拆

### 与 v2.1.1 的关系

v3.0 与 v2.1.1 **不兼容**, 是范围 / 形状的重新设定, 不是版本号 +0.1。删除了 v2.1.1 60% 的内容; 删除细节见附录 A。

---

## §1 架构总览

### 1.1 一句话架构

**Go 模块化单体 + Vue 单页应用 + PG 每租户独立 schema + 进程内事件总线**, 单一后端二进制 + 单一前端 dist, 通过 docker-compose 编排。

### 1.2 核心分类: 服务包 vs 有界上下文

借鉴 Khononov "Learning DDD" 第 10 章: **不同子域用不同建模方式**。

| 类别 | 子域类型 | 建模方式 | 目录 | 数量 |
|---|---|---|---|---|
| **服务包** | 通用子域 / 支持子域 | Active Record + Service + Repository, **无 domain 层** | `internal/services/<X>/` | 7 |
| **有界上下文** | 核心子域 (首期仅 OKR) | 完整四层 DDD (domain/application/infrastructure/interface) | `internal/contexts/<X>/` | 1 |

**判定标准** (避免一刀切):
- 用 DDD: 有显著业务规则、状态机、不变量约束 → OKR 的 Plan 多级分解、Report 汇总规则、PlanItem 权重约束
- 不用 DDD: 主要是 CRUD + 增删改查 + 简单工作流 → Tenancy / IAM / Audit / Notification / FileStorage / Dictionary / Localization

> 这条分界线**比目录结构更重要**。如果将来 Problem 进场, 它会作为第 2 个有界上下文; 但如果新增的是"日历"这种 CRUD, 就放服务包。

### 1.3 目录结构

```
iop/
├── server/                                  ← Go 后端
│   ├── cmd/
│   │   ├── server/main.go                   API 主进程 (单一二进制)
│   │   ├── migrate/main.go                  迁移工具
│   │   └── tenantctl/main.go                租户 CLI
│   │
│   ├── internal/
│   │   ├── services/                        ★ 服务包 (无 domain 层)
│   │   │   ├── tenancy/
│   │   │   │   ├── tenant.go                Tenant 类型 + Repository 接口
│   │   │   │   ├── member.go                TenantMember 类型 + Repository
│   │   │   │   ├── service.go               业务逻辑 (CreateTenant 等)
│   │   │   │   ├── pg_repo.go               PostgreSQL 仓储实现
│   │   │   │   ├── http.go                  Gin handler + 路由注册
│   │   │   │   └── provisioner.go           SchemaProvisioner (执行 CREATE SCHEMA)
│   │   │   ├── iam/                         IAM (auth + RBAC)
│   │   │   │   ├── user.go / session.go / role.go
│   │   │   │   ├── service.go               Login / Refresh / Enforce
│   │   │   │   ├── jwt.go / bcrypt.go / casbin.go
│   │   │   │   ├── pg_repo.go
│   │   │   │   ├── http.go / middleware.go  JWT/Session/RBAC 中间件
│   │   │   ├── audit/
│   │   │   │   ├── log.go
│   │   │   │   ├── service.go               异步 buffer + 订阅 eventbus
│   │   │   │   ├── pg_repo.go
│   │   │   │   └── http.go
│   │   │   ├── notification/
│   │   │   │   ├── notification.go
│   │   │   │   ├── service.go               订阅 OKR 事件触发通知
│   │   │   │   ├── pg_repo.go / inapp.go
│   │   │   │   └── http.go
│   │   │   ├── filestorage/
│   │   │   │   ├── attachment.go
│   │   │   │   ├── service.go               Upload/Download/List
│   │   │   │   ├── minio.go / pg_repo.go
│   │   │   │   └── http.go
│   │   │   ├── dictionary/
│   │   │   │   ├── dict.go
│   │   │   │   ├── service.go               Lookup / AddOverride / Reload
│   │   │   │   ├── pg_repo.go / cache.go
│   │   │   │   └── http.go
│   │   │   └── localization/
│   │   │       ├── translate.go             T(ctx, key, args...)
│   │   │       ├── yaml_bundle.go           YAML 语言包
│   │   │       └── http.go
│   │   │
│   │   ├── contexts/                        ★ 有界上下文 (完整 DDD)
│   │   │   └── okr/                         OKR 工作安排 (M4 落地)
│   │   │       ├── domain/                  纯业务 (无任何外部 import)
│   │   │       │   ├── plan.go              Plan 聚合根 + PlanItem 实体
│   │   │       │   ├── report.go            Report 聚合根 + ReportEntry 实体
│   │   │       │   ├── values.go            PlanLevel / ReportType / Progress 值对象
│   │   │       │   ├── cadence.go           Cadence 领域服务 (周期计算)
│   │   │       │   ├── events.go            领域事件
│   │   │       │   ├── repository.go        PlanRepo / ReportRepo 接口
│   │   │       │   └── errors.go            领域错误
│   │   │       ├── application/             用例编排, 事务边界
│   │   │       │   ├── create_plan.go
│   │   │       │   ├── decompose_plan.go
│   │   │       │   ├── submit_daily.go
│   │   │       │   ├── submit_weekly.go
│   │   │       │   ├── rollup_weekly.go     管理人员周汇总 (Query 直查 DB)
│   │   │       │   ├── list_my_plans.go     Query
│   │   │       │   └── remind_overdue.go    定时器触发
│   │   │       ├── infrastructure/          DB / 外部适配
│   │   │       │   ├── pg_plan_repo.go
│   │   │       │   ├── pg_report_repo.go
│   │   │       │   └── pg_rollup_query.go   直查 DB 投影 DTO
│   │   │       └── interface/               HTTP 接口
│   │   │           ├── http.go              /plans/* /reports/* /rollups/*
│   │   │           └── dto.go
│   │   │
│   │   ├── shared/                          ★ 共享 (严格最小化)
│   │   │   ├── kernel/                      跨服务/上下文通用语义
│   │   │   │   ├── ids.go                   ID 类型 + UUID v7 生成
│   │   │   │   ├── context.go               ctxutil: WithTrace/Tenant/Member
│   │   │   │   ├── time.go                  Clock 接口
│   │   │   │   └── pagination.go            Page/PageSize
│   │   │   ├── errors/                      错误模型
│   │   │   │   ├── kind.go                  ErrKind 枚举
│   │   │   │   └── error.go                 Error + Wrap/Unwrap
│   │   │   ├── eventbus/                    进程内事件总线 (Publish-Subscribe)
│   │   │   │   ├── bus.go                   接口
│   │   │   │   ├── inproc_bus.go            channel + worker pool 实现
│   │   │   │   └── event.go                 Event 通用结构
│   │   │   └── tenantdb/                    多租户 DB 抽象
│   │   │       ├── platform_db.go           访问 public schema
│   │   │       ├── tenant_db.go             SET LOCAL search_path
│   │   │       └── pool_hook.go             RESET 防御性钩子
│   │   │
│   │   ├── infrastructure/                  通用基础设施 (无业务语义)
│   │   │   ├── pg/                          pgx pool
│   │   │   │   ├── pool.go
│   │   │   │   └── slow_query_hook.go       ★ v3.1: 记录 >200ms 的 SQL + 指标
│   │   │   ├── redis/
│   │   │   ├── minio/
│   │   │   ├── logger/                      zap 配置 + 脱敏
│   │   │   ├── metrics/                     Prometheus
│   │   │   └── health/                      ★ v3.1: 健康检查注册表 (依赖矩阵驱动)
│   │   │
│   │   ├── interface/                       HTTP 横切
│   │   │   ├── middleware/
│   │   │   │   ├── request_id.go
│   │   │   │   ├── recover.go
│   │   │   │   ├── logger.go
│   │   │   │   ├── cors.go
│   │   │   │   ├── trace.go
│   │   │   │   ├── rate_limit.go            ★ v3.1: 租户级 + 用户级限流 (Redis sliding window)
│   │   │   │   └── idempotency.go           ★ v3.1: Idempotency-Key 中间件 (Redis 24h TTL)
│   │   │   ├── apiresp/                     统一响应封装
│   │   │   └── server.go                    Gin engine 装配
│   │   │
│   │   ├── config/                          viper 加载
│   │   └── app/                             顶层 DI 装配
│   │       └── app.go                       构造所有服务/上下文, 启动 server
│   │
│   ├── pkg/                                 (谨慎扩张)
│   ├── api/openapi.yaml                     单一 OpenAPI 契约
│   ├── migrations/
│   │   ├── public/                          跨租户元数据 + 服务包公共表
│   │   └── tenant_template/                 新租户 schema 模板 (含 OKR 表)
│   ├── configs/
│   ├── test/integration/                    harness + 命门测试
│   ├── go.mod
│   └── Makefile
│
├── web/                                     ← Vue 3 前端 (单一 Vite 工程, 不上 workspace)
│   ├── package.json
│   ├── vite.config.ts / tsconfig.json / index.html
│   ├── src/
│   │   ├── main.ts / App.vue / env.d.ts
│   │   ├── shell/                           基座壳 (与业务无关)
│   │   │   ├── layout/                      AppLayout / NavBar
│   │   │   ├── auth/                        LoginView / auth.store / guard
│   │   │   ├── tenant/                      TenantSwitcher / tenant.store
│   │   │   ├── notify/                      NotifyCenter / notify.store
│   │   │   ├── workspace/                   工作台首页
│   │   │   └── components/                  Icon / AvatarBadge 等共享
│   │   ├── modules/
│   │   │   └── okr/                         OKR 工作安排 (M4 落地)
│   │   │       ├── routes.ts
│   │   │       ├── views/                   PlansView / ReportsView / RollupView
│   │   │       ├── components/
│   │   │       ├── stores/
│   │   │       └── api/
│   │   ├── api/
│   │   │   ├── client.ts                    axios + JWT + 401 跳登录
│   │   │   └── generated/                   openapi-typescript 输出 (gitignored)
│   │   ├── router/index.ts                  聚合 shell + modules
│   │   ├── styles/                          tokens.css + global.css
│   │   └── utils/
│   ├── tests/                               vitest
│   └── public/
│
├── deployments/
│   ├── docker-compose.yml                   开发: db + redis + minio + server + web
│   ├── docker-compose.prod.yml
│   ├── nginx/nginx.conf
│   ├── backup/                              ★ v3.1: 备份策略
│   │   ├── pg_backup.sh                     每日 pg_dump + 7 天保留
│   │   ├── minio_backup.sh
│   │   └── restore_runbook.md
│   └── kingbase/                            M5: KingbaseV8 适配
│
├── docs/
├── scripts/                                 dev.sh / seed.sh / openapi-gen.sh
├── legacy/                                  v1 资产 (Spring Boot + Element Plus), 仅作迁移参考
└── README.md
```

### 1.4 依赖方向规则 (CI 静态检查保护)

**有界上下文内 4 层** (与 v2.1.1 一致, 适用 contexts/<X>/):

1. `domain/` 不 import 任何同上下文外部 + 任何第三方框架; 只 import 标准库 + `shared/{kernel,errors}`
2. `application/` 只 import 同上下文 `domain/` + `shared/{kernel,errors,eventbus}`
3. `infrastructure/` 实现 `domain/` 接口; 可 import `shared/*` + `infrastructure/*`
4. `interface/` 只 import 同上下文 `application/` + `shared/*` + 全局 `interface/`

**服务包内** (services/<X>/, 单层包):

5. 服务包内不强制四层, 但仍要遵守: `service.go` 不直接调 DB 库, 而是通过同包 `Repository` 接口 (实现在 `pg_repo.go`)
6. 服务包之间允许通过明确导出的接口互相调用 (例如 `okr` application 调 `notification.Notifier`), 但**不允许**互相 import 内部结构体

**跨上下文 / 跨服务**:

7. 跨边界协作首选 `shared/eventbus/`; 同步调用通过服务包暴露的接口注入 (DI), 而不是直接 import
8. **禁止**: 跨 schema JOIN, 跨上下文 SQL JOIN
9. **禁止**: 有界上下文之间互相 import (M4 只有一个 OKR, 此规则为未来准备)

**前端**:

10. `shell/` 与 `modules/<X>/` 之间不互相 import (eslint 规则)
11. `modules/<X>/` 与 `modules/<Y>/` 不互相 import

### 1.5 部署形态

`deployments/docker-compose.yml` 含 5 个服务:
- `db` (postgres:16) — 生产换 KingbaseV8
- `redis` — Session 黑名单 + 缓存
- `minio` — 对象存储 (FileStorage 服务包用)
- `server` — Go 二进制
- `web` — nginx + dist

生产: 单一 nginx 服务静态资源 + 反代 `/api`。

---

## §2 多租户基础设施

> 这块**完整保留** v2.1.1 §2 的设计 — 两次专家评审都明确指出这是 spec 最值钱的部分。

### 2.1 核心思路

JWT 验签后, 从 claims 取出 `tenant_id`, 加载租户元数据写入 `ctx`; 业务代码经过 `TenantDB` wrapper 时, **在事务开启的同一连接上 SET LOCAL search_path** 切到对应 schema。

### 2.2 关键决策

1. **`SET LOCAL` 绑定到事务, 不是连接级**: 事务 COMMIT/ROLLBACK 自动回滚 search_path, 不可能漏 reset
2. **双 DB 句柄 `PlatformDB` + `TenantDB`**: 共用 pgxpool, 接口分离
   | 句柄 | 用途 | search_path |
   |---|---|---|
   | `PlatformDB` | `public` 表: tenant / platform_user / casbin_rule / dict / platform_audit | `public` |
   | `TenantDB` | `tenant_<slug>` 表: member / okr_plan / okr_report / audit_log / notification | `tenant_<slug>, public` |
3. **防御性 pool hook**: 连接归池前 `RESET search_path` (双保险)
4. **跨 schema JOIN 禁止**: 需要 public 数据时用 `PlatformDB` 单独查再 Go 里组装

### 2.3 Schema 生命周期

| 时机 | 触发 | 动作 |
|---|---|---|
| 开通租户 | `tenantctl create` 或 Admin API | 插 `public.tenant` → `CREATE SCHEMA tenant_<slug>` → 跑模板迁移 → 写迁移历史 |
| 升级业务表 | `tenantctl migrate-all` | 遍历 active tenant, 每个 schema 跑增量迁移 |
| 冻结租户 | Admin API | 仅改 status; schema 保留; 中间件拒绝 |
| 销户 | Admin API + 30 天保留 + 手动确认 | `DROP SCHEMA tenant_<slug> CASCADE` |

> v3.0 回到**中央 migrations** (`migrations/{public,tenant_template}/`), 不再每模块自带。理由: 当前没有"卸载模块"的真实需求, 中央迁移更容易维护和回滚。

### 2.4 命门测试 (CI gate, 任一挂掉不发版)

1. **隔离**: A 租户的数据, B 租户的 JWT 看不到
2. **污染**: 100 并发交替切租户, 连接归池后 search_path 已 reset
3. **越权**: 伪造 tenant_id 的 JWT 在 tenant 中间件被拒
4. **状态**: suspended 租户的合法 JWT → 403
5. **跨 schema 静态扫描**: 扫源码确认无跨 schema JOIN

> v2.1.1 的"DDD 边界扫描 / 模块隔离扫描 / 多产品装配冒烟"全部删除, 因为对应的过度设计也删了。

---

## §3 服务包详细设计 (轻量, 不上 DDD)

每个服务包按 "**类型 + Service + Repository + HTTP**" 三件套描述, 不展开聚合/事件等 DDD 概念。

### 3.1 Tenancy (租户)

**职责**: 租户元数据管理 (CRUD) + schema 开通 / 销毁。

**类型**:
- `Tenant{ ID, Slug, Name, Status, CreatedAt, SuspendedAt, ClosedAt }`
- `TenantMember{ PlatformUserID, TenantID, MemberID, JoinedAt, Status }`

**Service 方法**: `CreateTenant / SuspendTenant / ResumeTenant / CloseTenant / JoinMember / ListActiveTenants / GetTenant`

**关键决策**:
- `Slug` 全局唯一约束在 DB 层 (UNIQUE INDEX)
- `(PlatformUserID, TenantID)` 唯一约束
- 状态机校验放 `service.go` (不做聚合根)
- `SchemaProvisioner` 是独立类型, 执行 `CREATE SCHEMA tenant_<slug>` + 跑模板迁移

**接口**: `POST /tenants`, `PATCH /tenants/{id}`, `GET /tenants` (平台管理员)

### 3.2 IAM (身份与访问)

**职责**: 用户 / 会话 / RBAC; 平台用户 + 租户成员双层。

**类型**:
- `PlatformUser{ ID, Email, PasswordHash, MfaSecret, Status, LastLoginAt }`
- `Session{ ID, PlatformUserID, TenantID, MemberID, IssuedAt, ExpiresAt, Revoked }`
- `Role{ ID, TenantID, Code, Name, Policies[] }`

**Service 方法**: `Login / Refresh / Logout / SwitchTenant / Enforce / ListMyTenants / AddRole / AddPolicy`

**关键决策**:
- 本地账号密码 + JWT (RS256), IdentityProvider 接口预留 OIDC/LDAP
- Casbin 适配 RBAC, policy 存 `public.casbin_rule`
- Session 黑名单走 Redis
- 密码策略: bcrypt cost=12, 长度 10+ 字母 + 数字
- 内置角色: `platform_admin / tenant_admin / tenant_member`

**接口**:
- `POST /auth/{login,refresh,logout,switch-tenant}`
- `GET /me`, `GET /me/tenants`
- 中间件: `JWTAuth() / SessionCheck() / RBAC(resource, action)`

### 3.3 Audit (审计)

**职责**: 异步收集 + 持久化业务事件; 双账本 (租户内 + 平台跨租户)。

**类型**:
- `AuditLog{ ID, TenantID, Actor, Action, ResourceType, ResourceID, Detail JSONB, CreatedAt }`
- `PlatformAuditLog` (跨租户, 平台管理员操作)

**Service 方法**: `RecordEvent / FlushBuffer / ListLogs`

**关键决策**:
- 订阅 `shared/eventbus` 所有业务事件 → 自动记录
- 写入走 `BufferedWriter` (channel + worker pool), 业务事务不受审计写入影响
- buffer 满走同步 (不丢)

**接口**: `GET /audit/logs` (内部用)

### 3.4 Notification (通知)

**职责**: 站内信; 订阅业务事件触发通知。

**类型**:
- `Notification{ ID, RecipientMemberID, Type, Title, Body, Read, CreatedAt }`

**Service 方法**: `Send / MarkRead / ListUnread`

**关键决策**:
- 订阅 OKR 事件 (例: `okr.weekly_overdue`) 触发通知
- Channel 抽象 (InApp v1, Email/SMS v1.5+)
- 前端 20s 轮询 `/notifications/unread` (SSE 在 v1.5+ 再考虑)

**接口**: `GET /notifications/unread`, `POST /notifications/{id}/read`

### 3.5 FileStorage (文件存储)

**职责**: 通过 MinIO 中转的文件上传 / 下载; 通过 `BizRef` 关联业务。

**类型**:
- `Attachment{ ID, BizModule, BizID, ObjectKey, Name, Size, MimeType, UploaderMemberID, CreatedAt }`
- `BizRef{ BizModule string, BizID string }` (例: `("okr.plan", "<uuid>")`)

**Service 方法**: `Upload / Download / List / Delete`

**关键决策**:
- 后端中转 (业务事务可关联文件 ID), v1.5+ 评估 presigned URL 直传
- 单文件 50MB 上限
- ObjectKey 路径: `<tenant_slug>/<biz_module>/<biz_id>/<uuid>`

**接口**: `POST /files/upload`, `GET /files/{id}/download`, `GET /files/biz/{module}/{id}`

### 3.6 Dictionary (字典)

**职责**: 平台默认字典 + 租户级 override; 给所有"有限枚举" (类型 / 状态 / 分类) 提供统一查询。

**类型**:
- `DictType{ Code, Name }`
- `DictItem{ TypeCode, Code, Name, SortOrder, Active }`
- `DictOverride{ TenantID, TypeCode, ItemCode, Override JSONB }`

**Service 方法**: `Lookup(typeCode) → []Item`, `AddOverride / RemoveOverride / ReloadCache`

**关键决策**:
- 平台默认在 `public.dict_*`, 租户 override 在 `tenant_<slug>.dict_override`
- Redis 缓存 (10 分钟 TTL), `ReloadCache` 立即失效
- 业务代码统一走 `dict.Lookup("plan_level")`, 不硬编码常量

**接口**: `GET /dict/{typeCode}` (含租户 override)

### 3.7 Localization (i18n)

**职责**: 国际化文案查找; v1 从 YAML 加载, v1.5+ 加 DB 管理 UI。

**Service 方法**: `T(ctx, key, args...) → string`

**关键决策**:
- 语言包按模块前缀: `okr.plan.level.year` / `iam.error.invalid_password`
- ctx 携带 locale, 默认 `zh-CN`
- 缺失 key 返回 key 本身 (不报错, 不阻塞)

---

## §4 OKR 有界上下文 (真 DDD)

> 这是 M4 的核心, 团队学透 DDD 战术模式的样本上下文。其他业务模块未来按需引入时, **不必复刻这套模板** — 简单的就用 Active Record + Service。

### 4.1 领域语言 (Ubiquitous Language)

- **Plan (计划)**: 时间维度的目标集合, 4 个级别 `Year / HalfYear / Month / Week`
- **PlanItem (条目)**: Plan 内的具体目标项, 含 owner / due_date / weight / status
- **Decompose (分解)**: 上级 Plan 拆出下级 Plan 的关系 (Year → HalfYear → Month → Week)
- **Report (报告)**: 用户提交的工作汇报, 类型 `Daily / Weekly`
- **ReportEntry (报告条目)**: 报告内一行, 可选关联 PlanItem
- **Rollup (汇总)**: 管理人员对下属周报的聚合视图
- **Cadence (节奏)**: 周期日历, 决定每周 / 月 / 半年 / 年的起止日期

### 4.2 聚合

修正 v2.1.1 的"聚合根爆炸"问题, 严格遵守 Vernon "IDDD" 的 Small Aggregates 原则:

#### Plan 聚合

- **聚合根**: `Plan`
- **包含**: `PlanItem` (实体, 不独立保存, 通过 `plan.AddItem()` 修改)
- **属性**: `ID, Level, OwnerMemberID, PeriodStart, PeriodEnd, Title, ParentID(可空), Items[], Status`
- **不变量**:
  - `PeriodStart < PeriodEnd`
  - 同 owner + level + period 全局唯一
  - PlanItem.weight 总和 ≤ 100
  - Status 转换: `draft → active → closed` (不可逆)
  - ParentID 必须指向更高级别的 Plan (Year > HalfYear > Month > Week)

#### Report 聚合

- **聚合根**: `Report`
- **包含**: `ReportEntry` (实体)
- **属性**: `ID, Type, OwnerMemberID, PeriodStart, PeriodEnd, Entries[], SubmittedAt, ReadBy[]`
- **不变量**:
  - Daily Report period = 1 天
  - Weekly Report period = 7 天 (跨周一到周日)
  - SubmittedAt 一旦设置不可改

#### 跨聚合引用

- `ReportEntry.PlanItemID` (可空) 通过 ID 引用 PlanItem, 不持有 PlanItem 对象
- 应用层负责跨聚合一致性 (例: 报告关联了不存在的 PlanItem → 应用层校验)

### 4.3 值对象

- `PlanLevel` 枚举: `Year / HalfYear / Month / Week`
- `ReportType` 枚举: `Daily / Weekly`
- `Period{ Start, End time.Time }` — 不可变
- `Progress{ Percent int, Summary string }` — Percent 0-100

### 4.4 领域服务

- **`Cadence`**: 给定 (level, date) 计算该周期的 `(start, end)`; 跨年 / 季 / 月边界正确处理
- **`PlanDecomposer`**: 校验 child plan 的 period 与 weight 是否 fit parent
- **`RollupView`**: 给定 (managerMemberID, periodEnd) 聚合下属 Weekly Reports → `RollupReport` 视图 (Query 侧, 直查 DB, 不经过聚合)

### 4.5 领域事件

- `PlanCreated{ PlanID, Level, OwnerMemberID, PeriodStart, PeriodEnd }`
- `PlanItemAdded{ PlanID, ItemID, Title }`
- `PlanItemCompleted{ PlanID, ItemID, Progress }`
- `PlanClosed{ PlanID, ClosedAt }`
- `DailySubmitted{ ReportID, OwnerMemberID, PeriodEnd }`
- `WeeklySubmitted{ ReportID, OwnerMemberID, PeriodEnd }`
- `WeeklyOverdue{ OwnerMemberID, PeriodEnd }` (cron 触发)

**事件命名**: 全部使用 topic `okr.<event_name>`, 全局唯一。

### 4.6 应用服务 (用例)

- `CreatePlan(cmd) → Plan` — 校验 + 发 `PlanCreated`
- `AddPlanItem(cmd) → ()`
- `CompletePlanItem(cmd) → ()`
- `ClosePlan(cmd) → ()`
- `DecomposeFrom(parentID, childLevel) → Plan` (复制 PlanItems 为草稿)
- `SubmitDailyReport(cmd) → Report`
- `SubmitWeeklyReport(cmd) → Report`
- `ListMyPlans(period, level) → []Plan` (Query)
- `RollupWeekly(managerID, periodEnd) → RollupReport` (Query, 直查 DB)
- `CommentReport(reportID, comment) → ()`
- `RemindOverdue(periodEnd) → ()` — cron 触发, 通过 `notification.Notifier` 接口发提醒

### 4.7 基础设施

- `pgPlanRepo` / `pgReportRepo` — 实现 domain 仓储接口
- `pgRollupQuery` — 直查 DB 投影 DTO (CQRS Query 侧的轻量实践)
- `cron` 调度: 每周一 9:00 跑 `RemindOverdue`; 用 `robfig/cron/v3`

### 4.8 接口

- `GET /plans?level=&period=` / `POST /plans` / `PATCH /plans/:id`
- `POST /plans/:id/items` / `PATCH /plans/:id/items/:itemId/complete`
- `POST /reports/daily` / `POST /reports/weekly`
- `GET /reports?type=&period=`
- `GET /rollups/weekly?period=&dept=` (要求 manager 角色)
- `POST /reports/:id/comments`

---

## §5 跨上下文 / 跨服务协作

### 5.1 事件总线 (Publish-Subscribe)

修正 v2.1.1 误用的"Customer-Supplier / Open Host Service"等术语, 统一称作 **Publish-Subscribe**:

- OKR (Publisher) 发 `okr.weekly_submitted` 等事件
- Audit (Subscriber) 订阅所有事件, 写审计
- Notification (Subscriber) 订阅特定事件, 触发通知
- 未来上下文 (Subscriber) 按需订阅

**eventbus 接口**:

```go
type Bus interface {
    Publish(ctx context.Context, topic string, data any) error
    Subscribe(topic string, handler Handler)
}

type Event struct {
    Topic      string
    OccurredAt time.Time
    TenantID   string
    Actor      string
    TraceID    string
    Data       any
}
```

v1 实现: 进程内 channel + worker pool。

**topic 命名规则**:
- `<source>.<event_name>`, 如 `okr.plan_created`, `iam.user_logged_in`, `tenancy.tenant_created`
- 全局唯一, 写入 `docs/architecture/event-catalog.md` 注册表

### 5.2 同步调用 (服务间)

- OKR application 需要发通知时, 通过 DI 注入 `notification.Notifier` 接口 (定义在 `services/notification/`)
- OKR application 需要查字典时, 注入 `dictionary.Lookuper`
- 不允许 import `services/notification` 内部类型, 只能 import 接口

### 5.3 反例 (避免做的事)

| 反例 | 为什么 | 正确做法 |
|---|---|---|
| `contexts/okr/domain/` import `services/iam/` | 跨层 import 破坏 domain 纯净 | application 层注入 IAM 接口 |
| `contexts/okr/domain/plan.go` 写 SQL | domain 不该感知 DB | 仓储接口在 domain, 实现在 infrastructure |
| 用 GORM hook 写审计 | 业务和审计强耦合 | domain 发事件, audit 订阅 |
| 跨 schema JOIN | 强耦合, 拆分难 | 拉两份再 Go 里组装 |
| 把 audit 写入放业务事务里 | 审计失败会回滚业务 | 业务事务发"待发事件", commit 后异步 publish |

---

## §6 横切关注点

### 6.1 错误模型

```go
type Error struct {
    Kind    ErrKind  // 业务/网络/数据库/权限/参数/未知
    Code    string   // 业务码: "okr.plan.invalid_period"
    Message string   // 给用户的中文
    Cause   error    // 底层错误
    TraceID string
}
```

错误码命名: `<source>.<resource>.<reason>`。错误码与 i18n key 一一对应。

### 6.2 结构化日志 (zap)

- 字段统一: `trace_id, tenant_id, member_id, action, latency_ms, error_code`
- 敏感字段脱敏中间件 (password / token / email)
- 生产环境 INFO+, 开发 DEBUG+

### 6.3 链路追踪

v1 只有 `trace_id` (RequestID 中间件生成 + 贯穿 ctx 全链路 + 日志字段)。grep trace_id 调试足够; OpenTelemetry 推迟。

### 6.4 测试策略

| 层 | 工具 | 覆盖 |
|---|---|---|
| **领域单测** (OKR domain) | testify | 100% (聚合不变量 / 领域服务 / 值对象) |
| **应用单测** (OKR application) | testify + mock repo | 用例编排; mock infra |
| **服务包单测** | testify + mock repo | service.go 业务逻辑 |
| **集成测试** | testify + docker PG | 仓储实现 + HTTP 接口 + 命门测试 |
| **前端单测** | vitest + Vue Test Utils | 组件 / store / 路由 guard |
| **E2E** | Playwright (M5+) | 5 个关键流程 (登录 / 建计划 / 提交日报 / 看汇总 / 切租户) |

### 6.5 命门测试 (CI gate)

见 §2.4, 5 条。

### 6.6 租户级限流 (v3.1)

**为什么必要**: 多租户最大的"邻居噪音"风险——单个失控租户 (恶意爬虫 / 客户脚本 bug / 业务峰值) 把所有租户拖垮。

**实现** (`interface/middleware/rate_limit.go`):

- **算法**: Redis sliding window (用 `redis-cell` 模块或自行实现); 不用 token bucket 是因为爆发容忍较弱
- **维度** (三层独立计数):
  - **租户级**: 默认 1000 req/min/租户; 走 `dictionary` 可按租户 override
  - **用户级**: 默认 60 req/min/用户; 防爬虫 + 防误操作
  - **公开端点**: 登录 / 公共分享, 默认 20 req/min/IP
- **响应**: 触发后 `429 Too Many Requests` + `Retry-After` 头; 写入 audit
- **白名单**: 平台管理员 token 不限流 (审计仍记录)

**配置**:
```yaml
rate_limit:
  tenant_default: 1000
  user_default: 60
  ip_login: 20
  storage: redis
```

**演化路径**: v1 单实例 Redis 够用; v1.5+ 评估是否分片。

### 6.7 幂等性 (v3.1)

**为什么必要**: 用户网络抖动 / 前端重试 / 客户脚本重发, 都会造成业务重复 (重复扣减 / 重复审计 / 重复通知)。

**实现** (`interface/middleware/idempotency.go`):

- **覆盖范围**: 所有 `POST` / `PATCH` / `DELETE` 路由
- **Key 来源**: 客户端必须传 `Idempotency-Key` header (UUID v4); 中间件强校验, 未传 → `400`
- **Key 维度**: `(tenant_id, member_id, idempotency_key)` 三元组, 防跨用户冲突
- **存储**: Redis SET, TTL 24 小时, 值 = 首次响应的 (status, body) JSON
- **行为**:
  - 首次请求: 执行, 把响应缓存
  - 重复请求 (24h 内): 直接返回缓存的响应, **不重新执行业务**
  - 业务执行中重复请求: 等待 (max 5s) 或返回 `409 Conflict`
- **白名单**: GET / 内部 webhook 不走幂等

**前端约定**: `@iop/api-client` 自动为所有非 GET 请求注入 UUID v4 Idempotency-Key (用户无感)。

### 6.8 慢查询防护 (v3.1)

**为什么必要**: 多租户 + DDD 多层抽象, 极易写出 N+1 或全表扫; 一条慢 SQL 拖垮共享 PG 池, 全体租户感知。

**实现** (`infrastructure/pg/slow_query_hook.go`):

- **pgx tracer hook**: 每条 SQL 执行后记录 `query_duration_ms`
- **阈值**:
  - `> 200ms`: WARN 日志 + Prometheus `pg_slow_query_total{kind="warn"}` 计数
  - `> 1000ms`: ERROR 日志 + 报警 (M5 接入告警系统时联动)
- **日志字段**: `sql_template (脱敏), tenant_id, duration_ms, rows_affected, caller`
- **保护性强约束**:
  - 所有 `SELECT * FROM <租户表>` 必须带 `LIMIT` 或显式分页 (lint 规则)
  - 仓储接口的 `List` 方法必须接受 `Pagination`, 不允许返回无界结果
  - 多租户表的索引设计审查清单 (写入 `docs/architecture/db-index-checklist.md`)
- **N+1 检测**: 单请求内同一 SQL 模板执行 > 10 次 → WARN

**Grafana 面板** (M5): 慢查询 Top 20、按 tenant 分布、按时间段分布。

### 6.9 健康检查依赖矩阵 (v3.1)

**为什么必要**: Kubernetes 时代的 livez/readyz 语义混乱是常见生产故障源。v3.0 只说"有 livez/readyz", 没说"Redis 挂了 readyz 返什么"。

**明确语义**:

| 端点 | 含义 | 失败动作 |
|---|---|---|
| `/livez` | 进程活着 | 进程崩溃 → 重启 |
| `/readyz` | 进程能处理"主路径"业务请求 | 失败 → 流量摘除 (load balancer 不转发) |
| `/healthz` | 内部诊断, 各依赖详细状态 (内部网络访问) | 仅用于人工诊断, 不影响流量 |

**依赖矩阵** (是否进 readyz 检查):

| 依赖 | livez | readyz | 失败时业务影响 |
|---|---|---|---|
| 进程自身 | ✅ | ✅ | 全停 |
| PG (PlatformDB) | ❌ | ✅ | 全停; 登录 / 租户加载都依赖 |
| PG (TenantDB) | ❌ | ✅ | 同上 (共用 pool) |
| Redis | ❌ | ⚠️ **部分** | Session 黑名单 / 限流 / 字典缓存依赖; 标记降级模式但不下流量 |
| MinIO | ❌ | ❌ | 仅文件上传/下载受影响; readyz 不关心 |
| SMTP (邮件) | ❌ | ❌ | 通知降级到站内信; v1 noop 默认 |

**降级策略**:
- Redis 挂 → 限流降级到内存计数 (单实例 OK, 多实例会失准, 记 WARN); 字典缓存直接穿透 PG (慢但能用)
- MinIO 挂 → 文件接口返 `503 Service Unavailable` + 明确错误码, 其他接口照常

**实现** (`infrastructure/health/registry.go`):
```go
type Check struct {
    Name        string
    Critical    bool                       // true → 进 readyz; false → 仅 healthz
    Check       func(ctx) error
    Degradation func()                     // 失败时执行的降级动作 (可空)
}
```

### 6.10 数据备份与恢复 (v3.1, M2 起就位)

**为什么必要**: v3.0 把备份放在 M5, 但**生产事故不等到 M5**; M2 一旦有真实租户登录, 就该有最低限度备份。

**最低标准** (M2 落地):

1. **PG 每日全备**: `pg_dump` 凌晨 3 点, 输出到独立卷
2. **保留期**: 7 天滚动 + 每周一份月备 (1 年)
3. **异机存储**: 备份文件每日同步到对象存储 (MinIO 单独 bucket, 或客户云上 S3)
4. **完整性校验**: 每次备份后跑 `pg_restore --list` 校验文件可读
5. **恢复演练**: M5 前至少 1 次端到端恢复 (在 staging 用最新备份还原, 跑命门测试)

**恢复 runbook** (`deployments/backup/restore_runbook.md`): 必须包含
- 备份文件位置 + 加密密钥位置 + 责任人
- 单租户级恢复步骤 (从全备拆出某 `tenant_<slug>` schema, 不影响其他租户)
- 全库恢复步骤 (灾难场景)
- RTO 目标: < 4 小时; RPO 目标: < 24 小时 (单实例非 HA 的合理值; v1.5+ HA 后改 RPO < 1h)

**v1.5+ 升级路径**:
- pg_basebackup + WAL 归档 → RPO < 5 分钟
- 主从复制 + 自动 failover → RTO < 30 分钟

---

## §7 里程碑路线图 (3 人团队, 现实节奏)

> 与 v2.1.1 的关键差异: M4 收敛到 OKR 一个业务模块 + 给足 10 周时间。其他业务模块全部推迟到 v1.5+。

### M1 — 骨架 + 基础设施 (~4 周)

**可交付**:
- `cmd/server` + `cmd/migrate` + `cmd/tenantctl`
- `shared/{kernel,errors,eventbus,tenantdb}` 全部
- `interface/middleware` 全链 (RequestID / Recover / Logger / CORS / Error)
- `services/{dictionary,localization}` 骨架 (供 M4 业务用)
- 初始 OpenAPI (`/livez /readyz /version /metrics`)
- `migrations/public/000001` 基础表 (含 `migration_history`)
- `web/` 单一 Vite 工程骨架 (shell 目录)
- `deployments/docker-compose.yml` 5 服务起得来
- `Makefile`: build/test/lint/run/migrate/openapi-gen

**验收**:
- `make dev` 一键起; livez/readyz/metrics 工作正常
- 故意 panic 不杀进程
- 日志含 trace_id 贯穿
- CI 跑通

### M2 — Tenancy + IAM + 命门测试 + B2B SaaS 基本盘 (~6 周, v3.1 比 v3.0 +1 周)

**可交付**:
- `services/tenancy` 完整 (Tenant + TenantMember + SchemaProvisioner)
- `services/iam` v1: 本地密码 + JWT + Session + 基础 RBAC
- `cmd/tenantctl`: create/suspend/resume/close/migrate-all
- `migrations/public/000002`: tenant / platform_user / membership
- `migrations/tenant_template/000001`: member
- `test/integration/`: harness + **5 个命门测试**
- `web/`: LoginView + auth.store + guard + TenantSwitcher + api/client
- ★ **v3.1 增**: `interface/middleware/rate_limit.go` (租户/用户/IP 三级 Redis 滑动窗口)
- ★ **v3.1 增**: `interface/middleware/idempotency.go` (Idempotency-Key 中间件, Redis 24h)
- ★ **v3.1 增**: `infrastructure/pg/slow_query_hook.go` (>200ms 记录 + 指标)
- ★ **v3.1 增**: `infrastructure/health/` (livez/readyz/healthz 三端点 + 依赖矩阵)
- ★ **v3.1 增**: `deployments/backup/` (pg_dump 每日 + 7 天保留 + 恢复 runbook v0.1)

**验收**:
- `tenantctl create --slug=acme` 后 schema 自动建好
- 登录拿 JWT 正确
- 5 个命门测试全绿
- 前端 F5 保持登录 / 401 跳登录 / 切租户后 URL 不变数据变
- **M2 末期一次轻量 KingbaseV8 嗅探** (拿命门测试 + auth 流程跑 KingbaseV8, 提前发现兼容差异)
- ★ **v3.1 增**: 限流测试 (单租户超阈值 → 429 + Retry-After); 幂等性测试 (同 Key 重发返回缓存响应)
- ★ **v3.1 增**: 慢查询 hook 触发测试 (人工注入 1s SQL, 验证日志 + 指标)
- ★ **v3.1 增**: 关掉 Redis 后 readyz 返**降级** (而不是 503), MinIO 关掉后 readyz 仍 200
- ★ **v3.1 增**: 备份恢复一次端到端演练 (在 staging 用今日备份还原, 跑命门测试)

### M3 — Audit + Notification + FileStorage + Dictionary (~4 周)

**可交付**:
- `services/audit` 完整 (含 BufferedWriter + 订阅 eventbus)
- `services/notification` 完整 (InApp 渠道, SMTP v1 noop)
- `services/filestorage` 完整 (MinIO + BizRef)
- `services/dictionary` 完整 (Redis 缓存 + 平台默认 seed)
- `services/iam` 扩展: 完整 RBAC + 角色 / 策略管理 API
- `migrations/public/000003`: casbin_rule
- `migrations/tenant_template/000002`: audit_log / notification / attachment / dict_override
- `web/`: NotifyCenter (20s 轮询) + 共享组件

**验收**:
- 无权角色被拒
- Casbin policy 改后立即生效
- 审计异步可见; buffer 满走同步不丢
- 文件上传 / 下载流程通

### M4 — OKR 闭环 (~10 周, 学透 DDD 的核心里程碑)

**可交付**:
- `contexts/okr/` 完整 (domain + application + infrastructure + interface)
- `migrations/tenant_template/000003`: okr_plan / okr_plan_item / okr_report / okr_report_entry / okr_rollup_view (5-6 张表)
- 领域单测 100% 覆盖聚合不变量
- 应用单测覆盖全部用例
- 集成测试覆盖端到端
- `web/src/modules/okr/`: PlansView / ReportsView / RollupView + 完整交互
- 种子数据: 6 个用户 + 4 级计划样例 + 2 周报告样例
- cron 定时器: 每周一 9:00 跑 RemindOverdue
- 关键文档: i18n key 命名规约 (`okr.*` 前缀)

**验收**:
- 4 级计划创建 + 分解 + 完成全流程通
- Daily / Weekly Report 提交; 跨周汇总视图正确
- 未提交周报自动通知
- Notification 收到 OKR 事件后正确发通知 (验证事件订阅链路)
- 越权: tenant_a 的 token 访问 tenant_b 的 plan URL → 404
- DDD 边界: domain 不依赖任何 infrastructure (`go list -deps` 验证)

**学习目标** (与代码交付同等重要):
- 团队亲手实现 Plan 聚合 + 不变量 + 领域事件 + 领域服务
- 团队体感到 `domain → application → infrastructure → interface` 四层各自的"为什么"
- 团队在 M4 末写一份 1-2 千字的"OKR 上下文 DDD 实战回顾", 沉淀经验

### M5 — 生产部署 + KingbaseV8 验证 (~3 周)

**可交付**:
- `deployments/` prod compose + Nginx HTTPS + 限流 + WAF + KingbaseV8 适配 + runbook
- `docs/operations/`: backup-restore + tenant-lifecycle + observability + incident-playbooks
- `docs/developer/`: getting-started + okr-walkthrough + adding-new-service (服务包) + adding-new-context (有界上下文)
- `docs/architecture/`: overview + context-map + event-catalog + 10 个 ADR
- `.github/workflows/release.yml`: tag 触发构建镜像
- `web/e2e/`: 5 个 Playwright 用例

**验收**:
- staging 一键部署 HTTPS 可访问
- **KingbaseV8 跑命门测试 + OKR 全套测试全绿**
- 备份恢复演练通
- E2E 5 个用例 CI 跑通

### 工作量与节奏

```
M1 (4w) → M2 (6w) → M3 (4w) → M4 (10w) → M5 (3w)
```

合计 ~27 周 (~6 月), 假设 3 人全栈团队连续投入。v3.1 比 v3.0 +1 周 (M2 多了 SaaS 基本盘 5 项), 但避免了"上线半年后才发现没限流被打挂"的真实事故, 是高 ROI 的投资。

**每个里程碑结束都是可发布点**:
- M1 → dev 环境可访问
- M2 → 内部联调可建租户
- M3 → 内部用户可见看板壳 (无业务)
- M4 → **首个真实租户可灰度试用 OKR**
- M5 → 正式上线

### v1.5+ 路线 (不在 v3.0 范围, 按真实业务需求触发)

- **第 2 个业务模块进场时**: 决定是否抽 `Module` 接口 (Khononov 原则: 等需求确定再抽象)
- **客户要 Problem 模块时**: 沿 OKR 模板做第 2 个有界上下文
- **客户要 CRUD 类模块** (Contacts/Documents/Assets/Announcements): 各加一个服务包, 不要 DDD
- **客户要桌面端**: 评估 Tauri 抽象层; 那时业务代码也成熟, 知道该抽什么
- **租户需要模块开关**: 设计 AppCenter
- **第 2 个产品形态出现**: 拆 `cmd/<product>`

---

## §8 关键决策 (精简版 ADR)

修正 v2.1.1 用 33 条 ADR "决定未来"的过度做法。v3.0 只列**已做的二选一**:

| # | 决策 | 选择 | 替代方案 / 理由 |
|---|---|---|---|
| 1 | 架构定位 | Go 模块化单体 | vs 微服务 (运维代价不值得; 拆分需求未到) |
| 2 | DDD 适用范围 | 仅 OKR 上下文; 其他全部服务包 | vs 全部 DDD (Khononov: 通用 / 支持子域用 DDD 战术是反模式) |
| 3 | 租户隔离 | PG 每租户独立 schema + SET LOCAL | vs row-level (查询复杂; SCHEMA 隔离更直观) |
| 4 | 用户 × 租户 | 双层 (PlatformUser + TenantMember) | 支持账号跨租户复用 |
| 5 | 鉴权 | 本地账号 + JWT + Casbin RBAC | OIDC/LDAP 通过 IdentityProvider 接口预留 |
| 6 | 事件总线 | 进程内 channel + worker pool | vs NATS (削峰未到; 进程内调试容易) |
| 7 | 错误处理 | Kind + Code + i18n key 三元组 | handler 不直接写响应 |
| 8 | 日志 | zap 结构化 + 脱敏 + trace_id | vs OTel (grep 够用) |
| 9 | 测试 | testify + 本地 docker PG + 5 命门 | 不用 testcontainers (Docker 客户端 API 兼容问题) |
| 10 | 部署 | docker-compose dev + prod + Nginx + 单二进制 | k8s 推迟 |
| 11★ | 租户级限流 (v3.1) | Redis 滑动窗口, 租户/用户/IP 三级独立 | vs token bucket (爆发容忍较弱不适合多租户) / 内存计数 (多实例失准) |
| 12★ | 幂等性 (v3.1) | 全部非 GET 请求强制 Idempotency-Key 头, Redis 24h 存响应 | vs 业务侧自行去重 (重复实现 + 易漏) / 不做 (生产事故必发) |
| 13★ | 慢查询防护 (v3.1) | pgx hook + >200ms 记录 + 强制分页 + N+1 检测 | vs APM 工具 (引入成本高) / 不做 (多租户必踩) |
| 14★ | 健康检查依赖矩阵 (v3.1) | livez (进程) / readyz (主路径依赖) / healthz (诊断) 三端点, 依赖矩阵明确 | vs 笼统的 /health (k8s 时代常见故障源) |
| 15★ | 备份策略前移到 M2 (v3.1) | pg_dump 每日 + 7 天 + 异机, M2 起就位 | vs M5 才做 (生产事故不等里程碑) |

★ = v3.1 在 v3.0 基础上新增的 B2B SaaS 基本盘决策。

---

## §9 已知约束与推迟项

### 已知约束

- **单实例**: PG / Redis 单实例 (200 租户 OK); 高可用 M5+
- **数据库**: 开发 PG 16; 生产 KingbaseV8 PG 兼容模式 (M2 末期嗅探确认)
- **业务模块**: v3.0 仅 OKR; 其他模块按真实需求逐个解锁, 不预设
- **MFA / SSO / 富文本 / 移动端 / 开放 API / SSE / 流程编辑**: v3.0 范围外
- **审计保留期**: 默认 7 年, 由后续合规决策细化
- **租户销户**: 30 天保留期 + 手动确认
- **团队**: 假设 3 人全栈; 增减人数对里程碑长度成比例影响
- ★ **v3.1**: 后台 cron (RemindOverdue / 慢查询采样等) 假定**单 server 实例**; 真要部署 ≥ 2 实例时必须切到 distributed cron, 否则定时任务重复触发 (例: 周一 9:00 推送 100 个用户 × N 实例 = N 倍消息)
- ★ **v3.1**: 限流降级为内存计数后 (Redis 挂场景), 多实例环境下计数不准, 仅作"半保护" — 短期可接受, 但必须**告警 + 监控 + 尽快恢复 Redis**
- ★ **v3.1**: 备份 v1 假定单实例 PG; v1.5+ 主从架构后, 备份策略需重新设计 (从从库备份避免影响主库)

### 明确推迟

按真实业务需求触发, 不预先建目录 / 不预先写 ADR / 不预先抽接口:

| 项 | 触发条件 | 预估代价 |
|---|---|---|
| Problem 模块 | 客户要"跨部门多阶段办理" | 4-6 周 (沿 OKR 模板) |
| Contacts / Documents / Assets / Announcements | 客户分别要 | 2-3 周 / 个 (服务包级别) |
| Party 模块 | 客户付费要求 + 法律合规评估通过 | 4-6 周 + 合规 |
| AppCenter (租户模块开关) | 至少 3 个业务模块 + 租户要自助开关 | 2 周 |
| `Module` 接口 + Registry | 至少 3 个业务模块 + 强烈的 DI 痛感 | 1 周 |
| `cmd/<product>` 多产品 | 真有第 2 个产品 | 半天 |
| pnpm workspace | ≥ 3 个前端模块 + 真互相干扰 | 1 周 |
| Tauri 抽象层 + 桌面端 | 客户要桌面版 | 4-6 周 (含签名公证) |
| 在线聊天 | 客户要 | 4-6 周 (含 SSE/WS 基础设施) |
| NATS 异步消息 | 真实削峰信号 | 2 周 |
| OpenTelemetry | grep trace_id 不够用 | 1 周 |
| 分布式 cron (leader election) | server 部署 ≥ 2 实例 | 1 周 (用 `go-zookeeper` 或 PG advisory lock) |
| PG 主从 + 自动 failover | RPO/RTO 需求收紧 (v1.5+) | 2 周 |
| WAL 归档 + PITR (Point-in-Time Recovery) | RPO < 5 分钟需求 | 1 周 |

---

## 附录 A: v2.1.1 → v3.0 删减明细

### 完整删除

- 多产品组装 (`cmd/iop-full` / `cmd/gallant`) — 单 `cmd/server`
- `Module` 接口 + `Registry` + `Deps` — 直接 DI, 不抽象
- 模块自带 `migrations/{platform,tenant}/` — 回到中央迁移
- `migration_history.module_name` 列 — 不需要
- pnpm workspace + `packages/` + `apps/` — 单 Vite 工程
- 桌面端 `@iop/shell/platform/*` 抽象层 + `@iop/platform-web` + `desktop/` — 全删
- AppCenter 占位 — 不建目录
- 7 个业务模块 (Problem / OKR 之外 6 个 + Party) → 仅 OKR
- 模块隔离 CI gate / 多产品装配冒烟 — 不再需要
- `embed.FS` 迁移 — 中央迁移就近文件即可
- IAM PersonalProfile 完整子节 — 推迟到客户要再做
- 33 个 ADR 中预测性条目 — 砍到 10 个真实决策

### 修正

- "Customer-Supplier / Open Host Service / Shared Kernel" 等误用术语 → 统一改 **Publish-Subscribe**
- Audit / Notification / FileStorage / Dictionary / Localization / PlatformConfiguration 从 "Bounded Context" 降级为 "服务包" (无 domain 层)
- Problem 聚合的 Measure / Dispute / Evaluation 不再标"独立聚合根" (M4 不做 Problem, 暂搁置; 未来做时按 Vernon 原则: 强属于 Problem 的实体)
- "Module 接口是 Published Language" 的混淆描述 → 删除 (Module 接口是工程契约, 与领域 Published Language 是两回事)

### v3.0 → v3.1 (补 B2B SaaS 基本盘 6 项)

参照公开资料里 Slack 早期 / Notion / Linear / Jira / GitHub Enterprise 等中等规模 B2B SaaS 的共性, v3.1 补齐以下"看似细节但生产必踩"的 6 项:

| 项 | 位置 | 落地里程碑 | 必要性 |
|---|---|---|---|
| 租户级限流 | §6.6, `middleware/rate_limit.go` | M2 | ★★★★★ 多租户最大的"邻居噪音"风险 |
| 幂等性 | §6.7, `middleware/idempotency.go` | M2 | ★★★★★ 网络抖动 / 前端重试必导致重复 |
| 慢查询防护 | §6.8, `pg/slow_query_hook.go` | M2 | ★★★★ 多租户 + DDD 抽象易踩 N+1 |
| 健康检查依赖矩阵 | §6.9, `infrastructure/health/` | M2 | ★★★★ k8s 时代故障源 |
| 备份策略前移 | §6.10, `deployments/backup/` | M2 (而非 M5) | ★★★★★ 生产事故不等里程碑 |
| 分布式 cron 路径 | §9 推迟项 + §6 备注 | M5+ | ★★★ 单实例期不急, 多实例时必备 |

**对比微信/支付宝/飞书 — 我们学了什么、没学什么**:
- **学**: "超时级联控制" 思想 (内部调用每层超时递减) — v1.5+ 引入 gRPC 内部调用时落地
- **学**: "业务/审计/通知分离" 思想 (支付宝 Saga 的轻量版) — v3.0 已有
- **学**: "多级缓存" 常识 (Redis + 本地 cache) — Dictionary 已用
- **不学**: 单元化 / LDC / 多机房多活 (规模差 4-5 个数量级)
- **不学**: 自研 RPC / 存储引擎 (团队规模差 100x)
- **不学**: 微服务拆分 (3 人团队拆 = 自杀)
- **不学**: 强一致分布式事务 (用最终一致 + 事件总线足够)

### 保留 (v2.1.1 真正值得保留的部分)

- PG 每租户 schema 隔离 + `SET LOCAL search_path` + pool RESET hook
- 5 个命门测试 (隔离 / 污染 / 越权 / 状态 / 跨 schema)
- JWT + tenant claim + 中间件加载租户上下文
- 错误模型 (Kind + Code + i18n)
- 字典驱动有限枚举
- 双 DB 句柄 (PlatformDB + TenantDB)
- 审计异步 buffered writer + 订阅事件
- trace_id 贯穿 + zap 结构化日志
- Casbin RBAC + Session Redis 黑名单
- OpenAPI 自动生成 TS SDK

---

## 附录 B: 学习路径建议 (给学习者的)

1. **先读 Khononov "Learning Domain-Driven Design" 第 10 章** (Designing Modules) — 理解为什么不是所有子域都用 DDD
2. **再读 Vernon "Implementing Domain-Driven Design" 第 10 章** (Aggregates) — 理解 Small Aggregates 原则, 避免聚合根爆炸
3. **M1-M3 期间**: 重点是把基础设施跑通, 不要纠结"是不是 DDD"; 这阶段所有服务包都不用 DDD
4. **M4 期间**: 在 OKR 上**严格**实践 DDD 四层 + 聚合 + 领域事件, 写完后回看哪些是真有用、哪些是仪式
5. **M5 之后**: 团队对 DDD 有了真实手感, 才回看是否要给下一个业务模块用 DDD; 凭手感判断, 不凭目录模板

---

**文档结束。**
