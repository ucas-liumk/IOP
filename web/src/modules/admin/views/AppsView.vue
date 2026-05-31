<template>
  <section class="admin-page">
    <PageHeader :title="pageTitle" :sub="pageSub">
      <template #actions>
        <div class="head-actions">
          <select v-if="isPlatform" class="tenant-select" v-model="selectedTenantId" @change="refreshCatalog">
            <option value="">选择租户</option>
            <option v-for="t in tenants" :key="t.id" :value="t.id">
              {{ t.name }}
            </option>
          </select>
          <button class="btn btn-ghost" @click="refresh">刷新</button>
        </div>
      </template>
    </PageHeader>

    <div v-if="pageError" class="page-error">
      {{ pageError }}
      <button class="page-error-close" @click="pageError = ''">×</button>
    </div>

    <p class="hint">
      启用后，应用会出现在成员的「应用市场」中，成员可从左下角「+ 添加」将其固定到左侧菜单栏。
      停用会移除本租户对该应用的访问（不会删除数据）。
    </p>

    <div v-if="isPlatform && !selectedTenantId && !loading" class="loading">请选择租户后管理应用</div>
    <div v-else-if="loading" class="loading">加载中…</div>

    <div v-for="group in categoryGroups" :key="group.category" class="cat-block">
      <div class="cat-title">
        <span class="cat-dot" :style="{ background: group.color }"></span>
        {{ group.category }}
        <span class="cat-count">{{ group.apps.length }}</span>
      </div>
      <div class="app-grid">
        <article v-for="app in group.apps" :key="app.code" class="app-card" :class="{ on: app.installed }">
          <div class="card-top">
            <div class="app-icon" :style="{ background: app.color }">
              <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor"><path :d="app.icon" /></svg>
            </div>
            <label class="switch" :class="{ on: app.installed, busy: busyCode === app.code }">
              <input type="checkbox" :checked="app.installed" :disabled="busyCode === app.code" @change="toggle(app)" />
              <span class="switch-track"><span class="switch-thumb"></span></span>
            </label>
          </div>
          <div class="app-name">{{ app.name }} <span class="ver">v{{ app.version }}</span></div>
          <div class="app-desc">{{ app.description || '—' }}</div>
          <label class="category-edit">
            <span>展示分类</span>
            <select :value="app.category" :disabled="busyCategoryCode === app.code" @change="changeCategory(app, $event)">
              <option v-for="cat in categories" :key="cat.code" :value="cat.name">{{ cat.name }}</option>
            </select>
          </label>
          <div class="app-meta">
            <span class="chip" :title="`${app.permissions?.length || 0} 个权限点`">
              🔑 {{ app.permissions?.length || 0 }}
            </span>
            <span class="chip" :title="`${app.events?.length || 0} 个事件`">
              ⚡ {{ app.events?.length || 0 }}
            </span>
            <code class="code-chip">{{ app.code }}</code>
          </div>
        </article>
      </div>
    </div>

    <EmptyState v-if="!loading && catalog.length === 0" title="尚无已注册应用" sub="开发者通过 Module 契约注册应用后将在此显示" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PageHeader, EmptyState } from "@/shell/components";
import {
  getAppCategories,
  getCatalog,
  getPlatformAppCategories,
  getPlatformCatalog,
  installApp,
  installPlatformApp,
  uninstallApp,
  uninstallPlatformApp,
  updateAppCategory,
  updatePlatformAppCategory,
  type AppCategory,
  type CatalogEntry,
} from "@/shell/appcenter/appstore";
import { listAllTenants, type PlatformTenant } from "@/modules/admin/api/admin";
import { useConfirm } from "@/shell/confirm";

const { confirm } = useConfirm();

const props = withDefaults(defineProps<{ scope?: "tenant" | "platform" }>(), {
  scope: "tenant",
});

const catalog = ref<CatalogEntry[]>([]);
const categories = ref<AppCategory[]>([]);
const tenants = ref<PlatformTenant[]>([]);
const selectedTenantId = ref("");
const loading = ref(true);
const busyCode = ref("");
const busyCategoryCode = ref("");
const pageError = ref("");

const isPlatform = computed(() => props.scope === "platform");
const installedCount = computed(() => catalog.value.filter((a) => a.installed).length);
const selectedTenant = computed(() => tenants.value.find((t) => t.id === selectedTenantId.value) ?? null);
const pageTitle = computed(() => (isPlatform.value ? "平台应用管理" : "应用管理"));
const pageSub = computed(() => {
  if (!isPlatform.value) return `平台共 ${catalog.value.length} 个应用 · 已为本租户启用 ${installedCount.value} 个`;
  const tenant = selectedTenant.value?.name ?? "未选择租户";
  return `${tenant} · 平台共 ${catalog.value.length} 个应用 · 已为该租户启用 ${installedCount.value} 个`;
});

const categoryRank = computed(() => new Map(categories.value.map((c) => [c.name, c.order])));
const categoryColor = computed(() => new Map(categories.value.map((c) => [c.name, c.color])));
const categoryGroups = computed(() => {
  const groups: Record<string, CatalogEntry[]> = {};
  for (const a of catalog.value) {
    const cat = a.category || "其他";
    (groups[cat] ||= []).push(a);
  }
  return Object.entries(groups)
    .sort(([a], [b]) => (categoryRank.value.get(a) ?? 999) - (categoryRank.value.get(b) ?? 999) || a.localeCompare(b))
    .map(([category, apps]) => ({
      category,
      apps,
      color: categoryColor.value.get(category) || apps[0]?.color || "var(--text-4)",
    }));
});

