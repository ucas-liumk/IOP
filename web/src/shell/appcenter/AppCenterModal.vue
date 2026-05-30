<template>
  <Teleport to="body">
    <div v-if="open" class="ac-overlay" @click.self="$emit('close')">
      <div class="ac-modal" @click.stop>
        <header class="ac-head">
          <div class="ac-h-title">
            <div class="h-ico">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/><rect x="14" y="14" width="7" height="7"/></svg>
            </div>
            应用中心
          </div>
          <div class="ac-search">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/></svg>
            <input v-model="q" placeholder="搜索应用..." />
          </div>
          <button class="ac-close" @click="$emit('close')">×</button>
        </header>

        <div class="ac-body">
          <!-- 我的应用 -->
          <div class="ac-sec-head">
            <span class="t">我的应用</span>
            <span class="hint">{{ myApps.length }} 个已安装 · 点击进入</span>
          </div>
          <div v-if="myApps.length > 0" class="my-apps-strip">
            <button v-for="a in myApps" :key="a.code" class="my-tile" @click="$emit('navigate', appHomeRoute(a.code))">
              <div class="m-ico" :style="{ background: a.color }">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="currentColor"><path :d="a.icon"/></svg>
              </div>
              <div class="m-name">{{ a.name }}</div>
            </button>
          </div>
          <div v-else class="empty-row">尚无安装应用 · 在下方应用市场添加</div>

          <!-- 应用市场 -->
          <div v-for="(apps, cat) in catalogByCategory" :key="cat" class="cat-block">
            <div class="cat-title">
              <span class="cat-dot" :style="{ background: apps[0]?.color || 'var(--text-4)' }"></span>
              {{ cat }}
              <span class="cat-count">{{ apps.length }}</span>
            </div>
            <div class="app-grid">
              <button
                v-for="a in apps"
                :key="a.code"
                class="app-cell"
                :class="{ 'is-installed': a.installed }"
                @click="toggle(a)"
              >
                <div class="c-ico" :style="{ background: a.color }">
                  <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor"><path :d="a.icon"/></svg>
                </div>
                <span v-if="a.installed" class="c-action c-added">✓</span>
                <span v-else class="c-action c-add">+</span>
                <div class="c-name">{{ a.name }}</div>
                <div class="c-version">v{{ a.version }}</div>
              </button>
            </div>
          </div>

          <div v-if="comingSoon.length > 0" class="cat-block">
            <div class="cat-title">
              <span class="cat-dot" style="background: var(--text-4);"></span>
              敬请期待
              <span class="cat-count">{{ comingSoon.length }}</span>
            </div>
            <div class="app-grid">
              <button v-for="a in comingSoon" :key="a.code" class="app-cell is-soon" disabled>
                <div class="c-ico"><svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="9"/><line x1="12" y1="7" x2="12" y2="13"/><circle cx="12" cy="17" r=".7" fill="currentColor"/></svg></div>
                <span class="c-tag soon">敬请期待</span>
                <div class="c-name">{{ a.name }}</div>
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
import { getCatalog, installApp, uninstallApp, appHomeRoute, type CatalogEntry, type Manifest } from "./appstore";

const props = defineProps<{ open: boolean }>();
defineEmits<{ (e: "close"): void; (e: "navigate", path: string): void }>();

const q = ref("");
const catalog = ref<CatalogEntry[]>([]);
const loading = ref(false);

