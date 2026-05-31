# 平台管理员 RBAC + 三员分立 设计(Phase 0 地基)

- 日期:2026-05-30
- 状态:已评审,待用户复核
- 范围:平台控制台(`/platform/*`)的权限地基。**不含**四个治理域的具体功能页(那些在 Phase 1–4 各自出 spec)。

## 1. 背景与目标

当前平台控制台只有 4 个页面(平台概览 / 组织机构平铺列表 / 全局用户平铺列表 / 注册申请),且"平台管理员"是一个**全权大管理员**(`platform_user.is_platform_admin` 全局布尔)。

目标:把平台管理员升级为**可细粒度授权、支持三员分立(等保)**的平台后台地基,让后续四个治理域的功能都能挂到角色上,由超管按权限分配给不同管理员。

四个治理域(后续 Phase):
- 组织与身份治理(组织层级树、用户按组织区分、租户生命周期、全局用户运维)
- 安全与审计
- 应用与系统配置
- 监控与运维工具

## 2. 三员分立模型(等保三权分立)

| 角色 | 能做 | 明确**不能**做(制约点) |
|---|---|---|
| 系统管理员 `sys_admin` | 账号/组织生命周期、组织层级、应用模块启停、参数/字典/公告、监控、定时任务、备份 | 不配置安全策略、**不授权**、不看审计 |
| 安全管理员 `sec_admin` | 给用户/角色**授权**、安全策略(密码/锁定/MFA/IP/限流)、强制下线、授超管资格 | 不操作业务/系统配置、**不能改/删审计** |
| 审计管理员 `audit_admin` | 查看/导出/管理全部审计+登录日志、监督前两员 | 不配置、不授权 |
| 超级管理员 `super_admin` | 初始化:创建三员角色、分配权限、指派用户;兜底 | 见 §3 治理模式 |

核心:**没有任何一个人能"既配置、又授权、又抹掉痕迹"**。

## 3. 治理模式(`platform_setting.governance_mode`)

| | `single_admin`(默认) | `three_member`(严格等保) |
|---|---|---|
| 超管 | 全权,`PlatformAuthz` 对超管短路放行 | **仍全权**(用户要求),但操作**强制写审计 + 必填 reason** |
| 三员 | 存在但通常空置,可作普通委派 | 各司其职;**日常授权归安全管理员** |
| `audit/purge` | 超管可 | **对任何人关闭**(含超管),审计只可归档 |
| 职责分离 | 不校验 | 默认**告警**:同一人不宜同时持多个三员 / 超管+三员(可配为硬约束) |
| 切换模式 | 超管专属,高危,留痕 | 切回 single 同样高危留痕 |

- 默认 `single_admin`,适合试用 / 小运维团队 —— 一个超管搞定。
- 正式上线 / 大团队 / 合规 → 切 `three_member`,严格按三员。

## 4. 数据模型

复用现有 `public.role`(`tenant_id IS NULL`=平台级)+ `public.role_policy`(`role_id, resource, action, effect`),新增三张表。迁移:`server/migrations/public/000009_platform_rbac.{up,down}.sql`。

