<template>
  <WidgetCard
    title="新通知"
    :icon="iconPath"
    :source="{ code: 'iop', name: '平台', color: 'var(--info)' }"
    :more="{ label: '查看全部', to: '/notifications', go: () => router.push('/notifications') }"
    :config-mode="configMode"
  >
    <div v-if="loading" class="row muted">加载中…</div>
    <div v-else-if="list.length === 0" class="row muted">没有未读消息</div>
    <ul v-else class="rows">
      <li v-for="n in list.slice(0, 4)" :key="n.id" class="row">
        <span class="dot"></span>
        <span class="msg">{{ n.title || n.body || '通知' }}</span>
        <span class="time">{{ formatTime(n.created_at || n.at) }}</span>
      </li>
    </ul>
  </WidgetCard>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { client } from "@/api/client";
import WidgetCard from "./WidgetCard.vue";

defineProps<{ configMode?: boolean }>();

const router = useRouter();
const iconPath = "M12 22a2 2 0 0 0 2-2h-4a2 2 0 0 0 2 2zm6-6V11c0-3.07-1.63-5.64-4.5-6.32V4a1.5 1.5 0 1 0-3 0v.68C7.64 5.36 6 7.92 6 11v5l-2 2v1h16v-1l-2-2z";

interface Note { id: string; title?: string; body?: string; created_at?: string; at?: string; }
const list = ref<Note[]>([]);
const loading = ref(true);

onMounted(async () => {
  try {
    const r = await client.get("/notifications/unread");
    list.value = r.data?.data?.notifications ?? [];
  } catch {} finally { loading.value = false; }
});

function formatTime(s?: string): string {
  if (!s) return "";
  const d = new Date(s);
  if (Number.isNaN(d.getTime())) return "";
  const diff = (Date.now() - d.getTime()) / 1000;
  if (diff < 60) return "刚刚";
  if (diff < 3600) return `${Math.floor(diff / 60)} 分前`;
  if (diff < 86400) return `${Math.floor(diff / 3600)} 时前`;
  return `${Math.floor(diff / 86400)} 天前`;
}
</script>

<style scoped>
.rows { list-style: none; margin: 0; padding: 0; }
.row {
  display: flex; align-items: center; gap: 8px;
  padding: 7px 0;
  font-size: 13px;
  color: var(--text);
  border-bottom: 1px dashed var(--border);
}
.row:last-child { border-bottom: 0; }
.row.muted {
  color: var(--text-3);
  font-size: 12.5px;
  padding: 14px 0;
  justify-content: center;
}
.dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  background: var(--info);
  flex-shrink: 0;
}
.msg {
  flex: 1; min-width: 0;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.time {
  color: var(--text-3);
  font-size: 11.5px;
  flex-shrink: 0;
}
</style>
