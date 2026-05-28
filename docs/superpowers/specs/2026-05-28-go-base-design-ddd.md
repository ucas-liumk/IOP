# Go 多租户 B 端基座 · DDD 设计文档 v2

| 项 | 内容 |
|---|---|
| 版本 | v2.1.1 (在 v2.1 基础上扩充首期业务模块集合) |
| 起草日期 | 2026-05-28 |
| 状态 | Draft, 待 spec 复核后转入 writing-plans |
| 范围 | `iop/` monorepo 顶层直挂 Go 后端 (`server/`) + Vue 前端 (`web/`) + 部署 / 文档 / 脚本 + 桌面端外壳预留 (`desktop/` 经 `web/desktop/` 持有) |
| 与 v1 spec 的关系 | 战术目录结构换骨, 战略目标/里程碑/命门测试/部署形态保持 (见 §A) |
| 首期落地业务模块 | **7 个**: Problem (问题协同) / OKR (工作安排) / Contacts (通讯录) / Announcements (时政热点) — **M4a**; Documents (资源平台) / Assets (资产管理) / Party (党务管理) — **M4b**。个人档案 (PersonalProfile) 作为 IAM 模块的扩展能力, 不单独成模块。 |
| 推迟到 v1.5+ | AppCenter (应用中心, 平台级模块开关) / Contacts 在线聊天 (需 SSE/WebSocket 基础设施) |
| v2.1 相对 v2.0 增量 | (1) 区分 `platform/` 与 `modules/` 两类上下文; (2) 引入 `Module` 接口与编译期组合, 支持单模块独立打包; (3) 前端 monorepo 化 (`packages/` + `apps/`), 每产品独立 Vite 入口; (4) 前端预留桌面端抽象层 (Web 默认实现, Tauri 后续接入); (5) 迁移随模块走, 每模块自带 migrations |
| v2.1.1 相对 v2.1 增量 | (a) 业务模块从 1 个扩充到 7 个, 分 M4a / M4b 两批; (b) IAM 扩 TenantMember profile (JSONB + FileStorage BizRef 关联非结构化数据), 个人档案功能内化为 IAM 能力; (c) v1.5 范围引入 AppCenter (租户级模块开关) 与 Contacts 聊天 |

---

## §0 目的与范围

### 目的

- 把架构图(Go 模块化单体 + PG 每租户独立 schema + 共享内核 + 模块化业务)落地为**一份按领域驱动设计 (DDD) 组织的工程实现路线**。
- v2 与 v1 的核心区别: 把"按技术分层"换成"**按业务上下文 + 上下文内分层**", 让业务语言主导代码组织, 避免后期业务复杂度上升后陷入"贫血模型 + 上帝 service"。
- 首期落地 = 通用多租户基座 + Problem 上下文作为 PoC; 未来在同一基座承载更多 B 端产品。
- 在不破坏基座可演进性的前提下, 把 gallant v1 (Spring Boot + Vue + Element Plus) 已落地的资产(状态机 / 看板 / 弹层 / 8 表单 / 设计令牌 / 13 个前端测试)平稳搬过来。

### 范围

**包含**:
- Go 后端骨架: `server/{cmd,internal,pkg,api,configs,test}`
- 区分 **平台上下文 (Platform Contexts)** 与 **业务模块 (Business Modules)** 两类:
  - **平台上下文 (8 个, 任何产品必带)**: `Tenancy / IAM / Audit / Notification / FileStorage / Dictionary / Localization / PlatformConfiguration`
  - **业务模块 (按产品装配; 首期 7 个)**:
    - **M4a 批 (4 个)**: `Problem (问题协同) / OKR (工作安排) / Contacts (通讯录) / Announcements (时政热点)`
    - **M4b 批 (3 个)**: `Documents (资源平台) / Assets (资产管理) / Party (党务管理)`
  - **v1.5+ 平台增强**: `AppCenter` (应用中心, 租户级模块开关, 归 platform 层)
  - **不单独成模块**: `PersonalProfile` (个人档案) → 内化为 `platform/iam.TenantMember` 的 profile 扩展 (JSONB + FileStorage 关联)
- 每个上下文/模块统一四层划分: `domain / application / infrastructure / interface`
- 每个模块**自带 migrations** (平台级 + 租户级), 不再有中央 `migrations/tenant_template/`
- 统一 `Module` 接口 + 编译期组合: 每个产品一个 `cmd/<product>/main.go`, 装配 「全部平台 + 选定业务模块」
- 共享内核 (Shared Kernel) **最小化**: 仅 `Tenant ID / Member ID / TraceID` 等通用 ID 与 ctx 访问器 + `Module` 接口定义
- Vue 前端 monorepo (pnpm workspace):
  - `packages/`: `shell` + `api-client` + `ui-tokens` + `platform-web` + 每个业务模块一个 `module-<name>` 包
  - `apps/`: 每个产品一个 Vite 入口 (`iop-full`, `gallant`, …), 引用所需 packages
  - `desktop/`: Tauri 壳骨架预留 (M5+ 接入真实集成), 每产品一份配置
- 前端 **平台能力抽象层** (`@iop/shell/platform/*`): `storage / notifier / fileDialog / clipboard / env`, Web 默认实现, Tauri 实现按需替换
- OpenAPI 自动生成 TS SDK (按模块分组, 装入 `module-<name>` 包)
- 部署: docker-compose (dev + prod) + Nginx + KingbaseV8 验证
- 多产品独立部署: 同一份 monorepo 可输出多个二进制 + 多个前端 dist + 多个桌面包

**不包含 (明确推迟)**:
- Tauri 真实集成与桌面端打包流程 (架构预留, 工程实现推迟到 M6; v2.1 只落地前端抽象层与默认 Web 实现)
- Go plugin / 运行期模块加载 (编译期组合已够, 不引入运行期插件机制)
- 移动端 / 小程序 (按需, v1.5+)
- 开放 API + API Key (M5 之后)
- SSE 实时推送替代轮询 (v1.5)
- 流程定义可视化编辑 (v2.0)
- IM 集成 (Slack / 钉钉 / 企微) (v2.0)
- NATS 异步消息 (出现真实削峰信号再引入)
- OpenTelemetry (grep trace_id 不够用时再引入)
- 单上下文水平拆分为微服务 (有真实独立伸缩信号再说)

---

## §1 架构总览 (DDD 视角)

### 1.1 战略设计 — 有界上下文地图

8 个上下文 + 1 个跨切集合, 按 DDD 的 "**核心域 / 支持子域 / 通用子域**" 分类:

```
┌─────────────────────────────────────────────────────────────────┐
│ 核心子域 (Core)                                                  │
│   Problem  ← 业务护城河, 状态机和办理流程是差异化关键 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 支持子域 (Supporting)                                             │
│   Tenancy        Audit         Notification     FileStorage      │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│ 通用子域 (Generic)                                                │
│   IAM            Dictionary    Localization     PlatformConfig   │
│   (身份与访问)    (数据字典)    (i18n)          (运行时配置)        │
└─────────────────────────────────────────────────────────────────┘
```

**上下文映射 (Context Map)**:

```
                       ┌───────────────────────────┐
                       │  Tenancy  (Upstream U/D)   │
                       │  发布 TenantID + Schema    │
                       └────┬──────────────────┬────┘
                            │                  │
                       Published Language  Published Language
                            │                  │
              ┌─────────────▼─────┐    ┌───────▼──────────┐
              │   IAM (Conformist)│    │  其他所有上下文      │
              │   保护 ID 模型     │    │  注入 TenantID 到 ctx│
              └─────────────┬─────┘    └───────┬──────────┘
                            │                  │
                  发布 JWT / Session            │
                            ▼                  ▼
                    ┌───────────────────────────────┐
                    │  Problem         │
                    │  发布 ProblemSubmitted /      │
                    │       StageAdvanced 等领域事件  │
                    └───┬────────────────┬──────────┘
                        │                │
                    Subscriber       Subscriber
                        ▼                ▼
                ┌──────────────┐  ┌──────────────┐
                │ Audit        │  │ Notification │
                │ (CRM-style   │  │ (Open Host   │
                │  Customer-   │  │  Service via │
                │  Supplier)   │  │  event bus)  │
                └──────────────┘  └──────────────┘

                ┌──────────────┐  ┌──────────────┐  ┌──────────────┐
                │ FileStorage  │  │ Dictionary   │  │ Localization │
                │ Separate Ways│  │ Shared Kernel│  │ Shared Kernel│
                │ (独立 minio) │  │ (dict_code   │  │ (i18n key 约定)│
                │              │  │  约定)       │  │              │
                └──────────────┘  └──────────────┘  └──────────────┘
```

**关系类型说明** (DDD Strategic Design 术语):

| 关系 | 上下文 | 含义 |
|---|---|---|
| Upstream/Downstream | Tenancy → 其他 | 租户上下文是其他所有上下文的上游, 发布 TenantID 这一通用语言 |
| Published Language | Tenancy → 所有 | TenantID/Schema 是稳定契约, 不允许下游反向影响 |
| Conformist | 其他 → Tenancy | 下游完全服从上游的 ID 模型, 不做适配 |
| Customer-Supplier | Problem → Audit | 业务负责发事件, 审计负责持久化; 业务变更需提前通知审计 |
| Open Host Service | Problem → Notification | 业务通过事件总线开放协议, 任何订阅者都能消费, 业务不感知 |
| Separate Ways | FileStorage 独立 | 文件存储不与其他上下文共享模型, 通过 (biz_module, biz_id) 元组关联 |
| Shared Kernel | Dictionary / Localization | 全平台共用术语, 但变更要协商 (字典 code 改名要走平台流程) |

### 1.2 战术设计 — 四层架构

每个上下文内部严格四层:

```
┌─────────────────────────────────────────────────────────────────┐
│ interface/   接口适配层                                          │
│   - HTTP handlers (Gin)                                         │
│   - 请求/响应 DTO                                                │
│   - 路由注册                                                     │
│   - 入参校验 (binding) + 出参 transform                          │
│   * 唯一允许调用 application/                                    │
└──────────────────────────┬──────────────────────────────────────┘
                           │ 依赖
┌──────────────────────────▼──────────────────────────────────────┐
│ application/   应用层 (用例编排)                                  │
│   - Command Handlers   (写, 编排聚合 + 仓储 + 领域事件发布)         │
│   - Query Handlers     (读, 跳过领域模型直查 DB → DTO)             │
│   - 事务边界 (Transaction Script per use case)                  │
│   * 不持有业务规则, 业务规则在 domain/                            │
│   * 依赖 domain/ 的接口 + 基础设施实现 (通过 DI 注入)              │
└──────────────────────────┬──────────────────────────────────────┘
                           │ 依赖
┌──────────────────────────▼──────────────────────────────────────┐
│ domain/   领域层 (业务核心, 不依赖任何外部)                        │
│   - Aggregates 聚合 (Problem, Tenant, PlatformUser, ...)        │
│   - Entities + Value Objects (Stage, Branch, Email, ...)         │
│   - Domain Services (StageEngine, PasswordPolicy, ...)          │
│   - Domain Events (ProblemSubmitted, StageAdvanced, ...)        │
│   - Repository Interfaces (ProblemRepo, TenantRepo, ...)        │
│   * 纯 Go, 不 import 任何 infrastructure / 任何外部框架            │
│   * 唯一允许出现的"基础设施"是标准库 + 通用 ID 工具                  │
└─────────────────────────────────────────────────────────────────┘
                           ▲ 实现
┌──────────────────────────┴──────────────────────────────────────┐
│ infrastructure/   基础设施层                                     │
│   - Repository 实现 (pgProblemRepo implements domain.ProblemRepo)│
│   - 外部服务适配器 (MinIO, Redis, SMTP, ...)                     │
│   - DB Schema 映射 / SQL                                        │
│   - 第三方库胶水代码                                              │
│   * 实现 domain/ 中声明的接口, 反向依赖                            │
│   * application/ 通过 DI 注入具体实现                             │
└─────────────────────────────────────────────────────────────────┘
```

**依赖方向** (DIP 依赖倒置 + Clean Architecture):
- `interface → application → domain ← infrastructure`
- 领域层不依赖任何外部, 包括同上下文的 infrastructure
- 仓储接口定义在 `domain/`, 实现在 `infrastructure/`
- 应用层通过构造函数注入接收 `domain.XxxRepo` 接口, 实际拿到的是 `infrastructure.pgXxxRepo`

### 1.3 目录结构 (monorepo)

顶层分工:

- `server/` Go 后端: `internal/platform/` (8 平台上下文) + `internal/modules/` (业务模块, 按需装配)
- `web/` 前端 pnpm workspace: `packages/` (可复用单元) + `apps/` (每产品一个 Vite 入口) + `desktop/` (Tauri 壳, M6+)
- `deployments/`, `scripts/`, `docs/` 同 v2.0

