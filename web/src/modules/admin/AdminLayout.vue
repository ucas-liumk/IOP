<template>
  <div class="admin-shell">
    <aside class="admin-nav">
      <div class="nav-title">组织控制台</div>
      <div class="nav-sub">{{ tenantName }}</div>

      <!-- Nav is driven by /me/menus (auth.store.menusTenant) — visibility follows
           the user's role policies + this tenant's enabled apps. -->
      <DynamicNav :menus="auth.menusTenant" />

      <div class="nav-group" v-if="auth.isPlatformAdmin">
        <div class="group-title">平台</div>
        <router-link to="/platform" class="nav-link platform-link">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>
          进入平台控制台
          <span class="badge-platform">平台</span>
        </router-link>
      </div>
    </aside>

    <div class="admin-main">
      <router-view />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import DynamicNav from "@/shell/layout/DynamicNav.vue";

const auth = useAuthStore();

const tenantName = computed(() => auth.tenant?.name ?? "—");

onMounted(async () => {
  // Ensure the tenant menu tree is loaded (a hard reload into /admin restores
  // tokens but the guard's restore() also fills this; load again if still empty).
  if (auth.menusTenant.length === 0) await auth.loadMenus();
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
