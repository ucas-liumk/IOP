# 安全策略 / Security Policy

感谢你帮助保障 IOP（企业多租户 B 端办公平台基座）的安全。本文档说明受支持的版本、如何私密上报漏洞，以及部署到生产前**必须**核对的安全配置。

Thanks for helping keep IOP secure. This document describes supported versions, how to privately report a vulnerability, and the security configuration you **must** review before any production deployment.

## 受支持的版本 / Supported Versions

| 版本 Version | 状态 Status | 安全更新 Security fixes |
|---|---|---|
| 0.1.x | ✅ 活跃 Active | ✅ 是 Yes |
| < 0.1.0 | ⚠️ 预发布 Pre-release | ❌ 否 No |

项目仍处于 `0.x` 阶段，API 与数据模型可能变更。安全修复仅在最新的次版本线上提供，建议始终运行最新的 `0.1.x`。

While the project is in `0.x`, APIs and data models may change. Security fixes are delivered only on the latest minor line — always run the most recent `0.1.x`.

## 上报漏洞 / Reporting a Vulnerability

**请勿** 通过公开 issue、PR、讨论区或社交媒体披露安全漏洞。

**Do not** disclose security issues via public issues, PRs, discussions, or social media.

请使用 GitHub 私密漏洞上报通道：

Please use GitHub's private vulnerability reporting:

1. 打开仓库的 **Security** 标签页 → **Report a vulnerability**（GitHub Security Advisories）。
2. 描述受影响的组件（server / web / migrations / 部署）、复现步骤、影响范围，以及可能的修复建议。
3. 如果涉及多租户隔离（命门），请尽量说明是否存在跨租户数据可达性。

Open the repo's **Security** tab → **Report a vulnerability** (GitHub Security Advisories), and include the affected component (server / web / migrations / deployment), reproduction steps, impact, and any suggested fix. For tenant-isolation (keystone) issues, note whether cross-tenant data is reachable.

### 响应 SLA / Response SLA

| 阶段 Stage | 目标时间 Target |
|---|---|
| 首次确认 Acknowledgement | 3 个工作日 / 3 business days |
| 初步评估与定级 Triage & severity | 7 个工作日 / 7 business days |
| 修复或缓解 Fix or mitigation | 严重 Critical: 30 天 / days；其他 Others: 90 天 / days |
| 协同披露 Coordinated disclosure | 修复发布后 Upon fix release |

我们遵循协同披露原则，并会在公告中署名上报者（除非你要求匿名）。

We follow coordinated disclosure and will credit reporters in the advisory unless you request anonymity.

## 安全须知 / Security Notes

以下为部署到**生产环境前必须**核对的高优先级项。默认配置面向本地开发，**不可**直接用于生产。

The following are high-priority items you **must** review before production. Defaults target local development and are **not** safe for production as-is.

### 1. 轮换默认管理员账号 / Rotate the default admin account

启动时会自动播种内置平台管理员账号（见 `server/internal/services/iam/seed.go`）：

A built-in platform admin is seeded on boot (see `server/internal/services/iam/seed.go`):

```
username: admin
password: Admin12345!
```

- ⚠️ 该默认口令为**公开已知**，生产环境必须**立即轮换**。
- 通过环境变量 `IOP_SEED_ADMIN_PASSWORD` 覆盖播种口令，避免使用内置默认值。
- 启用**首次登录强制改密**，确保任何遗留的默认口令在首次使用后立即失效。
- 该 admin 账号会加入系统租户（slug `system`）并被授予 `platform_admin` 角色，权限极高，务必妥善保护。

The default password is **publicly known**; **rotate it immediately** in production. Override the seeded password with `IOP_SEED_ADMIN_PASSWORD`, and enable **forced password change on first login** so any leftover default credential is invalidated on first use. This account joins the system tenant (slug `system`) and is granted the high-privilege `platform_admin` role — protect it accordingly.

### 2. 设置 JWT 签名密钥 / Set the JWT signing secret

- 生产环境必须通过 `IOP_AUTH_JWT_SECRET` 显式设置 JWT 签名密钥（环境变量前缀为 `IOP`，格式 `IOP_<SECTION>_<KEY>`）。
- 密钥长度**至少 32 个字符**，使用密码学安全的随机值，**禁止**复用开发默认值或弱口令。
- 切勿将密钥提交到版本库；通过环境变量或密钥管理服务注入。
- 轮换密钥会使现有令牌失效，请在维护窗口内操作。

Production must set the JWT signing secret via `IOP_AUTH_JWT_SECRET` (env prefix `IOP`, form `IOP_<SECTION>_<KEY>`). Use a cryptographically random value of **at least 32 characters**; never reuse the dev default or a weak value, and never commit it. Rotating it invalidates existing tokens — do it in a maintenance window.

### 3. 收紧网络与数据库配置 / Harden network & database config

- **CORS 不可使用 `*`**。生产环境必须将允许来源收敛到明确的前端域名白名单，避免凭据随通配源泄露。
- **PostgreSQL 必须启用 TLS**：生产连接串使用 `sslmode=require`（或更严格的 `verify-full`），不要使用 `disable`。
- Redis / MinIO 等依赖也应启用认证与传输加密，且不暴露到公网。
- 启用慢查询、限流、幂等等 B2B 基本盘防护（见 README 与运维文档）。

**CORS must not be `*`** — restrict allowed origins to an explicit frontend allowlist so credentials are not exposed to wildcard origins. **PostgreSQL must use TLS**: production DSNs should set `sslmode=require` (or stricter `verify-full`), never `disable`. Authenticate and encrypt Redis / MinIO and keep them off the public internet. Enable the B2B baseline protections (rate limiting, idempotency, slow-query guards) described in the README and ops docs.

### 多租户隔离 / Tenant isolation (keystone)

多租户隔离是本项目的「命门」。租户级数据访问必须经由 `tenantdb.TenantDB.Transaction`（自动 `SET LOCAL search_path`）。任何可能造成跨 schema / 跨租户数据可达的问题，请按上述流程私密上报。隔离回归测试见 `server/test/integration/tenant_isolation_test.go`。

Tenant isolation is the project's keystone. Tenant-scoped access must go through `tenantdb.TenantDB.Transaction` (auto `SET LOCAL search_path`). Report any cross-schema / cross-tenant reachability privately via the process above. Isolation regression tests live in `server/test/integration/tenant_isolation_test.go`.
