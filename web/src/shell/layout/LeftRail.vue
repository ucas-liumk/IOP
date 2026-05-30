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

    <!-- Installed apps from /me/apps — driven by tenant_app + Module Registry -->
    <router-link
      v-for="app in myApps"
      :key="app.code"
      :to="appHomeRoute(app.code)"
      class="rail-item installed"
      :class="{ active: $route.path.startsWith(appHomeRoute(app.code).split('/').slice(0, 2).join('/')) }"
      :title="app.name"
    >
      <div class="rail-ico" :style="{ background: app.color }">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor">
          <path :d="app.icon"/>
        </svg>
      </div>
      <div class="rail-label">{{ shortName(app.name) }}</div>
      <span v-if="app.code === 'okr' && planCount > 0" class="rail-badge">{{ planCount > 99 ? '99+' : planCount }}</span>
    </router-link>

    <router-link
      v-if="admin.is_tenant_admin || admin.has_platform_access"
      :to="admin.has_platform_access ? '/platform' : '/admin'"
      class="rail-item admin-item"
      :class="{ active: $route.path.startsWith('/admin') || $route.path.startsWith('/platform') }"
      :title="admin.has_platform_access ? '平台控制台' : '组织控制台'"
    >
      <div class="rail-ico">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/>
        </svg>
      </div>
      <div class="rail-label">{{ admin.has_platform_access ? '平台' : '管理' }}</div>
      <span v-if="admin.has_platform_access" class="platform-tag">P</span>
    </router-link>

    <div class="rail-spacer"></div>

    <button class="rail-add" title="添加应用" @click="appCenterOpen = true">
      <div class="ico-plus">
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
          <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
        </svg>
      </div>
      <div class="rail-label">添加</div>
    </button>

    <AppCenterModal
      :open="appCenterOpen"
      @close="appCenterOpen = false; reloadApps()"
      @navigate="goTo"
    />
  </aside>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { listPlans } from "@/modules/okr/api/okr";
import { getMyAdminFlags, type MeAdmin } from "@/modules/admin/api/admin";
import { getMyApps, appHomeRoute, type Manifest } from "@/shell/appcenter/appstore";
import AppCenterModal from "@/shell/appcenter/AppCenterModal.vue";

const router = useRouter();
const planCount = ref(0);
const admin = ref<MeAdmin>({ is_tenant_admin: false, is_platform_admin: false, has_platform_access: false });
const appCenterOpen = ref(false);
const myApps = ref<Manifest[]>([]);

onMounted(async () => {
  await reloadApps();
  try { planCount.value = (await listPlans("week")).length; } catch {}
  try { admin.value = await getMyAdminFlags(); } catch {}
});

async function reloadApps() {
  try { myApps.value = await getMyApps(); } catch { myApps.value = []; }
}

function goTo(path: string) {
  appCenterOpen.value = false;
  router.push(path);
  reloadApps();
}

function shortName(name: string): string {
  const m = name.match(/^[A-Za-z0-9]+/);
  if (m && m[0].length >= 2) return m[0];
  return name.length > 4 ? name.slice(0, 4) : name;
}
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
  background: var(--primary) !important;
  color: #fff;
  box-shadow: 0 4px 10px rgba(30,95,217,.32);
}
.rail-item.active .rail-label { font-weight: 600; }

.rail-item.installed .rail-ico { color: #fff; }
.rail-item.installed:hover .rail-ico { transform: translateY(-1px); box-shadow: 0 4px 10px rgba(30,95,217,.25); }

.rail-item.admin-item .rail-ico {
  background: linear-gradient(135deg, var(--text-2), #0d1b2e);
  color: #fff;
}
.rail-item.admin-item.active .rail-ico {
  background: linear-gradient(135deg, var(--primary), #0d1b2e) !important;
  box-shadow: 0 4px 10px rgba(13,27,46,.25);
}

.platform-tag {
  position: absolute;
  top: 4px; right: 8px;
  font-size: 8.5px; font-weight: 800;
  padding: 1px 4px;
  background: var(--purple);
  color: white;
  border-radius: 3px;
}

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
