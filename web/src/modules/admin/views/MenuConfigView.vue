<template>
  <section class="admin-page">
    <PageHeader title="菜单管理" :sub="`本租户可用菜单 · ${rows.length} 个节点`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
      </template>
    </PageHeader>

    <article class="card">
      <table class="tree-table">
        <thead>
          <tr>
            <th>菜单名称</th>
            <th>类型</th>
            <th>路由</th>
            <th>权限标识</th>
            <th>状态</th>
            <th>租户启用</th>
            <th>排序</th>
            <th class="op">操作</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="row in rows" :key="row.key">
            <td>
              <div class="name-cell" :style="{ paddingLeft: `${row.depth * 18}px` }">
                <span class="tree-mark">{{ row.children?.length ? '▾' : '' }}</span>
                <span class="menu-title">{{ row.title }}</span>
                <code>{{ row.key }}</code>
                <em v-if="row.built_in">内置</em>
              </div>
            </td>
            <td>{{ typeLabel(row.type) }}</td>
            <td><code v-if="row.path">{{ row.path }}</code><span v-else class="muted">-</span></td>
            <td><code v-if="row.perm" class="perm">{{ row.perm }}</code><span v-else class="muted">-</span></td>
            <td><span class="status" :class="row.status === 'active' ? 'on' : 'off'">{{ row.status === 'active' ? '正常' : '停用' }}</span></td>
            <td>
              <span class="status" :class="row.tenant_enabled !== false ? 'on' : 'off'">
                {{ row.tenant_enabled !== false ? '启用' : '禁用' }}
              </span>
            </td>
            <td>
              <input class="order-input" type="number" :value="row.order" @change="setOrder(row, $event)" />
            </td>
            <td class="op">
              <button class="link-btn" v-perm="'menu:write'" @click="toggle(row)">
                {{ row.tenant_enabled !== false ? '禁用' : '启用' }}
              </button>
            </td>
          </tr>
          <tr v-if="rows.length === 0"><td colspan="8" class="empty">暂无菜单</td></tr>
        </tbody>
      </table>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PageHeader } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import {
  getTenantMenuConfigTree, updateTenantMenuConfig, type MenuTreeNode,
} from "@/modules/platform/api/menus";

type Row = MenuTreeNode & { depth: number };

const notify = useNotification();
const menus = ref<MenuTreeNode[]>([]);
const rows = computed<Row[]>(() => flatten(menus.value));

onMounted(reload);

async function reload() {
  menus.value = await getTenantMenuConfigTree();
}
function flatten(nodes: MenuTreeNode[], depth = 0): Row[] {
  const out: Row[] = [];
  for (const n of nodes) {
    out.push({ ...n, depth });
    if (n.children?.length) out.push(...flatten(n.children, depth + 1));
  }
  return out;
}
async function toggle(row: Row) {
  try {
    await updateTenantMenuConfig(row.key, { enabled: row.tenant_enabled === false, order: row.order });
    await reload();
    notify.success("菜单配置已更新");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "更新失败");
  }
}
async function setOrder(row: Row, evt: Event) {
  const raw = (evt.target as HTMLInputElement).value;
  const order = Number(raw || 0);
  try {
    await updateTenantMenuConfig(row.key, { enabled: row.tenant_enabled !== false, order });
    await reload();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "排序更新失败");
  }
}
function typeLabel(t: string) {
  return ({ dir: "目录", menu: "菜单", button: "按钮", link: "外链", iframe: "iframe", micro: "微前端" } as Record<string, string>)[t] ?? t;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; overflow: hidden; }
.tree-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.tree-table th, .tree-table td { padding: 9px 10px; border-bottom: 1px solid var(--border-soft); text-align: left; vertical-align: top; }
.tree-table th { font-size: 11.5px; color: var(--text-3); font-weight: 700; text-transform: uppercase; letter-spacing: .4px; }
.name-cell { display: flex; align-items: center; gap: 7px; min-width: 260px; }
.tree-mark { width: 12px; color: var(--text-4); }
.menu-title { font-weight: 600; }
code { font-family: var(--ff-mono); font-size: 11px; color: var(--text-3); }
em { font-style: normal; font-size: 10px; color: var(--purple); background: var(--purple-soft); padding: 1px 5px; border-radius: 3px; }
.perm { color: var(--primary); background: var(--primary-soft); padding: 1px 5px; border-radius: 3px; }
.status { font-size: 11px; font-weight: 700; padding: 2px 6px; border-radius: 4px; }
.status.on { background: var(--success-soft); color: var(--success); }
.status.off { background: var(--surface-2); color: var(--text-3); }
.muted, .empty { color: var(--text-4); }
.empty { text-align: center; padding: 24px; }
.order-input { width: 72px; padding: 5px 7px; border: 1px solid var(--border); border-radius: 6px; background: var(--surface); }
.op { width: 80px; }
.link-btn { border: 0; background: transparent; color: var(--primary); cursor: pointer; font-size: 12px; padding: 3px 5px; }
.link-btn:hover { background: var(--primary-soft); border-radius: 4px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-ghost { background: transparent; }
</style>
