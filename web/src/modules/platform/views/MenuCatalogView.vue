<template>
  <section class="admin-page">
    <PageHeader title="菜单目录" sub="全平台菜单 / 权限目录（只读巡检）· 两套控制台">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
      </template>
    </PageHeader>

    <div class="console-tabs">
      <button class="tab" :class="{ on: activeConsole === 'platform' }" @click="activeConsole = 'platform'">
        平台控制台
        <span class="tab-count">{{ countMenus(platformMenus) }}</span>
      </button>
      <button class="tab" :class="{ on: activeConsole === 'tenant' }" @click="activeConsole = 'tenant'">
        组织控制台
        <span class="tab-count">{{ countMenus(tenantMenus) }}</span>
      </button>
    </div>

    <p class="hint">
      菜单目录由各模块在 Manifest 中声明、启动时汇总而成（目录 / 菜单 / 按钮三级，节点权限为 <code>resource:action</code>）。
      此处为只读巡检视图：平台角色的可见性由「平台角色」页的菜单勾选控制；组织内部菜单可见性由各组织管理员控制。
    </p>

    <article class="card tree-card">
      <div v-if="loading" class="loading">加载中…</div>
      <template v-else>
        <div v-if="tenantUnreachable && activeConsole === 'tenant'" class="page-error soft">
          组织控制台目录需要在某个组织上下文中读取（平台账号无组织上下文时不可用）。请在「我的组织」选择组织后于组织控制台查看，或在此查看平台控制台目录。
        </div>
        <TreeView
          v-else
          :nodes="activeMenus"
          id-key="key"
          label-key="title"
        >
          <template #label="{ node }">
            <span class="menu-node" :class="`type-${node.type}`">
              <span class="m-title">{{ node.title }}</span>
              <code v-if="node.path" class="m-path">{{ node.path }}</code>
              <span class="m-type" :class="`tt-${node.type}`">{{ typeLabel(node.type) }}</span>
              <code v-if="node.perm" class="perm-tag">{{ node.perm }}</code>
              <span v-else class="no-perm">仅登录</span>
              <code v-if="node.app" class="app-tag">{{ node.app }}</code>
            </span>
          </template>
          <template #empty>该控制台暂无菜单目录。</template>
        </TreeView>
      </template>
    </article>

    <!-- Registered apps & per-tenant enablement (governance reference) -->
    <article class="card apps-card">
      <div class="apps-head">
        <h3 class="card-title">可插拔应用 · 按组织启停</h3>
        <span class="apps-sub">应用启停（控制对应菜单可见）由各组织管理员在「组织控制台 → 应用管理」自治；平台侧此处仅供巡检。</span>
      </div>
      <div v-if="apps.length === 0" class="muted">尚无已注册的可插拔应用。</div>
      <div v-else class="app-grid">
        <div v-for="a in apps" :key="a.app" class="app-chip">
          <code class="app-code">{{ a.app }}</code>
          <span class="app-menus">{{ a.count }} 个菜单</span>
        </div>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PageHeader, TreeView } from "@/shell/components";
import { getPlatformMenuTree, getTenantMenuTree, type MenuTreeNode } from "../api/menus";

const platformMenus = ref<MenuTreeNode[]>([]);
const tenantMenus = ref<MenuTreeNode[]>([]);
const tenantUnreachable = ref(false);
const loading = ref(false);
const activeConsole = ref<"platform" | "tenant">("platform");

const activeMenus = computed(() => (activeConsole.value === "platform" ? platformMenus.value : tenantMenus.value));

// Aggregate pluggable apps (nodes with a non-empty `app`) across both consoles.
const apps = computed(() => {
  const counts: Record<string, number> = {};
  const tally = (nodes: MenuTreeNode[]) => {
    for (const n of nodes) {
      if (n.app) counts[n.app] = (counts[n.app] ?? 0) + 1;
      if (n.children?.length) tally(n.children);
    }
  };
  tally(platformMenus.value);
  tally(tenantMenus.value);
  return Object.entries(counts).map(([app, count]) => ({ app, count })).sort((a, b) => a.app.localeCompare(b.app));
});

