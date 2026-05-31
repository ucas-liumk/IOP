# IOP 后台重设计 — 通用 RBAC + ruoyi/jeecg 式双控制台

- 日期:2026-05-30
- 状态:已确认方向,进入实现
- 取代:`2026-05-28...platform-admin-rbac` 的「三员分立 + governance_mode」特例设计(PR#1)。本设计**废弃** governance_mode / super 短路 / 三员硬编码,改为**通用 用户→角色→(菜单权限 + 数据范围)**;三员只是可配置角色。

## 0. 核心决策(已与用户确认)

1. **通用 RBAC**:用户 → 角色 →(菜单权限 + 数据范围)。无 `governance_mode`、无代码级 admin 短路。内置管理员角色靠 `*`/`*` 通配策略获得全权(纯配置)。
2. **菜单/权限 = 模块声明 + 目录装配**:模块在 Manifest 声明 `Menus[]`(菜单树节点 + 按钮权限),启动汇总成目录树。角色编辑器从目录树勾选;平台按租户启停模块 = 控菜单可见。保留可插拔。
3. **组织内部**:`department`(树)+ `post`(岗位),落租户 schema。
4. **数据范围**:`role.data_scope` + `role_dept`(预留),统一 helper,本期不强制套用业务查询。
5. **范围**:两个控制台全做,分三期:核心 → 租户台 → 平台台。

## 1. 菜单/权限目录模型

权限授予底座仍是现有 `public.role_policy(role_id, resource, action, effect)` + `Enforce`/`matchPolicy`(不重造)。菜单树是覆盖在其上的可见性/展示层。

**Manifest 扩展**(`server/internal/shared/module/module.go` + 前端 `appstore.ts`):
```go
type MenuNode struct {
    Key     string `json:"key"`      // 唯一,如 "okr.plans"
    Title   string `json:"title"`    // "我的计划"
    Icon    string `json:"icon"`     // SVG path
    Path    string `json:"path"`     // 前端路由,如 "/okr/plans"(目录节点可空)
    Parent  string `json:"parent"`   // 父节点 key,空=顶级
    Type    string `json:"type"`     // "dir" | "menu" | "button"
    Console string `json:"console"`  // "platform" | "tenant" | "both"
    App     string `json:"app"`      // 所属模块 code(接 AppEnabled 门控),平台内置菜单可空
    Perm    string `json:"perm"`     // 需要的权限 "resource:action";空=仅需 App 启用/登录
    Order   int    `json:"order"`
}
```
- `Manifest.Menus []MenuNode` 新增字段(与现有 `Permissions[]` 并存:Permissions 给 API 网关,Menus 给 UI 可见性)。
- `Registry.AllMenus()` 汇总;`Registry.MenuTree(console)` 组装成树。
- **可见性**:用户能见某节点 ⟺ `Perm` 被其角色策略命中(拆成 resource:action 走 `Enforce` 匹配)+ 若 `App` 非空则该租户启用了该模块 + `Console` 匹配当前台。
- 新增接口:
  - `GET /me/menus?console=tenant|platform` → 当前用户可见菜单树(前端动态渲染导航)
  - `GET /me/perms` → 扁平权限 key 集 `["okr:plan:read", ...]`(前端按钮级 `v-perm` 门控)
  - `GET /admin/menus`(租户)/ `GET /platform/menus`(平台) → 完整目录树(角色编辑器勾选用)

## 2. 数据模型

**平台 `public`**(迁移 `000010_rbac_core.up.sql`):
- `role` ADD `data_scope TEXT NOT NULL DEFAULT 'all'`(`all`/`dept`/`dept_and_sub`/`self`/`custom`,预留)。
- `role` ADD `is_builtin BOOLEAN NOT NULL DEFAULT FALSE`(内置角色不可删/改 code)。
- 新增 `role_dept(role_id UUID, tenant_id UUID, dept_id UUID, PRIMARY KEY(role_id, dept_id))` — custom 数据范围(应用层校验,预留)。
- **去掉 `Enforce` 代码短路**:删 `service.go` 里 `if r.Code=="tenant_admin"||"platform_admin" {return nil}`;迁移给 `tenant_admin`、`platform_admin`/`super_admin` seed 一条 `role_policy(role_id, '*','*','allow')`。

**PR#1 处置**(迁移 `000010` 内 + 代码):
- **KEEP**:`role`/`role_policy`/`role_grant`、`platform_role_grant`(平台用户×角色)、`PlatformAuthz` 中间件(改造成纯权限检查)。
- **REMOVE**:`platform_setting(governance_mode)` 行 + `governance_mode` 全部逻辑;`EnforcePlatform` 的 super 短路 + audit/purge 锁;`defaultRolePolicies` 三员硬编码 seed。`EnforcePlatform` 简化为「列平台角色→匹配 role_policy」(与 `Enforce` 同构,无短路;内置 `super_admin` 角色靠 `*`/`*` 策略全权)。
- **ADAPT**:`platform_permission` 目录表 → 并入菜单/权限目录(由模块声明汇总)。`platform_audit_log` 保留(操作日志用)。

**租户 schema `tenant_template`**(迁移 `000005_org_core.up.sql`):
- `department(id UUID PK, name TEXT, parent_id UUID NULL, order_num INT DEFAULT 0, leader TEXT, phone TEXT, email TEXT, status TEXT DEFAULT 'active', created_at TIMESTAMPTZ DEFAULT now())` — 自引用树。
- `post(id UUID PK, code TEXT UNIQUE, name TEXT, order_num INT DEFAULT 0, status TEXT DEFAULT 'active', created_at)`。
- `member` ADD `dept_id UUID`(可空,FK→department);保留 `department TEXT` 兼容。
- `member_post(member_id UUID, post_id UUID, PRIMARY KEY(member_id, post_id))`。

## 3. 后端 API(第 1 期)

- **目录/菜单**:`/me/menus`、`/me/perms`、`/admin/menus`、`/platform/menus`(§1)。
- **部门(租户)**:`GET/POST/PATCH/DELETE /admin/depts`、`GET /admin/depts/tree`、`POST /admin/depts/:id/move`。
- **岗位(租户)**:`GET/POST/PATCH/DELETE /admin/posts`。
- **角色**:扩展 `POST/PATCH /admin/roles`、`/platform/roles` 接受 `data_scope`(+ custom `dept_ids`);列表返回 `data_scope`、`is_builtin`。菜单权限授予仍走 `/.../roles/:id/policies`(resource/action),但 UI 用菜单树勾选。
- **成员**:`PATCH /admin/members/:id` 接受 `dept_id`;`POST/DELETE /admin/members/:id/posts`;成员列表支持 `?dept_id=`(可选含子树)筛选 + 搜索。
- **鉴权**:`Enforce`(去短路)+ `EnforcePlatform`(去短路);新增 `DataScope(ctx, memberID, tenantID) → ScopeSpec`(预留 helper)。
- **审计/日志**:操作日志写 `platform_audit_log`(平台)+ 租户 `audit_log`(已有);登录日志(`iam.user_logged_in/out/login_failed` 事件已有 → 落库查询接口)。

## 4. 前端(第 1 期)

- `auth.store`:登录/restore 后加载 `/me/menus` + `/me/perms`,存 `menus`、`perms`。
- **`v-perm` 指令** + `hasPerm(key)` helper(`web/src/shell/auth/perm.ts`):按钮级门控。
- **动态导航**:把 `AdminLayout`/`PlatformLayout` 的硬编码 nav 换成由 `/me/menus` 渲染的菜单树组件(`web/src/shell/layout/DynamicNav.vue`)。
- **角色编辑器**:`RoleEditor` 用**菜单树勾选框**(替代扁平 by_resource 下拉)+ 数据范围选择(全部/本部门/本部门及以下/仅本人/自定义→选部门)。
- 复用 shell 组件(PageHeader/DataTable/TreeSelect 新增/notify/confirm)。

## 5. 三员 seed + 内置角色

- 平台内置角色(seed,`is_builtin=true`):`super_admin`(`*`/`*` 全权)。可选模板:`sys_admin`/`sec_admin`/`audit_admin`(给定菜单权限集,作为现成模板,纯配置、可改可删)。
- 租户内置角色:`tenant_admin`(`*`/`*`)、`tenant_member`(各模块 read/write 默认)。
- boot seed 幂等(沿用 `SeedDefaults`/`SeedPlatformRBAC` 思路,去掉 governance/三员特例)。

## 6. 第 2 期 — 租户台(组织内部,ruoyi 式)模块清单

页面均在 `/admin/*`,`TenantAdminRequired` + 各菜单权限网关:
- **部门管理**:左树 + 右详情,增删改、拖拽排序、停用。
- **岗位管理**:列表 CRUD。
- **用户(成员)管理**:左**部门树**筛选 + 右用户表;分配角色/岗位、设部门、重置密码、停用/启用、导入/导出(CSV)。
- **角色管理**:菜单树勾选 + 数据范围;内置角色只读 code。
- **字典管理**:字典类型 + 字典数据(复用现有 `dictionary` 服务,补 UI)。
- **通知公告**:`notice` 表 + 列表/发布/撤回。
- **操作日志 + 登录日志**:查询/筛选/详情。
- **在线用户**:列出活跃 session(`public.session`),强制下线。

## 7. 第 3 期 — 平台台(全局治理)模块清单

页面均在 `/platform/*`,`PlatformAccess` + 各菜单权限网关:
- **组织(租户)管理**:列表/开通/停用/关闭/详情(复用现有 tenancy)。
- **全局用户**:跨租户用户 CRUD/停用/重置(复用现有)。
- **平台角色**:同租户角色编辑器,平台级权限。
- **菜单目录管理**:查看汇总菜单/权限树;按租户启停模块(复用 appstore)。
- **字典 / 参数配置**:平台级 `platform_setting`(去 governance 后改为通用 KV 参数)。
- **通知公告 / 操作日志 / 登录日志 / 在线用户**:平台范围。
- **监控**:服务/DB/Redis/MinIO 健康(复用 `health.Registry`)、缓存查看。
- **定时任务**:任务列表/启停/执行记录(新增轻量 scheduler)。
- **代码生成**(可选,排最后):按表生成 CRUD 脚手架。

## 8. 测试 & 验收(每期)

- 后端:`go build ./...`、`go vet ./...`、`make build`、集成命门(角色仅勾部分菜单→只能访问对应权限;部门树/岗位/成员分配;去短路后内置角色靠通配策略仍全权;`/me/menus` 按角色过滤)。
- 前端:`vue-tsc --noEmit`、`npm run build`。
- 全程不破坏现有 5 条隔离命门 + smoke。

## 9. 非目标(本轮明确不做)

- 数据范围对所有业务查询的强制套用(预留,逐步接入)。
- 工作流/低代码/报表/大屏(jeecg 高级特性,后续)。
- 代码生成器在 P3 视进度为可选。