```
iop/                                             ← 项目根 (顶层直挂, 不再有 go_base/ 中间层)
├── server/                                      ← Go 后端
│   ├── cmd/                                     ★ 一个产品 = 一个二进制
│   │   ├── iop-full/main.go                     完整版 (全部 platform + 全部 modules)
│   │   ├── gallant/main.go                      仅问题协同独立产品 (全部 platform + modules/problem)
│   │   ├── migrate/main.go                      迁移工具 (支持 --module 选择子集)
│   │   └── tenantctl/main.go                    租户管理 CLI (跨产品共用; 各 module 可注册子命令)
│   │
│   │  注: M1-M3 阶段只构建 cmd/iop-full/, 不维护 gallant/; 第二个业务模块进场或正式产品化时再拆 cmd 入口
│   │
│   ├── internal/
│   │   ├── platform/                            ★ 平台基座: 任何产品都必须装配
│   │   │   ├── tenancy/                         租户
│   │   │   │   ├── domain/                      Tenant 聚合 + TenantMember + 值对象 + 事件 + 仓储接口
│   │   │   │   ├── application/                 CreateTenant / SuspendTenant / JoinMember / ListActiveTenants
│   │   │   │   ├── infrastructure/              pg 仓储 + LRU 缓存 + SchemaProvisioner
│   │   │   │   ├── interface/                   HTTP handler + DTO + middleware (TenantLoader)
│   │   │   │   ├── migrations/                  ★ 自带迁移
│   │   │   │   │   ├── platform/                ↳ public schema 部分 (tenant, migration_history 等)
│   │   │   │   │   └── tenant/                  ↳ 租户 schema 部分 (member 表)
│   │   │   │   └── module.go                    实现 shared/module.Module 接口
│   │   │   │
│   │   │   ├── iam/                             身份与访问 (auth + RBAC 合并)
│   │   │   │   ├── domain/                      PlatformUser / Session / Role + PolicyRule + IdentityProvider 接口
│   │   │   │   ├── application/                 Login / Refresh / Logout / SwitchTenant / Enforce
│   │   │   │   ├── infrastructure/              JWT signer / bcrypt / Local provider / Casbin enforcer / pg repos
│   │   │   │   ├── interface/                   /auth/* /me/* + JWT/Session/RBAC middleware
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── audit/
│   │   │   │   ├── domain/                      AuditLog (租户内) + PlatformAudit (跨租户) + 仓储
│   │   │   │   ├── application/                 RecordEvent / FlushBuffer / queries
│   │   │   │   ├── infrastructure/              pg repos + BufferedWriter + EventSubscriber (订阅 eventbus)
│   │   │   │   ├── interface/                   /audit/* (内部用)
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── notification/
│   │   │   │   ├── domain/                      Notification + Channel + Dispatcher
│   │   │   │   ├── application/                 SendNotification / MarkRead / ListUnread / EventHandlers
│   │   │   │   ├── infrastructure/              pg repo + InAppChannel + SmtpChannel (v1 noop)
│   │   │   │   ├── interface/                   /notifications/*
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── filestorage/
│   │   │   │   ├── domain/                      Attachment + ObjectKey + BizRef + StorageProvider 接口
│   │   │   │   ├── application/                 Upload / Download / List / Delete
│   │   │   │   ├── infrastructure/              MinioProvider + pg AttachmentRepo
│   │   │   │   ├── interface/                   /files/*
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── dictionary/                      字典
│   │   │   │   ├── domain/                      DictType + DictItem + Override (租户级)
│   │   │   │   ├── application/                 Lookup / AddOverride / ReloadCache
│   │   │   │   ├── infrastructure/              pg DictRepo + Redis cache
│   │   │   │   ├── interface/                   /dict/*
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── localization/                    i18n
│   │   │   │   ├── domain/                      Translation + Locale
│   │   │   │   ├── application/                 T(ctx, key, args...)
│   │   │   │   ├── infrastructure/              YAML bundle loader (M5+ DB 后端)
│   │   │   │   ├── interface/                   (M5+ 管理 UI)
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── platform_config/                 运行时配置 (跨切, 严格说不是上下文)
│   │   │   │   ├── domain/                      ConfigKey / ConfigValue
│   │   │   │   ├── application/                 Get(ctx, key) / SetTenantOverride
│   │   │   │   ├── infrastructure/              StaticProvider (YAML) + TenantOverride (M3+)
│   │   │   │   ├── migrations/{platform,tenant}/
│   │   │   │   └── module.go
│   │   │   │
│   │   │   └── appcenter/                       AppCenter 应用中心 (v1.5+, 不在 v2.1.1 范围) — 租户级模块开关
│   │   │       ├── domain/                      TenantModuleEnablement 聚合 (tenant_id, module_name, enabled, enabled_at)
│   │   │       ├── application/                 ListAvailable / Enable / Disable / GetMenu
│   │   │       ├── infrastructure/              pg repo + Registry 提供"已编译模块"列表
│   │   │       ├── interface/                   /appcenter/* + 中间件 EnablementGate (拒绝未启用模块的请求)
│   │   │       ├── migrations/platform/         tenant_module_enablement 表
│   │   │       └── module.go
│   │   │
│   │   ├── modules/                             ★ 业务模块: 按产品装配, 之间禁止互相 import
│   │   │   ├── problem/                   Problem (M4a)
│   │   │   │   ├── domain/                      Problem 聚合根 + Stage/Branch/Measure/Dispute/Evaluation
│   │   │   │   │                                + StageEngine 领域服务 + 事件 + 仓储接口
│   │   │   │   ├── application/                 8 阶段用例 + dashboard_query + list_problems
│   │   │   │   ├── infrastructure/              pg ProblemRepo + dashboard_query_pg
│   │   │   │   ├── interface/                   /problems/* /dashboard/* /messages/*
│   │   │   │   ├── migrations/tenant/           10 张表
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── okr/                    OKR 工作安排 (M4a) — 核心业务模块, 体量近似 problem
│   │   │   │   ├── domain/                      Plan 聚合 (Year/HalfYear/Month/Week 4 级) + PlanItem +
│   │   │   │   │                                Report 聚合 (Daily/Weekly) + RollupView 领域服务 + 事件
│   │   │   │   ├── application/                 CreatePlan / DecomposePlan / SubmitDaily / SubmitWeekly
│   │   │   │   │                                / RollupWeekly / CommentReport / KPI 查询
│   │   │   │   ├── infrastructure/              pg PlanRepo / ReportRepo + 汇总投影
│   │   │   │   ├── interface/                   /plans/* /reports/* /rollups/*
│   │   │   │   ├── migrations/tenant/           ~8 张表
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── contacts/                        Contacts 通讯录 (M4a) — 部门联系人(=IAM 投影)+ 个人联系人
│   │   │   │   ├── domain/                      OrgContact 视图 + PersonalContact 聚合 + ContactGroup 实体
│   │   │   │   │                                + Frequent/Favorite 标记 + 事件
│   │   │   │   ├── application/                 SearchOrg / ListPersonal / AddPersonalContact / MarkFavorite
│   │   │   │   │                                / Import vCard / Export vCard
│   │   │   │   ├── infrastructure/              pg PersonalContactRepo + 订阅 iam 事件维护 OrgContact 投影
│   │   │   │   ├── interface/                   /contacts/org /contacts/personal /contacts/favorites
│   │   │   │   ├── migrations/tenant/           ~3 张表 (personal_contact, contact_group, favorite)
│   │   │   │   └── module.go                    v1.5+ 加 chat 子聚合; v2.1.1 不含
│   │   │   │
│   │   │   ├── announcements/                        Announcements 时政热点 (M4a) — 轻量 CMS
│   │   │   │   ├── domain/                      Post 聚合 + Tag 值对象 + PublishWorkflow (草稿/审/发) + 事件
│   │   │   │   ├── application/                 CreatePost / Publish / Tag / List / GetDetail / IncrViewCount
│   │   │   │   ├── infrastructure/              pg PostRepo + redis 热点缓存
│   │   │   │   ├── interface/                   /news/posts/* /news/tags/*
│   │   │   │   ├── migrations/tenant/           ~3 张表 (post, tag, post_tag)
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── documents/                     Documents 资源平台 (M4b) — 用户上传/共享/检索
│   │   │   │   ├── domain/                      Resource 聚合 (含 FileStorage BizRef) + ShareScope 值对象
│   │   │   │   │                                + DownloadStats 实体 + 事件
│   │   │   │   ├── application/                 Upload / ToggleShare / Search / Download / ListMine
│   │   │   │   ├── infrastructure/              pg ResourceRepo (附件实际经 platform/filestorage)
│   │   │   │   ├── interface/                   /resources/*
│   │   │   │   ├── migrations/tenant/           ~3 张表 (resource, share_scope, download_log)
│   │   │   │   └── module.go
│   │   │   │
│   │   │   ├── assets/                       Assets 资产管理 (M4b) — 个人录入 + 部门可见
│   │   │   │   ├── domain/                      Asset 聚合 + AssetCategory 值对象 + DeptScope 领域服务 + 事件
│   │   │   │   ├── application/                 CreateAsset / UpdateAsset / RetireAsset / ListMyAssets
│   │   │   │   │                                / ListDeptAssets (部门管理员视角) / Export
│   │   │   │   ├── infrastructure/              pg AssetRepo + 部门可见性走 IAM 部门关系
│   │   │   │   ├── interface/                   /assets/*
│   │   │   │   ├── migrations/tenant/           ~3 张表 (asset, asset_category, asset_history)
│   │   │   │   └── module.go
│   │   │   │
│   │   │   └── party/                    Party 党务管理 (M4b) — 强中国特色, 按租户可选装
│   │   │       ├── domain/                      MemberDevelopment 聚合 (5 阶段: 申请/积极分子/发展/预备/正式)
│   │   │       │                                + PartyActivity 聚合 (含签到) + StudyMaterial 聚合 + 事件
│   │   │       ├── application/                 AdvanceStage / RecordActivity / Checkin / UploadStudy
│   │   │       │                                / IssueCertificate / Reports
│   │   │       ├── infrastructure/              pg repos + 附件经 filestorage
│   │   │       ├── interface/                   /party/members/* /party/activities/* /party/study/*
│   │   │       ├── migrations/tenant/           ~5 张表
│   │   │       └── module.go
│   │   │
│   │   │   (v1.5+) chat/, …  目前不存在
│   │   │
│   │   ├── shared/                              ★ 共享内核 (严格最小化)
│   │   │   ├── kernel/                          跨上下文通用语义
│   │   │   │   ├── ids.go                       ID 类型 + 生成 (UUID v7)
│   │   │   │   ├── context.go                   ctxutil: WithTrace/Tenant/Member + accessor
│   │   │   │   ├── time.go                      Clock 抽象 (测试可注入)
│   │   │   │   └── pagination.go                Page/PageSize 通用值对象
│   │   │   ├── errors/                          错误模型 (跨上下文共用)
│   │   │   │   ├── kind.go                      ErrKind 枚举
│   │   │   │   ├── error.go                     Error struct + Wrap/Unwrap
│   │   │   │   └── codes.go                     错误码区间约定 (各上下文/模块领一段)
│   │   │   ├── eventbus/                        进程内事件总线
│   │   │   │   ├── bus.go                       接口
│   │   │   │   ├── inproc_bus.go                channel + worker pool 实现
│   │   │   │   └── types.go                     Event 通用结构 (occurred_at / actor / trace / module)
│   │   │   ├── tenantdb/                        多租户 DB 抽象
│   │   │   │   ├── platform_db.go               访问 public schema
│   │   │   │   ├── tenant_db.go                 含 SET LOCAL search_path
│   │   │   │   └── pool_hook.go                 RESET search_path 防御性钩子
│   │   │   └── module/                          ★ Module 接口与装配 (v2.1 新增, 见 §1.6)
│   │   │       ├── module.go                    Module / Migration / EventSubscription / CLICommand 接口
│   │   │       ├── registry.go                  Registry: 排序 + 启动顺序 + 路由聚合 + 事件订阅聚合
│   │   │       └── deps.go                      公共依赖 (PlatformDB / TenantDB / EventBus / Logger / Metrics) 类型
│   │   │
│   │   ├── infrastructure/                      通用基础设施 (无业务语义)
│   │   │   ├── pg/                              pgx pool, 健康检查
│   │   │   ├── redis/
│   │   │   ├── minio/
│   │   │   ├── logger/                          zap 配置 + 脱敏中间件
│   │   │   └── metrics/                         Prometheus collector
│   │   │
│   │   ├── interface/                           HTTP 横切关注
│   │   │   ├── middleware/                      RequestID, Recover, Logger, CORS, Trace, AuditTag, BodyLimit
│   │   │   ├── apiresp/                         统一响应封装 + Render Transformer
│   │   │   ├── query_pipeline/                  QueryPipeline 抽象 (扩展点)
│   │   │   └── server.go                        Gin engine 装配 + Registry.RegisterRoutes 聚合
│   │   │
│   │   ├── config/                              viper 加载 (启动期)
│   │   └── app/                                 顶层 DI 装配工厂 (供各 cmd 复用)
│   │       ├── platform.go                      装配全部 platform Module 实例
│   │       ├── runtime.go                       Run(ctx, []Module): 迁移 / 路由 / 事件订阅 / Graceful shutdown
│   │       └── deps.go                          构造 module.Deps
│   │
│   ├── pkg/                                     可被外部 import 的工具 (谨慎扩张)
│   │   └── ctxutil/                             (薄封装, 真正实现在 shared/kernel/)
│   │
│   ├── api/openapi/                             接口契约 (按模块分目录, 生成时分别打包)
│   │   ├── platform.yaml                        所有 platform 模块汇总
│   │   ├── module-problem.yaml
│   │   └── (未来) module-xxx.yaml
│   ├── configs/
│   ├── test/integration/                        含 harness.go + 命门测试
│   ├── go.mod
│   └── Makefile                                 含 build-<product> / migrate-<product> 等目标
│
├── web/                                         ← Vue 3 前端 (pnpm workspace, 多产品多入口)
│   ├── pnpm-workspace.yaml
│   ├── package.json                             workspace root + 共享 devDeps (TS / Vite / Vitest)
│   ├── tsconfig.base.json
│   │
│   ├── packages/                                ★ 可复用单元 (库)
│   │   ├── shell/                               @iop/shell (基座壳, 与业务无关)
│   │   │   ├── package.json
│   │   │   └── src/
│   │   │       ├── layout/                      AppLayout / NavBar / ModuleNav
│   │   │       ├── auth/                        LoginView / auth.store / guard
│   │   │       ├── tenant/                      TenantSwitcher / tenant.store
│   │   │       ├── notify/                      NotifyCenter / notify.store
│   │   │       ├── workspace/                   工作台首页
│   │   │       ├── platform/                    ★ 平台能力抽象层 (Web/Tauri 共用契约)
│   │   │       │   ├── env.ts                   runtime config 注入 (API base URL, tenant 解析)
│   │   │       │   ├── storage.ts               KV 存储 (localStorage vs Keychain)
│   │   │       │   ├── notifier.ts              系统通知 (Web Notification vs OS notify)
│   │   │       │   ├── file_dialog.ts           文件选择 (input[type=file] vs Tauri dialog)
│   │   │       │   ├── clipboard.ts
│   │   │       │   └── deep_link.ts             深链回跳 (浏览器 hash vs custom scheme)
│   │   │       ├── components/                  共享 UI (Icon, StageChip, AvatarBadge)
│   │   │       └── stores/                      跨壳 Pinia store
│   │   │
│   │   ├── platform-web/                        @iop/platform-web (Web 默认实现, M1 落地)
│   │   ├── platform-tauri/                      @iop/platform-tauri (M6+ 接入)
│   │   │
│   │   ├── api-client/                          @iop/api-client (axios + JWT + 401 跳登录 + base URL 注入)
│   │   ├── ui-tokens/                           @iop/ui-tokens (tokens.css + global.css + 组件令牌)
│   │   │
│   │   ├── module-problem/                @iop/module-problem (M4a)
│   │   │   ├── package.json
│   │   │   └── src/{routes.ts, views/, components/, stores/, api/, sdk/}
│   │   │
│   │   ├── module-okr/                 @iop/module-okr (M4a) — 4 级计划 + 日/周报 + 汇总
│   │   ├── module-contacts/                    @iop/module-contacts (M4a) — 部门联系人 + 个人联系人 + 收藏
│   │   ├── module-announcements/                     @iop/module-announcements (M4a) — 列表/详情/标签
│   │   ├── module-documents/                  @iop/module-documents (M4b) — 上传/共享/检索
│   │   ├── module-assets/                    @iop/module-assets (M4b) — 资产录入/部门视图
│   │   ├── module-party/                 @iop/module-party (M4b) — 党员发展/活动/学习
│   │   │
│   │   │  注: 个人档案 UI 嵌入 @iop/shell (作为 /me/profile 路由), 不开独立 module-profile 包
│   │   │
│   │   └── (v1.5+) module-appcenter/, module-chat/, ...
│   │
│   ├── apps/                                    ★ 产品入口 (每个一个 Vite build)
│   │   ├── iop-full/                            完整版 (壳 + 全部业务模块)
│   │   │   ├── package.json                     deps: @iop/shell + @iop/platform-web + 全部 @iop/module-*
│   │   │   ├── vite.config.ts
│   │   │   ├── index.html
│   │   │   └── src/
│   │   │       ├── main.ts                      装配: shell + platform-web + 选定 modules
│   │   │       ├── App.vue
│   │   │       └── modules.ts                   ★ 选装列表 (装配口)
│   │   │
│   │   ├── gallant/                             仅问题协同独立产品
│   │   │   ├── package.json                     deps: @iop/shell + @iop/platform-web + @iop/module-problem
│   │   │   └── src/modules.ts                   只装一个模块
│   │   │
│   │   │  注: M1-M3 阶段先只搭 apps/iop-full/; 拆 apps/gallant/ 同步于后端 cmd 拆分时机
│   │   │
│   │   └── (未来) <product>/
│   │
│   ├── desktop/                                 ★ Tauri 壳骨架 (架构预留, M6+ 接入真实集成)
│   │   ├── iop-full/                            tauri.conf.json + Rust src + icon
│   │   │   └── (内部) 装载 apps/iop-full 的 dist + 替换 @iop/platform-web → @iop/platform-tauri
│   │   └── gallant/
│   │
│   ├── tests/                                   vitest workspace 共享 setup
│   └── e2e/                                     Playwright (M5+, 按产品分组)
│
├── deployments/
│   ├── docker-compose.yml                       db + redis + minio + server(iop-full) + web(iop-full)
│   ├── docker-compose.prod.yml
│   ├── docker-compose.gallant.yml               独立产品的可选 compose (与 iop-full 并存或单跑)
│   ├── nginx/nginx.conf
│   └── kingbase/                                M5: KingbaseV8 适配
│
├── docs/
├── scripts/                                     dev.sh / seed.sh / openapi-gen.sh / new-module.sh
├── legacy/                                      v1 Spring Boot + Element Plus 资产, 仅作迁移参考, 不参与构建
└── README.md
```

**关键变化与 v2.0 对比**:

| 维度 | v2.0 | v2.1 |
|---|---|---|
| 后端顶层 | `internal/contexts/<8 个>/` 平铺 | `internal/platform/<8 个 + v1.5 appcenter>/` + `internal/modules/<v2.1.1 首期 7 个>/` 分层 |
| cmd 入口 | 单一 `cmd/server/` | 每产品一个 `cmd/<product>/`; M1-M3 暂只建 `iop-full` |
| Migrations | 中央 `migrations/{public,tenant_template}/` | **每 module 自带** `migrations/{platform,tenant}/`; `migration_history` 表多一列 `module_name` |
| Module 装配 | 各上下文 `module.go` 自管 (无统一接口) | 统一 `shared/module.Module` 接口 + `Registry` 装配 |
| 前端工程 | 单一 Vite 工程 `web/src/{shell,modules}` | pnpm workspace `web/{packages,apps,desktop}`, 每产品独立入口 |
| 桌面端 | 不在范围 | 架构预留: `@iop/shell/platform/*` 抽象层 + `desktop/` 骨架 (M6+ 真实接入) |

### 1.4 依赖方向规则 (架构防腐)

下文 `<X>` 泛指 `platform/<name>` 或 `modules/<name>` 任一上下文/模块。

**层内规则** (与 v2.0 相同, 四层内部依赖方向):

1. **`internal/{platform,modules}/<X>/domain/`** 不允许 import:
   - 任何其他上下文/模块 (无论 platform 还是 modules)
   - 任何 `internal/infrastructure/`
   - 任何同 `<X>` 的 `infrastructure/`, `application/`, `interface/`
   - 任何第三方框架 (Gin, GORM, pgx, Casbin, ...)
   - 只能 import: 标准库 + `internal/shared/kernel/` + `internal/shared/errors/`

2. **`internal/{platform,modules}/<X>/application/`** 只能 import:
   - 同 `<X>` 的 `domain/`
   - `internal/shared/{kernel,errors,eventbus,module}`
   - **不能** import 其他上下文/模块的 `domain/` 或任何 `infrastructure/`

3. **`internal/{platform,modules}/<X>/infrastructure/`** 实现 `domain/` 中的接口:
   - 允许 import 同 `<X>` 的 `domain/`
   - 允许 import `internal/shared/*` + `internal/infrastructure/*`
   - **不能** import 同 `<X>` 的 `application/`, `interface/`
   - **不能** import 其他上下文/模块的任何子包

4. **`internal/{platform,modules}/<X>/interface/`** 只能 import:
   - 同 `<X>` 的 `application/` (调用用例)
   - 同 `<X>` 的 `domain/` (仅做 DTO ↔ 领域对象转换时)
   - `internal/interface/` (中间件、apiresp)
   - **不能** import 同 `<X>` 的 `infrastructure/` (走 DI 由 application 注入)

**跨上下文/模块规则** (v2.1 收紧):

5. **跨上下文/模块协作** 只能通过:
   - `internal/shared/eventbus/` 异步发/订事件 (首选; 发布方通过 `event_module` 字段标识来源)
   - 其他上下文/模块 `interface/` 暴露的 HTTP API (远期拆服务的过渡)
   - **禁止**: 跨上下文/模块 import domain 或 application
   - **禁止**: 跨 schema JOIN, 跨上下文 SQL JOIN

6. **平台 ↔ 业务模块 的方向约束** (新增, CI gate):
   - ✅ `modules/<X>` 可以 import `platform/<Y>/interface/` 的事件订阅契约或公开类型 (例: 订阅 `iam.UserLoggedIn` 事件 schema)
   - ❌ `platform/<X>` **不能** import 任何 `modules/<Y>`
   - ❌ `modules/<X>` **不能** import 任何 `modules/<Y>`
   - 平台对模块的依赖是「能力提供」(通过 `module.Deps` 注入), 不是 import

7. **跨 schema JOIN 禁止**: 需要 public 数据时用 `PlatformDB` 单独查再在 Go 里组装。

8. **`shared/module/` 的稳定性**: `Module` 接口属于发布语言 (Published Language), 任何变更都视作 breaking change, 需走 ADR 流程。

**前端对称规则** (pnpm workspace 实施):

- `apps/<product>` 是唯一允许装配 `@iop/shell` + `@iop/platform-*` + `@iop/module-*` 的位置
- `@iop/module-<X>` **不能** depend on `@iop/module-<Y>` (workspace 依赖 graph 强制)
- `@iop/shell` **不能** depend on 任何 `@iop/module-*` (反向依赖必定打死)
- `@iop/platform-web` / `@iop/platform-tauri` 实现 `@iop/shell/platform/*` 暴露的接口, 不互相 import
- `packages/shell/components/` 仅放真正"基座"组件; 业务专属组件留在 `module-<X>/components/`

### 1.5 部署形态

**v2.1 关键概念**: 同一份 monorepo 可以产出**多个产品**, 每个产品是 (后端二进制, 前端 dist, 桌面壳) 的一组制品。各产品共享数据库实例但**不共享租户数据** (租户内只有该产品所在 module 的表存在)。