async function refresh() {
  loading.value = true;
  pageError.value = "";
  try {
    if (isPlatform.value) {
      await loadTenants();
      await refreshCatalog();
    } else {
      const [apps, cats] = await Promise.all([getCatalog(), getAppCategories()]);
      catalog.value = apps;
      categories.value = cats;
    }
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "加载失败";
  } finally {
    loading.value = false;
  }
}
onMounted(refresh);

async function loadTenants() {
  tenants.value = await listAllTenants();
  if (selectedTenantId.value && tenants.value.some((t) => t.id === selectedTenantId.value)) return;
  selectedTenantId.value = tenants.value.find((t) => t.status === "active")?.id ?? tenants.value[0]?.id ?? "";
}

async function refreshCatalog() {
  if (isPlatform.value && !selectedTenantId.value) {
    catalog.value = [];
    categories.value = await getPlatformAppCategories();
    return;
  }
  const [apps, cats] = isPlatform.value
    ? await Promise.all([getPlatformCatalog(selectedTenantId.value), getPlatformAppCategories()])
    : await Promise.all([getCatalog(), getAppCategories()]);
  catalog.value = apps;
  categories.value = cats;
}

async function toggle(app: CatalogEntry) {
  busyCode.value = app.code;
  pageError.value = "";
  const turningOn = !app.installed;
  try {
    if (turningOn) {
      if (isPlatform.value) await installPlatformApp(selectedTenantId.value, app.code);
      else await installApp(app.code);
    }
    else {
      if (!(await confirm({ title: "确认", message: `确认停用「${app.name}」？成员将无法再访问该应用。`, danger: true }))) { busyCode.value = ""; return; }
      if (isPlatform.value) await uninstallPlatformApp(selectedTenantId.value, app.code);
      else await uninstallApp(app.code);
    }
    app.installed = turningOn;
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "操作失败";
  } finally {
    busyCode.value = "";
  }
}

async function changeCategory(app: CatalogEntry, event: Event) {
  const next = (event.target as HTMLSelectElement).value;
  const prev = app.category;
  app.category = next;
  busyCategoryCode.value = app.code;
  pageError.value = "";
  try {
    const updated = isPlatform.value
      ? await updatePlatformAppCategory(selectedTenantId.value, app.code, next)
      : await updateAppCategory(app.code, next);
    app.category = updated.category || next;
  } catch (e: any) {
    app.category = prev;
    pageError.value = e.response?.data?.error?.message ?? "分类保存失败";
  } finally {
    busyCategoryCode.value = "";
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.head-actions { display: flex; align-items: center; gap: 10px; }
.tenant-select {
  min-width: 220px;
  height: 34px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-2);
  padding: 0 10px;
  font: inherit;
}
.hint { font-size: 12.5px; color: var(--text-3); line-height: 1.6; margin: 0; }
.loading { color: var(--text-3); font-size: 13px; padding: 20px 0; }
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close { border: 0; background: transparent; color: inherit; font-size: 18px; line-height: 1; cursor: pointer; }

.cat-block { display: flex; flex-direction: column; gap: 10px; }
.cat-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 600; color: var(--text-2);
}
.cat-dot { width: 8px; height: 8px; border-radius: 50%; }
.cat-count {
  font-size: 11px; color: var(--text-3);
  background: var(--surface-2); padding: 0 7px; border-radius: 999px;
}

.app-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.app-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
  display: flex; flex-direction: column; gap: 8px;
  transition: border-color .15s, box-shadow .15s;
}
.app-card.on { border-color: var(--primary); box-shadow: 0 0 0 2px var(--primary-soft); }
.card-top { display: flex; align-items: flex-start; justify-content: space-between; }
.app-icon {
  width: 40px; height: 40px; border-radius: 11px;
  display: grid; place-items: center; color: #fff;
  box-shadow: 0 3px 8px rgba(13,27,46,.14);
}
.app-name { font-size: 14px; font-weight: 600; color: var(--text); }
.app-name .ver { font-size: 11px; color: var(--text-3); font-family: var(--ff-mono); font-weight: 400; margin-left: 4px; }
.app-desc { font-size: 12px; color: var(--text-3); line-height: 1.5; min-height: 30px; }
.category-edit {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  font-size: 12px;
  color: var(--text-3);
}
.category-edit select {
  min-width: 108px;
  height: 30px;
  border: 1px solid var(--border);
  border-radius: 8px;
  background: var(--surface);
  color: var(--text-2);
  padding: 0 8px;
  font: inherit;
}
.category-edit select:disabled { opacity: .55; }
.app-meta { display: flex; align-items: center; gap: 8px; }
.chip { font-size: 11.5px; color: var(--text-2); }
.code-chip {
  margin-left: auto;
  font-size: 11px; font-family: var(--ff-mono);
  background: var(--surface-2); color: var(--text-3);
  padding: 1px 7px; border-radius: 4px;
}

/* toggle switch */
.switch { position: relative; display: inline-block; cursor: pointer; }
.switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.switch-track {
  display: block; width: 38px; height: 22px; border-radius: 999px;
  background: var(--border); transition: background .2s;
}
.switch-thumb {
  position: absolute; top: 2px; left: 2px;
  width: 18px; height: 18px; border-radius: 50%;
  background: #fff; box-shadow: 0 1px 3px rgba(13,27,46,.25);
  transition: transform .2s;
}
.switch.on .switch-track { background: var(--success); }
.switch.on .switch-thumb { transform: translateX(16px); }
.switch.busy { opacity: .5; pointer-events: none; }
</style>
