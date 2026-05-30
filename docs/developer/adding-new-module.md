# 添加一个新业务模块

> v3.1 起，平台是「基座 + 可插拔模块」架构。任意业务模块（订单 / CRM / HR…）通过统一的 `Module` 契约接入，**框架内的修改 ≈ 一行注册代码**。

## 1 分钟新增一个模块

```bash
./scripts/new-module.sh <code> "<中文名>" "<分类>"
# 例:
./scripts/new-module.sh crm "客户管理 CRM" "业务管理"
```

脚本生成 9 个文件：

```
server/internal/contexts/crm/          ← 后端模块代码 (DDD 四层)
├── module.go                          实现 module.Module 接口
├── domain/types.go                    业务实体
├── application/service.go             用例编排
├── infrastructure/repo.go             PG 实现（脚手架空壳）
└── interface/http.go                  HTTP handler

server/migrations/tenant_template/
└── 004_crm.up.sql                     租户级表迁移

web/src/modules/crm/                   ← 前端模块代码
├── manifest.ts                        声明 routePrefix / homeRoute
├── routes.ts                          路由（vue-router 格式）
├── api/crm.ts                         axios 客户端
└── views/IndexView.vue                首页（用 PageHeader + DataTable）
```

## 接入步骤

### 后端 1 行注册

`server/internal/app/app.go`:

```go
import "github.com/leo/iop/server/internal/contexts/crm"

// 在 registry.Register(okr.New(deps)) 旁边加一行：
registry.Register(crm.New(deps))
```

### 跑迁移 + 启动

```bash
cd server && ./bin/migrate up && ./bin/server
```

### 安装到当前租户（任选其一）

- 浏览器：左侧 rail 点 **+ 添加** → 应用中心找到「客户管理 CRM」点击安装
- API: `POST /api/admin/apps/crm/install`
- CLI: `./bin/tenantctl app install --tenant=<id> --code=crm`（待实现）

### 完工

- 左侧 rail 自动出现新 app 图标
- `/crm` 路由自动可访问
- `/admin/permissions` 自动列出新模块的 `crm.item:read/write/delete`，角色编辑器可直接选用
- 模块发布的事件（`crm.item_created`）自动被审计 + 通知订阅

## 框架契约对照

### 后端 Module 接口 (`shared/module/module.go`)

```go
type Module interface {
    Manifest() Manifest      // 应用信息 + 权限 + 事件声明
    RegisterRoutes(api *gin.RouterGroup, deps Deps)  // 挂路由到 /api/apps/<code>/
}
```

### Manifest 字段

| 字段 | 含义 | 示例 |
|---|---|---|
| `Code` | 全局唯一标识 | `"crm"` |
| `Name` | 用户可见名称 | `"客户管理 CRM"` |
| `Description` | 应用市场描述 | `"销售线索到合同"` |
| `Icon` | SVG path data | `"M12 2 L22 12 ..."` |
| `Color` | CSS color | `"var(--cat-biz)"` |
| `Category` | 应用类别 | `"业务管理"` |
| `Version` | 模块版本 | `"1.0.0"` |
| `Permissions` | RBAC 资源×动作清单 | `[{Resource: "crm.lead", Action: "read", Label: "查看线索"}]` |
| `Events` | 本模块发布的事件 topic | `["crm.lead_created"]` |

### 前端 manifest

`modules/<code>/manifest.ts`:

```ts
export const manifest = {
  code: "crm",
  name: "客户管理 CRM",
  routePrefix: "/crm",      // 所有路由挂在这个前缀下
  homeRoute: "/crm",        // 左侧 rail 点击时跳到这里
};
```

`modules/<code>/routes.ts`:

```ts
import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "",       name: "crm.home",    component: () => import("./views/IndexView.vue") },
  { path: "leads",  name: "crm.leads",   component: () => import("./views/LeadsView.vue") },
];
```

`router/index.ts` 用 `import.meta.glob('@/modules/*/manifest.ts', { eager: true })` **自动发现并挂载**，**不需要手改 router**。

## 模块可用的 Deps 注入

```go
type Deps struct {
    Pool     *pgxpool.Pool          // 原始 pgx pool
    Tenant   *tenantdb.TenantDB     // 自动 SET LOCAL search_path
    Platform *tenantdb.PlatformDB   // 跨租户表（如 platform_user）
    Bus      eventbus.Bus           // 发事件 + 订阅
    Logger   *zap.Logger
    Clock    kernel.Clock           // 可测试的时钟
}
```

模块**不应**直接访问全局变量或其他模块的代码。

## 模块间通信

通过 **事件总线**，不要直接调用：

```go
// CRM 模块发事件
deps.Bus.Publish(ctx, "crm.lead_converted", map[string]any{
    "lead_id": id, "amount": amount,
})

// OKR 模块订阅（在 OKR 自己的 Wire() 方法里）
bus.Subscribe("crm.lead_converted", func(ctx, e) error {
    // 把成交线索作为本周亮点写入 OKR 报告...
    return nil
})
```

## 复用前端组件

`@/shell/components` 提供 4 个基础积木：

```vue
<template>
  <section>
    <PageHeader title="客户线索" sub="销售管理模块">
      <template #actions>
        <button class="btn btn-primary">+ 新建线索</button>
      </template>
    </PageHeader>

    <div class="stat-grid">
      <StatCard label="本月新增" :value="42" delta="+12%" />
      <StatCard label="待跟进" :value="8" color="var(--warning)" bg="var(--warning-soft)" />
    </div>

    <DataTable :columns="columns" :rows="leads" rowKey="id">
      <template #cell-name="{ row }">
        <strong>{{ row.name }}</strong>
      </template>
    </DataTable>

    <EmptyState v-if="!leads.length" title="尚无线索" sub="点击新建添加" />
  </section>
</template>

<script setup lang="ts">
import { PageHeader, StatCard, DataTable, EmptyState, type Column } from "@/shell/components";
// …
</script>
```

## 不要做的事

| ❌ 不要 | ✅ 改用 |
|---|---|
| `import "github.com/leo/iop/server/internal/contexts/okr"` (在 CRM 内) | 通过 `Bus.Subscribe` 监听 OKR 事件 |
| 在 frontend 跨模块 `import` | 通过事件总线（前端版可走 mitt 或 store） |
| 共享 schema 表（如直接读 `okr_plan`） | 监听 `okr.plan_*` 事件维护自己的投影表 |
| 手改 `router/index.ts` 加模块路由 | 用 `manifest.ts` + `routes.ts`，自动发现 |
| 手改 `LeftRail.vue` 加图标 | `Manifest.Icon` + 安装后自动出现 |

## 完整 demo

跑下面 5 条命令，10 分钟做出一个可登录、可创建数据、有权限、能审计的新业务模块：

```bash
# 1. 脚手架
./scripts/new-module.sh leads "销售线索" "业务管理"

# 2. 注册 (1 行)
sed -i.bak 's/registry.Register(okr.New(deps))/&\n\tregistry.Register(leads.New(deps))/' server/internal/app/app.go
echo 'import "github.com/leo/iop/server/internal/contexts/leads"' >> server/internal/app/app.go.imports  # 手动加 import 行

# 3. 构建 + 启动
make -C server build
./server/bin/server &

# 4. 安装到租户
curl -X POST http://localhost:8080/api/admin/apps/leads/install \
  -H "Authorization: Bearer $TOKEN"

# 5. 看效果
open http://localhost:5174/leads
```
