<template>
  <section class="admin-page">
    <PageHeader title="我的审批" sub="待办 / 已办 / 我发起 / 抄送我">
      <template #actions>
        <RouterLink class="btn btn-primary" to="/approval/new">+ 发起审批</RouterLink>
      </template>
    </PageHeader>

    <div class="seg-ctrl ap-tabs">
      <button v-for="t in tabs" :key="t.key" class="seg-btn" :class="{ active: tab === t.key }" @click="select(t.key)">
        {{ t.label }}
      </button>
    </div>

    <div v-if="loading" class="ap-loading">加载中…</div>
    <EmptyState v-else-if="rows.length === 0" :title="emptyTitle" sub="这里还没有内容" icon="✓" />

    <ul v-else class="ap-list">
      <li v-for="row in rows" :key="rowKey(row)" class="ap-card card clickable" @click="open(row)">
        <div class="ap-card-main">
          <div class="ap-card-title">{{ titleOf(row) }}</div>
          <div class="ap-card-sub">{{ subOf(row) }}</div>
        </div>
        <div class="ap-card-side">
          <span class="badge" :class="statusBadge(statusOf(row))">{{ statusLabel(statusOf(row)) }}</span>
          <span class="ap-card-time">{{ fmt(timeOf(row)) }}</span>
        </div>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { PageHeader, EmptyState } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import * as api from "../api";
import type { InboxType, Task, Instance } from "../api";

const router = useRouter();
const notify = useNotification();

const tabs: { key: InboxType; label: string }[] = [
  { key: "todo", label: "待办" },
  { key: "done", label: "已办" },
  { key: "initiated", label: "我发起" },
  { key: "cc", label: "抄送我" },
];

const tab = ref<InboxType>("todo");
const rows = ref<(Task | Instance)[]>([]);
const loading = ref(false);

const emptyTitle = computed(() => tabs.find((t) => t.key === tab.value)?.label ?? "");

function isTask(row: Task | Instance): row is Task {
  return "instance_id" in row;
}

function rowKey(row: Task | Instance): string {
  return isTask(row) ? row.id : (row as Instance).id;
}
function instanceIdOf(row: Task | Instance): string {
  return isTask(row) ? row.instance_id : (row as Instance).id;
}
function titleOf(row: Task | Instance): string {
  return isTask(row) ? row.form_name ?? "审批" : (row as Instance).form_name;
}
function subOf(row: Task | Instance): string {
  if (isTask(row)) {
    const t = row as Task;
    return t.type === "cc" ? "抄送给我" : "等待我审批";
  }
  const i = row as Instance;
  return `我发起 · 进行到第 ${i.current_node + 1} 步`;
}
function statusOf(row: Task | Instance): string {
  if (isTask(row)) {
    // For todo/cc the relevant status is the instance state.
    return (row as Task).instance_status ?? (row as Task).status;
  }
  return (row as Instance).status;
}
function timeOf(row: Task | Instance): string {
  return isTask(row) ? row.created_at : (row as Instance).created_at;
}

const statusLabels: Record<string, string> = {
  pending: "审批中",
  approved: "已通过",
  rejected: "已拒绝",
  canceled: "已撤回",
  read: "已阅",
};
function statusLabel(s: string): string {
  return statusLabels[s] ?? s;
}
function statusBadge(s: string): string {
  switch (s) {
    case "approved": return "badge-success";
    case "rejected": return "badge-danger";
    case "canceled": return "badge-info";
    case "read": return "badge-teal";
    default: return "badge-warning";
  }
}

function fmt(s: string): string {
  if (!s) return "";
  const d = new Date(s);
  return d.toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}

async function load() {
  loading.value = true;
  try {
    const res = await api.inbox(tab.value);
    rows.value = res.items as (Task | Instance)[];
  } catch {
    notify.error("加载失败");
    rows.value = [];
  } finally {
    loading.value = false;
  }
}

function select(t: InboxType) {
  tab.value = t;
  void load();
}

function open(row: Task | Instance) {
  router.push(`/approval/instances/${instanceIdOf(row)}`);
}

onMounted(load);
</script>

<style scoped>
.ap-tabs { margin-bottom: var(--sp-4); }
.ap-loading { padding: var(--sp-6); color: var(--text-3); }
.ap-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--sp-3); }
.ap-card {
  display: flex; align-items: center; justify-content: space-between;
  padding: var(--sp-4) var(--sp-5);
}
.ap-card-main { min-width: 0; }
.ap-card-title { font-weight: 600; color: var(--text); }
.ap-card-sub { font-size: 13px; color: var(--text-3); margin-top: 2px; }
.ap-card-side { display: flex; align-items: center; gap: var(--sp-3); flex-shrink: 0; }
.ap-card-time { font-size: 12px; color: var(--text-4); }
</style>