`deployments/docker-compose.yml` (开发, 默认起 `iop-full` 一套) 含 5 个服务:
- `db` (postgres:16) — 开发; 生产换 KingbaseV8
- `redis` (redis:7-alpine) — Session 黑名单 + 缓存
- `minio` — 对象存储 (FileStorage 模块使用)
- `server` (Go 二进制, 默认 `iop-full`) — API 主进程
- `web` (nginx + apps/iop-full/dist)

生产形态可选:
- **单产品独立部署** (例: 客户 A 只买问题协同 → `cmd/gallant` + `apps/gallant` + 独立 DB)
- **多产品共栈部署** (内部全功能版本 → `cmd/iop-full` + `apps/iop-full`, 同 DB 同租户内多业务模块)
- **桌面端分发** (M6+, 例: `desktop/gallant` Tauri 打包客户端, 后端仍远端 `cmd/gallant`)

生产: nginx 既服务 web 静态资源也反代 `/api` 到 server。每产品一套 nginx server block, 或共用 nginx 按路径分流。

### 1.6 模块化与产品组装 (v2.1 新增)

#### 1.6.1 核心思想

- **平台 (Platform)** 与 **业务模块 (Modules)** 都实现统一的 `Module` 接口
- 产品 = 一个 `cmd/<product>/main.go` (后端) + 一个 `apps/<product>` (前端) 的组合, 编译期决定装配集合
- 装配差异只在最外层入口, 平台/模块/前端包内部对装配无感知
- **不使用** Go plugin / 配置文件动态加载, 因为编译期组合已经覆盖所有可见需求, 且对桌面端 / 跨平台 / 升级流程友好得多

#### 1.6.2 Go 端: `Module` 接口

```go
// internal/shared/module/module.go
package module

type Module interface {
    Name() string                                 // 全局唯一, 如 "platform/iam" / "modules/problem"
    Description() string

    // 迁移按"作用域"分组返回; Registry 负责按时机调用
    PlatformMigrations() []Migration              // 作用 public schema (跨租户)
    TenantMigrations() []Migration                // 作用每租户 schema (新建租户时跑 + 升级时按 module 增量跑)

    // HTTP 路由注册; r 是已带 module 前缀的 RouterGroup
    RegisterRoutes(r *gin.RouterGroup)

    // 事件订阅 (订阅其他 module 发布的事件); 自身发布不需要在此声明
    SubscribeEvents(bus eventbus.Bus)

    // 可选: 向 tenantctl 注册子命令 (例: problem 注册 'seed' 命令)
    CLICommands() []*cobra.Command

    // 健康检查 (Registry 会聚合到 /readyz)
    HealthCheck(ctx context.Context) error
}

type Migration struct {
    ID   string  // 可排序, 如 "000001_initial"
    Up   string  // SQL
    Down string  // SQL (M5+ 强制; M1-M4 可空字符串)
}
```

`Module` 通过构造函数接收 `Deps`, 平台/模块都从同一份 `Deps` 取依赖, 不依赖全局变量:

```go
// internal/shared/module/deps.go
type Deps struct {
    PlatformDB  *tenantdb.PlatformDB
    TenantDB    *tenantdb.TenantDB
    EventBus    eventbus.Bus
    Logger      *zap.Logger
    Metrics     *prometheus.Registry
    Clock       kernel.Clock
    Dict        DictClient        // platform/dictionary 提供的查字典接口 (抽象在 shared/module/)
    File        FileClient        // platform/filestorage 提供的文件接口
    Notify      NotifyClient      // platform/notification 提供的通知接口
    // 注: Dict/File/Notify 接口定义在 shared/module/, 实现注入由 app/runtime.go 完成,
    //     这样 modules/<X> 可以使用平台能力而无需 import platform/<Y>
}
```

`Registry` 负责生命周期:

```go
// internal/shared/module/registry.go
type Registry struct { modules []Module }

func (r *Registry) Run(ctx context.Context, deps Deps, engine *gin.Engine) error {
    // 1. 按拓扑序排列 (platform 全部先于 modules; 同类内按声明顺序)
    // 2. 跑 platform migrations (作用 public)
    // 3. 跑 tenant migrations (对每个 active tenant 跑各 module 的 tenant 部分; 跳过已应用版本)
    // 4. 注册路由: engine.Group("/api") → for each mod: mod.RegisterRoutes(group)
    // 5. 注册事件订阅: for each mod: mod.SubscribeEvents(bus)
    // 6. 注册 CLI 子命令 (tenantctl 启动时调用)
    // 7. 启动后台 worker (audit buffered writer 等) — 通过 Module 暴露的 lifecycle hook
}
```

#### 1.6.3 后端装配示例

```go
// cmd/iop-full/main.go (M4b 完工后状态; M4a 中间态只装 4 个)
func main() {
    deps := app.BuildDeps(loadConfig())
    
    platform := app.PlatformModules(deps)  // tenancy, iam, audit, notification, filestorage, dict, locale, platform_config
    business := []module.Module{
        problem.New(deps),
        okr.New(deps),
        contacts.New(deps),
        announcements.New(deps),
        documents.New(deps),
        assets.New(deps),
        party.New(deps),
        // v1.5+ 在此追加: chat.New(deps), ...
    }
    
    registry := module.NewRegistry(append(platform, business...)...)
    app.Run(ctx, registry, deps)
}

// cmd/gallant/main.go (M5 单产品拆分演练时落地)
func main() {
    deps := app.BuildDeps(loadConfig())
    platform := app.PlatformModules(deps)
    business := []module.Module{ problem.New(deps) }  // ★ 只装一个
    
    registry := module.NewRegistry(append(platform, business...)...)
    app.Run(ctx, registry, deps)
}
```

> 现状提示: M1-M3 阶段, 业务模块尚未实现, `cmd/iop-full/` 装配 = 空业务模块列表 + 全部 platform, 仍然可以跑通登录/租户/字典 流程作为壳。M4a 增加前 4 个; M4b 补齐到 7 个。`cmd/gallant/` 留到 M5 单产品拆分演练时落地。

#### 1.6.4 前端装配 (pnpm workspace)

每个 `apps/<product>/src/main.ts` 是装配点:

```ts
// apps/iop-full/src/main.ts (M4b 完工后状态)
import { createShell } from '@iop/shell'
import { webPlatform } from '@iop/platform-web'        // Web 默认实现
import { problemModule } from '@iop/module-problem'
import { okrModule } from '@iop/module-okr'
import { contactsModule } from '@iop/module-contacts'
import { announcementsModule } from '@iop/module-announcements'
import { documentsModule } from '@iop/module-documents'
import { assetsModule } from '@iop/module-assets'
import { partyModule } from '@iop/module-party'

const app = createShell({
    platform: webPlatform,                              // 桌面端构建时换 @iop/platform-tauri
    modules: [
        problemModule,
        okrModule,
        contactsModule,
        announcementsModule,
        documentsModule,
        assetsModule,
        partyModule,
        // v1.5+: appCenterModule, chatModule,
    ],
})

app.mount('#app')
```

`createShell` 内部职责:
- 装配 Pinia / 路由 (`@iop/shell/router` 聚合 shell 自有路由 + 各 module.routes)
- 把 `platform` 注入到 provide/inject 顶层, 供 shell 与 module 通过 `usePlatform()` 取用
- 初始化 `@iop/api-client` (从 `platform.env.apiBaseURL` 取 base URL)
- 渲染 `AppLayout` (shell), 路由切换决定挂载哪个 module 视图

#### 1.6.5 平台能力抽象层 (Web vs Tauri)

`@iop/shell/platform/*` 定义契约, `@iop/platform-web` / `@iop/platform-tauri` 实现:

| 能力 | 接口 | Web 默认实现 | Tauri 实现 |
|---|---|---|---|
| `env` | `getApiBaseURL() / getTenantSlug()` | 读 `window.__IOP_CONFIG__` 或 `/api` | 读 Tauri 启动参数 / 配置文件 |
| `storage` | `get/set/remove(key)` | `localStorage` | Tauri Keychain plugin |
| `notifier` | `notify({title, body, ...})` | Web `Notification` API | Tauri `notification` plugin |
| `fileDialog` | `openFile() / saveFile()` | `<input type=file>` | Tauri `dialog` plugin |
| `clipboard` | `read() / write()` | `navigator.clipboard` | Tauri `clipboard` plugin |
| `deepLink` | `subscribe(handler)` | `hashchange` | Tauri `deep-link` plugin (`iop://`) |

业务模块代码 **永远不直接调用** `localStorage`/`Notification`/`navigator.*`, 只通过 `usePlatform()` 获取抽象接口, 桌面端零改动接入。

#### 1.6.6 OpenAPI 与 SDK 生成

- 后端按 module 维护 OpenAPI 片段, 在 `server/api/openapi/` 下:
  - `platform.yaml` (平台 8 模块汇总)
  - `module-problem.yaml` (业务模块)
- `scripts/openapi-gen.sh` 按片段独立生成 TS SDK, 输出到对应前端包的 `sdk/` 子目录:
  - `packages/shell/src/sdk/` ← 平台 SDK
  - `packages/module-problem/src/sdk/` ← 模块 SDK
- 各 module 的前端 API 调用层 import 自己包内的 SDK; **禁止跨包 import sdk**, 避免出现"问题协同包"引用"订单包"的 API

#### 1.6.7 新增业务模块流程 (`scripts/new-module.sh` 脚手架)

1. `scripts/new-module.sh <name>` 生成:
   - `server/internal/modules/<name>/{domain,application,infrastructure,interface,migrations,module.go}` 骨架
   - `web/packages/module-<name>/` 骨架 + `package.json` 含必要 deps
   - `server/api/openapi/module-<name>.yaml` 空模板
2. 决定要装载该模块的产品: 修改对应 `cmd/<product>/main.go` 和 `apps/<product>/src/main.ts`
3. 写 domain → application → infrastructure → interface (TDD 推荐)
4. 加迁移 (`migrations/{platform,tenant}/000001_*.sql`)
5. 跑 `make migrate-iop-full && make test`, 验证: 路由出现、迁移已应用、隔离测试通过

---

## §2 多租户基础设施 (跨上下文)

### 2.1 核心思路

JWT 验签后, 从 claims 取出 `tenant_id`, 加载租户元数据写入 `ctx`; 业务代码经过 `TenantDB` wrapper 时, **在事务开启的同一连接上 SET LOCAL search_path** 切到对应 schema。

**DDD 视角**: 这是**跨上下文的基础设施**, 不属于任何单一上下文。放在 `internal/shared/tenantdb/`。Tenancy 上下文负责"租户"这个领域概念 (谁是租户、状态、生命周期), TenantDB 负责"如何在 SQL 层物理切到对的 schema"。两者职责分离。

### 2.2 数据流时序

```
HTTP 请求
  │  Authorization: Bearer <JWT>
  ▼
[middleware: JWT 验签] (iam 上下文提供 middleware)
  │  解出 claims = { platform_user_id, tenant_id, member_id, session_id }
  │  ctx = WithClaims(ctx, claims)
  ▼
[middleware: Session 校验] (iam 上下文)
  │  Redis 黑名单
  ▼
[middleware: Tenant 加载] (tenancy 上下文 + shared/tenantdb 协作)
  │  tenancyApp.GetTenant(claims.tenant_id) → TenantContext (LRU 缓存)
  │  状态校验: status == "active"
  │  ctx = WithTenant(ctx, tc)
  ▼
[middleware: RBAC 鉴权] (iam 上下文)
  │  iamApp.Enforce(ctx, resource, action)
  ▼
[Gin handler] → [application service]
  │  applicationSvc.SubmitProblem(ctx, cmd)
  │    └── tenantDB.Transaction(ctx, func(tx) {
  │          ★ tx.Exec("SET LOCAL search_path TO tenant_acme, public")
  │          ... domain operation + repo + event publish ...
  │       })
  ▼
[PG 单实例]
  │  事务内 search_path = tenant_acme, public
  ▼
事务 COMMIT → SET LOCAL 自动回滚 → 连接归池 (pool hook RESET 双保险)
```

### 2.3 关键设计决策

**1. 用 `SET LOCAL` 绑定到事务, 不是连接级**
- `SET LOCAL` 跟事务生命周期, COMMIT/ROLLBACK 自动回滚, **不可能漏 reset**
- 连接级 SET 容易因池复用造成跨租户数据穿透

**2. 双 DB 句柄**: `PlatformDB` 与 `TenantDB` 共用 pgxpool 但接口分离

| 句柄 | 用途 | search_path |
|------|------|-------------|
| `PlatformDB` | 访问 `public`: tenant, platform_user, casbin_rule, platform_audit, dict | `public` |
| `TenantDB` | 访问当前租户 schema: member, problem, audit_log, notification | `tenant_<slug>, public` (按 ctx) |

