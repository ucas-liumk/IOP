<template>
  <div class="me-shell">
    <aside class="me-nav">
      <div class="nav-title">个人中心</div>
      <div class="nav-sub">{{ greetingName }}</div>

      <div class="nav-group">
        <div class="group-title">账户</div>
        <router-link to="/me/profile" class="nav-link" :class="{ active: $route.path.startsWith('/me/profile') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M20 21v-2a4 4 0 0 0-4-4H8a4 4 0 0 0-4 4v2"/><circle cx="12" cy="7" r="4"/></svg>
          个人资料
        </router-link>
        <router-link to="/me/security" class="nav-link" :class="{ active: $route.path.startsWith('/me/security') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0 1 10 0v4"/></svg>
          安全设置
        </router-link>
        <router-link to="/me/sessions" class="nav-link" :class="{ active: $route.path.startsWith('/me/sessions') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="3" width="20" height="14" rx="2"/><line x1="8" y1="21" x2="16" y2="21"/><line x1="12" y1="17" x2="12" y2="21"/></svg>
          登录会话
        </router-link>
      </div>

      <div class="nav-group">
        <div class="group-title">工作区</div>
        <router-link to="/me/tenants" class="nav-link" :class="{ active: $route.path.startsWith('/me/tenants') }">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 9 12 2l9 7v11a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2z"/><polyline points="9 22 9 12 15 12 15 22"/></svg>
          我的工作区
        </router-link>
      </div>

      <div class="nav-foot">
        <router-link to="/" class="back-link">
          <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="19" y1="12" x2="5" y2="12"/><polyline points="12 19 5 12 12 5"/></svg>
          返回工作台
        </router-link>
      </div>
    </aside>

    <main class="me-main">
      <router-view />
    </main>
  </div>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";

const auth = useAuthStore();
const greetingName = computed(() => auth.user?.username || auth.user?.email || auth.user?.phone || "—");
</script>

<style scoped>
.me-shell {
  display: grid;
  grid-template-columns: 220px 1fr;
  gap: 0;
  min-height: calc(100vh - 56px);
  margin: -22px -28px -40px -28px;
}

.me-nav {
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

.nav-foot { margin-top: auto; padding-top: 10px; border-top: 1px solid var(--border); }
.back-link {
  display: flex; align-items: center; gap: 7px;
  padding: 7px 10px;
  font-size: 12.5px;
  color: var(--text-3);
  border-radius: 7px;
  text-decoration: none;
}
.back-link:hover { color: var(--primary); background: var(--primary-soft); }

.me-main { padding: 28px 32px; background: var(--bg); min-width: 0; }
</style>
