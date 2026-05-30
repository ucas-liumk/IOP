#!/usr/bin/env bash
# Scaffolds a new business module across backend + frontend.
#
# Usage:
#   ./scripts/new-module.sh <code> "<Chinese name>" "<category>"
#
# Example:
#   ./scripts/new-module.sh crm "客户管理 CRM" "业务管理"
#
# What it generates:
#   server/internal/contexts/<code>/{module.go, application/service.go,
#                                    infrastructure/repo.go, interface/http.go,
#                                    domain/types.go}
#   server/migrations/tenant_template/<NNNN>_<code>.up.sql + .down.sql
#   web/src/modules/<code>/{manifest.ts, routes.ts, api.ts, views/IndexView.vue}
#
# After running:
#   1. Edit server/internal/app/app.go — add `registry.Register(<code>.New(deps))`
#   2. Run `make -C server migrate` then `make -C server build`
#   3. Run `npm --prefix web run dev` and visit /<code>
#   4. Install for your tenant via AppCenter, OR via API:
#      POST /api/admin/apps/<code>/install
set -euo pipefail

CODE="${1:-}"
NAME="${2:-}"
CATEGORY="${3:-协同办公}"

if [[ -z "$CODE" || -z "$NAME" ]]; then
  echo "Usage: $0 <code> \"<Chinese name>\" [\"<category>\"]"
  echo "Example: $0 crm \"客户管理 CRM\" \"业务管理\""
  exit 1
fi

if [[ ! "$CODE" =~ ^[a-z][a-z0-9_]*$ ]]; then
  echo "Code must be lowercase letters/digits/_  e.g. crm, hr_pro"
  exit 1
fi

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

SERVER_DIR="$ROOT/server/internal/contexts/$CODE"
MIGRATION_DIR="$ROOT/server/migrations/tenant_template"
WEB_DIR="$ROOT/web/src/modules/$CODE"

if [[ -d "$SERVER_DIR" || -d "$WEB_DIR" ]]; then
  echo "Module '$CODE' already exists. Aborting."
  exit 1
fi

# Capitalize first letter (bash 3.x compatible — works on macOS).
CODE_CAP=$(awk -v s="$CODE" 'BEGIN { printf "%s%s", toupper(substr(s,1,1)), substr(s,2) }')

# Pick a category color
case "$CATEGORY" in
  "协同办公") COLOR_VAR="var(--cat-collab)" ;;
  "业务管理") COLOR_VAR="var(--cat-biz)" ;;
  "财务税务") COLOR_VAR="var(--cat-finance)" ;;
  "人力资源") COLOR_VAR="var(--cat-hr)" ;;
  "数据分析") COLOR_VAR="var(--cat-data)" ;;
  *)         COLOR_VAR="var(--cat-admin)" ;;
esac

