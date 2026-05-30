<template>
  <header class="topbar">
    <div class="brand">
      <div class="brand-logo">I</div>
      <div class="brand-name">一站通办</div>
    </div>

    <div class="tb-search">
      <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <circle cx="11" cy="11" r="7"/><path d="m21 21-4.3-4.3"/>
      </svg>
      <input placeholder="搜索应用、文件、人员、待办…" />
      <kbd>⌘K</kbd>
    </div>

    <div class="tb-actions">
      <button class="icon-btn" title="帮助">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/><line x1="12" y1="17" x2="12.01" y2="17"/>
        </svg>
      </button>
      <button class="icon-btn" title="历史">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="10"/><polyline points="12 6 12 12 16 14"/>
        </svg>
      </button>
      <button class="icon-btn" title="通知">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M6 8a6 6 0 0 1 12 0c0 7 3 9 3 9H3s3-2 3-9"/><path d="M10.3 21a1.94 1.94 0 0 0 3.4 0"/>
        </svg>
        <span class="dot">3</span>
      </button>

      <div class="tenant-switcher" :class="{ open: tenantOpen }">
        <button class="tenant-chip" @click="tenantOpen = !tenantOpen">
          <div class="t-badge">{{ tenantInitial }}</div>
          <span>{{ auth.tenant?.name ?? '选择租户' }}</span>
          <svg class="chev" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
            <polyline points="6 9 12 15 18 9"/>
          </svg>
        </button>
        <div class="tenant-dropdown" v-show="tenantOpen || hoverDropdown" @mouseenter="hoverDropdown = true" @mouseleave="hoverDropdown = false">
          <div class="tenant-section-title">当前租户</div>
          <div class="tenant-row current" v-if="auth.tenant">
            <div class="t-logo" :style="{ background: 'linear-gradient(135deg,#1e5fd9,#4a85ee)' }">{{ tenantInitial }}</div>
            <div class="t-info">
              <div class="t-name">{{ auth.tenant.name }}<span class="tag">租户管理员</span></div>
              <div class="t-sub">{{ auth.tenant.slug }} · 已加入</div>
            </div>
            <svg class="check" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <polyline points="20 6 9 17 4 12"/>
            </svg>
          </div>
          <hr/>
          <div class="tenant-section-title">我的其他租户</div>
          <div v-for="t in otherTenants" :key="t.id" class="tenant-row" @click="switchTo(t.id)">
            <div class="t-logo" :style="{ background: gradientFor(t.name) }">{{ t.name[0] }}</div>
            <div class="t-info">
              <div class="t-name">{{ t.name }}</div>
              <div class="t-sub">{{ t.slug }}</div>
            </div>
          </div>
          <hr/>
          <div class="add-tenant">
            <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5" stroke-linecap="round" stroke-linejoin="round">
              <line x1="12" y1="5" x2="12" y2="19"/><line x1="5" y1="12" x2="19" y2="12"/>
            </svg>
            申请加入新租户
          </div>
        </div>
      </div>

      <button class="user-avatar" :title="auth.user?.email" @click="logout">{{ initials }}</button>
    </div>
  </header>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "@/shell/auth/auth.store";

const auth = useAuthStore();
const router = useRouter();
const tenantOpen = ref(false);
const hoverDropdown = ref(false);

const initials = computed(() => (auth.user?.email ?? "?").slice(0, 2).toUpperCase());
const tenantInitial = computed(() => auth.tenant?.name?.[0] ?? "?");
const otherTenants = computed(() => auth.tenants.filter((t) => t.id !== auth.tenant?.id));

function gradientFor(name: string) {
  const seed = name.split("").reduce((s, c) => s + c.charCodeAt(0), 0);
  const palette = [
    ["#1e5fd9","#4a85ee"],["#7c4ddb","#5a2db5"],["#0fa8a3","#0a7e7a"],
    ["#e8920e","#b86d05"],["#1aa971","#0e7b51"],["#41526b","#0d1b2e"],
  ];
  const [a,b] = palette[seed % palette.length];
  return `linear-gradient(135deg, ${a}, ${b})`;
}
async function switchTo(id: string) {
  tenantOpen.value = false;
  await auth.switchTenant(id);
  // If we're inside the tenant console but lost tenant_admin in the new tenant,
  // leave it (the /admin beforeEnter guard doesn't re-run on an in-place switch).
  if (router.currentRoute.value.path.startsWith("/admin") && !auth.isTenantAdmin) {
    router.push("/");
  }
}
async function logout() {
  await auth.logout();
  router.push("/login");
}
</script>

