<template>
  <section class="home" :class="{ 'is-config': configMode }">
    <!-- Welcome strip -->
    <div class="welcome" :class="{ 'is-config': configMode }">
      <div class="welcome-left">
        <h1>
          {{ greetingPrefix }}，{{ greetingName }}
          <span class="wave">👋</span>
        </h1>
        <div class="welcome-meta">
          {{ todayLabel }} · {{ auth.tenant?.name ?? '—' }}
        </div>
      </div>
      <div class="welcome-right">
        <span class="env-tag">{{ env }}</span>
        <button class="config-btn" :class="{ active: configMode }" @click="configMode = !configMode">
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
            <rect x="3" y="3" width="7" height="7" rx="1"/><rect x="14" y="3" width="7" height="7" rx="1"/>
            <rect x="3" y="14" width="7" height="7" rx="1"/><rect x="14" y="14" width="7" height="7" rx="1"/>
          </svg>
          {{ configMode ? '完成配置' : '配置工作台' }}
        </button>
      </div>
    </div>

    <!-- Installed apps -->
    <article class="card" :class="{ 'is-config': configMode }">
      <header class="card-header">
        <span class="card-title">
          我的应用
          <span class="sub">· {{ myApps.length }} 个已安装</span>
        </span>
        <button class="card-link primary" @click="openAppCenter">浏览应用市场 →</button>
      </header>

      <div v-if="myApps.length === 0" class="apps-empty">
        <div class="empty-icon">◌</div>
        <div class="empty-title">尚未安装任何应用</div>
        <div class="empty-sub">点击右下角「+ 添加」打开应用市场</div>
      </div>

      <div v-else class="apps-grid">
        <router-link
          v-for="app in myApps"
          :key="app.code"
          :to="appHomeRoute(app.code)"
          class="app-tile"
        >
          <div class="app-icon" :style="{ background: app.color }">
            <svg width="22" height="22" viewBox="0 0 24 24" fill="currentColor">
              <path :d="app.icon"/>
            </svg>
          </div>
          <div class="app-name">{{ app.name }}</div>
          <div class="app-version">v{{ app.version }}</div>
        </router-link>
      </div>
    </article>

    <!-- User-configured widgets, rendered in pref order -->
    <component
      :is="WIDGET_COMPONENTS[code]"
      v-for="code in visibleWidgets"
      :key="code"
      :config-mode="configMode"
    />

    <!-- Platform info -->
    <article class="card" :class="{ 'is-config': configMode }">
      <header class="card-header">
        <span class="card-title">平台</span>
      </header>
      <div class="info-grid">
        <div class="info-item">
          <div class="info-label">租户</div>
          <div class="info-value">{{ auth.tenant?.name ?? '—' }} <code>{{ auth.tenant?.slug }}</code></div>
        </div>
        <div class="info-item">
          <div class="info-label">账号</div>
          <div class="info-value">{{ auth.user?.username || auth.user?.phone || auth.user?.email || '—' }}</div>
        </div>
        <div class="info-item">
          <div class="info-label">角色</div>
          <div class="info-value">
            <span v-if="admin.has_platform_access" class="role-tag platform">平台角色</span>
            <span v-else-if="admin.is_tenant_admin" class="role-tag tenant">租户管理员</span>
            <span v-else class="role-tag">成员</span>
          </div>
        </div>
        <div class="info-item">
          <div class="info-label">版本</div>
          <div class="info-value"><code>{{ version }}</code></div>
        </div>
      </div>
    </article>

    <WidgetGallery :open="configMode" @close="configMode = false" />
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import { getMyApps, appHomeRoute, type Manifest } from "@/shell/appcenter/appstore";
import { getMyAdminFlags, type MeAdmin } from "@/modules/admin/api/admin";
import { useWidgetPrefs } from "./widgets/prefs";
import { type WidgetCode } from "./widgets/types";
import WidgetGallery from "./widgets/WidgetGallery.vue";
import WidgetNotifications from "./widgets/WidgetNotifications.vue";
import WidgetTodos from "./widgets/WidgetTodos.vue";
import WidgetAgenda from "./widgets/WidgetAgenda.vue";
import WidgetRecent from "./widgets/WidgetRecent.vue";
import WidgetAnnouncement from "./widgets/WidgetAnnouncement.vue";

const auth = useAuthStore();
const configMode = ref(false);

const { prefs } = useWidgetPrefs();
const visibleWidgets = computed(() => prefs.value.visible);

const WIDGET_COMPONENTS: Record<WidgetCode, unknown> = {
  notifications: WidgetNotifications,
  todos: WidgetTodos,
  agenda: WidgetAgenda,
  recent: WidgetRecent,
  announcement: WidgetAnnouncement,
};

const now = new Date();
const greetingName = computed(() =>
  auth.user?.username || auth.user?.email?.split("@")[0] || "同学"
);
const greetingPrefix = computed(() => {
  const h = now.getHours();
  if (h < 6) return "凌晨好"; if (h < 12) return "上午好"; if (h < 14) return "中午好";
  if (h < 18) return "下午好"; return "晚上好";
});
const todayLabel = now.toLocaleDateString("zh-CN", {
  year: "numeric", month: "long", day: "numeric", weekday: "long",
});
const env = import.meta.env.MODE === "development" ? "dev" : "prod";

const myApps = ref<Manifest[]>([]);
const admin = ref<MeAdmin>({ is_tenant_admin: false, is_platform_admin: false, has_platform_access: false });
const version = ref<string>("—");

