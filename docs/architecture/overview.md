# IOP 架构总览 (v3.1)

> 这是给新加入团队的开发者快速建立心智模型的文档.
> 完整 design spec: `docs/superpowers/specs/2026-05-28-go-base-design-v3.md`

## 一句话架构

**Go 模块化单体 + Vue 单页应用 + PG 每租户独立 schema + 进程内事件总线**

单一后端二进制 (`cmd/server`) + 单一前端 SPA. 通过 docker-compose 编排.

## 物理拓扑

```
┌──────────────────────────────────────────────────────────────┐
│                          Nginx                                │
│   (HTTPS 443 + per-IP rate limit + 静态资源 + /api 反代)       │
└──────────────┬──────────────────────────┬───────────────────┘
               │                          │
               ▼                          ▼
        ┌─────────────┐           ┌─────────────┐
        │  Web (Vite  │           │   Server    │
        │   dist via  │           │  (Go bin)   │
        │   nginx)    │           │             │
        └─────────────┘           └──┬──────────┘
                                      │
                          ┌───────────┼─────────────┐
                          ▼           ▼             ▼
                     ┌────────┐  ┌────────┐    ┌────────┐
                     │  PG    │  │ Redis  │    │ MinIO  │
                     │ (1 实例)│  │ (1实例)│    │ (1实例)│
                     └────────┘  └────────┘    └────────┘
                          │
                          ▼
                     ┌─────────┐
                     │ Backup  │ (daily pg_dump)
                     └─────────┘
```

## 逻辑分层 (Go 后端)

```
┌─ contexts/                有界上下文 (业务核心) — 真 DDD
│  └─ okr/  domain ▸ application ▸ infrastructure ▸ interface
├─ services/                服务包 (通用/支持子域) — Active Record + Service
│  ├─ tenancy/              租户 + member + SchemaProvisioner
│  ├─ iam/                  PlatformUser + Session + Role + JWT + RBAC
│  ├─ audit/                异步 buffered writer + bus 订阅
│  ├─ notification/         站内信 + 业务事件订阅
│  ├─ filestorage/          MinIO + BizRef
│  ├─ dictionary/           平台默认 + 租户 override
│  └─ localization/         YAML i18n
├─ shared/                  跨层最小内核
│  ├─ kernel/               ID, ctx, time, pagination
│  ├─ errors/               Kind + Code + i18n key
│  ├─ eventbus/             Publish-Subscribe (in-proc channel)
│  └─ tenantdb/             PlatformDB + TenantDB (SET LOCAL)
├─ infrastructure/          基础设施 (无业务语义)
│  ├─ pg/                   pgxpool + slow query tracer + RESET hook
│  ├─ redis/                go-redis client
│  ├─ logger/               zap + sanitize
│  ├─ metrics/              Prometheus
│  └─ health/               check registry
├─ interface/               HTTP 横切
│  ├─ middleware/           RequestID/Recover/Logger/CORS/RateLimit/Idempotency
│  ├─ apiresp/              统一响应 envelope
│  └─ server.go             Gin engine
├─ config/                  viper
└─ app/                     顶层 DI 装配
```

## 数据隔离

- **public schema**: 全平台共享 (tenant, platform_user, session, role, migration_history)
- **tenant_<slug> schema**: 每租户独立 (member, audit_log, notification, attachment, okr_plan, okr_report 等)
- 切换通过 `TenantDB.Transaction` 内的 `SET LOCAL search_path TO "tenant_<slug>", public`
- 连接归池前 `RESET search_path` 双保险 (pool hook)

## 请求生命周期 (典型)

```
[client] POST /api/plans
   │ Authorization: Bearer <jwt>
   │ Idempotency-Key: <uuid>
   ▼
[nginx] HTTPS termination + rate limit (per-IP)
   ▼
[middleware] RequestID → Recover → Logger → CORS → RateLimit (per-tenant/user) → Idempotency
   ▼
[iam.JWTAuth] 解 jwt, 注入 claims (platform_user, tenant, member) 进 ctx
   ▼
[iam.TenantLoader] 用 claims.tenant_id 查 Tenant, 写 TenantContext 进 ctx
   ▼
[okr.iface] HTTP handler → application.CreatePlan
   ▼
[okr.application] domain.NewPlan(...) → planRepo.Save(...) → bus.Publish(okr.plan_created)
   ▼
[infra.pgPlanRepo] tenantdb.TenantDB.Transaction:
   • SET LOCAL search_path TO "tenant_<slug>", public
   • INSERT INTO okr_plan ...
   • COMMIT (LOCAL search_path 自动回滚)
   ▼
[eventbus.worker] dispatch event:
   • audit subscriber → INSERT audit_log
   • notification subscriber → (未来) Send 通知
   ▼
[apiresp] 包装为 envelope, 返回 201
```

## 事件流

```
tenancy.tenant_created     ─┐
tenancy.member_joined      ─┼─→ audit (写 audit_log)
iam.user_logged_in         ─┤
okr.plan_created           ─┤
okr.plan_item_completed    ─┤
okr.daily_submitted        ─┤
okr.weekly_submitted       ─┤
okr.weekly_overdue         ─┴─→ notification (Send 站内信)
```

Topic 全 registry 见 `docs/architecture/event-catalog.md`.

## 关键决策 (10 条)

| # | 决策 | 选择 |
|---|---|---|
| 1 | 架构定位 | Go 模块化单体 |
| 2 | DDD 适用范围 | 仅 OKR; 其他服务包 |
| 3 | 租户隔离 | PG 每租户独立 schema + SET LOCAL |
| 4 | 用户 × 租户 | 双层 (PlatformUser + TenantMember) |
| 5 | 鉴权 | HS256 JWT + 自研 RBAC matcher |
| 6 | 事件总线 | 进程内 channel + worker pool |
| 7 | 错误处理 | Kind + Code + i18n key 三元组 |
| 8 | 日志 | zap 结构化 + 脱敏 + trace_id |
| 9 | 备份 | pg_dump 每日 + 7 天 + 异机 |
| 10 | 部署 | docker-compose dev + prod + Nginx |

详细见 spec v3.1 §8.

## 学习地图

- 新人第一周: 把 `cmd/server/main.go` 起步, 看 `app.Build`, 跑 OKR 一个用例端到端
- 第二周: 读 `contexts/okr/domain/plan.go`, 写一个新 PlanItem 不变量
- 第三周: 加一个新业务模块 (走 `docs/developer/adding-new-service.md`)

## 故事时间

- v1 (Java/Spring): 单体, 行级隔离, 服务在 spring container 里. M2 末期废弃.
- v2.0/v2.1.1 (废弃 spec): 试图同时上 8 platform + 7 业务模块, 评审认定过度工程.
- **v3.0/v3.1 (现行)**: 砍掉 60% 范围, 仅 OKR 走真 DDD, 其余降级为服务包. 这是工程能跑完的版本.