# Next migration sequence
NEXT_SEQ=$(ls "$MIGRATION_DIR"/*.up.sql 2>/dev/null | wc -l | xargs)
NEXT_SEQ=$(printf "%03d" $((NEXT_SEQ + 1)))

echo "[scaffold] code=$CODE name=$NAME category=$CATEGORY color=$COLOR_VAR seq=$NEXT_SEQ"

# === Backend ===
mkdir -p "$SERVER_DIR"/{application,domain,infrastructure,interface}

# domain/types.go
cat > "$SERVER_DIR/domain/types.go" <<EOF
package domain

import (
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

// Item is the seed aggregate for the $CODE module. Rename/extend as needed.
type Item struct {
	ID        kernel.ID \`json:"id"\`
	Title     string    \`json:"title"\`
	Body      string    \`json:"body"\`
	CreatedAt time.Time \`json:"created_at"\`
}
EOF

# application/service.go
cat > "$SERVER_DIR/application/service.go" <<EOF
package application

import (
	"context"

	"github.com/leo/iop/server/internal/contexts/$CODE/domain"
	"github.com/leo/iop/server/internal/shared/eventbus"
	"github.com/leo/iop/server/internal/shared/kernel"
	"github.com/leo/iop/server/internal/shared/tenantdb"
)

type Service struct {
	tenant *tenantdb.TenantDB
	bus    eventbus.Bus
	clock  kernel.Clock
}

func NewService(tenant *tenantdb.TenantDB, bus eventbus.Bus, clk kernel.Clock) *Service {
	return &Service{tenant: tenant, bus: bus, clock: clk}
}

// CreateItem inserts a row into ${CODE}_item and publishes ${CODE}.item_created.
func (s *Service) CreateItem(ctx context.Context, title, body string) (*domain.Item, error) {
	// TODO: replace with real DB-backed implementation.
	item := &domain.Item{
		ID:        kernel.NewID(),
		Title:     title,
		Body:      body,
		CreatedAt: s.clock.Now(),
	}
	_ = s.bus.Publish(ctx, "$CODE.item_created", map[string]any{
		"item_id": item.ID, "title": item.Title,
	})
	return item, nil
}
EOF

# infrastructure/repo.go (skeleton)
cat > "$SERVER_DIR/infrastructure/repo.go" <<EOF
package infrastructure

// PG repository implementations go here.
// See contexts/okr/infrastructure for the full pattern.
EOF

# interface/http.go
cat > "$SERVER_DIR/interface/http.go" <<EOF
package iface

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/$CODE/application"
	"github.com/leo/iop/server/internal/interface/apiresp"
	"github.com/leo/iop/server/internal/shared/errors"
)

// RegisterRoutes mounts /items under the caller's group (typically /api/apps/$CODE).
func RegisterRoutes(r *gin.RouterGroup, svc *application.Service) {
	r.GET("/items", func(c *gin.Context) {
		apiresp.OK(c, gin.H{"items": []any{}, "message": "$CODE module bootstrapped"})
	})
	r.POST("/items", func(c *gin.Context) {
		var req struct{ Title, Body string }
		if err := c.ShouldBindJSON(&req); err != nil {
			apiresp.Fail(c, errors.Wrap(errors.KindParam, "$CODE.invalid_request", "请求格式错误", err))
			return
		}
		item, err := svc.CreateItem(c.Request.Context(), req.Title, req.Body)
		if err != nil {
			apiresp.Fail(c, err)
			return
		}
		apiresp.Created(c, item)
	})
}
EOF

# module.go (the contract entry point)
cat > "$SERVER_DIR/module.go" <<EOF
package $CODE

import (
	"github.com/gin-gonic/gin"

	"github.com/leo/iop/server/internal/contexts/$CODE/application"
	iface "github.com/leo/iop/server/internal/contexts/$CODE/interface"
	"github.com/leo/iop/server/internal/shared/module"
)

type Module struct {
	app *application.Service
}

func New(deps module.Deps) module.Module {
	return &Module{
		app: application.NewService(deps.Tenant, deps.Bus, deps.Clock),
	}
}

func (m *Module) Manifest() module.Manifest {
	return module.Manifest{
		Code:        "$CODE",
		Name:        "$NAME",
		Description: "由脚手架生成 · 替换为真实描述",
		Icon:        "M12 2 L22 12 L12 22 L2 12 Z",
		Color:       "$COLOR_VAR",
		Category:    "$CATEGORY",
		Version:     "0.1.0",
		Permissions: []module.Permission{
			{Resource: "$CODE.item", Action: "read",   Label: "查看"},
			{Resource: "$CODE.item", Action: "write",  Label: "新建/编辑"},
			{Resource: "$CODE.item", Action: "delete", Label: "删除"},
		},
		Events: []string{
			"$CODE.item_created",
		},
	}
}

func (m *Module) RegisterRoutes(api *gin.RouterGroup, _ module.Deps) {
	iface.RegisterRoutes(api, m.app)
}
EOF

# Migration
cat > "$MIGRATION_DIR/${NEXT_SEQ}_${CODE}.up.sql" <<EOF
-- $CODE: $NAME
-- Seed table. Add columns / additional tables as the domain grows.
CREATE TABLE IF NOT EXISTS ${CODE}_item (
    id          UUID PRIMARY KEY,
    title       TEXT NOT NULL,
    body        TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
EOF

cat > "$MIGRATION_DIR/${NEXT_SEQ}_${CODE}.down.sql" <<EOF
DROP TABLE IF EXISTS ${CODE}_item;
EOF

# === Frontend ===
mkdir -p "$WEB_DIR/views" "$WEB_DIR/api"

cat > "$WEB_DIR/manifest.ts" <<EOF
export const manifest = {
  code: "$CODE",
  name: "$NAME",
  routePrefix: "/$CODE",
  homeRoute: "/$CODE",
};
EOF

cat > "$WEB_DIR/routes.ts" <<EOF
import type { RouteRecordRaw } from "vue-router";

export const routes: RouteRecordRaw[] = [
  { path: "", name: "$CODE.home", component: () => import("./views/IndexView.vue") },
];
EOF

cat > "$WEB_DIR/api/${CODE}.ts" <<EOF
import { client } from "@/api/client";

export interface ${CODE_CAP}Item {
  id: string;
  title: string;
  body: string;
  created_at: string;
}

export async function listItems(): Promise<${CODE_CAP}Item[]> {
  const r = await client.get("/apps/$CODE/items");
  return r.data?.data?.items ?? [];
}

export async function createItem(title: string, body: string): Promise<${CODE_CAP}Item> {
  const r = await client.post("/apps/$CODE/items", { title, body });
  return r.data?.data;
}
EOF

cat > "$WEB_DIR/views/IndexView.vue" <<'VUEEOF'
<template>
  <section>
    <PageHeader title="MODULE_NAME_PLACEHOLDER" sub="由脚手架生成 · 替换为真实功能">
      <template #actions>
        <button class="btn btn-primary" @click="showCreate = !showCreate">+ 新建</button>
      </template>
    </PageHeader>

    <article v-if="showCreate" class="card create-form">
      <form @submit.prevent="create">
        <div class="row">
          <input class="input" v-model="form.title" placeholder="标题" required />
          <input class="input" v-model="form.body" placeholder="内容" />
          <button class="btn btn-primary" type="submit">提交</button>
        </div>
      </form>
    </article>

    <DataTable :columns="columns" :rows="items" rowKey="id" emptyText="暂无数据">
      <template #cell-title="{ row }">
        <strong>{{ row.title }}</strong>
      </template>
      <template #cell-created_at="{ row }">
        <span class="time">{{ row.created_at?.slice(0, 10) }}</span>
      </template>
    </DataTable>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, DataTable, type Column } from "@/shell/components";
import { createItem, listItems, type MODULE_TYPE_PLACEHOLDER } from "../api/MODULE_CODE_PLACEHOLDER";

const items = ref<MODULE_TYPE_PLACEHOLDER[]>([]);
const showCreate = ref(false);
const form = reactive({ title: "", body: "" });

const columns: Column[] = [
  { key: "title", label: "标题" },
  { key: "body", label: "内容" },
  { key: "created_at", label: "创建时间", width: "140px" },
];

onMounted(async () => { items.value = await listItems(); });

async function create() {
  if (!form.title) return;
  try {
    await createItem(form.title, form.body);
    form.title = ""; form.body = "";
    showCreate.value = false;
    items.value = await listItems();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "创建失败"); }
}
</script>

<style scoped>
.create-form { padding: 14px; margin-bottom: 14px; }
.row { display: grid; grid-template-columns: 1fr 2fr auto; gap: 10px; }
.input { padding: 8px 12px; border: 1px solid var(--border-strong); border-radius: 7px; font-size: 13px; }
.btn { padding: 8px 16px; border-radius: 7px; font-size: 13px; cursor: pointer; border: 1px solid var(--border); background: var(--surface); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.time { font-family: var(--ff-mono); color: var(--text-3); font-size: 12px; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
</style>
VUEEOF

# Substitute placeholders in the Vue file (sed -i differs Mac vs Linux)
sed -i.bak "s/MODULE_NAME_PLACEHOLDER/$NAME/g; s/MODULE_TYPE_PLACEHOLDER/${CODE_CAP}Item/g; s/MODULE_CODE_PLACEHOLDER/$CODE/g" "$WEB_DIR/views/IndexView.vue"
rm "$WEB_DIR/views/IndexView.vue.bak"

echo
echo "✅ Module '$CODE' scaffolded."
echo
echo "Next steps:"
echo "  1. Wire it into app.go:"
echo "       import \"github.com/leo/iop/server/internal/contexts/$CODE\""
echo "       registry.Register($CODE.New(deps))"
echo "  2. cd server && make build && ./bin/migrate up"
echo "  3. Install for your tenant via AppCenter (\"+ 添加\" button), or:"
echo "       curl -X POST http://localhost:8080/api/admin/apps/$CODE/install \\"
echo "         -H \"Authorization: Bearer \$TOKEN\""
echo "  4. Visit http://localhost:5174/$CODE"
echo
echo "Generated files:"
echo "  - $SERVER_DIR/"
echo "  - $MIGRATION_DIR/${NEXT_SEQ}_${CODE}.{up,down}.sql"
echo "  - $WEB_DIR/"
