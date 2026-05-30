<template>
  <div class="admin-shell">
    <aside class="admin-nav">
      <div class="nav-title">平台控制台</div>
      <div class="nav-sub">全平台治理 · {{ auth.user?.username || '平台管理员' }}</div>

      <div class="nav-group">
        <div class="group-title">概览</div>
        <router-link to="/platform" class="nav-link" :class="{ active: $route.path === '/platform' }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="7" height="9"/><rect x="14" y="3" width="7" height="5"/><rect x="14" y="12" width="7" height="9"/><rect x="3" y="16" width="7" height="5"/></svg>
          平台概览
        </router-link>
      </div>

      <div class="nav-group">
        <div class="group-title">治理</div>
        <router-link to="/platform/organizations" class="nav-link" :class="{ active: $route.path.startsWith('/platform/organizations') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="6" width="18" height="15" rx="1"/><path d="M3 10h18"/><path d="M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2"/></svg>
          组织机构
        </router-link>
        <router-link to="/platform/users" class="nav-link" :class="{ active: $route.path.startsWith('/platform/users') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>
          全局用户
        </router-link>
        <router-link to="/platform/registrations" class="nav-link" :class="{ active: $route.path.startsWith('/platform/registrations') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
          注册申请
          <span v-if="pending > 0" class="badge-count">{{ pending > 99 ? '99+' : pending }}</span>
        </router-link>
        <router-link to="/platform/rbac" class="nav-link" :class="{ active: $route.path.startsWith('/platform/rbac') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          权限管理
        </router-link>
      </div>
    </aside>

    <div class="admin-main">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import { getPlatformStats } from "@/modules/admin/api/admin";

const auth = useAuthStore();
const pending = ref(0);

onMounted(async () => {
  try { pending.value = (await getPlatformStats()).pending_registrations; } catch {}
});
</script>

<style scoped>
/* Mirrors AdminLayout so the platform console reads as one with the rest of the
   app shell (top bar + left rail + this sub-sidebar). */
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
  letter-spacing: .8px; padding: 4px 8px;
}
.nav-link {
  display: flex; align-items: center; gap: 9px;
  padding: 7px 10px; font-size: 13px; color: var(--text-2);
  border-radius: 7px; text-decoration: none; cursor: pointer;
  transition: background .12s, color .12s;
}
.nav-link:hover { background: var(--bg); color: var(--text); text-decoration: none; }
.nav-link.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.nav-link.active svg { color: var(--primary); }
.badge-count {
  margin-left: auto; min-width: 18px; height: 18px; padding: 0 5px;
  font-size: 10.5px; font-weight: 700; background: var(--danger); color: #fff;
  border-radius: 9px; display: inline-grid; place-items: center;
}
.admin-main { padding: 28px 32px; background: var(--bg); min-width: 0; }
</style>
