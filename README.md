# IOP · 一站通办

> 多租户 B 端办公平台**基座（framework）** — 像 ruoyi / jeecg 一样开箱即用，但天生多租户、可插拔业务模块。
> A multi-tenant B2B office-platform **framework**: schema-isolated tenants, pluggable business modules, JWT+RBAC, and a clean platform/tenant admin split.

---

## ✨ 特性

- **多租户隔离** — 每个组织独占一个 PostgreSQL schema（`tenant_<slug>`），物理隔离；5 个「命门」隔离测试守护。
- **平台 / 组织双控制台** — 平台管理员（全局身份）治理所有组织/用户/注册申请；组织管理员只管本组织内部（成员/角色/部门/设置）。
- **可插拔业务模块** — 一个 `Module` 契约（Manifest + 路由），注册一行即接入；前端按约定自动发现挂载。内置两个参考模块：**OKR 工作安排**、**任务清单（仿滴答清单）**。
- **认证与授权** — 用户名/手机号登录、JWT(HS256) + 刷新、RBAC（资源×动作，模块声明、角色编辑器授予并在路由层强制）、暴力破解锁定、停用即时吊销会话、首登强制改密。
- **注册-审批制** — 自助申请加入组织 → 管理员审批后开通账号。
- **应用中心** — 管理员按组织启用/停用应用；成员从左下角「添加」固定到左侧菜单。
- **B 端基本盘** — Redis 滑动窗口限流、幂等、慢查询钩子、审计日志、健康检查（livez/readyz）、Prometheus `/metrics`、优雅退出、安全响应头。
- **前端体验** — Vue 3 + Pinia + Vite，统一外壳（顶栏 + 左侧应用栏 + 工作台），全局 toast/确认弹窗、404/403、骨架屏。

## 🧱 技术栈

| 层 | 技术 |
|---|---|
| 后端 | Go 1.26 · gin · pgx/v5 · 模块化单体（DDD 四层）· 进程内事件总线 |
| 前端 | Vue 3 · TypeScript · Pinia · Vue Router · Vite |
| 存储 | PostgreSQL 16（schema 隔离）· Redis 7 · MinIO（S3 兼容） |
| 运维 | Docker Compose · Nginx · 迁移工具 · Prometheus 指标 |

## 🚀 快速开始

**前置**：Go ≥ 1.26、Node ≥ 18、Docker（含 Compose）。

### 一条命令（推荐）

```bash
git clone https://github.com/ucas-liumk/IOP.git
cd IOP
./scripts/dev.sh
```

`dev.sh` 会：启动 db/redis/minio → 跑数据库迁移 → 首次自动 `npm install` → 同时启动后端(:8080)与前端(:5174)。

打开 **http://localhost:5174**，用内置平台管理员登录：

```
用户名: admin
密码:   Admin12345!     # 首次登录会强制要求修改
```

> 登录后即进入**平台控制台**：先「组织机构」开通一个组织，再「全局用户」为其创建一个组织管理员，该管理员登录后即可进入**组织控制台**管理本组织。

### 手动分步（等价）

```bash
# 1) 基础设施
cd deployments && docker compose up -d db redis minio && cd ..
# 2) 迁移 + 启动后端（读取 server/configs/dev.yaml）
cd server && go run ./cmd/migrate up && go run ./cmd/server
# 3) 另开一个终端：前端
cd web && npm install && npm run dev
```

### 全 Docker（含已构建的前后端镜像）

```bash
cd deployments && docker compose --profile full up --build
# 前端 http://localhost:5173 · 后端 http://localhost:8080
```

## 🗺️ 架构与模型

```
平台 (Platform)
└── 组织 (Organization ≡ Tenant, 1:1, 独立 schema)
    └── 部门 (Department, 组织内部架构)
        └── 成员 (Member ← 关联全局 platform_user)
```