**哪些上下文用哪个**:
- `platform/tenancy.infrastructure.*Repo` → `PlatformDB` (租户元数据在 public)
- `platform/iam.infrastructure.userRepo / sessionRepo / roleRepo` → `PlatformDB`
- `platform/audit.infrastructure.platformAuditRepo` → `PlatformDB`
- `platform/dictionary.infrastructure.dictRepo` (类型/默认项) → `PlatformDB`
- `platform/platform_config.infrastructure.*` → `PlatformDB`
- 其他所有 `infrastructure/*Repo` (platform 租户层: audit/notification/filestorage; modules/*: problem, ...) → `TenantDB`

**3. 防御性 pool hook**: 连接归池前 `RESET search_path` (双保险)。

**4. 跨 schema JOIN 禁止**: 用 PlatformDB 单独查再 Go 里组装。同样, 跨上下文 JOIN 也禁止 — 即便它们在同一个 schema 里。

### 2.4 关键代码骨架

```go
// internal/shared/tenantdb/tenant_db.go
package tenantdb

type TenantContext struct {
    ID         string  // public.tenant.id (UUID)
    Slug       string  // 'acme'
    SchemaName string  // 'tenant_acme'
    Status     string  // active / suspended / closed
}
func WithTenant(ctx context.Context, tc *TenantContext) context.Context
func FromContext(ctx context.Context) (*TenantContext, bool)

type TenantDB struct { db *gorm.DB }

func (t *TenantDB) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
    tc, ok := FromContext(ctx)
    if !ok { return errors.New(errors.KindInternal, "tenant_context_missing") }
    return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
        if err := tx.Exec(
            fmt.Sprintf(`SET LOCAL search_path TO %s, public`, quoteIdent(tc.SchemaName)),
        ).Error; err != nil {
            return err
        }
        return fn(tx)
    })
}
```

### 2.5 Schema 生命周期 (v2.1: 迁移随模块)

**核心变化**: 没有中央 `migrations/tenant_template/`, 每个 module 自带 `migrations/{platform,tenant}/`。`public.migration_history` 表多一列 `module_name` 标识来源:

```sql
CREATE TABLE public.migration_history (
    id           UUID PRIMARY KEY,
    tenant_id    UUID NULL,                -- NULL 表示作用于 public schema
    module_name  TEXT NOT NULL,            -- e.g. 'platform/iam', 'modules/problem'
    migration_id TEXT NOT NULL,            -- e.g. '000001_initial'
    applied_at   TIMESTAMPTZ NOT NULL,
    checksum     TEXT NOT NULL,            -- 防止历史迁移被改
    UNIQUE (tenant_id, module_name, migration_id)
);
```

| 时机 | 触发 | 动作 | 负责 |
|------|------|------|-----------|
| **首次启动 / 升级二进制** | `migrate up` (启动自动 + CLI 手动) | 对当前二进制装配的**每个 module**, 跑 `PlatformMigrations()` 增量到最新 (作用 public schema) | `app.Run` 启动时 `Registry.MigratePlatform()` |
| **开通租户** | `tenantctl create` 或 Admin API | 插 `public.tenant` → `CREATE SCHEMA tenant_<slug>` → **对每个 module 跑 `TenantMigrations()` 全量** → 写 history | `platform/tenancy.app.CreateTenant` 调 `SchemaProvisioner`, `SchemaProvisioner` 通过 `Registry` 拿到所有 module 的 tenant migrations |
| **升级业务表 (跨租户)** | `tenantctl migrate-all [--module=X]` | 遍历 active tenant, 对每 module 跑未应用的 tenant 迁移 | `platform/tenancy.app.MigrateAllTenants` 协调 `Registry.MigrateTenant(tenantCtx)` |
| **首次装载新模块** (产品升级新装一个 module) | 升级二进制 + `tenantctl migrate-all --module=<new>` | 该 module 的 platform 迁移 + 给每个 active tenant 跑该 module 的 tenant 迁移 | 同上, `--module` 过滤 |
| **冻结租户** | Admin API | 仅改 status; schema 保留; 中间件拒绝 | `platform/tenancy.app.SuspendTenant` |
| **销户** | Admin API + 30 天保留 + 手动确认 | `DROP SCHEMA tenant_<slug> CASCADE`; 标记 closed_at; `migration_history` 该 tenant 行级清理 | `platform/tenancy.app.CloseTenant` |

**关键约束**:
- 同一个 module 在不同产品 (`cmd/iop-full`, `cmd/gallant`) 中的 migration 序列必须一致, 不允许产品分叉。同步通过 module embedded 文件 (`embed.FS`) 强制。
- 跨 module 的 schema 依赖**禁止** (例: `modules/problem` 不能引用 `platform/iam.user` 表外键)。如需关联, 用 `platform_user_id` 列做"软引用", 应用层 JOIN。
- 迁移 checksum 校验已应用历史不被改, 改了启动报错。

### 2.6 命门测试 (CI gate)

放在 `server/test/integration/`, 任一挂掉不让发版:

1. **隔离**: A 租户的数据, B 租户的 JWT 看不到
2. **污染**: 100 并发交替切租户, 连接归池后 search_path 已 reset
3. **越权**: 伪造 tenant_id 的 JWT 在 tenant 中间件被拒
4. **状态**: suspended 租户的合法 JWT → 403
5. **跨 schema 静态扫描**: 扫 `internal/{platform,modules}/<X>/infrastructure/` 下 SQL 字面值确认无跨 schema JOIN
6. **DDD 边界静态扫描**: 用 `go list -deps` + 脚本验证 §1.4 各条依赖规则无违反
7. **模块隔离扫描** (v2.1 新增): 验证
   - 没有 `internal/platform/<X>` import 任何 `internal/modules/`
   - 没有 `internal/modules/<X>` import 任何 `internal/modules/<Y>` (`X != Y`)
   - 前端 workspace 依赖图: `@iop/module-*` 之间无 depend; `@iop/shell` 不 depend `@iop/module-*`
8. **多产品装配冒烟** (v2.1 新增): 每个 `cmd/<product>/` 至少跑一次 `make build && ./server migrate up` 验证可装配可启动可迁移; 第二个产品出现后才生效

---

## §3 上下文/模块详细设计

每个上下文/模块按 "**领域语言 → 聚合 → 领域服务 → 领域事件 → 应用服务 → 基础设施 → 接口**" 七层叙述。

**分类**:

| 类别 | 子节 | 数量 | 说明 |
|---|---|---|---|
| **平台上下文 (Platform)** | §3.1 §3.2 §3.4–§3.9 | 8 必带 | 任何产品都必须装配 |
| **平台增强 (v1.5+)** | §3.16 | 1 (占位) | AppCenter 应用中心, v2.1.1 仅留架构占位 |
| **业务模块 (Modules) — M4a** | §3.3 §3.10 §3.11 §3.12 | 4 | Problem / OKR / Contacts / Announcements |
| **业务模块 (Modules) — M4b** | §3.13 §3.14 §3.15 | 3 | Documents / Assets / Party |

> v2.0 → v2.1 的位置变化: **§3.3 Problem 从「核心域上下文」改为「首个业务模块 (modules/problem)」**, 内部领域设计完全不变, 仅目录位置由 `internal/contexts/problem/` 改为 `internal/modules/problem/`。其他 7 节 (Tenancy / IAM / Audit / Notification / FileStorage / Dictionary / Localization / PlatformConfiguration) 由 `internal/contexts/<X>/` 改为 `internal/platform/<X>/`。
>
> v2.1 → v2.1.1 的扩充: 新增 6 个业务模块 §3.10–§3.15 (OKR / Contacts / Announcements / Documents / Assets / Party) 与 1 个平台增强 §3.16 (AppCenter, 占位)。§3.2 IAM 内加入 §3.2.x PersonalProfile 扩展能力。命名采用行业标准 (`problem` / `okr` / `contacts` / `announcements` / `documents` / `assets` / `party` / `appcenter`)。

### 3.1 Tenancy (租户)

**领域语言** (Ubiquitous Language):
- Tenant (租户): 一家使用平台的企业组织
- Tenant Slug: 租户的 URL 友好标识 (如 `acme`)
- Tenant Schema: 该租户独占的 PostgreSQL schema (如 `tenant_acme`)
- Tenant Member (租户成员): 某平台用户在某租户内的成员档案
- Tenant Status: `active / suspended / closed`

**聚合**:
- `Tenant` (聚合根) — 含 ID, Slug, Name, Status, CreatedAt, SuspendedAt, ClosedAt
  - 不变量: `Slug` 全局唯一; `Status` 转换符合 `active → suspended → active | closed`
- `TenantMember` (聚合根) — 含 PlatformUserID, TenantID, MemberID, JoinedAt, Status
  - 不变量: `(PlatformUserID, TenantID)` 唯一

**领域服务**:
- `SchemaProvisioner` — 给定 TenantSlug, 执行 `CREATE SCHEMA` + 跑模板迁移 + 写迁移历史

**领域事件**:
- `TenantCreated{ TenantID, Slug, Name, OccurredAt }`
- `TenantSuspended{ TenantID, Reason, OccurredAt }`
- `TenantResumed{ TenantID, OccurredAt }`
- `TenantClosed{ TenantID, OccurredAt }`
- `MemberJoined{ TenantID, MemberID, PlatformUserID, OccurredAt }`

**应用服务 (用例)**:
- `CreateTenant(cmd) → Tenant` — 创建租户, 执行 schema provisioning, 发 `TenantCreated`
- `SuspendTenant(cmd) → ()` — 冻结, 发 `TenantSuspended`
- `ResumeTenant(cmd) → ()`
- `CloseTenant(cmd) → ()` — 销户, 30 天后 DROP SCHEMA
- `JoinMember(cmd) → TenantMember`
- `ListActiveTenants() → []Tenant` (Query)
- `GetTenant(id) → Tenant` (Query, LRU cache 30s TTL)

**基础设施**:
- `pgTenantRepo` (用 `PlatformDB`)
- `pgMemberRepo` (用 `PlatformDB`)
- `lruTenantCache` 装饰 `TenantRepo` 加 30s TTL 缓存
- `schemaProvisioner` (实现 domain 的接口, 真实执行 SQL)

**接口**:
- `POST /tenants` — 平台管理员开通租户
- `PATCH /tenants/{id}` — 改状态
- `GET /tenants` — 列举 (平台管理员)

### 3.2 IAM (身份与访问)

**领域语言**:
- Platform User (平台用户): 跨租户的身份 (email + 密码)
- Identity Provider: 提供身份验证的实体 (Local/OIDC/LDAP)
- Session: 一次登录后的会话, 关联 JWT
- Role: 租户内的角色
- Policy Rule: 资源-动作-效果 规则
- Permission: 对某资源的某动作的允许/拒绝

**聚合**:
- `PlatformUser` — 含 ID, Email, PasswordHash, MfaSecret, Status, LastLoginAt
  - 不变量: `Email` 全局唯一; 密码满足 `PasswordPolicy`
- `Session` — 含 ID, PlatformUserID, TenantID, MemberID, IssuedAt, ExpiresAt, Revoked
  - 不变量: `ExpiresAt > IssuedAt`
- `Role` — 含 ID, TenantID, Code, Name, Policies[]
  - 内含 `PolicyRule` 值对象 (Resource, Action, Effect)

**值对象**:
- `Email` — 校验格式 (RFC 5322 子集)
- `Password` — 校验长度+复杂度 (最低 10 位含字母+数字)
- `HashedPassword` — bcrypt cost=12 (不可逆)
- `TokenPair{ AccessToken, RefreshToken, AccessExpiresAt, RefreshExpiresAt }`

**领域服务**:
- `PasswordPolicy` — 校验密码; 检查锁定 (5 次失败锁 15min)
- `TokenSigner` (接口) — 签发/验签 JWT; 实现是 RS256/HS256
- `IdentityProvider` (接口) — `Authenticate(credentials) → Identity`; v1 实现 `LocalProvider`
- `Enforcer` (接口) — `Check(subject, resource, action) → bool`; 实现是 Casbin 适配器

**领域事件**:
- `UserLoggedIn{ PlatformUserID, TenantID, OccurredAt, IPAddress }`
- `UserLoggedOut{ SessionID, OccurredAt }`
- `LoginFailed{ Email, Reason, OccurredAt }` (审计敏感)
- `RoleGranted{ MemberID, RoleCode, GrantedBy, OccurredAt }`
- `PasswordChanged{ PlatformUserID, OccurredAt }`

**应用服务**:
- `Login(cmd) → TokenPair` — 调 IdentityProvider + 发 JWT + 发事件
- `Refresh(cmd) → TokenPair`
- `Logout(cmd) → ()`
- `SwitchTenant(cmd) → TokenPair` — 校验 membership 后重发 JWT
- `Enforce(ctx, resource, action) → ()` — 用于 RBAC 中间件
- `ListMyTenants() → []TenantBrief` (Query, 经 Tenancy 上下文反查)
- `AddRole(cmd) → ()`, `AddPolicy(cmd) → ()`

**基础设施**:
- `jwtSigner` 实现 TokenSigner (用 `github.com/golang-jwt/jwt/v5`)
- `bcryptHasher` 实现密码哈希
- `localProvider` 实现 IdentityProvider (查 pgUserRepo + bcrypt)
- `casbinEnforcer` 实现 Enforcer; policy 从 `public.casbin_rule` 加载
- `pgUserRepo`, `pgSessionRepo` (Session 黑名单走 Redis), `pgRoleRepo`

**接口**:
- `POST /auth/login | /auth/refresh | /auth/logout`
- `POST /auth/switch-tenant`
- `GET /me | /me/tenants`
- **中间件** (interface/middleware.go):
  - `JWTAuth()` — 验签 + 注入 claims 到 ctx
  - `SessionCheck()` — Redis 黑名单
  - `RBAC(resource, action)` — `r.GET("/problems/:id", iam.RBAC("problem:problem", "read"), h.Get)`

**内置角色 v1**: `platform_admin` (公司侧管理员) / `tenant_admin` (租户内最高权限) / `tenant_member` (普通成员)。租户内更细的角色由 tenant_admin 在 v1.5 自助配置。

#### 3.2.x PersonalProfile 扩展 (M4a 范围, 内化于 IAM)

**背景**: 个人档案 (头像/简历/技能/教育/工作经历/证书/导出/分享) 与 `TenantMember` 数据天然在同一聚合内, 故不开独立 module, 作为 IAM 模块的扩展能力存在。

**聚合扩展**:
- `TenantMember` 增加字段:
  - `profile JSONB` — 结构化扩展信息 (姓名/性别/出生年月/部门/职位/电话/简介/技能数组/教育经历数组/工作经历数组)
  - `avatar_attachment_id UUID` — 通过 `platform/filestorage` BizRef `(iam.member, member_id)` 关联头像
  - `certificates_attachment_ids UUID[]` — 同样走 FileStorage 关联证书/资质 PDF 等非结构化数据

**值对象**:
- `Profile{ Name, Gender, BirthDate, DeptID, Position, Phone, Bio, Skills[], Education[], Experience[] }` — 不变量: Skills/Education/Experience 数组长度上限 50; Bio ≤ 2000 字符
- `ShareToken{ TokenID, MemberID, IssuedAt, ExpiresAt, ReadFields[] }` — 临时只读分享链接令牌

**领域服务**:
- `ProfileValidator` — 校验 profile JSON schema; 校验附件归属 (member 只能挂自己的附件)

**领域事件**:
- `ProfileUpdated{ MemberID, ChangedFields[], OccurredAt }`
- `ProfileShared{ MemberID, ShareTokenID, IssuedTo, OccurredAt }` (审计)
- `AvatarUpdated{ MemberID, OldAttachmentID, NewAttachmentID, OccurredAt }`

**应用服务**:
- `UpdateProfile(cmd) → ()` — 校验后更新 profile JSONB, 发事件
- `UpdateAvatar(cmd) → ()` — 上传走 filestorage, 仅写 attachment_id 引用
- `AddCertificate / RemoveCertificate(cmd) → ()`
- `ExportProfile(memberID, format=pdf|excel|vcard) → bytes` — 模板渲染, 走文件流响应
- `ShareProfile(cmd) → ShareToken` — 颁发临时只读 token, 默认 7 天过期
- `RevokeShare(tokenID) → ()`
- `GetSharedProfile(token) → ProfileView` — 公开端点 (无需登录, 走 token 校验)

**基础设施**:
- 沿用 `pgMemberRepo` (profile 列 + attachment id 列)
- `redis` 存 ShareToken (TTL 自然过期)
- `pdfRenderer` (M4a 引入轻量模板引擎, 例如 wkhtmltopdf 或 gofpdf)
- 与 `platform/filestorage` 通过 `module.Deps.File` 客户端通信, 不直接 import filestorage

**接口**:
- `GET /me/profile` — 查自己
- `PUT /me/profile` — 改自己 (整个 profile JSONB)
- `POST /me/profile/avatar` — 多 part 上传 (内部转发 filestorage)
- `POST /me/profile/certificates` / `DELETE /me/profile/certificates/{id}`
- `GET /me/profile/export?format=pdf` — 导出
- `POST /me/profile/share` → ShareToken
- `DELETE /me/profile/share/{tokenId}`
- `GET /public/profile/{token}` — 凭 token 查 (走 share token 中间件, 不走 JWT)

**为什么不单独成模块**:
- 数据 (`profile` 列) 强属于 `TenantMember` 聚合, 拆出去会违反聚合一致性
- 共享 IAM 已有的 RBAC + 部门关系, 不需要再发一遍事件
- 单独成模块的产品价值不足以抵消跨模块查询代价 (没有客户会"只买个人档案"产品)

### 3.3 Problem (问题协同) — ★ 业务模块 #1 (`modules/problem`, M4a)

**领域语言**:
- Problem (问题): 跨部门重大事项, 需多阶段协同办理
- Stage (阶段): 问题办理流程的步骤; 8 个枚举 (submit/review/propose/meeting/arbitrate/consult/implement/evaluate)
- Branch (分支): 经过 propose 后, 走 `dispute` (会商→裁决) 还是 `consensus` (征求意见)
- Measure (举措): propose/implement 阶段的具体行动项
- Dispute (争议): meeting 阶段记录的分歧
- DisputePosition (争议立场): 某成员在某争议中的观点
- Evaluation (评价): evaluate 阶段的满意度评分 + 评论
- Action (动作): 推进阶段的 HTTP 入口 (8 个 stage 共用 `POST /problems/:id/actions/:stage`)

**聚合**:
- `Problem` (聚合根) — 含 ID, TenantID, Title, Body, Category, Priority, CurrentStage, Branch, SubmitterMemberID, AssigneeMemberID, CreatedAt, ClosedAt
  - 内含: `StageHistory[]` (子实体, 不可独立保存)
  - 不变量: `CurrentStage` 转换必须经 `StageEngine.Advance` 合法; `Branch` 仅在 propose 之后才能设置
- `Measure` (聚合根, 独立, 通过 ProblemID 关联) — 因为可能批量增删, 与 Problem 解耦
- `Dispute` (聚合根, 独立) — 内含 `DisputePosition[]` 值对象
- `Evaluation` (聚合根, 独立)
- `Message` (聚合根, 独立) — 问题相关的协作消息流

**值对象**:
- `Stage` — 8 阶段枚举 + 元数据 (label, sequence, on_branch)
- `Branch` — `dispute` / `consensus`
- `ProblemCategory` — dict_code (走 Dictionary 上下文, 字面如 `strategy_planning`)
- `Priority` — dict_code (`urgent` / `high` / `normal`)
- `EvaluationScore` — 1..5 整数

**领域服务**:
- `StageEngine` — 纯函数, 无 receiver:
  - `Advance(cur Stage, branchChoice Branch) (Stage, error)` — 状态转换
  - `ValidateAction(p *Problem, requested Stage) error` — 校验 action 合法性
  - `DeriveStatus(s Stage, evaluated bool) string` — 派生展示状态

**领域事件**:
- `ProblemSubmitted{ ProblemID, SubmitterMemberID, OccurredAt }`
- `ProblemReviewed{ ProblemID, ReviewerMemberID, AssigneeMemberID, OccurredAt }`
- `ProposalMade{ ProblemID, MeasureIDs[], OccurredAt }`
- `BranchChosen{ ProblemID, Branch, OccurredAt }`
- `MeetingHeld{ ProblemID, DisputeID, OccurredAt }`
- `Arbitrated{ ProblemID, DecisionMakerMemberID, ResolutionText, OccurredAt }`
- `Consulted{ ProblemID, OccurredAt }`
- `Implemented{ ProblemID, OccurredAt }`
- `Evaluated{ ProblemID, Score, OccurredAt }`
- `ProblemClosed{ ProblemID, OccurredAt }`

**应用服务**:
- 8 个推进阶段的 Command Handlers, 每个统一模式:
  ```
  TenantDB.Transaction →
    1. 加载 Problem 聚合 (含 stage_history)
    2. 调 StageEngine.ValidateAction 校验
    3. 写新数据 (措施/争议/评价)
    4. 调 problem.Advance() 推进 stage
    5. 追加 StageHistory
    6. 仓储 Save
    7. 收集领域事件, 在 commit 后异步 publish 到 eventbus
  ```
- `DashboardQuery()` — CQRS 读模型, 直查 DB 投影到 DTO, 不走聚合 (性能优化)
- `ListProblems(filter)`, `GetProblem(id)` (Query)

**基础设施**:
- `pgProblemRepo` — 实现 ProblemRepo, 用 TenantDB
- `dashboardQueryPg` — 直接 SQL 聚合查询, 输出 DashboardDTO

**接口**:
- `GET /dashboard/overview`
- `GET/POST /problems`, `GET /problems/:id`
- `POST /problems/:id/actions/:stage` — 8 个 stage 共用一个 endpoint, 内部分发
- `GET /messages/problem/:id`
- 与 gallant v1 路径完全一致 (前端零路径改动); 渐进改造 `/files/problem/:id` → `/files?biz=problem&id=:id`

### 3.4 Audit

**领域语言**: Audit Log (审计流水), Audit Action, Audit Actor (执行者), Audit Target (操作对象).

**聚合**:
- `AuditLog` (聚合根, 租户内) — ID, TenantID, ActorMemberID, Action, TargetType, TargetID, Diff (JSONB), TraceID, OccurredAt
- `PlatformAudit` (聚合根, 跨租户) — ID, ActorPlatformUserID, Action, IPAddress, TraceID, OccurredAt

**领域服务**: `AuditWriter` (接口) — 写审计 (异步)

**领域事件**: 本上下文**不发**事件 (它是订阅者, 不是发布者), 反向订阅其他上下文的领域事件。

**应用服务**:
- `RecordEvent(cmd)` — 同步入 buffer (默认 channel 1000)
- `FlushBuffer()` — 后台 worker 周期 flush (200ms 或 50 条), buffer 满走同步写防丢
- `ListAuditLogs(filter)` (Query, 合规检索)

**基础设施**:
- `bufferedWriter` 实现 AuditWriter (channel + worker pool + 同步兜底)
- `eventSubscriber` — 启动时订阅 eventbus 上所有上下文的领域事件, 转换成 AuditLog 写入

**接口**: 主要内部使用; 仅暴露 `GET /audit/logs` 给 platform_admin。

### 3.5 Notification

**领域语言**: Notification (通知), Channel (站内信/邮件/SMS), Recipient (接收者).

**聚合**:
- `Notification` (聚合根) — ID, RecipientMemberID, Type, Title, Body, Link, ReadAt, OccurredAt
  - 不变量: `ReadAt` 一旦设置不可清除

**值对象**:
- `NotifyType` — dict_code (`urgent / mention / approval / comment / system / app`)
- `RecipientID` — MemberID

**领域服务**: `NotificationDispatcher` — 决定一条 Notification 走哪些 Channel.

**领域事件**:
- `NotificationSent{ NotificationID, RecipientMemberID, Channels[], OccurredAt }`
- `NotificationRead{ NotificationID, ReadAt }`

**应用服务**:
- `SendNotification(cmd)` — 显式发送 (其他上下文调用)
- `ListUnread(memberID) → []Notification` (Query)
- `MarkRead(memberID, ids[])`
- **事件订阅 handlers**:
  - 订阅 `problem.ProblemReviewed` → 给 Submitter 发"你的问题已被分办"
  - 订阅 `problem.Arbitrated` → 给参会成员发"裁决结果"
  - 订阅 `tenancy.TenantSuspended` → 给租户所有成员发"账号暂停"
  - ... (订阅关系在 `notification/application/event_handlers.go` 集中声明)

**基础设施**:
- `pgNotifRepo` (TenantDB)
- `inAppChannel` — 站内信 (写 DB + WebSocket M5+)
- `smtpChannel` — v1 noop, 预留 SMTP 实现

**接口**: `GET /notifications/unread`, `POST /notifications/mark-read`.

### 3.6 FileStorage

**领域语言**: Attachment (附件), Object Key (对象存储键), Biz Reference (业务关联).

**聚合**:
- `Attachment` (聚合根) — ID, TenantID, BizModule, BizID, Stage, FileName, FileSize, ContentType, ObjectKey, UploaderMemberID, UploadedAt

**值对象**:
- `ObjectKey` — `tenants/<tenant_slug>/<biz_module>/<biz_id>/<stage>/<uuid>-<safe_name>`
- `BizRef` — (BizModule, BizID) 元组, 通用关联到任意业务对象

**领域服务**:
- `StorageProvider` (接口) — `Upload(key, reader) → ()`, `Download(key) → reader`, `Delete(key)`

**领域事件**:
- `FileUploaded{ AttachmentID, BizRef, OccurredAt }`
- `FileDeleted{ AttachmentID, OccurredAt }`

**应用服务**: `UploadFile / DownloadFile / ListFiles / DeleteFile`.

**基础设施**:
- `minioProvider` 实现 StorageProvider
- `pgAttachmentRepo` (TenantDB)

**接口**: `GET/POST/DELETE /files`, `GET /files?biz=<module>&id=<id>`.

### 3.7 Dictionary

**领域语言**: Dict Type (字典类型), Dict Item (字典项), Override (租户级覆盖).

**聚合**:
- `DictType` (聚合根, public) — Code, Name, Scope (`platform` / `tenant`)
- `DictItem` (聚合根, public) — TypeCode, Code, Label, ParentCode, Sort, Enabled
- `DictItemOverride` (聚合根, tenant) — TypeCode, Code, LabelOverride

**应用服务**:
- `Lookup(ctx, typeCode) → []DictItem` (合并平台默认 + 租户 override)
- `AddOverride(cmd)`
- `ReloadCache()`

**基础设施**:
- `pgDictRepo` (PlatformDB for type/item, TenantDB for override)
- `redisDictCache` — 缓存 lookup 结果, 失效后回源

**接口**: `GET /dict/:type_code`.

**约定** (重要, 跨上下文): 所有有限枚举字段约定存 `dict_code` 而非字面值。例:
```sql
problem.category VARCHAR(32)  -- 存 'strategy_planning' (code), 不存 '战略规划' (label)
```
未来打开"租户自定义字典"零业务改动。

### 3.8 Localization (i18n)

**领域语言**: Locale (语言区域), Translation (翻译), Key (i18n 键).

**聚合**: `Translation` (聚合根) — Locale, Key, Value, UpdatedAt.

**应用服务**: `Translate(ctx, key, args...) → string`.

**基础设施**: v1 从 `configs/i18n/<locale>/*.yaml` 静态加载; M5+ 可加 DB 后端。

**约定**: 所有面向用户的字符串走 `localization.T(ctx, "key", args...)`; v1 仅 zh-CN, 但 key 体系已建立。错误消息走错误 Kind 对应的 code 自动查 i18n。

### 3.9 PlatformConfiguration (跨切关注集合, 严格说不是上下文)

放在 `shared/platform_config/`, 因为它没有自己的业务规则, 只是一个 key-value 抽象。

**接口**: `Get(ctx, key) (any, error)` — v1 从 YAML 读; M3+ 加 `tenant_<slug>.config_override` 表 + Redis 缓存做租户级覆盖。

**使用约定**: 所有"政策性参数" (auth.jwt.access_ttl, audit.buffer_size, notify.poll_interval, ...) 走 `config.Get(ctx, "...")` 而非硬编码或读 `cfg.Auth.JwtAccessTTL`。

---

### 3.10 OKR (工作安排) — ★ 业务模块 #2 (`modules/okr`, M4a)

**领域语言**:
- Plan (计划): 时间维度的目标集合; 4 级: `Year / HalfYear / Month / Week`
- PlanItem (条目): 一条具体计划项; 含 owner / due_date / weight / status
- Decompose (分解): 上级 Plan 拆为下级 Plan 的关系
- Report (报告): 用户提交的工作汇报; 类型 `Daily / Weekly`
- ReportEntry (报告条目): 报告内的一行, 关联 PlanItem (可选)
- Rollup (汇总): 管理人员对下属周报的聚合视图
- Cadence (节奏): 周/月/季的开始结束日, 走 `dictionary` 的日历定义

**聚合**:
- `Plan` (聚合根) — Level, OwnerMemberID, PeriodStart, PeriodEnd, Title, ParentID(可空), Items[], Status (`draft/active/closed`)
  - 不变量: `PeriodStart < PeriodEnd`; 同 owner 同 level 同 period 唯一; PlanItem.weight 总和 ≤ 100
- `Report` (聚合根) — Type, OwnerMemberID, PeriodStart, PeriodEnd, Entries[], SubmittedAt, ReadBy[]
  - 不变量: Weekly Report 的 period 必须跨越 7 天; Daily Report 跨 1 天

**值对象**:
- `PlanLevel` 枚举: Year / HalfYear / Month / Week
- `ReportType` 枚举: Daily / Weekly
- `Progress{ percent, summary }`

**领域服务**:
- `Cadence` — 给定 (level, date) 计算周/月/半年/年的 (start, end)
- `RollupView` — 给定 manager + week, 聚合下属周报为 RollupReport (按部门/按人/按 PlanItem 三种维度)
- `PlanDecomposer` — 校验 child plan 的 period 与 weight 不超出 parent

**领域事件**:
- `PlanCreated / PlanItemAdded / PlanItemCompleted / PlanClosed`
- `DailySubmitted / WeeklySubmitted{ ReportID, OwnerMemberID, PeriodEnd }`
- `WeeklyOverdue` (定时器触发, 提醒未提交)

**应用服务**:
- `CreatePlan(cmd) → Plan` / `AddPlanItem` / `CompletePlanItem` / `ClosePlan`
- `DecomposeFrom(parentID, childLevel) → Plan` — 复制 PlanItems 提示新建
- `SubmitDailyReport(cmd) → Report` / `SubmitWeeklyReport(cmd) → Report`
- `ListMyPlans(period, level) → []Plan`
- `RollupWeekly(managerID, periodEnd) → RollupReport` — Query 侧, 直查 DB 不走聚合
- `CommentReport(reportID, comment) → ()`
- `RemindOverdue(periodEnd) → ()` — cron 触发, 发未交周报通知

**基础设施**:
- `pgPlanRepo / pgReportRepo / pgRollupQueryPG`
- `cron` 调度器 (轻量, 走 `robfig/cron/v3`); 每周一 9:00 触发 RemindOverdue
- 通知通过 `module.Deps.Notify`, 不直接 import `platform/notification`

**接口**:
- `GET /plans?level=&period=` / `POST /plans` / `PATCH /plans/:id` / `POST /plans/:id/items` / `PATCH /plans/:id/items/:itemId/complete`
- `POST /reports/daily` / `POST /reports/weekly` / `GET /reports?type=&period=`
- `GET /rollups/weekly?period=&dept=` (要求 manager 角色)
- `POST /reports/:id/comments`

**业务模块间不互相 import**: 与 problem 无直接关联; 如需"问题 → 计划项"双向链接, 用通用的 BizRef + 应用层组装。

---

### 3.11 Contacts (通讯录) — ★ 业务模块 #3 (`modules/contacts`, M4a)

**领域语言**:
- OrgContact (组织联系人): 来自 IAM TenantMember 的投影 (姓名/部门/职位/邮箱/电话)
- PersonalContact (个人联系人): 用户自己维护的外部联系人
- ContactGroup (分组): 用户对自己常联系人的分组
- Favorite (收藏): 用户对 OrgContact 或 PersonalContact 的快速访问标记
- ContactsQuery (检索): 跨 OrgContact + PersonalContact 的统一搜索

**聚合**:
- `PersonalContact` (聚合根) — OwnerMemberID, Name, Phone, Email, Company, Notes, Tags[], CreatedAt
  - 不变量: (OwnerMemberID, Phone) 不强制唯一 (允许同人多号)
- `ContactGroup` (聚合根) — OwnerMemberID, Name, MemberRefs[] (含 type=org/personal + id)
- `Favorite` (实体, 不是聚合根) — OwnerMemberID, TargetType, TargetID

**值对象**:
- `ContactRef{ Type: org|personal, ID }` (区分投影与个人)
- `ContactCard{ Name, Phone, Email, Avatar, Title, Dept }` (查询/导出的统一视图)

**领域服务**:
- `OrgContactProjector` — 订阅 IAM 事件 (`MemberJoined / ProfileUpdated / MemberLeft`) 维护本地投影 (避免每次查询都跨 schema)

**领域事件**:
- `PersonalContactAdded / PersonalContactUpdated / PersonalContactDeleted`
- `GroupCreated / GroupMemberAdded`
- `FavoriteToggled`
- v1.5+: `ChatMessageSent` (聊天功能)

**应用服务**:
- `SearchOrg(query, dept?) → []ContactCard` — 走 OrgContact 投影 + 全文索引
- `ListPersonal / AddPersonal / UpdatePersonal / DeletePersonal`
- `ListGroups / CreateGroup / AddToGroup / RemoveFromGroup`
- `ToggleFavorite(ref)` / `ListFavorites`
- `ImportVCard(file) → []PersonalContact` / `ExportVCard(scope) → bytes`

**基础设施**:
- `pgPersonalContactRepo / pgContactGroupRepo / pgFavoriteRepo`
- `pgOrgContactProjector` — 跑 EventSubscriber, 维护 `tenant_<slug>.org_contact_projection` 表
- `pgvector` 或 `pg_trgm` 索引支持模糊检索 (可选, 性能扩展点)

**接口**:
- `GET /contacts/org?q=&dept=` / `GET /contacts/org/:memberId`
- `GET /contacts/personal` / `POST /contacts/personal` / `PATCH/DELETE /contacts/personal/:id`
- `GET /contacts/groups` / `POST /contacts/groups` / `PATCH /contacts/groups/:id`
- `POST /contacts/favorites` / `DELETE /contacts/favorites/:targetType/:targetId`
- `POST /contacts/import/vcard` / `GET /contacts/export/vcard`

**v1.5+ 留口**: `chat/` 子聚合 (Conversation + Message + Participant), 通过 SSE 推送, 路由 `/contacts/chat/*`。该接入点在 §3.11 聚合层的 `ContactCard` 值对象中已为"快捷发起会话"按钮保留 hook, 不影响 v2.1.1 范围。

---

### 3.12 Announcements (时政热点) — ★ 业务模块 #4 (`modules/announcements`, M4a)

**领域语言**:
- Post (帖文): 一篇资讯; 含标题/正文/封面/标签/发布时间/作者
- Tag (标签): 简单字符串分类
- PublishWorkflow (发布流): `draft → review → published / rejected`
- Hot (热度): 浏览次数 / 评论数 加权计算

**聚合**:
- `Post` (聚合根) — Title, BodyMarkdown, CoverAttachmentID(可空), AuthorMemberID, Tags[], Status, PublishedAt, ViewCount
  - 不变量: Title ≤ 200 字; published 状态后不允许改 status 回 draft (要新建 revise)

**值对象**:
- `PostStatus` 枚举: Draft / Review / Published / Rejected / Archived
- `Tag` (字符串, 最多 5 个/帖)

**领域服务**:
- `HotScore` — 给定 Post, 计算热度 = `log10(views+1) * 2 + comment_count * 0.5 - days_since_publish * 0.1`

**领域事件**:
- `PostDrafted / PostSubmittedForReview / PostPublished / PostRejected / PostArchived`
- `PostViewed{ PostID, ViewerMemberID, OccurredAt }` (异步批量计数)

**应用服务**:
- `CreatePost / SubmitForReview / Approve / Reject` (走简单审批流, 角色由 RBAC 决定)
- `Publish / Archive / Revise (clone draft)`
- `ListPosts(filter: tag/status/period, sort: latest|hot, page)` (Query)
- `GetPostDetail(id) → PostView` — 自带 IncrViewCount 副作用
- `AddTag / RemoveTag`

**基础设施**:
- `pgPostRepo` + `redisHotCache` (热度排行榜每 10 分钟刷新)
- `mdRenderer` (Markdown → safe HTML, 走 `bluemonday` 防 XSS)
- 封面附件经 `module.Deps.File`

**接口**:
- `GET /news/posts?tag=&sort=&page=` / `GET /news/posts/:id`
- `POST /news/posts` (draft) / `POST /news/posts/:id/submit` / `POST /news/posts/:id/publish` (RBAC: editor)
- `POST /news/posts/:id/archive`
- `GET /news/tags`

**Notification 集成**: `PostPublished` 事件被 `platform/notification` 订阅, 推送给已订阅"时政热点"频道的用户 (订阅关系由用户在 NotifyCenter 自助开关, 走 platform_config + member 偏好)。

---

### 3.13 Documents (资源平台) — ★ 业务模块 #5 (`modules/documents`, M4b)

**领域语言**:
- Resource (资源): 用户上传的一份资料 (含文件 + 元数据)
- ShareScope (共享范围): `private / dept / tenant`
- DownloadLog (下载日志): 谁、何时、下载了哪份
- ResourceCategory (分类): 字典驱动 (走 `platform/dictionary`)

**聚合**:
- `Resource` (聚合根) — UploaderMemberID, Title, Description, Category, Tags[], AttachmentID (走 FileStorage), Size, ShareScope, Downloads (计数), CreatedAt
  - 不变量: 私有资源不出现在他人检索结果; 部门共享只对同部门可见

**值对象**:
- `ShareScope` 枚举
- `ResourceQuery{ keyword, category, tagsAny[], sharedOnly, minSize, maxSize, sort }`

**领域事件**:
- `ResourceUploaded / ShareScopeChanged / ResourceDownloaded / ResourceDeleted`

**应用服务**:
- `Upload(cmd: multipart) → Resource` — 内部转发 FileStorage 拿 AttachmentID
- `UpdateMetadata / ChangeShareScope / Delete`
- `Search(query, viewerCtx) → []Resource` — viewerCtx 决定可见范围
- `Download(id, viewerCtx) → stream` — 写 DownloadLog
- `ListMine() / ListShared()`

**基础设施**:
- `pgResourceRepo / pgDownloadLogRepo`
- 检索: 先 LIKE 起步, M4b 末观察容量再决定是否上 `pg_trgm` / 倒排
- 文件流走 `module.Deps.File` 提供的 download presign (或后端中转, 同 problem 选择)

**接口**:
- `POST /resources` (multipart) / `GET /resources?...` / `GET /resources/:id`
- `PATCH /resources/:id` (改元数据/share scope) / `DELETE /resources/:id`
- `GET /resources/:id/download` (走 FileStorage 鉴权)
- `GET /resources/mine` / `GET /resources/categories` (代理到 dictionary)

---

### 3.14 Assets (资产管理) — ★ 业务模块 #6 (`modules/assets`, M4b)

**领域语言**:
- Asset (资产): 个人录入的一项资产
- AssetCategory (类别): 字典驱动 (设备/家具/不动产/金融/其他)
- AssetHistory (变更历史): 价值/状态变更追踪
- DeptScope (部门范围): 部门管理员视角的可见性

**聚合**:
- `Asset` (聚合根) — OwnerMemberID, Name, Category, AcquiredAt, Cost, CurrentValue, Status (`active/retired/lost`), SerialNumber, Location, AttachmentIDs[] (照片/发票), Notes
  - 不变量: OwnerMemberID 不可变 (转移走 retire + 新增); CurrentValue ≥ 0

**值对象**:
- `Money{ Amount, Currency }` (走 dictionary 默认 CNY)
- `AssetStatus` 枚举

**领域服务**:
- `DeptScope` — 给定 viewerMemberID, 查 IAM 部门关系 → 返回可见 ownerMemberID 集合; 部门管理员看本部门, 普通员工只看自己

**领域事件**:
- `AssetCreated / AssetUpdated / AssetRetired / AssetTransferred`

**应用服务**:
- `CreateAsset / UpdateAsset / RetireAsset / RecordTransfer`
- `ListMyAssets(filter)` / `ListDeptAssets(deptID, filter)` — 后者需角色为 dept_admin
- `Export(scope: mine|dept, format: excel|csv)`

**基础设施**:
- `pgAssetRepo / pgAssetHistoryRepo`
- 部门关系来自 IAM, 经 `module.Deps` 上 `DeptClient.GetDept(memberID)` 接口 (定义在 `shared/module/`, 实现在 iam)
- 报表导出走 excelize 库

**接口**:
- `GET /assets/mine` / `POST /assets` / `PATCH /assets/:id` / `POST /assets/:id/retire`
- `GET /assets/dept/:deptId` (要求 dept_admin)
- `GET /assets/export?scope=&format=`

**隐私边界**: 普通员工看不到他人资产; 部门管理员看本部门 (含金额); 跨部门 / 跨租户 不可见。该规则在 `DeptScope` 领域服务 + RBAC 双重保护。

---

### 3.15 Party (党务管理) — ★ 业务模块 #7 (`modules/party`, M4b)

**领域语言**:
- MemberDevelopment (党员发展): 跟踪某成员从申请到正式党员的过程
- DevelopmentStage (发展阶段): 5 步枚举 `applicant (申请) → activist (积极分子) → development (发展对象) → probationary (预备党员) → full (正式党员)`
- PartyActivity (党日活动): 一次活动 (主题/时间/地点/主持人/签到簿)
- Checkin (签到): 单个党员在某活动的签到记录
- StudyMaterial (学习资料): 理论学习的文件 + 元数据
- StudyRecord (学习记录): 个人对某资料的学习状态 (打卡/笔记)

**聚合**:
- `MemberDevelopment` (聚合根) — MemberID, CurrentStage, StageHistory[], Sponsors[](入党介绍人), JoinDate(空表示未到正式)
  - 不变量: Stage 严格按 5 步推进, 不允许跳跃; 介绍人必须是 full 党员
- `PartyActivity` (聚合根) — Title, Theme, StartAt, EndAt, Location, OrganizerMemberID, Checkins[], MinutesAttachmentID (会议纪要)
- `StudyMaterial` (聚合根) — Title, Category, AttachmentID, RequiredFor[] (`applicant/activist/...`), Tags[]
- `StudyRecord` (实体) — MemberID, MaterialID, StudiedAt, Note

**值对象**:
- `DevelopmentStage` 枚举
- `StageTransition{ FromStage, ToStage, OccurredAt, ApproverMemberID, EvidenceAttachmentIDs[] }`

**领域服务**:
- `StageAdvancer` — 校验跨阶段的硬性条件 (例: development → probationary 需 2 名介绍人都是 full 党员, 且申请人有 6 个月以上的 activist 状态)
- `StudyComplianceChecker` — 给定 memberID, 列出"必读但未读"的资料

**领域事件**:
- `DevelopmentAdvanced{ MemberID, FromStage, ToStage }` / `JoinedParty{ MemberID, JoinDate }`
- `ActivityScheduled / Checkedin / ActivityClosed`
- `MaterialUploaded / Studied{ MemberID, MaterialID }`

**应用服务**:
- `RegisterApplicant / AdvanceStage(memberID, evidence)` — StageAdvancer 校验
- `IssueCertificate(memberID) → bytes (PDF)` — 入党证明
- `ScheduleActivity / Checkin(activityID, memberID) / CloseActivity`
- `UploadStudyMaterial / DownloadStudyMaterial / RecordStudy`
- `Reports`: 党员花名册 / 学习达标率 / 月度活动统计

**基础设施**:
- `pgMemberDevelopmentRepo / pgActivityRepo / pgStudyMaterialRepo / pgStudyRecordRepo`
- 附件走 `module.Deps.File`
- PDF 渲染同 IAM 的 ProfileExport (复用 pdfRenderer)

**接口**:
- `GET /party/members` / `GET /party/members/:id` / `POST /party/members` / `POST /party/members/:id/advance`
- `GET/POST /party/activities` / `POST /party/activities/:id/checkin` / `POST /party/activities/:id/close`
- `GET/POST /party/study/materials` / `POST /party/study/materials/:id/record`
- `GET /party/reports/{roster|study-rate|monthly}`

**装载提示**: 该模块特定于中国国情, 出海产品 (`cmd/<oversea-product>`) 不应装载。这是验证模块独立打包必要性的最强用例。

---

### 3.16 AppCenter (应用中心) — ★ 平台增强 (v1.5+, 不在 v2.1.1 范围)

> 说明: AppCenter **不是业务模块**, 是平台基座的增强能力, 归 `internal/platform/appcenter/`。v2.1.1 仅保留架构占位 (目录 + module.go 空壳), 完整实现推迟到 v1.5。

**领域语言**:
- TenantModuleEnablement (租户模块开关): 一条 (tenant_id, module_name, enabled) 记录, 控制租户内该模块是否对终端用户可见可调用
- ModuleCatalog (模块清单): 当前二进制装载的所有模块的元信息 (name / description / icon / version)
- EnablementGate (启用门): HTTP 中间件, 检查请求模块是否对当前租户启用, 否则 404

**聚合**:
- `TenantModuleEnablement` (聚合根) — TenantID, ModuleName, Enabled, EnabledAt, EnabledBy
  - 不变量: 同 (TenantID, ModuleName) 唯一; 平台模块强制 enabled, 不可禁用

**领域服务**:
- `ModuleCatalog` — 从 `Registry` 拉取已装载模块列表 + 元信息

**应用服务**:
- `ListCatalog() → []ModuleInfo` (平台 + 业务, 标注是否可切换)
- `Enable(tenantID, moduleName) / Disable(tenantID, moduleName)` (要求 tenant_admin)
- `GetMyMenu(tenantID, memberID) → []MenuEntry` — 按启用状态 × RBAC 过滤的最终菜单

**领域事件**:
- `ModuleEnabledForTenant / ModuleDisabledForTenant` (审计)

**基础设施**:
- `pgEnablementRepo` (位于 `public.tenant_module_enablement`)
- `EnablementGate` 中间件 (位于 `interface/middleware`), 在 RBAC 之前执行
- 与 `Registry` 协作拿到模块装载列表

**接口**:
- `GET /appcenter/catalog` — 列出本二进制装载的模块
- `POST /appcenter/enable / POST /appcenter/disable` (tenant_admin)
- `GET /me/menu` — 前端 shell 调用, 拿到本租户本用户应显示的菜单

**与编译期组合的关系**: AppCenter 不"安装"新模块 — 它只能对**已编译进二进制**的模块进行启停切换。要新增模块仍需升级二进制 (升级流程同 ADR 24)。这维持了 ADR 24 + ADR 26 (桌面端策略) 的一致性, 同时给租户管理员一个轻量自助能力。

---

## §4 跨上下文协作

### 4.1 事件驱动协作 (首选)

业务上下文 (Problem) 发领域事件 → eventbus → 多个订阅者:
- Audit 订阅所有领域事件 → 写审计
- Notification 订阅特定事件 → 触发通知 (订阅关系在 notification/application/event_handlers.go 集中声明)
- 其他业务上下文 (M5+) 可以订阅 → 触发自己的工作流

**eventbus 接口** (位于 `shared/eventbus/bus.go`):
```go
type Bus interface {
    Publish(ctx, eventType string, data any) error
    Subscribe(eventType string, handler Handler)
}
type Handler func(ctx context.Context, event Event) error
type Event struct {
    Type       string
    OccurredAt time.Time
    Trace      string
    Tenant     string
    Actor      string
    Data       any   // 具体领域事件
}
```

**v1 实现**: 进程内 channel + worker pool (默认 8) + panic recover + 扇出多订阅者; 同步发布异步消费, panic 不影响业务事务。

**事件类型常量**: 集中在每个上下文的 `domain/events.go`, 跨上下文订阅时 import 常量, 避免拼字符串。

### 4.2 防腐层 (Anti-Corruption Layer)

当某个外部系统或上下文的模型与本上下文不一致时, 在本上下文的 `infrastructure/` 内建立 ACL 翻译。例:

- Problem 需要"判断某成员是否有 review 权限": 通过 IAM 的 HTTP API 或事件查询时, 在 `problem/infrastructure/iam_acl.go` 翻译 IAM 的 `Permission` 到 problem 自己的 `ReviewerRole`.
- 远期接入企业微信通知: 在 `notification/infrastructure/wecom_acl.go` 翻译 Notification 模型到企业微信 API.

### 4.3 共享内核范围 (最小化)

只允许放在 `internal/shared/`:
1. **kernel/ids.go** — `TenantID, MemberID, PlatformUserID` 等通用 ID 类型
2. **kernel/context.go** — ctx 读写约定 (Trace/Tenant/Member)
3. **errors/** — 错误模型 (Kind + Error struct), 各上下文有自己的错误码段
4. **eventbus/** — 事件总线接口 + 进程内实现
5. **tenantdb/** — `PlatformDB` / `TenantDB` 抽象
6. **platform_config/** — Get(ctx, key) 抽象

**不允许放入共享内核**:
- 任何业务概念 (Tenant 聚合在 tenancy 上下文, 不在共享内核)
- 任何 HTTP / DTO / SDK 客户端
- 任何具体的第三方库胶水代码 (放 `internal/infrastructure/`)

### 4.4 反例 (避免做的事)

| 反例 | 为什么 | 正确做法 |
|---|---|---|
| `modules/problem/domain/` import `platform/iam/domain/` 拿 Role 类型 | 跨上下文 import domain 破坏边界 | 在 application 通过 ctx 拿 member 信息, 或订阅 IAM 事件 |
| 在 `modules/problem/domain/problem.go` 写 SQL | domain 不该感知 DB | 仓储接口在 domain, 实现在 infrastructure |
| 用 GORM hook 写审计 | 业务和审计强耦合, 难测 | domain 发事件, audit 订阅 |
| 跨上下文 JOIN | 强耦合, 拆分难 | 拉两份再 Go 里组装 |
| 把 audit 写入放业务事务里 | 审计失败会回滚业务 | 业务事务发"待发事件", commit 后异步 publish |
| `platform/iam` 直接 import `modules/problem` 的事件类型 | 平台反向依赖业务模块, 产品无法只装载部分模块 | 业务模块订阅 IAM 事件, 而非 IAM 主动给业务模块发通知 |
| 两个业务模块互相 import (例: `modules/order` import `modules/problem`) | 编译期组合失效, 一个产品想砍掉另一个会编译失败 | 通过 eventbus 异步通信; 真有强同步依赖说明边界划错了, 应合并或抽到 platform |

### 4.5 模块打包契约 (v2.1 新增)

为支持「按需装配多产品 / 单业务模块独立打包」, 平台基座必须对模块作出以下稳定性承诺, 写入 ADR 流程后变更需走兼容性评审:

**契约 A · 一个业务模块只依赖平台契约, 不依赖具体实现**
- 模块 import 限于: 标准库 + `internal/shared/{kernel,errors,eventbus,module}` + 同模块自己的子包
- 平台能力通过 `module.Deps` 的接口字段注入 (`Dict / File / Notify`), 这些接口定义在 `shared/module/`, 实现由平台模块在 `app/runtime.go` 装配时反向注册

**契约 B · 模块自治: 自己的迁移、路由、事件、CLI 子命令全部自带**
- `migrations/{platform,tenant}/` 跟模块走, 用 `embed.FS` 打进二进制
- 路由前缀由 `Registry` 统一加 (`/api/<module-shortname>/`), 模块只负责相对路径
- 事件 topic 命名: `<module-name>.<event>`, 例 `modules.problem.problem_submitted` / `platform.iam.user_logged_in`

**契约 C · 平台保证最小产品集装配可启动**
- 即"全部 platform + 零业务模块"必须能编译、启动、跑迁移、跑命门测试; 这是 M2 验收的隐含要求
- 业务模块可装载列表 = `cmd/<product>/main.go` 显式列出的集合, 与租户/数据库内已有数据无关 (产品升级新装一个模块时, 旧租户走 `tenantctl migrate-all --module=<new>`)

**契约 D · 事件发布是发布语言 (Published Language)**
- 任何模块发布的领域事件 schema 变更 = breaking change
- 事件结构含强制字段: `event_id`, `event_module`, `event_topic`, `tenant_id`, `actor`, `occurred_at`, `trace_id`, `version`
- 新增可选字段不算 break; 改字段语义 / 改类型 / 改 topic 名 = 必须新事件 + 弃用旧事件 + 双发一段时间

**契约 E · 前端模块包契约对应**
- 一个 `@iop/module-<X>` 包暴露: `{ name, routes, stores, register(shell) }` 五项 (TypeScript 类型在 `@iop/shell/module-contract.ts`)
- 不允许向 `@iop/shell` 写入除"注册路由 / 注册菜单项 / 注册通知处理器"以外的副作用
- 模块包不能 import 其他模块包 (workspace 依赖图禁止)

**契约 F · 拆服务的退出预案 (远期)**
- 如某模块未来要拆成独立微服务, 走的路径是: 把该模块所在的 `internal/modules/<X>/` 复制到新仓库 + 把它对平台能力的依赖从「Deps 注入」改为「HTTP 调用」; 由于契约 A 限制 import, 这一过程不需要触碰其他业务模块
- 这是设计意图, 不是 v2.1 范围内的工程

---

## §5 横切关注点

### 5.1 错误模型

```go
// internal/shared/errors
type Kind int
const (
    KindValidation Kind = iota  // 400
    KindUnauthorized            // 401
    KindForbidden               // 403
    KindNotFound                // 404
    KindConflict                // 409
    KindInternal                // 500
    KindUpstream                // 502
)
type Error struct {
    Kind    Kind
    Code    string  // i18n key, 如 "problem.action.invalid_stage"
    Message string
    Args    []any
    Cause   error
}
```

约定:
- 领域层 `return errors.New(KindConflict, "problem.action.invalid_stage")`
- 应用层 `return errors.Wrap(cause, KindInternal, "...")`
- handler 不直接写响应, 用 `c.Error(err)` 推到 ErrorMiddleware

### 5.2 错误码区间 (按上下文分段)

| 区间 | 含义 |
|---|---|
| `0` | 成功 |
| `40001-40099` | 通用入参校验 |
| `40100-40199` | IAM 认证错 (token 过期 / session 注销) |
| `40300-40399` | IAM 鉴权错 (RBAC 拒绝) |
| `40400-40499` | 资源不存在 |
| `40900-40999` | 业务状态冲突 |
| `41000-41999` | Tenancy 上下文错 |
| `41100-41199` | IAM 上下文错 (登录失败/锁定) |
| `41200-41299` | Audit 上下文错 |
| `41300-41399` | Notification 上下文错 |
| `41400-41499` | FileStorage 上下文错 |
| `41500-41599` | Dictionary 上下文错 |
| `41600-41699` | Localization 上下文错 |
| `42000-42999` | Problem 上下文错 |
| `43000+` | 后续业务上下文每个一段 (M5+) |
| `50000` | 服务异常 |
| `50200` | 上游不可用 |

### 5.3 结构化日志 (zap)

每条日志必带字段: `trace_id, tenant_id, member_id, path, method, status, latency_ms, context (上下文名)`.

敏感字段黑名单脱敏: `password / password_hash / mfa_secret / token / refresh_token / authorization`.

生产: JSON 行 → stdout → ELK/Loki; 开发: 彩色 console.

### 5.4 链路追踪

**v1**: 仅 `trace_id` (UUID v4) 贯穿, response header 回写 `X-Trace-Id`. 所有 log/audit/error/event 都带 trace_id。

**v1.5 (按需)**: 接 OpenTelemetry. 当且仅当 grep trace_id 不够用时再上。

### 5.5 指标 (Prometheus `/metrics`)

| 指标 | 用途 |
|---|---|
| `http_request_duration_seconds{path,method,status,tenant,context}` | P50/P95/P99 |
| `http_requests_total{...}` | 流量与错误率 |
| `db_query_duration_seconds{op,table,tenant_schema}` | DB 慢查询 |
| `tenant_active_total` | 在线租户数 |
| `audit_buffer_size` | 审计队列水位 (>800 报警) |
| `event_handler_errors_total{event_type,handler_context}` | 事件处理错误 |
| `domain_event_published_total{event_type,context}` | 领域事件吞吐 |
| `business_problem_advanced_total{from_stage,to_stage,branch}` | 业务流转 |

200 租户量级, `tenant` label cardinality 可控。

### 5.6 健康检查

- `GET /livez` — 进程存活, 永远 200 (除非死锁)
- `GET /readyz` — DB + Redis + MinIO ping
- `GET /version` — commit + build time + 各上下文 schema 版本

### 5.7 Panic Recovery

中间件链最里层 `Recover()`: 捕 panic → Error 日志含 stack → metric `panic_total` → 500 + trace_id → **不杀进程**。

---

## §6 测试策略 (DDD 视角)

### 6.1 测试金字塔

```
       E2E             ~5      完整闭环 (Playwright)
       跨上下文集成     ~10     租户隔离, iam+tenancy 链路
       上下文集成      ~30     每上下文 application+infrastructure
       应用层单测      ~40     mock repo, 验证用例编排
       领域层单测      ~80     纯函数, 验证业务规则 (覆盖 100%)
```

约 165 个测试, 对照 gallant v1 现有 32 个。

### 6.2 各层测试规约

**领域层** — 纯单测, 无 DB 无网络, 100% 覆盖目标:
- `stage_engine_test.go`: 9 个状态转移用例
- `password_policy_test.go`: 长度/复杂度/锁定边界
- `problem_test.go`: 聚合不变量保护
- 直接用 `testify/assert`, 不用 mock

**应用层** — mock repo + mock eventbus, 验证用例编排:
- `submit_problem_test.go`: mock ProblemRepo + 验 publish 了正确的 ProblemSubmitted
- 用 `testify/mock` 或手写 fake

**基础设施层** — 集成测试, 连真实 PG:
- `pg_problem_repo_test.go`: CRUD + 复杂查询
- 测试 fixture: 每测试随机临时 schema → 跑迁移 → seed → 完成 DROP, 支持 `go test -parallel 8`

**接口层** — HTTP testify, 验证 routing + DTO:
- `http_handler_test.go`: 用 `httptest.NewRecorder` + 假 JWT, 验响应格式

### 6.3 选型

- 断言: `testify/assert + require`
- HTTP: `httptest + gin.TestEngine`
- Mock: 优先用真依赖; 必须时用 `testify/mock`
- 测试容器: **不用 testcontainers** (沿用 gallant 决策)
- 快照: `cupaloy` (dashboard JSON 回归)

### 6.4 命门测试 (CI gate)

`test/integration/tenant_isolation_test.go` 6 个用例:
1. 隔离
2. 污染
3. 越权
4. 状态 (suspended)
5. 跨 schema 静态扫描
6. **DDD 边界静态扫描** (新增): 验证 §1.4 六条依赖规则

### 6.5 前端测试

沿用 gallant 13 个 Vitest (helpers / stages / StageChip / AvatarBadge)。

新增 shell/ 测试: `auth.store`, `tenant.store`, `api/client` 拦截器。

M5 加 Playwright E2E: 登录 → 切租户 → 创建 → 办结 → 评价。

### 6.6 CI

`.github/workflows/ci.yml`:
- `backend`: go test -race + vet + golangci-lint + 命门测试 gate + **DDD 边界扫描 gate**
- `frontend`: npm test + tsc --noEmit + build
- `contract`: migrate -dry-run + redocly lint openapi.yaml

### 6.7 覆盖率目标

| 层 | 目标 |
|---|---|
| 领域层 (所有上下文) | 100% (业务规则零容忍) |
| 应用层 | 90% |
| 基础设施层 | 80% |
| 接口层 | 70% (主路径 + 边界) |
| 共享内核 (tenant 命门部分) | 95% |
| 前端 stores/utils | 70% |

CI 不卡覆盖率, 每周报告趋势。

---

## §7 里程碑路线图 (M1 → M2 → M3 → M4a → M4b → M5 + M6 桌面 PoC + v1.5)

### M1 — 骨架 + 共享内核 + Module 装配 + 前端 monorepo

**可交付** (v2.1 增量已标 ★):
- `server/`: `cmd/iop-full/` + `cmd/migrate/` + `cmd/tenantctl/`; config + infrastructure (pgx/redis/minio)
- `shared/{kernel,errors,eventbus,tenantdb}` 全部
- ★ `shared/module/`: `Module` 接口 + `Migration` / `Deps` 类型 + `Registry` 装配器 + 单元测试
- ★ `app/`: `BuildDeps` + `PlatformModules` + `Run` 顶层装配工厂
- `platform/{dictionary,localization,platform_config}` 模块骨架 (扩展点)
- `interface/middleware` 全链 (RequestID/Recover/Logger/CORS/Error)
- `interface/apiresp` 含 Transformer 钩子
- `interface/query_pipeline` 含 ScopeFn 注册接口
- 初始 OpenAPI (/livez /readyz /version /metrics) 走 Registry 聚合
- `public.migration_history` 表 (含 `module_name` 列)
- ★ `web/` 重组为 pnpm workspace:
  - `packages/shell` (含 `platform/*` 抽象层接口)
  - `packages/platform-web` (默认实现)
  - `packages/api-client`, `packages/ui-tokens`
  - `apps/iop-full` 空入口 (装配 shell + platform-web, 暂无业务模块)
- `desktop/iop-full/` 目录骨架占位 (不接 Tauri, 仅 README 说明 M6 接入)
- `deployments/docker-compose.yml`: 5 服务起得来
- ★ `scripts/new-module.sh`: 新模块脚手架 (后端 + 前端 + OpenAPI)
- `Makefile`: build/test/lint/run/migrate/openapi-gen, 含 `build-iop-full` 等产品级目标

**验收**:
- `make dev` 一键起; livez/readyz/metrics 工作正常 (空业务模块也能跑)
- 故意 panic 测试路由不杀进程
- 日志含 trace_id 贯穿
- CI 跑通: lint/test/build/openapi 校验 + DDD 边界扫描 + ★ 模块隔离扫描
- ★ 前端 workspace 依赖图扫描: `@iop/module-*` 数量 = 0 时 build 成功
- ★ 跑 `scripts/new-module.sh demo` 能生成可编译的空业务模块

**风险**:
- DDD 目录嵌套深 → 工具/IDE 跳转友好度
- viper 嵌套配置 + env override 优先级
- openapi 字段命名约定 (json:"snake_case")
- ★ pnpm workspace 学习曲线 + Vite 多入口热重载体验

### M2 — Tenancy + IAM (含命门测试)

**可交付**:
- `platform/tenancy/` 模块完整 (domain/application/infrastructure/interface + 自带 migrations)
- `platform/iam/` 模块 v1: 本地账号密码 + JWT + Session + 基础 RBAC + 自带 migrations
- `cmd/tenantctl`: create/suspend/resume/close/migrate-all [--module]
- `platform/tenancy/migrations/platform/000001`: tenant/platform_user/membership
- `platform/tenancy/migrations/tenant/000001`: member
- `test/integration/`: harness + **8 个命门测试** (含 v2.1 新增的模块隔离 + 多产品装配冒烟)
- `packages/shell/auth/`: LoginView + auth.store + guard
- `packages/shell/tenant/`: TenantSwitcher (完善)
- `packages/api-client/`: 走 `platform.env.apiBaseURL` 取 base URL

**验收**:
- `tenantctl create --slug=acme` 后 schema 自动建好
- 登录拿 JWT、token 解析正确
- 6 个命门测试全绿 (CI gate)
- 前端 F5 保持登录 / 401 跳登录 / 切租户后 URL 不变数据变

**风险**:
- bcrypt cost 在低配机器登录慢
- Redis 持久化策略
- **M2 末期增加一次轻量 KingbaseV8 嗅探** (拿命门测试 + auth 流程跑 KingbaseV8 实例, 提前发现兼容差异)

### M3 — Audit + Notification + FileStorage + Dictionary 完整

**可交付**:
- `platform/audit/` 完整 (含订阅 eventbus 的 subscriber + buffered_writer + 自带 migrations)
- `platform/notification/` 完整 (含 event_handlers, 预留对将来 modules 事件的订阅 + 自带 migrations)
- `platform/filestorage/` 完整 (含 MinIO adapter + 通用 BizRef 关联 + 自带 migrations)
- `platform/dictionary/` 完整 (含 Redis 缓存 + 平台默认项 seed + 自带 migrations)
- `platform/iam/` 扩展: 完整 RBAC + 角色/策略管理 API + casbin_rule 表 (在 iam 自己的 platform migrations 里)
- `packages/shell/notify/`: NotifyCenter (20s 轮询)
- `packages/shell/components/`: 共享组件预留位 (Icon, AvatarBadge 等)

**验收**:
- 无权角色被拒 (40300)
- Casbin policy 改后立即生效
- 审计异步可见; buffer 满走同步不丢
- 文件上传/下载流程通; object_key 路径正确
- problem 事件触发 notification, audit 全部落库
- 跨上下文耦合静态检查: 各上下文 domain 包之间无 import (CI gate)

**风险**:
- Casbin reload 性能
- audit 写入失败兜底
- 事件订阅注册时机 (启动期 vs 懒加载)

### M4a — 首期业务模块第一批 (Problem / OKR / Contacts / Announcements)

**可交付**:
- ★ `modules/problem/` 完整 (8 用例 + StageEngine + 看板 + migrations/tenant 10 张表) — 平迁 gallant v1
- ★ `modules/okr/` 完整 (4 级计划 + 日/周报 + 周汇总 + cron 提醒 + migrations/tenant ~8 张表)
- ★ `modules/contacts/` 完整 (OrgContact 投影 + PersonalContact + 分组 + 收藏 + vCard 导入导出 + migrations/tenant ~3 张表)
- ★ `modules/announcements/` 完整 (Post 聚合 + 简单审批流 + tag + 热度排行 + Markdown 渲染 + migrations/tenant ~3 张表)
- ★ `platform/iam` PersonalProfile 扩展: TenantMember.profile JSONB + 头像/证书附件 + 导出 (PDF/vCard) + 临时分享 token
- 4 个 `packages/module-<name>/` workspace 包同步落地; `apps/iop-full/src/modules.ts` 装入 4 个模块
- 种子数据: 每模块 ~10 条样例数据
- 关键文档: i18n key 命名规约 (`platform.*.*` / `modules.*.*`) 落地

**验收**:
- Problem: 跑通争议闭环 + 共识闭环 + 看板 KPI 全绿
- OKR: 跨周提交日报触发 RollupWeekly 视图正确; 未提交触发 RemindOverdue 通知
- Contacts: IAM `MemberJoined` 事件被 OrgContactProjector 消费, 投影表 24h 内一致性 100%
- Announcements: `PostPublished` 事件被 Notification 订阅, 用户偏好开关生效
- IAM Profile: `/me/profile/export` 输出 PDF + `/me/profile/share` 颁发 token 后 `/public/profile/:token` 7 天内可访问, 8 天后 404
- 越权测试: 跨租户 token 访问任一模块的资源都 → 404
- DDD + 模块隔离扫描 + 平台→业务零依赖扫描通过
- ★ 装配验证: 注释掉 4 个业务模块中任 1 个, `apps/iop-full` 仍能 build 通过, 其他 3 个模块无副作用

**风险**:
- 4 个模块并行开发需要清晰的接口冻结时间表 (建议 M4a 第 1 周冻结所有事件 schema)
- IAM 扩展 profile 字段后, M2 已经写的 IAM 测试需要回归
- 前端 monorepo 在 4 个 module 包并存时, vite HMR 性能可能需要调优

### M4b — 首期业务模块第二批 (Documents / Assets / Party)

**可交付**:
- ★ `modules/documents/` 完整 (Resource 聚合 + ShareScope 可见性 + 下载日志 + 检索 + migrations/tenant ~3 张表)
- ★ `modules/assets/` 完整 (Asset 聚合 + DeptScope 服务 + 部门可见性 + Excel 导出 + migrations/tenant ~3 张表)
- ★ `modules/party/` 完整 (MemberDevelopment 5 阶段 + PartyActivity + StudyMaterial + 报表 + migrations/tenant ~5 张表)
- 3 个 `packages/module-<name>/` 同步落地; `apps/iop-full/src/modules.ts` 增至 7 个模块
- ★ **模块独立打包冒烟**: 拿 `modules/party` 这个国情化模块作样本, 验证 `cmd/<oversea-product>` 不装载该模块时, 二进制能正常编译启动, schema 无该模块表

**验收**:
- Documents: 私有/部门/租户三级共享范围在跨用户访问时全部正确
- Assets: 部门管理员看到本部门资产, 普通员工只看自己; 部门外不可见
- Party: 5 阶段推进硬规则 (介绍人必须是 full + activist 6 个月以上) 校验生效; 月度活动统计准确
- 全部 7 个业务模块 + 8 个平台模块装配后, 启动时间 ≤ 5s; readyz 全绿
- 跨模块零隐式耦合: 拿 `modules/party` 做对照, 移除该模块后其他 6 个模块功能不受影响

**风险**:
- 资产管理的部门可见性走 `module.Deps.DeptClient`, 该客户端接口必须 M4a 末期冻结 (否则 M4b 卡 IAM 提供方)
- 党务模块的 PDF 证书生成与 IAM profile export 复用 pdfRenderer, 模板隔离要做好
- M4a + M4b 合计新增 7 个模块, 一次 PR review 不现实, 必须按模块拆 PR 并配齐自动化 lint/test

### M5 — 生产部署 + KingbaseV8 验证 + 文档 + 单产品拆分演练

**可交付**:
- `deployments/` prod compose + Nginx HTTPS+限流+WAF + KingbaseV8 适配 + 部署 runbook
- `docs/operations/`: backup-restore + tenant-lifecycle + observability + incident-playbooks (5 个故障处置)
- `docs/developer/`: getting-started + adding-new-module (★ 替换 v2.0 的 adding-new-context) + i18n-and-dict 规范 + DDD-coding-standards
- `docs/architecture/`: overview + DDD-context-map + ★ platform-vs-modules + ★ product-composition + 15+ 个 ADR
- `.github/workflows/release.yml`: tag 触发构建镜像 (每产品独立 image tag, 例 `iop-server-iop-full:v1.0.0` / `iop-server-gallant:v1.0.0`)
- `e2e/`: 5 个 Playwright 用例 (针对 `apps/iop-full`)
- ★ **单产品拆分演练**: 真正拆出 `cmd/gallant/` + `apps/gallant/` 并跑通端到端 (验证模块打包契约真的有效, 不是纸上谈兵)

**验收**:
- staging 一键部署 HTTPS 可访问
- **KingbaseV8 跑命门测试 + 32 个 problem 测试全绿**
- 备份恢复演练通
- E2E 5 个用例 CI 跑通
- 完整灰度发版演练 (tag → staging → 烟雾测试 → prod)
- ★ `apps/gallant` build 出独立 dist, `cmd/gallant` 二进制大小 ≤ `cmd/iop-full` 的 90% (验证编译期裁剪生效, 实际差距取决于业务模块数)
- ★ 一个独立 PG 实例上, `cmd/gallant` 走完租户创建 → 提报问题 → 评价全流程

**风险**:
- KingbaseV8 vs PostgreSQL 差异 (CREATE SCHEMA, JSONB, SET LOCAL) — **已在 M2 末期嗅探, M5 收尾确认**
- ★ 单产品拆分演练遇到隐藏耦合 (例: 平台模块意外发布了业务术语的事件名, 单产品反而拖累)

### M6 — 桌面端 PoC (可选, 视市场需求开启)

**目标**: 把 `apps/iop-full` 或 `apps/gallant` 中之一打包成 Tauri 桌面应用, 验证 v2.1 预留的平台抽象层有效。

**可交付**:
- `packages/platform-tauri/`: 完整实现 `@iop/shell/platform/*` 全部接口 (storage / notifier / fileDialog / clipboard / env / deepLink)
- `desktop/<product>/`: tauri.conf.json + Rust src + 应用图标 + 自动更新配置
- 构建脚本: `make tauri-<product>` 输出 dmg/msi/AppImage
- 桌面端 deep link 回调流程 (`iop://callback?token=...`) + Keychain 安全存储 JWT
- 桌面端独有的功能 (托盘图标 / OS 通知 / 文件拖拽)
- `docs/developer/desktop-distribution.md`: 签名、公证、分发流程

**验收**:
- macOS / Windows 双平台安装包, 登录 → 创建问题 → 接收通知 全流程通
- 与 Web 版同源代码 (业务模块代码改动 = 0; 仅 platform 实现切换 + 桌面壳配置)
- 离线场景: 后端不可达时, UI 提示离线、token 不丢、网络恢复后自动重连

**风险**:
- Tauri 2.x 在中文 Windows 输入法 / macOS 公证 / 自动更新链路上的实际坑
- 业务模块中是否藏有隐式 Web API 依赖 (例: 直接 `window.location`), 需要 M1 抽象层覆盖到位
- 桌面端安全模型 (CSP, IPC 白名单) 与 Web 不同, 路由 / 跨域配置要适配

### 工作量

```
M1 → M2 → M3 → M4a → M4b → M5 → (M6 可选)
```

相对比 ≈ **4 : 3 : 3 : 5 : 4 : 2 : 2.5** (v2.1.1 拆出 M4a/M4b; M4a 含 4 个业务模块 + IAM profile 扩展, 体量是原 M4 的 ~1.25x; M4b 再补 3 个模块, 体量略小)

每个里程碑结束都是可发布点:
- M1 → dev 环境可访问 (空业务模块也跑得起来)
- M2 → 内部联调可建租户
- M3 → 内部用户可见看板壳 (无业务模块)
- **M4a → 首个真实租户可灰度试用 4 个核心模块 (Problem/OKR/Contacts/Announcements) + 个人档案**
- **M4b → 7 个业务模块全部上线**, 内部 dogfooding 验证完整产品形态
- M5 → 正式上线 (`iop-full` 主推); 同时跑通独立产品 `gallant` (仅 Problem 模块) 形态
- M6 → 桌面端 PoC 上线 (视市场需求)

**v1.5 阶段** (M5 后 1-3 月):
- AppCenter (`platform/appcenter`) 完整实现, 租户管理员可自助开关业务模块
- Contacts 在线聊天 (引入 SSE/WebSocket 基础设施, 是 v1.5 的最大底座升级)
- 视情况新增垂直行业模块

---

## §8 决策清单 (ADR 总览)

| # | 决策 | 选择 | 替代方案 / 理由 |
|---|---|---|---|
| 1 | 基座定位 | 通用基座 + 优先迁移 gallant | vs Go 重写 gallant / vs 全新 ERP |
| **2★** | **代码组织 (v2 替换 v1)** | **DDD 四层 × 8 有界上下文** | v1 选了"扁平 Go 风格 (模块=包)", 但 Problem 业务规则复杂度高, 不分层会迅速出现 god service; v2 改用 DDD |
| 3 | monorepo | `iop/{server,web,deployments,docs,scripts}` 顶层直挂 (v2.1.1 去掉 go_base 中间层) | 前后端契约同步 + OpenAPI 自动生成; 维护时一眼看清目录 |
| 4 | 前端 UI 库 | Vue 3 + Element Plus (保留) | 架构图取字面 Ant Design 工作量过大 |
| 5 | 租户识别 | JWT claim 携带 tenant_id | vs 子域名 / Header / URL path |
| 6 | 用户×租户 | 双层 (platform_user + member) | 支持账号跨租户复用 + 成员档案租户内隔离 |
| 7 | Schema 隔离 | 每租户独立 schema + SET LOCAL search_path | 事务级, 归池不污染 |
| 8 | DB 句柄 | PlatformDB + TenantDB 双句柄 | 共用 pool, wrapper 接口分离 |
| 9 | 鉴权 v1 | 本地账号密码 + JWT + IdentityProvider 抽象 | SSO 预留接口, 不一次到位 |
| 10 | RBAC 实现 | Casbin (with domains, domain = tenant_id) | vs 自研轻量 RBAC |
| 11 | 审计 | 显式调用 + 异步队列 + 双账本 + 订阅 eventbus | v1 是"显式调用", v2 增加"通过 eventbus 自动捕获领域事件"作为主路径 |
| 12 | 事件总线 | v1 进程内 channel + worker pool, 接口稳定 | vs NATS (出现削峰信号再切) |
| 13 | 错误处理 | 三层 Kind + 统一中间件 + 错误码按上下文分段 | handler 不直接写响应 |
| 14 | 日志 | zap 结构化 + 敏感脱敏 + trace_id 贯穿 | vs zerolog |
| 15 | 可观测性 v1 | trace_id + Prometheus + 上下文标签 | OpenTelemetry 按需 |
| 16 | 测试基础 | testify + 本地 docker PG + 随机 schema 并行 | 不用 testcontainers |
| 17 | 命门测试 | 6 个用例独立 CI gate (新增 DDD 边界扫描) | 任一挂掉不发版 |
| 18 | 上线节奏 | 5 个里程碑, 基座优先 | vs 业务驱动 3 步 |
| 19 | KingbaseV8 验证 | M2 末期嗅探 + M5 完整验证 | 不拖到最后 |
| **20★** | **扩展点放置** (v2 调整) | dict / i18n / config → 独立上下文 / shared; QueryPipeline / RenderTransformer → interface 层 | v1 放在共享内核, v2 把字典/i18n 升格为上下文以承载未来扩展; QueryPipeline/Transformer 留在 interface 层做横切 |
| **21★** | **共享内核范围** (新增) | 仅 ID / ctx / errors / eventbus / tenantdb / platform_config | 严格最小化, 业务概念全部归各自上下文 |
| **22★** | **CQRS 局部使用** (新增) | 仅 dashboard 等聚合查询使用; 普通 CRUD 不走 | 避免无用的复杂度; 看板查询绕过聚合直查 DB 是合理的局部优化 |
| **23☆** | **平台 vs 业务模块分层** (v2.1) | `internal/platform/<X>` 与 `internal/modules/<X>` 显式分目录; 平台不依赖业务, 业务依赖平台契约 | v2.0 把 8 个全平铺为 contexts, 一旦要做单产品/产品矩阵就要回头拆; v2.1 提前分层, CI gate 强制方向 |
| **24☆** | **产品组装: 编译期 vs 运行期** (v2.1) | 编译期组合: `cmd/<product>/main.go` + `apps/<product>/src/main.ts` 显式列模块 | vs Go plugin / YAML 配置动态加载; 编译期: 类型安全、桌面友好、二进制裁剪有效; 运行期: 复杂、调试难、桌面端不能用 |
| **25☆** | **每模块自带 migrations** (v2.1) | `internal/<X>/migrations/{platform,tenant}/` embedded; `migration_history` 加 `module_name` 列 | vs 集中 `migrations/tenant_template/`; 模块可独立装拆, 新增/删除一个模块不污染其他模块迁移历史 |
| **26☆** | **桌面端: 外壳模式 + 抽象层** (v2.1) | Tauri 只壳前端 SPA, 后端仍远端; 前端 `@iop/shell/platform/*` 抽象 Web/Tauri 差异 | vs 内嵌后端 (与多租户 PG schema 隔离冲突, 桌面端只跑单用户单租户); 外壳模式架构改动几乎为 0 |
| **27☆** | **前端 pnpm workspace + 每产品独立 Vite 入口** (v2.1) | `web/{packages,apps,desktop}`; `apps/<product>` 装配 shell + 选定 modules | vs 单 Vite 工程 + 路由级懒加载; workspace 强制模块边界 (依赖图禁止), 多产品独立 dist 体积可控 |
| **28◆** | **首期业务模块集合** (v2.1.1) | 7 个: Problem / OKR / Contacts / Announcements (M4a) + Documents / Assets / Party (M4b) | vs 单一 Problem 模块; B 端办公平台典型需求集合; Party 单留作"国情化模块独立打包"的真实验证用例 |
| **29◆** | **PersonalProfile 内化于 IAM** (v2.1.1) | 不开独立 `modules/personalprofile`, 扩 `platform/iam.TenantMember` 加 `profile JSONB` + FileStorage BizRef | 数据强属于 TenantMember 聚合, 拆出去会违反聚合一致性, 也无"只买档案管理"的产品形态; 用 FileStorage 关联非结构化数据兼顾结构化 + 文件 |
| **30◆** | **AppCenter 是平台开关, 不是模块管理器** (v2.1.1) | 归 `platform/appcenter` (v1.5+), 仅控制"已编译模块对租户是否启用", 不做运行期下载安装 | vs 真·模块市场 (与 ADR 24 编译期组合冲突, 桌面端不可用); 给租户管理员轻量自助权, 同时不破坏架构纪律 |
| **31◆** | **项目根目录扁平化** (v2.1.1) | `iop/{server,web,deployments,docs,scripts,legacy}` 顶层直挂, 去掉 `go_base/` 中间层 | 维护时一眼看清; 与团队习惯 (顶层即工程) 对齐; legacy 单独保留作迁移参考 |
| **32◆** | **M4 拆为 M4a + M4b** (v2.1.1) | 7 个业务模块分两批落地, 不一次性塞进单 M4 | 控制单里程碑风险面 + 让 M4a 末期可做一次中间灰度发布 (4 模块已是可用产品) |
| **33◆** | **模块命名采用行业标准** (v2.1.1) | Go 包名: problem / okr / contacts / announcements / documents / assets / party / appcenter | vs 中文拼音 / 自创术语; 便于国际化、便于代码评审、便于团队招聘对齐 |

★ = v2.0 新增或替换的决策。  ☆ = v2.1 在 v2.0 基础上的增量决策。  ◆ = v2.1.1 在 v2.1 基础上的增量决策 (业务模块扩充)。

---

## §9 待办与已知约束

### 待办 (spec 阶段之后由 writing-plans 拆分)
- 每个里程碑的详细 task 拆分
- 每个上下文/模块的领域聚合不变量详细 review (M2 前完成 tenancy/iam, M3 前完成 audit/notify/file/dict, M4 前完成 problem)
- migrations SQL 的字段级 review (按模块分组)
- KingbaseV8 嗅探脚本编写
- Casbin policy 的初始 seed 集合
- DDD 边界 + ★ 模块隔离 扫描脚本编写 (CI gate)
- 各上下文/模块 i18n key 命名规约 (一致前缀, 例 `platform.iam.*` / `modules.problem.*`)
- ★ `Module` 接口 v1 草案 + 评审 (M1 前完成, 接口稳定性是后续所有模块的前提)
- ★ `shared/module/{Dict,File,Notify}Client` 客户端接口冻结 (M3 前完成, 业务模块靠这层访问平台能力)
- ★ 事件 schema 注册表 (`platform.*.*` 与 `modules.*.*` topic 列表, 含字段说明), 进 `docs/architecture/event-catalog.md`
- ★ `scripts/new-module.sh` 脚手架编写 (M1 末完成)
- ★ M6 (桌面端) 启停决策点 (M5 完成后回看市场需求决定是否开)
- ◆ **业务模块事件 schema 互不冲突的命名空间审计** (M4a 启动前): 7 个模块的事件 topic 列表需先确认无冲突且语义清晰
- ◆ **M4a 4 个模块的接口冻结时间表** (M4a 第 1 周, IAM 提供的 `module.Deps.DeptClient` / `MemberClient` 必须在此时点 stable)
- ◆ **PDF 渲染器选型** (M4a 中, IAM profile export + Party 证书共用): wkhtmltopdf vs gofpdf vs Typst 三选一
- ◆ **Markdown 安全渲染选型** (M4a 中, Announcements 用): bluemonday + goldmark 组合默认

### 已知约束
- **单实例**: PG 单实例 (200 租户, ~50 连接 OK), Redis 单实例; 高可用方案见 M5 文档
- **数据库选型**: 开发 PG 16; 生产可切 KingbaseV8 PG 兼容模式 (已在 M2 末期嗅探确认兼容)
- **业务范围**: v2.1 落地业务模块仅 `problem` 一个; 订单/库存/审批/财务等模块为架构图举例, 未来加新模块走 `docs/developer/adding-new-module.md` 流程
- **MFA / SSO / 富文本 / 移动端 / 开放 API / SSE / 流程定义编辑**: 均在 v2.1 范围外
- **审计保留期**: 默认 7 年, 由后续合规决策细化
- **租户销户**: 30 天保留期 + 手动确认, 避免误删
- **DDD 学习成本**: 团队首次落地 DDD, M1 留 1-2 周做培训 + 代码 walkthrough
- ★ **同租户内多模块的"装载状态"无 UI 配置**: 哪些模块装载由二进制 (`cmd/<product>`) 决定, 不在租户管理 UI 暴露开关 (避免"租户买了但二进制没装"的不一致); 升级二进制时如新增模块, 走 `tenantctl migrate-all --module=<new>` 给老租户补迁移
- ★ **同一 module 跨产品 (iop-full / gallant) migration 序列必须一致**: 用 `embed.FS` 强制随模块走, 不允许在不同产品里"自定义"
- ★ **桌面端不在 v2.1 工程范围**: 仅落地前端抽象层 + 默认 Web 实现 + `desktop/` 占位; Tauri 真实接入推迟到 M6
- ★ **`Module` 接口的演化**: 一旦发布 (M1 验收), 接口字段的"删除/改类型"视为 break, 必须走 ADR 流程; 新增可选字段不算 break
- ◆ **业务范围 v2.1.1 扩充**: 首期 7 个业务模块覆盖了"OA 基本盘 + 国情化模块". 不在首期: 财务 / 流程审批 / HR / 客户关系 / 进销存 (这些是垂直模块, 走"加新模块"流程, 不影响基座)
- ◆ **Party 模块仅对国内租户启用**: 出海产品的 `cmd/<oversea>/main.go` 不装载 `modules/party`, 这是模块独立打包的真实价值落点
- ◆ **AppCenter 在 v2.1.1 仅占位**: `platform/appcenter/` 目录 + 空 `module.go` 存在, 但 RegisterRoutes 返回空, 不暴露任何 API; M4a/M4b 期间业务模块的菜单走前端硬编码, v1.5 后切到 AppCenter 驱动
- ◆ **聊天功能依赖 SSE/WebSocket 基础设施**: v1.5 引入时需要决定:(a) SSE 单向通知 (轻量) 还是 (b) 完整 WebSocket 双工 (RFC 6455). 选 (a) 维持轮询补丁路径, 选 (b) 需在 platform 层加 connection manager

### 不在范围内但需要警觉的事
- Casbin policy 数量增长后缓存策略 (200 租户 × ~10 角色 × ~20 policy ≈ 40k 行, 单机 OK; 100k+ 才需要分片)
- audit 表年增量 (按 10k 操作/天/租户 × 200 租户 × 365 天 ≈ 7.3 亿行/年), M5 前要定**分区策略** (按月分区是首选)
- attachment 大文件下载经后端中转的压力 (gallant 现状), M5 评估是否引入 presigned URL 直传/直下
- 事件总线 v2 切 NATS 时的"消息顺序"和"重放"语义; v1 进程内有序, v2 需要在事件 schema 上预留 `sequence` 字段
- 跨上下文/模块事件订阅的"组合爆炸": 一个业务事件被 N 个订阅者消费, 调试链路靠 trace_id 串
- ★ 模块数量 ≥ 5 后 monorepo 构建时间 (pnpm + Vite + Go build): 评估是否引入 `turbo` / `nx` 增量构建
- ★ 产品矩阵增长 (≥ 3 个产品) 后 CI 资源消耗 (每产品独立 build + 独立 e2e)
- ★ 平台 → 业务方向的反向依赖诱惑: `platform/notification` 可能想"知道是问题协同的什么事件来定制模板", 必须用通用 template 机制而不是 import 业务模块

---

## 附录 A: 与 v1 spec (modular monolith) 的差异 + v2.0 → v2.1 增量

### A.1 v1 → v2 (DDD 重构)

| 维度 | v1 (扁平 Go 风格) | v2 (DDD 四层) |
|---|---|---|
| 顶层目录 | `internal/platform/* + internal/modules/*` | `internal/contexts/<X>/{domain,application,infrastructure,interface} + internal/shared/*` |
| 模块边界 | 包级别约定 | **每上下文四层, CI 静态扫描强制依赖方向** |
| 共享内核范围 | 7 子模块 (tenant/auth/rbac/audit/notify/file/event) | **最小化**: 仅 ID/ctx/errors/eventbus/tenantdb/config |
| 业务模块写法 | `modules/problem/` 单层 8 文件 | 多层多文件 |
| 业务规则放置 | 散落在 service / handler | **集中在 domain/, 100% 单测覆盖** |
| 跨模块协作 | interface + 事件总线 (混用) | **领域事件总线为主, 单向上下游为辅** |
| 字典/i18n 位置 | 共享内核扩展点 | 独立上下文 (Dictionary / Localization) |
| 命门测试 | 5 个 | **6 个** (加 DDD 边界扫描) |
| 错误码段 | 平台 / 业务两段 | **按上下文细分 (10 段以上)** |
| 学习曲线 | 低 | 中等 (团队需要 1-2 周熟悉 DDD 术语和分层) |

**为什么 v2 改用 DDD**:
- Problem 业务规则复杂度 (8 阶段状态机 + 4 类协作消息 + 看板聚合) 在 v1 的扁平结构里容易堆成 god service
- 未来加新业务模块 (订单/审批/CRM) 时, 有四层骨架可以直接复刻, 不必每次想"该放哪个包"
- DDD 的"领域语言"约束帮助产品和开发对齐: 例 `Stage` `Branch` `Measure` 等词在代码、API、文档、PRD 里同名同义
- 静态依赖检查可机器验证, 不依赖人工 code review

**为什么不全盘照搬 DDD 全套**:
- 不引入 CQRS+Event Sourcing (太重, 当前业务不需要)
- 不引入 Sagas / Process Manager (没有分布式事务诉求)
- 不引入 Hexagonal "端口适配器"全套术语 (用四层 + DI 已足够清晰)
- 不为通用 CRUD 强加聚合 (例如 Dictionary 的 DictItem 直接用作 DTO 也行, 不必每次走聚合)

### A.2 v2.0 → v2.1 (引入多产品 / 模块独立打包 / 桌面端预留)

| 维度 | v2.0 | v2.1 |
|---|---|---|
| 顶层目录 | `internal/contexts/<X>/` 平铺 8 个 | `internal/platform/<X>/` (8 个平台上下文) + `internal/modules/<X>/` (业务模块, 按需) |
| 装配口 | 单 `cmd/server/`, 全部上下文统一启动 | 每产品一个 `cmd/<product>/`, 显式列模块; M1-M3 只建 `iop-full` |
| Module 接口 | 各上下文 `module.go` 自管, 无统一签名 | 统一 `shared/module.Module` 接口 + `Registry` 装配 |
| 迁移位置 | 中央 `migrations/{public,tenant_template}/` | 每 module 自带 `migrations/{platform,tenant}/`, embed.FS |
| migration_history | 二元组 `(tenant_id, migration_id)` | 三元组 `(tenant_id, module_name, migration_id)` |
| 前端工程 | 单 Vite 工程 `web/src/{shell,modules}` | pnpm workspace `web/{packages,apps,desktop}`, 每产品独立 Vite 入口 |
| 前端模块 | 路由懒加载即可 | `@iop/module-<X>` 独立 workspace 包, 包级依赖图防互相 import |
| 桌面端 | 不在范围 | 架构预留: `@iop/shell/platform/*` 抽象层 + `@iop/platform-web` 默认实现 + `desktop/` 占位; Tauri 真实接入 M6+ |
| 命门测试 | 6 个 | **8 个** (加模块隔离扫描 + 多产品装配冒烟) |
| 里程碑数 | M1-M5 | M1-M5 + 可选 M6 (桌面 PoC) |
| ADR 数 | 22 | 27 (新增 5 条 v2.1 决策) |

**为什么 v2.0 → v2.1 升级**:
- 业务路径明确要做多个业务模块, 提前分平台/业务两层比事后拆便宜得多 (经验法则: 第二个模块进场前不拆, 拆的成本就翻倍)
- 桌面端外壳模式的代价仅是「前端抽象层」, 现在不投, 将来想做时业务代码已经写死了 `localStorage` / `window.*`, 改造成本就高了
- 模块自带 migrations 后, 业务模块可以"装载即生效, 卸载即清除", 是产品矩阵的前提

**为什么不一步到位上 Tauri (M1 集成)**:
- 增加 M1 体量, 但桌面端是不确定需求 (M5 前可能不开)
- 抽象层 + 默认 Web 实现的代价已经覆盖"架构不漏未来", 真实集成可推迟
- Tauri 工具链 (Rust + 签名 + 公证) 学习成本独立, 与基座学习曲线叠加会拖慢 M1

### A.3 v2.1 → v2.1.1 (首期业务模块扩充 + 命名标准化 + 顶层扁平化)

| 维度 | v2.1 | v2.1.1 |
|---|---|---|
| 首期业务模块数 | 1 (Problem) | **7**: Problem / OKR / Contacts / Announcements (M4a) + Documents / Assets / Party (M4b) |
| 个人档案 | 未规划 | 内化为 IAM 扩展 (TenantMember.profile JSONB + FileStorage BizRef 关联非结构化) |
| 应用中心 | 未规划 | 平台层 (`platform/appcenter`), 仅作开关用, 推迟到 v1.5 实施 |
| 项目根目录 | `iop/go_base/{server,web,...}` | `iop/{server,web,deployments,docs,scripts,legacy}` 顶层直挂 |
| Go 包命名 | 中文拼音 / 自创术语 | 行业标准: problem / okr / contacts / announcements / documents / assets / party / appcenter |
| 里程碑 | M1-M5 + M6 | M1 → M2 → M3 → **M4a** → **M4b** → M5 + M6 + v1.5 |
| ADR 数 | 27 | **33** (新增 6 条 v2.1.1 决策, 标 ◆) |
| 在线聊天 | 未规划 | 明确推迟到 v1.5+ (需 SSE/WebSocket 基础设施) |

**为什么 v2.1 → v2.1.1 升级**:
- 业务目标明确从"一个问题协同模块"扩为"OA 综合办公基座", 7 个模块同期上线是产品差异化的最小集
- 模块命名标准化降低国际化和团队招聘成本, 同时提升代码评审效率 (`workplanning` 不如 `okr` 一目了然)
- 顶层扁平化只是命名问题但显著降低维护时的认知负担

**为什么 PersonalProfile 不开模块、AppCenter 单独算平台增强**:
- PersonalProfile 数据强属于 TenantMember 聚合, 拆出去违反聚合一致性, 也无客户"只买档案管理"的产品场景
- AppCenter 是"对模块的元管理", 不是业务模块, 归 platform 层符合 DDD 分类; 同时它的"安装"语义被收敛为"启用已编译模块", 不破坏 ADR 24 编译期组合决策

**v1.5 后续路线**:
- AppCenter 完整实现 → 租户自助开关
- Contacts 聊天 → 引入 SSE/WebSocket 底座升级
- 视垂直行业扩 (财务 / 审批 / CRM / 进销存)
- 桌面端 Tauri (M6 PoC 通过后量产)

---

## 附录 B: DDD 概念中英文映射 + 团队术语表

| 英文 | 中文 (本项目统一用法) | 简释 |
|---|---|---|
| Bounded Context | 有界上下文 | 一个内聚业务子域, 内部用统一语言 |
| Aggregate | 聚合 | 一组绑定一致性的对象, 通过聚合根访问 |
| Aggregate Root | 聚合根 | 对外暴露的唯一入口, 例 `Problem` |
| Entity | 实体 | 有身份且可变, 例 `Measure` |
| Value Object | 值对象 | 不可变, 按属性判等, 例 `Email` `Stage` |
| Domain Service | 领域服务 | 跨聚合的纯业务逻辑, 无状态, 例 `StageEngine` |
| Domain Event | 领域事件 | 业务上有意义的过去时事件, 例 `ProblemSubmitted` |
| Repository | 仓储 | 类集合的聚合访问接口, 接口在 domain 实现在 infrastructure |
| Application Service | 应用服务 | 编排用例, 事务边界, 例 `SubmitProblem` |
| Command / Query (CQRS) | 命令 / 查询 | 写走聚合, 读可直查 DB; 本项目仅 dashboard 走 CQRS |
| Anti-Corruption Layer (ACL) | 防腐层 | 翻译外部模型到本上下文模型, 例 `iam_acl.go` |
| Shared Kernel | 共享内核 | 多上下文共用且变更需协商的极小代码集 |
| Published Language | 发布语言 | 上下文对外稳定契约 (例 TenantID 类型) |
| Context Map | 上下文映射 | 上下文之间关系的描述图 |
| Ubiquitous Language | 统一语言 | 产品、开发、文档共用的业务术语 |
| Conformist | 服从者 | 完全接受上游模型不做适配 |
| Customer-Supplier | 客户-供应商 | 上下游协作, 上游优先满足下游 |
| Open Host Service | 开放主机服务 | 上游通过协议 (如事件总线) 开放给任意下游 |
| Separate Ways | 各行其道 | 两上下文不直接协作 |

---

**文档结束。**
