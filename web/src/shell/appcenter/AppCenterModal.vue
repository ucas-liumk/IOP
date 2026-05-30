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
          <div class="ac-sec-head">
            <span class="t">我的应用</span>
            <span class="hint">点击启动 · 拖拽可调整顺序 (mock)</span>
          </div>
          <div class="my-apps-strip">
            <button v-for="a in myApps" :key="a.code" class="my-tile" @click="a.path && $emit('navigate', a.path)">
              <div class="m-ico" :style="{ background: a.color }">
                <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" v-html="a.svg"></svg>
              </div>
              <div class="m-name">{{ a.name }}</div>
            </button>
          </div>

          <div v-for="(cat, ci) in categories" :key="cat.title" class="cat-block">
            <div class="cat-title">
              <span class="cat-dot" :style="{ background: cat.color }"></span>
              {{ cat.title }}
            </div>
            <div class="app-grid">
              <button
                v-for="a in filteredCat(ci)"
                :key="a.code"
                class="app-cell"
                :class="{ 'is-soon': a.status === 'soon' }"
                @click="install(a)"
              >
                <div class="c-ico" :style="{ background: a.status === 'soon' ? '' : cat.color }">
                  <svg width="22" height="22" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" v-html="a.svg"></svg>
                </div>
                <span v-if="installedSet.has(a.code)" class="c-action c-added">✓</span>
                <span v-else-if="a.status === 'online'" class="c-action c-add">+</span>
                <span v-if="a.status === 'soon'" class="c-tag soon">敬请期待</span>
                <span v-else-if="a.status === 'beta'" class="c-tag beta">内测</span>
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
import { computed, ref } from "vue";

defineProps<{ open: boolean }>();
const emit = defineEmits<{ (e: "close"): void; (e: "navigate", path: string): void }>();
void emit;

const q = ref("");

interface App {
  code: string; name: string; svg: string;
  status: "online" | "soon" | "beta";
  path?: string;
  color?: string;
}

const myApps: App[] = [
  { code: "okr", name: "OKR", color: "var(--cat-collab)", path: "/okr/plans", status: "online",
    svg: '<circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="4"/><circle cx="12" cy="12" r="1.2" fill="currentColor"/>' },
  { code: "admin", name: "管理", color: "var(--text-2)", path: "/admin", status: "online",
    svg: '<circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33"/>' },
];

const installedSet = computed(() => new Set(myApps.map((a) => a.code)));

const categories: { title: string; color: string; apps: App[] }[] = [
  {
    title: "协同办公", color: "var(--cat-collab)",
    apps: [
      { code: "approval", name: "审批流程", status: "beta",
        svg: '<polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>' },
      { code: "doc", name: "文档协作", status: "soon",
        svg: '<path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>' },
      { code: "meeting", name: "会议室预订", status: "soon",
        svg: '<rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/>' },
      { code: "wiki", name: "知识库", status: "soon",
        svg: '<path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/>' },
    ],
  },
  {
    title: "业务管理", color: "var(--cat-biz)",
    apps: [
      { code: "order", name: "订单管理", status: "soon",
        svg: '<circle cx="9" cy="21" r="1"/><circle cx="20" cy="21" r="1"/><path d="M1 1h4l2.7 13.4a2 2 0 0 0 2 1.6h9.7a2 2 0 0 0 2-1.6L23 6H6"/>' },
      { code: "inventory", name: "库存管理", status: "soon",
        svg: '<path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/>' },
      { code: "crm", name: "客户管理 CRM", status: "soon",
        svg: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>' },
      { code: "contract", name: "智能合同", status: "soon",
        svg: '<polygon points="14 2 18 6 7 17 3 17 3 13 14 2"/>' },
    ],
  },
  {
    title: "财务税务", color: "var(--cat-finance)",
    apps: [
      { code: "finance", name: "财务管理", status: "soon",
        svg: '<line x1="12" y1="1" x2="12" y2="23"/><path d="M17 5H9.5a3.5 3.5 0 0 0 0 7h5a3.5 3.5 0 0 1 0 7H6"/>' },
      { code: "tax", name: "税务申报", status: "soon",
        svg: '<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 9h6v6H9z"/>' },
    ],
  },
  {
    title: "人力资源", color: "var(--cat-hr)",
    apps: [
      { code: "hr", name: "HR 一体化", status: "soon",
        svg: '<path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/>' },
      { code: "attendance", name: "考勤打卡", status: "soon",
        svg: '<circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>' },
    ],
  },
  {
    title: "数据分析", color: "var(--cat-data)",
    apps: [
      { code: "report", name: "报表中心", status: "soon",
        svg: '<line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/>' },
      { code: "bi", name: "BI 看板", status: "soon",
        svg: '<rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18M9 21V9"/>' },
    ],
  },
];

function filteredCat(idx: number) {
  const apps = categories[idx].apps;
  if (!q.value) return apps;
  return apps.filter((a) => a.name.includes(q.value));
}

function install(a: App) {
  if (a.status === "soon") {
    alert(`${a.name} 敬请期待 · 加入候补名单时会通知你`);
    return;
  }
  alert(`已添加 "${a.name}" 到我的应用 (mock)`);
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
  flex-shrink: 0;
}
.ac-close:hover { background: var(--bg); color: var(--text); }

.ac-body {
  padding: 20px 22px 24px;
  overflow-y: auto;
}

.ac-sec-head {
  display: flex; align-items: baseline; gap: 10px;
  margin: 4px 0 12px;
}
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
.my-tile {
  position: relative;
  display: flex; flex-direction: column; align-items: center;
  gap: 7px;
  padding: 10px 4px 8px;
  border-radius: 10px;
  background: transparent;
  border: 0;
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
.app-grid {
  display: grid;
  grid-template-columns: repeat(6, 1fr);
  gap: 6px;
}
.app-cell {
  position: relative;
  display: flex; flex-direction: column; align-items: center;
  gap: 8px;
  padding: 14px 6px 12px;
  border-radius: 12px;
  background: transparent; border: 0;
  cursor: pointer;
  transition: background .12s, transform .12s;
}
.app-cell:hover { background: var(--surface-2); transform: translateY(-1px); }
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
.app-cell .c-tag.beta { background: var(--warning); color: #fff; }
</style>