// Future-app placeholders to make the marketplace feel populated.
const comingSoon = ref<Manifest[]>([
  { code: "approval", name: "审批流程", description: "通用审批引擎", icon: "M9 11l3 3L22 4M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11", color: "var(--cat-collab)", category: "协同办公", version: "0.9.0", permissions: [], events: [] },
  { code: "crm", name: "客户管理 CRM", description: "销售线索到合同", icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 7a4 4 0 1 1 0 0", color: "var(--cat-biz)", category: "业务管理", version: "0.5.0", permissions: [], events: [] },
  { code: "order", name: "订单管理", description: "销售订单全生命周期", icon: "M9 21h9.5l1.5-13H4l1.5 13z", color: "var(--cat-biz)", category: "业务管理", version: "0.5.0", permissions: [], events: [] },
  { code: "finance", name: "财务管理", description: "收支与对账", icon: "M12 1v23M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6", color: "var(--cat-finance)", category: "财务税务", version: "0.5.0", permissions: [], events: [] },
  { code: "hr", name: "HR 一体化", description: "人事考勤薪酬", icon: "M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2M9 7a4 4 0 1 1 0 0", color: "var(--cat-hr)", category: "人力资源", version: "0.5.0", permissions: [], events: [] },
  { code: "report", name: "报表中心", description: "跨应用数据看板", icon: "M18 20V10M12 20V4M6 20v-6", color: "var(--cat-data)", category: "数据分析", version: "0.5.0", permissions: [], events: [] },
]);

const myApps = computed(() => catalog.value.filter((a) => a.installed));
const filtered = computed(() => {
  if (!q.value) return catalog.value;
  return catalog.value.filter((a) =>
    a.name.includes(q.value) || a.code.includes(q.value.toLowerCase())
  );
});
const catalogByCategory = computed(() => {
  const out: Record<string, CatalogEntry[]> = {};
  for (const a of filtered.value) {
    (out[a.category] ??= []).push(a);
  }
  return out;
});

onMounted(reload);
watch(() => props.open, (v) => { if (v) reload(); });

async function reload() {
  loading.value = true;
  try { catalog.value = await getCatalog(); }
  catch { catalog.value = []; }
  finally { loading.value = false; }
}

async function toggle(a: CatalogEntry) {
  try {
    if (a.installed) {
      if (!confirm(`从工作台移除 "${a.name}"？`)) return;
      await uninstallApp(a.code);
    } else {
      await installApp(a.code);
    }
    await reload();
  } catch (e: any) {
    alert(e.response?.data?.error?.message ?? "操作失败");
  }
}
</script>

<style scoped>
.ac-overlay {
  position: fixed; inset: 0;
  background: rgba(13,27,46,.42);
  backdrop-filter: blur(3px);
  z-index: 500;
  display: flex; align-items: center; justify-content: center;
  padding: 32px;
}
.ac-modal {
  width: 100%;
  max-width: 880px;
  max-height: 86vh;
  background: var(--surface);
  border-radius: 18px;
  box-shadow: 0 24px 64px rgba(13,27,46,.28);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
.ac-head {
  display: flex; align-items: center;
  gap: 16px;
  padding: 18px 22px;
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}
.ac-h-title {
  font-size: 17px; font-weight: 700; color: var(--text);
  letter-spacing: -.2px;
  display: flex; align-items: center; gap: 9px;
}
.h-ico {
  width: 28px; height: 28px;
  border-radius: 8px;
  background: linear-gradient(135deg, var(--primary), #4a85ee);
  color: #fff;
  display: grid; place-items: center;
}
.ac-search {
  flex: 0 1 280px;
  margin-left: auto;
  height: 36px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 8px;
  display: flex; align-items: center;
  padding: 0 12px; gap: 8px;
  color: var(--text-3);
}
.ac-search:focus-within { border-color: var(--primary); background: var(--surface); box-shadow: 0 0 0 3px var(--primary-soft); }
.ac-search input { flex: 1; border: 0; outline: 0; background: transparent; font-family: inherit; font-size: 13px; color: var(--text); }
.ac-search input::placeholder { color: var(--text-4); }
.ac-close {
  width: 34px; height: 34px;
  border-radius: 8px;
  display: grid; place-items: center;
  background: transparent; border: 0;
  color: var(--text-3);
  font-size: 22px;
  cursor: pointer;
}
.ac-close:hover { background: var(--bg); color: var(--text); }

.ac-body { padding: 20px 22px 24px; overflow-y: auto; }

.ac-sec-head { display: flex; align-items: baseline; gap: 10px; margin: 4px 0 12px; }
.ac-sec-head .t { font-size: 14px; font-weight: 600; color: var(--text); }
.ac-sec-head .hint { font-size: 12px; color: var(--text-3); }

.my-apps-strip {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 8px;
  padding: 14px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 12px;
  margin-bottom: 26px;
}
.empty-row { padding: 26px 0; text-align: center; color: var(--text-3); font-size: 13px; }

.my-tile {
  position: relative;
  display: flex; flex-direction: column; align-items: center;
  gap: 7px;
  padding: 10px 4px 8px;
  border-radius: 10px;
  background: transparent; border: 0;
  cursor: pointer;
  transition: background .12s, box-shadow .12s;
}
.my-tile:hover { background: var(--surface); box-shadow: var(--sh-1); }
.my-tile .m-ico {
  width: 44px; height: 44px;
  border-radius: 12px;
  display: grid; place-items: center;
  color: #fff;
  box-shadow: 0 3px 8px rgba(13,27,46,.14);
}
.my-tile .m-name { font-size: 12px; font-weight: 500; color: var(--text); }

.cat-block { margin-bottom: 22px; }
.cat-title {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 600;
  color: var(--text-2);
  margin-bottom: 12px;
}
.cat-dot { width: 8px; height: 8px; border-radius: 2px; }
.cat-count {
  font-size: 11px;
  background: var(--surface-2);
  color: var(--text-3);
  padding: 1px 6px;
  border-radius: 999px;
  font-weight: 600;
}

.app-grid { display: grid; grid-template-columns: repeat(6, 1fr); gap: 6px; }
.app-cell {
  position: relative;
  display: flex; flex-direction: column; align-items: center;
  gap: 8px;
  padding: 14px 6px 10px;
  border-radius: 12px;
  background: transparent; border: 0;
  cursor: pointer;
  transition: background .12s, transform .12s;
}
.app-cell:hover:not(:disabled) { background: var(--surface-2); transform: translateY(-1px); }
.app-cell:disabled { cursor: not-allowed; }
.app-cell .c-ico {
  width: 46px; height: 46px;
  border-radius: 13px;
  display: grid; place-items: center;
  color: #fff;
  position: relative;
}
.app-cell.is-soon .c-ico { background: var(--bg-deep) !important; color: var(--text-4); }
.app-cell .c-name {
  font-size: 12px; font-weight: 500;
  color: var(--text);
  text-align: center;
  line-height: 1.25;
}
.app-cell .c-version {
  font-size: 10px; color: var(--text-4);
  font-family: var(--ff-mono);
}
.app-cell.is-soon .c-name { color: var(--text-3); }
.app-cell .c-action {
  position: absolute;
  top: 8px; right: 50%;
  transform: translateX(34px);
  width: 20px; height: 20px;
  border-radius: 50%;
  display: grid; place-items: center;
  font-size: 11px;
  border: 1.5px solid var(--surface);
  font-weight: 700;
}
.app-cell .c-add {
  background: var(--primary); color: #fff;
  opacity: 0; transition: opacity .12s;
}
.app-cell:hover .c-add { opacity: 1; }
.app-cell .c-added { background: var(--success); color: #fff; }
.app-cell .c-tag {
  position: absolute;
  top: 10px; left: 50%;
  transform: translateX(-38px);
  font-size: 8.5px; font-weight: 700;
  padding: 1px 4px;
  border-radius: 3px;
}
.app-cell .c-tag.soon { background: var(--bg-deep); color: var(--text-3); }
</style>