onMounted(async () => {
  try { myApps.value = await getMyApps(); } catch {}
  try { admin.value = await getMyAdminFlags(); } catch {}
  try {
    const r = await fetch("/version");
    const j = await r.json();
    version.value = j?.version ?? "—";
  } catch {}
});

function openAppCenter() {
  const btn = document.querySelector(".rail-add") as HTMLButtonElement | null;
  btn?.click();
}
</script>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
  max-width: 1100px;
  margin: 0 auto;
}

/* Welcome */
.welcome {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 22px 28px;
  background: linear-gradient(110deg, #eef3fe 0%, #f1ecfb 50%, #e3f6f5 100%);
  border-radius: 14px;
  border: 1px solid rgba(255,255,255,.6);
  position: relative;
  overflow: hidden;
  transition: outline 0.18s;
  outline: 2px dashed transparent;
  outline-offset: 2px;
}
.welcome.is-config { outline-color: var(--primary); }
.welcome::before {
  content: "";
  position: absolute; top: -50%; right: -8%;
  width: 280px; height: 280px;
  background: radial-gradient(circle, rgba(30,95,217,.10), transparent 60%);
  border-radius: 50%;
  pointer-events: none;
}
.welcome-left { position: relative; z-index: 1; }
.welcome h1 {
  margin: 0;
  font-size: 22px;
  font-weight: 700;
  color: var(--text);
  letter-spacing: -.3px;
}
.welcome h1 .wave {
  display: inline-block;
  animation: wave 2.4s ease-in-out infinite;
  transform-origin: 70% 70%;
}
@keyframes wave {
  0%,60%,100% { transform: rotate(0); }
  10% { transform: rotate(14deg); } 20% { transform: rotate(-8deg); }
  30% { transform: rotate(14deg); } 40% { transform: rotate(-4deg); }
  50% { transform: rotate(10deg); }
}
.welcome-meta {
  margin-top: 6px;
  font-size: 13px;
  color: var(--text-2);
}
.welcome-right {
  position: relative; z-index: 1;
  display: flex; align-items: center; gap: 10px;
}
.env-tag {
  display: inline-block;
  padding: 4px 10px;
  background: rgba(255,255,255,.75);
  border: 1px solid rgba(255,255,255,.9);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  letter-spacing: .5px;
  text-transform: uppercase;
  color: var(--primary);
  font-family: var(--ff-mono);
}
.config-btn {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 7px 13px;
  background: rgba(255,255,255,.85);
  border: 1px solid rgba(255,255,255,.95);
  border-radius: 8px;
  color: var(--primary);
  font-size: 12.5px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.18s;
}
.config-btn:hover {
  background: #fff;
  box-shadow: 0 2px 8px rgba(13,27,46,.10);
  transform: translateY(-1px);
}
.config-btn.active {
  background: var(--primary);
  color: #fff;
  border-color: var(--primary);
}

/* Card */
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--sh-1);
  overflow: hidden;
  transition: outline 0.18s;
  outline: 2px dashed transparent;
  outline-offset: 2px;
}
.card.is-config { outline-color: var(--primary); }
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 22px;
  border-bottom: 1px solid var(--border);
}
.card-title {
  font-size: 15px;
  font-weight: 600;
  color: var(--text);
}
.card-title .sub {
  font-weight: 400;
  color: var(--text-3);
  font-size: 13px;
  margin-left: 4px;
}
.card-link {
  font-size: 13px;
  border: 0;
  background: transparent;
  cursor: pointer;
  color: var(--text-3);
}
.card-link:hover { color: var(--primary); }
.card-link.primary { color: var(--primary); font-weight: 500; }

/* Apps grid */
.apps-empty {
  text-align: center;
  padding: 50px 0;
  color: var(--text-3);
}
.empty-icon { font-size: 36px; color: var(--text-4); margin-bottom: 10px; }
.empty-title { font-size: 14px; font-weight: 600; color: var(--text-2); }
.empty-sub { font-size: 12.5px; margin-top: 4px; }

.apps-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(140px, 1fr));
  gap: 10px;
  padding: 16px;
}
.app-tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 18px 10px 12px;
  border-radius: 12px;
  text-decoration: none;
  color: inherit;
  border: 1px solid var(--border);
  transition: all 0.15s;
  background: var(--surface);
}
.app-tile:hover {
  transform: translateY(-2px);
  box-shadow: var(--sh-2);
  border-color: var(--border-strong);
  text-decoration: none;
}
.app-icon {
  width: 48px; height: 48px;
  border-radius: 13px;
  display: grid;
  place-items: center;
  color: white;
  box-shadow: 0 3px 8px rgba(13,27,46,.14);
}
.app-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--text);
  text-align: center;
}
.app-version {
  font-size: 10.5px;
  color: var(--text-3);
  font-family: var(--ff-mono);
}

/* Platform info */
.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: 1px;
  background: var(--border);
}
.info-item {
  background: var(--surface);
  padding: 14px 22px;
}
.info-label {
  font-size: 11.5px;
  font-weight: 500;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: .5px;
}
.info-value {
  font-size: 13.5px;
  color: var(--text);
  margin-top: 4px;
  display: flex;
  align-items: center;
  gap: 8px;
}
.info-value code {
  background: var(--surface-2);
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--ff-mono);
  font-size: 11.5px;
  color: var(--text-2);
}

.role-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  background: var(--bg-deep);
  color: var(--text-3);
}
.role-tag.tenant {
  background: var(--info-soft);
  color: var(--info);
}
.role-tag.platform {
  background: var(--purple-soft);
  color: var(--purple);
}
</style>
