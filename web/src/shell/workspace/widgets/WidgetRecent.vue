<template>
  <WidgetCard
    title="最近访问"
    :icon="iconPath"
    :source="{ code: 'iop', name: '平台', color: 'var(--text-3)' }"
    :config-mode="configMode"
  >
    <div v-if="tiles.length === 0" class="row muted">还没有访问记录</div>
    <div v-else class="recent-grid">
      <router-link
        v-for="t in tiles"
        :key="t.code"
        :to="t.route"
        class="r-tile"
      >
        <div class="r-ico" :style="{ background: t.color }">
          <svg width="16" height="16" viewBox="0 0 24 24" fill="currentColor"><path :d="t.icon"/></svg>
        </div>
        <div class="r-name">{{ t.name }}</div>
      </router-link>
    </div>
  </WidgetCard>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRecentApps } from "./recentApps";
import { getMyApps, appHomeRoute, type Manifest } from "@/shell/appcenter/appstore";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ configMode?: boolean }>();

const iconPath = "M12 2a10 10 0 1 0 10 10A10 10 0 0 0 12 2zm0 18a8 8 0 1 1 8-8 8 8 0 0 1-8 8zm.5-13H11v6l5.25 3.15.75-1.23-4.5-2.67z";

const { recent } = useRecentApps();
const apps = ref<Manifest[]>([]);
onMounted(async () => {
  try { apps.value = await getMyApps(); } catch {}
});

const tiles = computed(() => {
  return recent.value
    .map((r) => {
      const a = apps.value.find((x) => x.code === r.code);
      if (!a) return null;
      return { code: a.code, name: a.name, icon: a.icon, color: a.color, route: appHomeRoute(a.code) };
    })
    .filter((x): x is NonNullable<typeof x> => x !== null)
    .slice(0, 6);
});
</script>

<style scoped>
.recent-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(86px, 1fr));
  gap: 10px;
}
.r-tile {
  display: flex; flex-direction: column; align-items: center; gap: 6px;
  padding: 10px 6px;
  border-radius: 10px;
  border: 1px solid var(--border);
  text-decoration: none;
  color: inherit;
  background: var(--surface);
  transition: all 0.15s;
}
.r-tile:hover {
  transform: translateY(-1px);
  box-shadow: var(--sh-2);
  border-color: var(--border-strong);
  text-decoration: none;
}
.r-ico {
  width: 32px; height: 32px;
  border-radius: 8px;
  display: grid; place-items: center;
  color: #fff;
  box-shadow: 0 2px 4px rgba(13,27,46,.10);
}
.r-name {
  font-size: 11.5px;
  font-weight: 500;
  color: var(--text);
  text-align: center;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  max-width: 100%;
}
.row.muted {
  color: var(--text-3); font-size: 12.5px;
  padding: 14px 0; text-align: center;
}
</style>
