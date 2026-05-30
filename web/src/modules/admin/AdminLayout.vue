<template>
  <div class="admin-shell">
    <aside class="admin-nav">
      <div class="nav-title">管理后台</div>
      <div class="nav-sub">{{ tenantName }}</div>

      <div class="nav-group">
        <div class="group-title">概览</div>
        <router-link to="/admin" class="nav-link" :class="{ active: $route.path === '/admin' }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="9"/><rect x="14" y="3" width="7" height="5"/><rect x="14" y="12" width="7" height="9"/><rect x="3" y="16" width="7" height="5"/></svg>
          仪表盘
        </router-link>
      </div>

      <div class="nav-group">
        <div class="group-title">租户</div>
        <router-link to="/admin/members" class="nav-link" :class="{ active: $route.path.startsWith('/admin/members') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>
          成员
        </router-link>
        <router-link to="/admin/roles" class="nav-link" :class="{ active: $route.path.startsWith('/admin/roles') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
          角色权限
        </router-link>
        <router-link to="/admin/departments" class="nav-link" :class="{ active: $route.path.startsWith('/admin/departments') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="7"/><rect x="14" y="3" width="7" height="7"/><rect x="3" y="14" width="7" height="7"/></svg>
          部门
        </router-link>
        <router-link to="/admin/settings" class="nav-link" :class="{ active: $route.path.startsWith('/admin/settings') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33"/></svg>
          租户设置
        </router-link>
      </div>

      <div class="nav-group">
        <div class="group-title">系统</div>
        <router-link to="/admin/audit" class="nav-link" :class="{ active: $route.path.startsWith('/admin/audit') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="9" y1="13" x2="15" y2="13"/></svg>
          审计日志
        </router-link>
        <router-link to="/admin/dict" class="nav-link" :class="{ active: $route.path.startsWith('/admin/dict') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20"/><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z"/></svg>
          字典管理
        </router-link>
      </div>

      <div class="nav-group" v-if="admin.is_platform_admin">
        <div class="group-title">平台</div>
        <router-link to="/admin/platform/tenants" class="nav-link" :class="{ active: $route.path.startsWith('/admin/platform') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          租户管理
          <span class="badge-platform">平台</span>
        </router-link>
      </div>

      <div class="nav-group">
        <div class="group-title">个人</div>
        <router-link to="/me/settings" class="nav-link" :class="{ active: $route.path.startsWith('/me/settings') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          个人设置
        </router-link>
      </div>
    </aside>

    <div class="admin-main">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import { getMyAdminFlags, type MeAdmin } from "./api/admin";

const auth = useAuthStore();
const admin = ref<MeAdmin>({ is_tenant_admin: false, is_platform_admin: false });

const tenantName = computed(() => auth.tenant?.name ?? "—");

onMounted(async () => {
  admin.value = await getMyAdminFlags();
});
</script>

<style scoped>
.admin-shell {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 0;
  min-height: calc(100vh - 56px);
  margin: -22px -28px -40px -28px;
}

.admin-nav {
  background: var(--surface);
  border-right: 1px solid var(--border);
  padding: 22px 14px;
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.nav-title { font-size: 16px; font-weight: 700; color: var(--text); padding: 0 8px; }
.nav-sub { font-size: 12px; color: var(--text-3); padding: 0 8px; margin-top: -8px; }
.nav-group { display: flex; flex-direction: column; gap: 2px; margin-top: 6px; }
.group-title {
  font-size: 10.5px; font-weight: 600;
  color: var(--text-4); text-transform: uppercase;
  letter-spacing: .8px;
  padding: 4px 8px;
}
.nav-link {
  display: flex; align-items: center; gap: 9px;
  padding: 7px 10px;
  font-size: 13px;
  color: var(--text-2);
  border-radius: 7px;
  text-decoration: none;
  cursor: pointer;
  transition: background .12s, color .12s;
}
.nav-link:hover { background: var(--bg); color: var(--text); text-decoration: none; }
.nav-link.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.nav-link.active svg { color: var(--primary); }
.badge-platform {
  margin-left: auto;
  font-size: 9.5px; font-weight: 700;
  padding: 1px 5px;
  background: var(--purple-soft); color: var(--purple);
  border-radius: 3px;
  letter-spacing: .3px;
}

.admin-main { padding: 28px 32px; background: var(--bg); min-width: 0; }
</style>
