# IOP

企业内部多租户 B 端办公平台基座. v3.1 设计 spec 在 `docs/superpowers/specs/`, M1 实施 plan 在 `docs/superpowers/plans/`.

## 当前状态

- ✅ **M1 完成**: 基座 + 基础设施 + livez/readyz/version/metrics + 字典 + i18n
- ⏳ M2 (next): Tenancy + IAM + 命门测试 + B2B SaaS 基本盘 (限流/幂等/慢查询/备份)
- M3: Audit + Notification + FileStorage + Dictionary 完整
- M4: OKR 完整闭环 (核心学 DDD 里程碑)
- M5: 生产部署 + KingbaseV8

## 目录

```
iop/
├── server/                  Go 后端 (单 cmd/server 二进制)
│   ├── cmd/{server,migrate,tenantctl}/
│   ├── internal/
│   │   ├── shared/{kernel,errors,eventbus,tenantdb}/    跨上下文最小内核
│   │   ├── services/{dictionary,localization}/          服务包 (M3 加更多)
│   │   ├── infrastructure/{pg,redis,minio,logger,metrics,health}/
│   │   ├── interface/{middleware,apiresp,server.go}
│   │   ├── config/  app/
│   ├── migrations/{public,tenant_template}/
│   ├── api/openapi.yaml
│   ├── configs/dev.yaml + i18n/
│   └── test/integration/
├── web/                     Vue 3 + Vite 单 SPA
│   ├── src/{shell,api,router,styles,utils}/
│   ├── package.json + vite.config.ts
├── deployments/             docker-compose dev + nginx
├── docs/superpowers/        spec + plan
├── scripts/                 dev.sh / openapi-gen.sh
└── legacy/                  v1 资产 (Spring Boot), 仅作迁移参考
```

## 快速开始

需要 Go 1.22+, Node 20+, Docker.

```bash
# 一键启动 (推荐):
./scripts/dev.sh
```

手动:

```bash
# 1. 起依赖
cd deployments && docker compose up -d db redis minio

# 2. 跑迁移
cd ../server && go run ./cmd/migrate up

# 3. 起后端 (终端 A)
go run ./cmd/server     # → http://localhost:8080

# 4. 起前端 (终端 B)
cd ../web && npm install && npm run dev   # → http://localhost:5174
```

## 端口映射

本地 SSH 隧道占用 5432/6379/9000/5173, 因此本项目使用替代端口:

| 服务 | 容器内 | 主机端口 |
|---|---|---|
| PostgreSQL | 5432 | **5433** |
| Redis      | 6379 | **6380** |
| MinIO API  | 9000 | **9100** |
| MinIO UI   | 9001 | **9101** |
| Server     | -    | **8080** |
| Web (vite) | -    | **5174** |

## 主要命令

```bash
# Backend
make -C server build        # 构建 server / migrate / tenantctl
make -C server test         # 单测 (要求 PG 起着)
make -C server lint         # golangci-lint
make -C server dev          # go run ./cmd/server
make -C server migrate      # go run ./cmd/migrate up

# Integration smoke
cd server && IOP_INTEGRATION=1 go test ./test/integration/... -v

# Frontend
cd web && npm run build
cd web && npm run dev

# OpenAPI
make -C server openapi-gen  # M1 stub; M3+ 自动生成前端 SDK
```

## M1 验证

```bash
curl http://localhost:8080/livez                 # {"status":"live"}
curl http://localhost:8080/readyz                # {"live":true,"ready":true}
curl http://localhost:8080/version               # {"version":"dev"}
curl http://localhost:8080/healthz               # 详细依赖矩阵
curl http://localhost:8080/api/dict/plan_level   # envelope 包裹的字典项
curl -H "X-Request-Id: abc" http://localhost:8080/version  # trace_id 回响
```

打开 http://localhost:5174 看工作台首页, 包含版本号 + readyz 状态 + 字典样例 + 错误 envelope 演示.

## 开发约定

- 提交粒度小, TDD 优先 (见 `docs/superpowers/plans/`)
- 跨服务通信走 `internal/shared/eventbus` (Publish-Subscribe), 不直接 import
- 错误统一 `internal/shared/errors`, code = `<source>.<resource>.<reason>`
- 日志 trace_id 贯穿; 不打 password / token / secret (logger.Sanitize)
- 多租户 DB 访问走 `tenantdb.TenantDB.Transaction` (自动 SET LOCAL search_path)

## 设计文档

- `docs/superpowers/specs/2026-05-28-go-base-design-v3.md` v3.1 现行 spec (取代 v2.1.1)
- `docs/superpowers/plans/2026-05-29-m1-foundation.md` M1 实施计划 (24 task)

## 测试覆盖 (M1)

```
internal/shared/kernel        13 tests (ID, ctx, time, pagination)
internal/shared/errors         4 tests (Kind, Wrap, Unwrap, As)
internal/shared/eventbus       2 tests (publish/subscribe, multi-subscriber)
internal/shared/tenantdb       3 tests (SET LOCAL search_path, missing ctx, SQL inject guard)
internal/infrastructure/pg     2 tests (pool connect, invalid DSN)
internal/infrastructure/logger 2 tests (JSON output, sanitize sensitive keys)
internal/infrastructure/health 3 tests (healthy, critical down, noncritical down)
internal/interface/middleware  3 tests (RequestID, Recover panic catch)
internal/interface/apiresp     2 tests (OK envelope, Fail kind→status)
internal/services/dictionary   3 tests (lookup, unknown, filter inactive)
internal/services/localization 3 tests (known, fallback, template args)
test/integration               1 test  (9 subtests, end-to-end)
```