<style scoped>
.topbar {
  height: 56px;
  background: var(--surface);
  border-bottom: 1px solid var(--border);
  position: sticky; top: 0; z-index: 100;
  display: flex; align-items: center;
  padding: 0 20px;
  gap: 20px;
}
.brand { display: flex; align-items: center; gap: 10px; min-width: calc(var(--rail-w) - 20px); }
.brand-logo {
  width: 32px; height: 32px;
  background: linear-gradient(135deg, var(--primary), #4a85ee);
  border-radius: 8px;
  display: grid; place-items: center;
  color: #fff; font-weight: 700; font-size: 16px;
  letter-spacing: -.5px;
}
.brand-name { font-weight: 600; font-size: 14.5px; letter-spacing: .2px; }

.tb-search {
  flex: 0 1 440px;
  margin-left: auto;
  height: 36px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 8px;
  display: flex; align-items: center;
  padding: 0 12px; gap: 8px;
  color: var(--text-3);
  transition: border-color .15s, background .15s, box-shadow .15s;
}
.tb-search:hover { border-color: var(--border-strong); }
.tb-search:focus-within {
  border-color: var(--primary); background: var(--surface);
  box-shadow: 0 0 0 3px var(--primary-soft);
}
.tb-search input {
  flex: 1; background: transparent; border: none; outline: none;
  font-family: inherit; font-size: 13px; color: var(--text);
}
.tb-search input::placeholder { color: var(--text-4); }
.tb-search kbd {
  font-family: inherit; font-size: 11px; color: var(--text-3);
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 4px; padding: 1px 6px;
}

.tb-actions { display: flex; align-items: center; gap: 4px; }
.icon-btn {
  width: 36px; height: 36px; border-radius: 8px;
  display: grid; place-items: center;
  background: transparent; border: 0;
  color: var(--text-2); position: relative;
  transition: background .15s, color .15s;
  cursor: pointer;
}
.icon-btn:hover { background: var(--bg); color: var(--text); }
.icon-btn .dot {
  position: absolute; top: 4px; right: 4px;
  min-width: 16px; height: 16px; padding: 0 4px;
  background: var(--accent); border: 2px solid var(--surface);
  border-radius: 999px;
  color: #fff; font-size: 10px; font-weight: 700;
  display: grid; place-items: center; line-height: 1;
}

.tenant-switcher { position: relative; margin-left: 8px; }
.tenant-chip {
  display: flex; align-items: center; gap: 8px;
  padding: 6px 10px 6px 8px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 8px;
  font-size: 13px; font-weight: 500; color: var(--text);
  transition: background .15s, border-color .15s;
  cursor: pointer;
}
.tenant-chip:hover, .tenant-switcher.open .tenant-chip {
  background: var(--primary-soft);
  border-color: var(--primary-softer);
  color: var(--primary);
}
.tenant-chip .t-badge {
  width: 22px; height: 22px;
  background: linear-gradient(135deg,#41526b,#0d1b2e);
  border-radius: 5px;
  color: #fff; font-size: 11px; font-weight: 700;
  display: grid; place-items: center;
}
.tenant-chip .chev { color: var(--text-3); transition: transform .2s; }
.tenant-switcher.open .chev { transform: rotate(180deg); }

.tenant-dropdown {
  position: absolute; top: calc(100% + 8px); right: 0;
  width: 300px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  box-shadow: var(--sh-3);
  padding: 8px;
  z-index: 200;
}
.tenant-row {
  display: flex; align-items: center; gap: 10px;
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;
  transition: background .12s;
}
.tenant-row:hover { background: var(--bg); }
.tenant-row .t-logo {
  width: 36px; height: 36px;
  border-radius: 8px;
  display: grid; place-items: center;
  color: #fff; font-weight: 700; font-size: 14px;
  flex-shrink: 0;
}
.tenant-row .t-info { flex: 1; min-width: 0; }
.tenant-row .t-name {
  font-size: 13px; font-weight: 600; color: var(--text);
  display: flex; align-items: center; gap: 6px;
}
.tenant-row .tag {
  font-size: 10px; padding: 1px 5px;
  background: var(--primary-soft); color: var(--primary);
  border-radius: 3px; font-weight: 600;
}
.tenant-row .t-sub { font-size: 12px; color: var(--text-3); margin-top: 2px; }
.tenant-row.current { background: var(--primary-soft); }
.tenant-row .check { color: var(--primary); flex-shrink: 0; }
.tenant-section-title {
  font-size: 11px; font-weight: 600; color: var(--text-3);
  text-transform: uppercase; letter-spacing: .8px;
  padding: 10px 10px 4px;
}
.tenant-dropdown hr { border: none; border-top: 1px solid var(--border); margin: 6px 4px; }
.tenant-dropdown .add-tenant {
  display: flex; align-items: center; gap: 8px;
  padding: 10px;
  font-size: 13px; color: var(--primary); font-weight: 500;
  border-radius: 8px; cursor: pointer;
}
.tenant-dropdown .add-tenant:hover { background: var(--primary-soft); }

.user-avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary), #4a85ee);
  display: grid; place-items: center;
  color: #fff; font-weight: 600; font-size: 13px;
  margin-left: 4px;
  cursor: pointer;
  transition: box-shadow .15s;
  border: 0;
}
.user-avatar:hover { box-shadow: 0 0 0 3px var(--primary-soft); }
</style>