onMounted(reload);

async function reload() {
  loading.value = true;
  tenantUnreachable.value = false;
  try {
    platformMenus.value = await getPlatformMenuTree().catch(() => []);
    try {
      tenantMenus.value = await getTenantMenuTree();
    } catch {
      tenantMenus.value = [];
      tenantUnreachable.value = true;
    }
  } finally { loading.value = false; }
}

function countMenus(nodes: MenuTreeNode[]): number {
  let n = 0;
  const walk = (list: MenuTreeNode[]) => { for (const x of list) { n++; if (x.children?.length) walk(x.children); } };
  walk(nodes);
  return n;
}
function typeLabel(t: string): string {
  return ({ dir: "目录", menu: "菜单", button: "按钮" } as Record<string, string>)[t] ?? t;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.console-tabs { display: flex; gap: 8px; }
.tab {
  display: inline-flex; align-items: center; gap: 8px;
  padding: 8px 16px; font-size: 13px; font-weight: 600;
  border: 1px solid var(--border); border-radius: 8px;
  background: var(--surface); color: var(--text-2); cursor: pointer;
}
.tab:hover { background: var(--surface-2); }
.tab.on { background: var(--primary-soft); color: var(--primary); border-color: var(--primary); }
.tab-count { font-size: 11px; font-weight: 700; background: var(--surface-2); color: var(--text-3); border-radius: 999px; padding: 1px 7px; }
.tab.on .tab-count { background: var(--surface); color: var(--primary); }

.hint { font-size: 12.5px; color: var(--text-3); line-height: 1.6; margin: 0; }
.hint code { font-family: var(--ff-mono); font-size: 11.5px; background: var(--surface-2); padding: 1px 5px; border-radius: 3px; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; box-shadow: var(--sh-1); }
.tree-card { padding: 12px 14px; }
.loading { font-size: 13px; color: var(--text-3); padding: 18px; }
.page-error.soft { background: var(--warning-soft); color: var(--warning); font-size: 12.5px; padding: 12px 14px; border-radius: 8px; line-height: 1.5; }

.menu-node { display: inline-flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.m-title { color: var(--text); }
.menu-node.type-dir .m-title { font-weight: 700; }
.m-path { font-family: var(--ff-mono); font-size: 10.5px; color: var(--text-3); }
.m-type { font-size: 10px; font-weight: 700; border-radius: 3px; padding: 1px 5px; }
.tt-dir { background: var(--purple-soft); color: var(--purple); }
.tt-menu { background: var(--info-soft); color: var(--info); }
.tt-button { background: var(--surface-2); color: var(--text-3); }
.perm-tag { font-family: var(--ff-mono); font-size: 10.5px; color: var(--primary); background: var(--primary-soft); padding: 1px 5px; border-radius: 3px; }
.no-perm { font-size: 10.5px; color: var(--text-4); }
.app-tag { font-family: var(--ff-mono); font-size: 10.5px; color: var(--success); background: var(--success-soft); padding: 1px 5px; border-radius: 3px; }

.apps-card { padding: 16px 18px; }
.apps-head { display: flex; flex-direction: column; gap: 3px; margin-bottom: 12px; }
.card-title { font-size: 14px; font-weight: 600; margin: 0; }
.apps-sub { font-size: 12px; color: var(--text-3); line-height: 1.5; }
.muted { color: var(--text-4); font-size: 12.5px; }
.app-grid { display: flex; flex-wrap: wrap; gap: 8px; }
.app-chip { display: inline-flex; align-items: center; gap: 8px; border: 1px solid var(--border); border-radius: 8px; padding: 6px 10px; background: var(--surface); }
.app-code { font-family: var(--ff-mono); font-size: 12px; color: var(--text); }
.app-menus { font-size: 11px; color: var(--text-3); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-ghost { background: transparent; }
</style>
