<template>
  <section class="me-page">
    <PageHeader title="登录会话" sub="查看你所有的登录设备，可强制退出非当前设备" />

    <DataTable :columns="columns" :rows="sessions" rowKey="id">
      <template #cell-device="{ row }">
        <div class="device-cell">
          <span class="d-icon">{{ deviceIcon(row.user_agent) }}</span>
          <div>
            <div class="d-name">{{ deviceName(row.user_agent) }}</div>
            <div class="d-meta">{{ row.ip_address || '—' }}</div>
          </div>
        </div>
      </template>
      <template #cell-issued_at="{ row }">
        <span class="time">{{ formatTime(row.issued_at) }}</span>
      </template>
      <template #cell-status="{ row }">
        <span v-if="row.current" class="status-tag current">当前设备</span>
        <span v-else-if="row.revoked" class="status-tag revoked">已注销</span>
        <span v-else class="status-tag active">活跃</span>
      </template>
      <template #cell-actions="{ row }">
        <button v-if="!row.current && !row.revoked" class="btn btn-ghost btn-sm danger" @click="revoke(row.id)">
          强制退出
        </button>
        <span v-else class="muted">—</span>
      </template>
    </DataTable>

    <EmptyState v-if="sessions.length === 0 && !loading" title="没有活跃会话" sub="" />
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { PageHeader, DataTable, EmptyState, type Column } from "@/shell/components";
import { listSessions, revokeSession, type Session } from "@/modules/admin/api/admin";
import { useConfirm } from "@/shell/confirm";

const { confirm } = useConfirm();

const sessions = ref<Session[]>([]);
const loading = ref(true);

const columns: Column[] = [
  { key: "device",    label: "设备 / IP",  width: 320 },
  { key: "issued_at", label: "登录时间",   width: 180 },
  { key: "status",    label: "状态",       width: 120 },
  { key: "actions",   label: "操作",       width: 120, align: "right" },
];

async function refresh() {
  loading.value = true;
  try { sessions.value = await listSessions(); }
  finally { loading.value = false; }
}
onMounted(refresh);

async function revoke(id: string) {
  if (!(await confirm({ title: "确认", message: "确认强制退出该设备？", danger: true }))) return;
  await revokeSession(id);
  await refresh();
}

function formatTime(iso?: string) {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const m = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${m(d.getMonth()+1)}-${m(d.getDate())} ${m(d.getHours())}:${m(d.getMinutes())}`;
}
function deviceIcon(ua: string) {
  if (/mobile|iphone|android/i.test(ua)) return "📱";
  if (/mac/i.test(ua)) return "💻";
  if (/windows/i.test(ua)) return "🖥️";
  return "🌐";
}
function deviceName(ua: string) {
  if (!ua) return "未知设备";
  if (/iphone/i.test(ua)) return "iPhone";
  if (/ipad/i.test(ua)) return "iPad";
  if (/android/i.test(ua)) return "Android";
  if (/mac/i.test(ua)) return "macOS";
  if (/windows/i.test(ua)) return "Windows";
  return ua.slice(0, 40) + (ua.length > 40 ? "…" : "");
}
</script>

<style scoped>
.me-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.device-cell { display: flex; align-items: center; gap: 10px; }
.d-icon { font-size: 18px; }
.d-name { font-size: 13.5px; font-weight: 500; color: var(--text); }
.d-meta { font-size: 11.5px; color: var(--text-3); font-family: var(--ff-mono); margin-top: 2px; }
.time { font-size: 12.5px; color: var(--text-2); font-family: var(--ff-mono); }
.status-tag {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 600;
}
.status-tag.current { background: var(--primary-soft); color: var(--primary); }
.status-tag.active { background: var(--success-soft); color: var(--success); }
.status-tag.revoked { background: var(--bg-deep); color: var(--text-3); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }
.muted { color: var(--text-4); }
</style>
