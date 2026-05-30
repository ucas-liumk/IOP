<template>
  <section class="admin-page">
    <PageHeader title="在线用户" :sub="`${sessions.length} 个活跃会话 · 全平台跨组织范围`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
      </template>
    </PageHeader>

    <DataTable :columns="columns" :rows="sessions" row-key="session_id" empty-text="当前没有活跃会话">
      <template #cell-username="{ row }">
        <span class="name">{{ row.username || '—' }}</span>
        <span v-if="row.display_name" class="sub"> · {{ row.display_name }}</span>
      </template>
      <template #cell-tenant_name="{ row }">
        <span class="tag" v-if="row.tenant_name">{{ row.tenant_name }}</span>
        <span class="muted" v-else>无组织上下文</span>
      </template>
      <template #cell-ip_address="{ row }">
        <code>{{ row.ip_address || '—' }}</code>
      </template>
      <template #cell-issued_at="{ row }">
        <span class="time">{{ row.issued_at }}</span>
      </template>
      <template #cell-expires_at="{ row }">
        <span class="time">{{ row.expires_at }}</span>
      </template>
      <template #cell-actions="{ row }">
        <button class="link-btn danger" v-perm="'session:revoke'" @click="kick(row)">强制下线</button>
      </template>
    </DataTable>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { PageHeader, DataTable, type Column } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import { listOnlineSessions, kickSession, type PlatformOnlineSession } from "../api/system";

const notify = useNotification();
const { confirm } = useConfirm();

const columns: Column[] = [
  { key: "username", label: "用户" },
  { key: "tenant_name", label: "所属组织", width: "180px" },
  { key: "ip_address", label: "IP 地址", width: "150px" },
  { key: "issued_at", label: "登录时间", width: "170px" },
  { key: "expires_at", label: "过期时间", width: "170px" },
  { key: "actions", label: "操作", width: "110px" },
];

const sessions = ref<PlatformOnlineSession[]>([]);

onMounted(reload);

async function reload() {
  sessions.value = await listOnlineSessions();
}

async function kick(s: PlatformOnlineSession) {
  const who = s.username || s.display_name || "该会话";
  const ok = await confirm({
    title: "强制下线",
    message: `确认将「${who}」的会话强制下线？该用户当前会话将立即失效。`,
    danger: true,
    confirmText: "强制下线",
  });
  if (!ok) return;
  try {
    await kickSession(s.session_id);
    notify.success("已强制下线");
    await reload();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.name { font-weight: 600; color: var(--text); }
.sub { font-size: 12px; color: var(--text-3); }
.tag { font-size: 11.5px; padding: 2px 8px; border-radius: 4px; background: var(--surface-2); color: var(--text-2); }
.muted { color: var(--text-4); font-size: 12px; }
.time { font-family: var(--ff-mono); font-size: 12px; color: var(--text-3); }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; color: var(--text-2); }
.link-btn { background: transparent; border: 0; font-size: 12px; color: var(--primary); cursor: pointer; padding: 4px 8px; border-radius: 4px; }
.link-btn.danger { color: var(--danger); }
.link-btn.danger:hover { background: var(--danger-soft); }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
</style>
