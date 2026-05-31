<template>
  <div class="admin-shell">
    <aside class="admin-nav">
      <div class="nav-title">平台控制台</div>
      <div class="nav-sub">全平台治理 · {{ auth.user?.username || '平台管理员' }}</div>

      <!-- Nav is driven by /me/menus?console=platform (auth.store.menusPlatform)
           — visibility follows the user's platform-role policies. -->
      <DynamicNav :menus="auth.menusPlatform" />
    </aside>

    <div class="admin-main">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import DynamicNav from "@/shell/layout/DynamicNav.vue";

const auth = useAuthStore();

onMounted(async () => {
  if (auth.menusPlatform.length === 0) await auth.loadMenus();
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
.admin-main { padding: 28px 32px; background: var(--bg); min-width: 0; }
</style>
