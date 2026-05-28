<template>
  <header class="navbar">
    <div class="brand">IOP</div>
    <nav class="links">
      <router-link to="/">工作台</router-link>
      <router-link to="/okr/plans">计划</router-link>
      <router-link to="/okr/reports">报告</router-link>
      <router-link to="/okr/rollup">汇总</router-link>
    </nav>
    <div class="right">
      <TenantSwitcher />
      <span class="user">{{ auth.user?.email ?? '' }}</span>
      <button v-if="auth.loggedIn" class="logout" @click="logout">退出</button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { useRouter } from "vue-router";
import { useAuthStore } from "@/shell/auth/auth.store";
import TenantSwitcher from "@/shell/tenant/TenantSwitcher.vue";

const auth = useAuthStore();
const router = useRouter();

async function logout() {
  await auth.logout();
  router.push("/login");
}
</script>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  gap: var(--space-6);
  padding: var(--space-3) var(--space-6);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  box-shadow: var(--shadow-sm);
}
.brand {
  font-weight: 600;
  font-size: 18px;
  color: var(--color-primary);
}
.links {
  display: flex;
  gap: var(--space-4);
  flex: 1;
}
.links a {
  color: var(--color-text);
}
.links a.router-link-active {
  color: var(--color-primary);
  font-weight: 600;
}
.right {
  display: flex;
  gap: var(--space-3);
  align-items: center;
  font-size: 13px;
  color: var(--color-text-muted);
}
.logout {
  padding: 2px 8px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}
</style>
