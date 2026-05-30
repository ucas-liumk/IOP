<template>
  <WidgetCard
    title="本周议程"
    :icon="iconPath"
    :source="{ code: 'okr', name: 'OKR 工作安排', color: 'var(--cat-collab)' }"
    :more="{ label: '查看周计划', to: '/okr/plans?level=week', go: () => router.push('/okr/plans') }"
    :config-mode="configMode"
  >
    <div v-if="loading" class="row muted">加载中…</div>
    <div v-else-if="items.length === 0" class="row muted">本周还没有安排</div>
    <ul v-else class="agenda">
      <li v-for="(item, idx) in items.slice(0, 4)" :key="idx" class="day-row">
        <div class="day-tag">{{ item.day }}</div>
        <div class="day-body">
          <div class="day-title">{{ item.title }}</div>
          <div class="day-meta">{{ item.meta }}</div>
        </div>
      </li>
    </ul>
  </WidgetCard>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { listPlans, type Plan } from "@/modules/okr/api/okr";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ configMode?: boolean }>();

const router = useRouter();
const iconPath = "M19 4h-1V2h-2v2H8V2H6v2H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zm0 16H5V10h14v10zm0-12H5V6h14v2z";

interface AgendaItem { day: string; title: string; meta: string; }
const items = ref<AgendaItem[]>([]);
const loading = ref(true);

function dayLabel(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  const weekday = ["周日", "周一", "周二", "周三", "周四", "周五", "周六"][d.getDay()];
  return `${d.getMonth() + 1}/${d.getDate()} ${weekday}`;
}

onMounted(async () => {
  try {
    const plans: Plan[] = await listPlans();
    const now = Date.now();
    const weekFromNow = now + 7 * 86400000;
    const open: AgendaItem[] = [];
    for (const p of plans) {
      if (p.status === "closed") continue;
      const end = new Date(p.period?.end ?? "").getTime();
      if (Number.isNaN(end)) continue;
      if (end >= now && end <= weekFromNow) {
        open.push({
          day: dayLabel(p.period.end),
          title: p.title,
          meta: `${p.level === "week" ? "周计划" : p.level} · 截止`,
        });
      }
    }
    open.sort((a, b) => a.day.localeCompare(b.day));
    items.value = open;
  } catch {} finally { loading.value = false; }
});
</script>

<style scoped>
.agenda { list-style: none; margin: 0; padding: 0; }
.day-row {
  display: flex; gap: 10px;
  padding: 8px 0;
  border-bottom: 1px dashed var(--border);
}
.day-row:last-child { border-bottom: 0; }
.day-tag {
  font-size: 11px;
  font-weight: 600;
  background: var(--cat-collab-soft, var(--info-soft));
  color: var(--cat-collab, var(--info));
  padding: 3px 8px;
  border-radius: 6px;
  white-space: nowrap;
  font-family: var(--ff-mono);
  height: fit-content;
  margin-top: 2px;
}
.day-body { flex: 1; min-width: 0; }
.day-title {
  font-size: 13px;
  color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.day-meta {
  font-size: 11.5px;
  color: var(--text-3);
  margin-top: 2px;
}
.row.muted {
  color: var(--text-3); font-size: 12.5px;
  padding: 14px 0; text-align: center;
}
</style>
