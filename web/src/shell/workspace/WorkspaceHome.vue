<template>
  <section class="home">
    <!-- Welcome strip — real greeting, no fake stats -->
    <div class="welcome">
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
      </div>
    </div>

    <!-- Installed apps — driven by /me/apps -->
    <article class="card">
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

    <!-- Platform info — keep minimal, real data -->
    <article class="card">
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
          <div class="info-value">{{ auth.user?.email }}</div>
        </div>
        <div class="info-item">
          <div class="info-label">角色</div>
          <div class="info-value">
            <span v-if="admin.is_platform_admin" class="role-tag platform">平台管理员</span>
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
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import { getMyApps, appHomeRoute, type Manifest } from "@/shell/appcenter/appstore";
import { getMyAdminFlags, type MeAdmin } from "@/modules/admin/api/admin";

const auth = useAuthStore();

const now = new Date();
const greetingName = computed(() => auth.user?.email?.split("@")[0] ?? "同学");
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
const admin = ref<MeAdmin>({ is_tenant_admin: false, is_platform_admin: false });
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
  // The AppCenter modal lives in LeftRail; simulate the click.
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
}
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
.welcome-right { position: relative; z-index: 1; }
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

/* Card */
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--sh-1);
  overflow: hidden;
}
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