```sql
-- 权限目录:可分配的平台权限点(供 UI 勾选 + 后端校验)。boot 时由代码注册表 upsert。
CREATE TABLE IF NOT EXISTS public.platform_permission (
    resource     TEXT NOT NULL,
    action       TEXT NOT NULL,
    domain       TEXT NOT NULL,          -- org_identity / security / audit / app_config / monitoring
    label        TEXT NOT NULL,
    is_high_risk BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (resource, action)
);

-- 平台角色 → platform_user 直接赋予(现有 role_grant 是 member×tenant,不适用无租户场景)。
CREATE TABLE IF NOT EXISTS public.platform_role_grant (
    role_id          UUID NOT NULL REFERENCES public.role(id) ON DELETE CASCADE,
    platform_user_id UUID NOT NULL REFERENCES public.platform_user(id) ON DELETE CASCADE,
    granted_by       UUID,
    granted_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (role_id, platform_user_id)
);
CREATE INDEX IF NOT EXISTS platform_role_grant_user_idx ON public.platform_role_grant(platform_user_id);

-- 平台级配置(governance_mode 首用;Phase 3 全局参数复用)。
CREATE TABLE IF NOT EXISTS public.platform_setting (
    key        TEXT PRIMARY KEY,
    value      JSONB NOT NULL,
    updated_by UUID,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

迁移还要:
1. 内置角色:**新增** `super_admin`、`sys_admin`、`sec_admin`、`audit_admin`(均 `tenant_id IS NULL`)。现有 `platform_admin` 角色为遗留(000008 起 auth 已不消费),保留不动、不再使用,避免改名牵连旧 seed。
2. 写三员默认权限(§5)到 `role_policy`。
3. 回填:`UPDATE`/`INSERT` 让现有 `is_platform_admin=true` 的用户在 `platform_role_grant` 中持有 `super_admin`。
4. `platform_setting('governance_mode', '"single_admin"')`。
5. `is_platform_admin` flag 暂留作镜像 / 引导账号兜底;鉴权改读 `platform_role_grant`。

**零破坏**:默认单一超管模式,现网行为不变。

## 5. 权限目录与三员默认切分

权限点 = `(resource, action)`,与租户侧 `RBAC(resource, action)` 对称。`[S]`=sys_admin `[Sec]`=sec_admin `[A]`=audit_admin;`super_admin` 模板=全集。⚠=高危。

**org_identity**:`org/read`[S·Sec·A] · `org/create|update|suspend|close`[S] · `org/hierarchy`[S] · `user/read`[S·Sec·A] · `user/create|update|disable|resetpwd`[S] · `user/impersonate`⚠[S] · `membership/assign`[S]

**security**:`role/manage`[Sec] · `authz/grant`[Sec] · `platform_admin/grant`⚠[Sec] · `security_policy/manage`[Sec] · `session/read`[Sec·A] · `session/revoke`[Sec]

**audit**:`audit/read`[A] · `audit/export`[A] · `audit/config`[A] · `audit/purge`⚠[A,strict 下连超管也无] · `login_log/read`[A]

**app_config**:`app/manage`[S] · `param/manage`[S] · `dict/manage`[S] · `announce/manage`[S] · `branding/manage`[S]

**monitoring**:`monitor/read`[S·A] · `job/manage`[S] · `cache/manage`[S] · `schema/sync`[S] · `codegen/use`[S] · `backup/manage`⚠[S]

高危点(⚠)v1:**强制审计 + 必填 reason**;完整**双人会签**列为后续可选增强,不进 v1。

## 6. 强制中间件(与租户侧对称)

- `PlatformAdminRequired` → **`PlatformAccess`**:放行**持有任一平台角色**的用户(读 `platform_role_grant`,不再只认 `is_platform_admin`)。
- 新增 **`PlatformAuthz(resource, action)`**:解析用户平台角色 → `role_policy` 权限并集 → 是否含该点。
  - `single_admin`:super_admin 短路放行。
  - `three_member`:super_admin 模板=全集仍放行,但写审计;命中高危点要求 reason(缺 → 400)。
- 权限解析做请求级缓存;角色/权限变更下次请求即时生效。

## 7. 接口(`/api/platform/rbac/*`)

| 接口 | 网关 | 作用 |
|---|---|---|
| `GET /rbac/permissions` | `role/manage` | 权限目录(UI 勾选) |
| `GET/POST/PATCH/DELETE /rbac/roles` | `role/manage` | 平台角色 CRUD(内置不可删、可克隆) |
| `PUT /rbac/roles/:id/permissions` | `role/manage` | 设角色权限点 |
| `POST/DELETE /rbac/roles/:id/members` | `authz/grant` | 给角色赋/撤 platform_user |
| `GET /rbac/me` | 登录即可 | 当前用户平台角色+权限(前端门控) |
| `GET/PUT /rbac/governance-mode` | 超管专属 | 读/切治理模式 |

平台组其余路由(组织/用户/审计…)逐条挂对应 `PlatformAuthz(...)`。

## 8. 审计接入

复用 `audit` 服务。每个 `PlatformAuthz` 命中的写操作落事件:`actor, 角色, resource/action, 目标, reason, mode, ip, ts`。
现有 `audit.ListByTenant` 是租户域,需新增 **`ListPlatform`**(平台域审计,供审计管理员 Phase 2 查看)。`three_member` 下审计写入为强约束。

## 9. 前端

- 平台控制台新增 **"权限管理"** 页 `web/src/modules/platform/views/RbacView.vue`:角色列表 + 权限勾选矩阵(按 domain 分组)+ 成员分配;顶部显示当前治理模式。
- `auth.store` 接入 `GET /rbac/me`;菜单/按钮按权限门控(后端为权威)。
- 平台 API 层加 `rbac` 调用。

## 10. 命门测试(`server/test/integration`,对标现有 5 个隔离测试)

1. 职责切分:sys_admin 调 `authz/*`、`audit/*` → 403;sec_admin 调业务/配置 → 403;audit_admin 调配置/授权 → 403。
2. 审计不可毁:`three_member` 下 `audit/purge` 对超管也 403。
3. 超管行为:`single_admin` 超管全通;`three_member` 超管仍通但每次写审计、高危缺 reason → 400。
4. 回填:迁移后原 `is_platform_admin` 用户持有 `super_admin`、可进平台台。
5. 门控权威:无平台角色用户进 `/api/platform/*` → 403(`PlatformAccess`)。
6. 模式切换:非超管切 `governance_mode` → 403。

## 11. 代码落点

- 迁移:`server/migrations/public/000009_platform_rbac.{up,down}.sql`
- 后端:`server/internal/services/iam/platformrbac.go`(权限注册表 + 角色/授权/模式 service);`repo.go` 增平台角色/grant 查询
- 中间件:`admin_http.go`(`PlatformAdminRequired`→`PlatformAccess`、新增 `PlatformAuthz`);`middleware.go` 复用解析
- 路由:`admin_http.go` 新增 `/platform/rbac/*`;`app.go` 平台组改 `PlatformAccess`,各路由挂 `PlatformAuthz`
- 审计:`audit/audit.go` 增 `ListPlatform`
- 前端:`web/src/modules/platform/views/RbacView.vue`、`platform/api`、`shell/auth/auth.store.ts`(`rbac/me` 门控)

## 12. 非目标(明确不做 / 后续)

- 四个治理域的具体功能页(Phase 1–4)。
- 完整双人会签 / 审批流(后续可选)。
- 组织层级树、用户按组织区分(属 Phase 1,本 spec 仅为其预留 `org/hierarchy`、`membership/assign` 权限点)。