- **组织 ≡ 租户**：业务上叫「组织」，技术上是一个隔离的「租户 schema」，一一对应。
- **platform_admin 是全局身份**（`platform_user.is_platform_admin`），不归属任何组织，只在平台层治理。
- **租户管理员**在「组织控制台」自治本组织；平台管理员**不直接介入**组织内部业务数据。

两套控制台（都在统一外壳内，左侧栏切换）：

| | 平台控制台 `/platform` | 组织控制台 `/admin` |
|---|---|---|
| 谁 | 平台管理员（全局） | 组织管理员 |
| 范围 | 全部组织 / 全部用户 / 全部注册申请 | 仅本组织 |
| 内容 | 概览、组织机构、全局用户、注册申请 | 成员、角色、部门、设置、注册申请、应用、字典、审计 |

详见 [`docs/`](docs/)（架构 spec、运维手册、开发指南）。

## 🧩 新增一个业务模块（≈ 1 行）

```bash
./scripts/new-module.sh crm "客户管理 CRM" "业务管理"   # 生成后端 DDD 四层 + 前端模块骨架
```

然后在 `server/internal/app/app.go` 注册一行：

```go
registry.Register(crm.New(deps))
```

重启即生效：路由挂到 `/api/apps/crm/*`，权限自动进角色编辑器，事件自动被审计/通知订阅，前端按 `manifest.ts` 自动发现挂载，管理员在「应用管理」启用后成员即可使用。完整说明见 [`docs/developer/adding-new-module.md`](docs/developer/adding-new-module.md)。

## 📁 目录

```
IOP/
├── server/                      Go 后端（cmd/server | migrate | tenantctl）
│   ├── internal/
│   │   ├── shared/{kernel,errors,eventbus,tenantdb,module}/   最小内核 + 模块契约
│   │   ├── services/{iam,tenancy,audit,dictionary,notification,filestorage,appstore,...}/
│   │   ├── contexts/{okr,tasks}/                              业务模块（DDD 四层）
│   │   ├── infrastructure/{pg,redis,minio,metrics,health,logger}/
│   │   ├── interface/{middleware,apiresp,server}/  config/  app/
│   ├── migrations/{public,tenant_template}/                  平台 + 每租户迁移
│   └── configs/dev.yaml
├── web/                         Vue 3 SPA
│   └── src/{shell,modules/{admin,platform,me,okr,tasks},router,api}/
├── deployments/                 docker-compose · Dockerfile · nginx · .env.example · backup
├── scripts/                     dev.sh · new-module.sh · ...
└── docs/                        spec · 运维 · 开发指南
```

## 🔐 安全与默认账号

- 默认平台管理员 `admin / Admin12345!` **仅用于首次引导**，首登强制改密；生产请用 `IOP_SEED_ADMIN_PASSWORD` 指定。
- 生产环境**必须**设置 `IOP_AUTH_JWT_SECRET`（≥32 位）、CORS 不为 `*`、PG `sslmode=require` —— 否则服务启动即失败（fail-fast）。
- 配置覆盖：环境变量 `IOP_<SECTION>_<KEY>` 覆盖 `configs/<env>.yaml`，见 [`deployments/.env.example`](deployments/.env.example)。
- 漏洞上报与安全说明见 [SECURITY.md](SECURITY.md)。

## 🧪 测试

```bash
cd server && go test ./...                              # 单元
IOP_INTEGRATION=1 IOP_TEST_DB_DSN="postgres://iop:iop_dev@localhost:5433/iop?sslmode=disable" \
  go test ./test/integration/...                        # 集成（含 5 个租户隔离命门测试，需 docker 起 db）
cd web && npx vue-tsc --noEmit                           # 前端类型检查
```

## 🤝 贡献 / 许可

- 贡献指南：[CONTRIBUTING.md](CONTRIBUTING.md) · 行为准则：[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md)
- 用 Claude Code 继续开发？仓库根的 [`CLAUDE.md`](CLAUDE.md) 是给 AI 的项目向导，换设备克隆后会自动加载。
- 许可：[Apache-2.0](LICENSE)
