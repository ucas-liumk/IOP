<template>
  <header class="nav">
    <div class="nav-brand">
      <div class="nav-brand-mark">I</div>
      <div>
        <div class="brand-name">IOP</div>
        <div class="brand-sub">企业协同平台</div>
      </div>
    </div>

    <div class="nav-tabs">
      <router-link to="/" class="nav-tab" :class="{ active: $route.path === '/' }">
        <span class="nav-tab-icon">▦</span>工作台
      </router-link>
      <router-link to="/okr/plans" class="nav-tab" :class="{ active: $route.path.startsWith('/okr/plans') }">
        <span class="nav-tab-icon">◎</span>计划
      </router-link>
      <router-link to="/okr/reports" class="nav-tab" :class="{ active: $route.path.startsWith('/okr/reports') }">
        <span class="nav-tab-icon">✎</span>报告
      </router-link>
      <router-link to="/okr/rollup" class="nav-tab" :class="{ active: $route.path.startsWith('/okr/rollup') }">
        <span class="nav-tab-icon">▲</span>汇总
      </router-link>
    </div>

    <div class="nav-spacer"></div>

    <div class="nav-tools">
      <div class="tenant-pill">
        <span class="tenant-pill-dot"></span>
        <TenantSwitcher />
      </div>
      <div class="nav-user" v-if="auth.loggedIn">
        <router-link to="/me/profile" class="user-link" title="个人中心">
          <div class="avatar">{{ initials }}</div>
          <span class="user-email">{{ displayName }}</span>
        </router-link>
        <button class="btn-ghost-icon" @click="logout" title="退出">⏻</button>
      </div>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/shell/auth/auth.store";
import TenantSwitcher from "@/shell/tenant/TenantSwitcher.vue";

const auth = useAuthStore();
const router = useRouter();

const displayName = computed(() => {
  const u = auth.user;
  return u?.username || u?.email || u?.phone || "未登录";
});
const initials = computed(() => {
  return displayName.value.slice(0, 2).toUpperCase();
});

async function logout() {
  await auth.logout();
  router.push("/login");
}
</script>

<style scoped>
.brand-name { font-size: 16px; font-weight: 700; letter-spacing: -0.01em; line-height: 1.1; }
.brand-sub { font-size: 11px; color: var(--text-3); font-weight: 500; }

.nav-tab-icon {
  display: inline-block;
  font-size: 12px;
  opacity: 0.85;
}

.tenant-pill {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 4px 5px 12px;
  background: var(--surface-3);
  border-radius: 999px;
  border: 1px solid var(--border);
}
.tenant-pill-dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: var(--success);
  box-shadow: 0 0 0 3px rgba(26,169,113,0.18);
}

.avatar {
  width: 28px;
  height: 28px;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--primary) 0%, var(--purple) 100%);
  color: white;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  font-size: 11px;
  letter-spacing: -0.01em;
}
.user-email {
  font-size: 13px;
  color: var(--text-2);
  font-weight: 500;
}
.user-link {
  display: flex; align-items: center; gap: var(--sp-2);
  text-decoration: none;
  padding: 4px 8px;
  border-radius: var(--r-sm);
  transition: background 0.15s;
}
.user-link:hover {
  background: var(--surface-2);
  text-decoration: none;
}
.btn-ghost-icon {
  background: transparent;
  border: 0;
  font-size: 16px;
  color: var(--text-3);
  padding: 4px 8px;
  border-radius: var(--r-sm);
  transition: all 0.15s;
}
.btn-ghost-icon:hover {
  background: var(--danger-soft);
  color: var(--danger);
}
</style>
