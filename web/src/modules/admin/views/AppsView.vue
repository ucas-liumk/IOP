<template>
  <section class="admin-page">
    <PageHeader title="应用管理" :sub="`平台共 ${catalog.length} 个应用 · 已为本租户启用 ${installedCount} 个`">
      <template #actions>
        <button class="btn btn-ghost" @click="refresh">刷新</button>
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

    <div v-if="loading" class="loading">加载中…</div>

    <div v-for="(apps, cat) in byCategory" :key="cat" class="cat-block">
      <div class="cat-title">
        <span class="cat-dot" :style="{ background: apps[0]?.color || 'var(--text-4)' }"></span>
        {{ cat }}
        <span class="cat-count">{{ apps.length }}</span>
      </div>
      <div class="app-grid">
        <article v-for="app in apps" :key="app.code" class="app-card" :class="{ on: app.installed }">
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
import { getCatalog, installApp, uninstallApp, type CatalogEntry } from "@/shell/appcenter/appstore";
import { useConfirm } from "@/shell/confirm";

const { confirm } = useConfirm();

const catalog = ref<CatalogEntry[]>([]);
const loading = ref(true);
const busyCode = ref("");
const pageError = ref("");

const installedCount = computed(() => catalog.value.filter((a) => a.installed).length);

const byCategory = computed(() => {
  const groups: Record<string, CatalogEntry[]> = {};
  for (const a of catalog.value) {
    const cat = a.category || "其他";
    (groups[cat] ||= []).push(a);
  }
  return groups;
});

async function refresh() {
  loading.value = true;
  try {
    catalog.value = await getCatalog();
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "加载失败";
  } finally {
    loading.value = false;
  }
}
onMounted(refresh);

async function toggle(app: CatalogEntry) {
  busyCode.value = app.code;
  pageError.value = "";
  const turningOn = !app.installed;
  try {
    if (turningOn) await installApp(app.code);
    else {
      if (!(await confirm({ title: "确认", message: `确认停用「${app.name}」？成员将无法再访问该应用。`, danger: true }))) { busyCode.value = ""; return; }
      await uninstallApp(app.code);
    }
    app.installed = turningOn;
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "操作失败";
  } finally {
    busyCode.value = "";
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
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
