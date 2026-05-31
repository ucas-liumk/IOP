<template>
  <WidgetCard
    title="我的待办"
    :icon="iconPath"
    :source="{ code: 'okr', name: 'OKR 工作安排', color: 'var(--cat-task)' }"
    :more="{ label: '打开 OKR', to: '/okr/plans', go: () => router.push('/okr/plans') }"
    :config-mode="configMode"
  >
    <div v-if="loading" class="row muted">加载中…</div>
    <div v-else-if="todos.length === 0" class="row muted">暂无待办，恭喜你</div>
    <ul v-else class="rows">
      <li v-for="t in todos.slice(0, 5)" :key="t.id" class="row">
        <span class="check" :style="{ borderColor: progressColor(t.progress) }">
          <span v-if="t.progress > 0" class="check-fill" :style="{ background: progressColor(t.progress), width: t.progress + '%' }"></span>
        </span>
        <span class="title">{{ t.title }}</span>
        <span class="pct">{{ t.progress }}%</span>
      </li>
    </ul>
  </WidgetCard>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { listPlans } from "@/modules/okr/api/okr";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ configMode?: boolean }>();

const router = useRouter();
const iconPath = "M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z";

interface Todo { id: string; title: string; progress: number; }
const todos = ref<Todo[]>([]);
const loading = ref(true);

onMounted(async () => {
  try {
    const plans = await listPlans();
    const open: Todo[] = [];
    for (const p of plans) {
      if (p.status === "closed") continue;
      for (const it of p.items ?? []) {
        if (it.progress_pct < 100) {
          open.push({ id: it.id, title: it.title, progress: it.progress_pct });
        }
      }
    }
    todos.value = open;
  } catch {} finally { loading.value = false; }
});

function progressColor(p: number): string {
  if (p >= 80) return "var(--success)";
  if (p >= 40) return "var(--info)";
  if (p > 0) return "var(--warning)";
  return "var(--border-strong)";
}
</script>

<style scoped>
.rows { list-style: none; margin: 0; padding: 0; }
.row {
  display: flex; align-items: center; gap: 10px;
  padding: 7px 0;
  font-size: 13px;
  color: var(--text);
  border-bottom: 1px dashed var(--border);
}
.row:last-child { border-bottom: 0; }
.row.muted {
  color: var(--text-3); font-size: 12.5px;
  padding: 14px 0; justify-content: center;
}
.check {
  width: 14px; height: 14px;
  border: 1.5px solid var(--border-strong);
  border-radius: 4px;
  flex-shrink: 0;
  position: relative;
  overflow: hidden;
}
.check-fill {
  position: absolute;
  left: 0; top: 0; bottom: 0;
  transition: width 0.3s;
}
.title {
  flex: 1; min-width: 0;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.pct {
  font-size: 11.5px;
  color: var(--text-3);
  font-family: var(--ff-mono);
  flex-shrink: 0;
}
</style>
