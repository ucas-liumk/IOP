<template>
  <aside class="rail">
    <router-link to="/" class="rail-item" :class="{ active: $route.path === '/' }" title="工作台">
      <div class="rail-ico">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <rect x="3" y="3" width="7" height="9"/><rect x="14" y="3" width="7" height="5"/>
          <rect x="14" y="12" width="7" height="9"/><rect x="3" y="16" width="7" height="5"/>
        </svg>
      </div>
      <div class="rail-label">工作台</div>
    </router-link>

    <div class="rail-sep"></div>

    <router-link to="/okr/plans" class="rail-item installed" :class="{ active: $route.path.startsWith('/okr') }" title="OKR 工作安排">
      <div class="rail-ico">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="4"/><circle cx="12" cy="12" r="1.2" fill="currentColor"/>
        </svg>
      </div>
      <div class="rail-label">OKR</div>
      <span v-if="planCount > 0" class="rail-badge">{{ planCount > 99 ? '99+' : planCount }}</span>
    </router-link>

    <button class="rail-item beta" title="审批流程（内测中）">
      <div class="rail-ico">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <polyline points="9 11 12 14 22 4"/><path d="M21 12v7a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h11"/>
        </svg>
      </div>
      <div class="rail-label">审批</div>
      <span class="beta-tag">内测</span>
    </button>

    <div class="rail-spacer"></div>

    <button class="rail-add" title="添加应用">
      <div class="ico-plus">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
      </div>
      <div class="rail-label">添加</div>
    </button>
  </aside>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { listPlans } from "@/modules/okr/api/okr";

const planCount = ref(0);
onMounted(async () => {
  try { planCount.value = (await listPlans("week")).length; } catch {}
});
</script>

<style scoped>
.rail {
  background: var(--surface);
  border-right: 1px solid var(--border);
  padding: 14px 10px;
  display: flex;
  flex-direction: column;
  gap: 4px;
  position: sticky;
  top: 56px;
  height: calc(100vh - 56px);
  align-self: start;
  z-index: 50;
}
.rail-item {
  position: relative;
  width: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  padding: 9px 4px 8px;
  border-radius: 10px;
  cursor: pointer;
  color: var(--text-2);
  background: transparent;
  border: 0;
  text-decoration: none;
  transition: background .15s, color .15s;
}
.rail-item:hover { background: var(--bg); color: var(--text); }
.rail-item .rail-ico {
  width: 28px; height: 28px;
  border-radius: 8px;
  display: grid; place-items: center;
  background: var(--bg-deep);
  color: var(--text-2);
  transition: background .15s, color .15s, box-shadow .2s, transform .12s;
}
.rail-item .rail-label {
  font-size: 11px;
  font-weight: 500;
  letter-spacing: .2px;
  line-height: 1.2;
}
.rail-item.active {
  background: var(--primary-soft);
  color: var(--primary);
}
.rail-item.active .rail-ico {
  background: var(--primary);
  color: #fff;
  box-shadow: 0 4px 10px rgba(30,95,217,.32);
}
.rail-item.active .rail-label { font-weight: 600; }

.rail-item.installed .rail-ico { background: var(--cat-collab); color: #fff; }
.rail-item.installed:hover .rail-ico { transform: translateY(-1px); box-shadow: 0 4px 10px rgba(30,95,217,.25); }

.rail-item .rail-badge {
  position: absolute;
  top: 4px; right: 10px;
  min-width: 16px; height: 16px;
  padding: 0 4px;
  background: var(--accent);
  border: 2px solid var(--surface);
  border-radius: 999px;
  color: #fff; font-size: 10px; font-weight: 700;
  display: grid; place-items: center; line-height: 1;
}

.rail-item.beta .rail-ico {
  background: var(--warning-soft);
  color: var(--warning);
  border: 1px dashed var(--warning);
}
.rail-item.beta .rail-label { color: var(--text-3); }
.rail-item.beta .beta-tag {
  position: absolute;
  top: 4px; right: 6px;
  font-size: 8.5px;
  padding: 1px 4px;
  background: var(--warning);
  color: #fff;
  border-radius: 3px;
  font-weight: 700;
  letter-spacing: .3px;
}

.rail-sep { height: 1px; background: var(--border); margin: 6px 8px; }
.rail-spacer { flex: 1; }

.rail-add {
  position: relative;
  width: 100%;
  padding: 10px 4px;
  border-radius: 10px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  color: var(--text-3);
  cursor: pointer;
  border: 1.5px dashed var(--border-strong);
  background: transparent;
  transition: background .15s, border-color .15s, color .15s;
}
.rail-add:hover {
  border-color: var(--primary);
  color: var(--primary);
  background: var(--primary-soft);
}
.rail-add .ico-plus {
  width: 28px; height: 28px;
  border-radius: 8px;
  display: grid; place-items: center;
  background: var(--surface);
  border: 1px solid var(--border);
}
.rail-add:hover .ico-plus {
  background: var(--primary); color: #fff; border-color: var(--primary);
}
.rail-add .rail-label { font-size: 10.5px; }
</style>
