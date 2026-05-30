<template>
  <section class="admin-page">
    <PageHeader title="注册申请" :sub="`待审批 ${pending.length} 条 · 共 ${all.length} 条`">
      <template #actions>
        <div class="tab-group">
          <button v-for="t in tabs" :key="t.key" class="tab" :class="{ active: tab === t.key }" @click="tab = t.key">
            {{ t.label }}
          </button>
        </div>
      </template>
    </PageHeader>

    <DataTable :columns="columns" :rows="rows" rowKey="id">
      <template #cell-applicant="{ row }">
        <div class="applicant-cell">
          <div class="ap-name">{{ row.real_name }}</div>
          <div class="ap-meta">
            <code>{{ row.username }}</code>
            <span v-if="row.phone" class="phone">· {{ row.phone }}</span>
          </div>
        </div>
      </template>
      <template #cell-organization="{ row }">
        <div class="org-cell">{{ row.organization }}</div>
      </template>
      <template #cell-applied_at="{ row }">
        <span class="time">{{ formatTime(row.applied_at) }}</span>
      </template>
      <template #cell-status="{ row }">
        <span class="status-tag" :class="`status-${row.status}`">
          {{ statusLabel(row.status) }}
        </span>
        <div v-if="row.status === 'approved' && row.granted_role" class="status-detail">
          {{ row.granted_role === 'tenant_admin' ? '管理员' : '成员' }}
        </div>
        <div v-if="row.status === 'rejected' && row.reject_reason" class="status-detail muted">
          {{ row.reject_reason }}
        </div>
      </template>
      <template #cell-actions="{ row }">
        <div v-if="row.status === 'pending'" class="row-actions">
          <button class="btn btn-primary btn-sm" @click="openApprove(row)">通过</button>
          <button class="btn btn-ghost btn-sm" @click="openReject(row)">拒绝</button>
        </div>
        <span v-else class="muted">—</span>
      </template>
    </DataTable>

    <EmptyState v-if="rows.length === 0 && !loading" :title="emptyTitle" sub="" />

    <!-- Approve modal -->
    <div v-if="approving" class="modal-overlay" @click.self="approving = null">
      <div class="modal">
        <h3>通过 {{ approving.real_name }} 的申请</h3>
        <p class="modal-sub">
          将进入「<strong>{{ approving.organization }}</strong>」，由申请人在注册时选定。
        </p>
        <label class="field">
          <span class="label">分配角色</span>
          <select v-model="approveRole" class="input">
            <option value="tenant_member">租户成员</option>
            <option value="tenant_admin">租户管理员</option>
          </select>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="approving = null">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmApprove">
            {{ busy ? '处理中…' : '确认通过' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Reject modal -->
    <div v-if="rejecting" class="modal-overlay" @click.self="rejecting = null">
      <div class="modal">
        <h3>拒绝 {{ rejecting.real_name }} 的申请</h3>
        <label class="field">
          <span class="label">拒绝原因 <span class="optional">（可选，仅自己可见）</span></span>
          <textarea v-model="rejectReason" class="input textarea" rows="3" maxlength="200"
                    placeholder="例如：单位不属实 / 重复申请"></textarea>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="rejecting = null">取消</button>
          <button class="btn btn-danger" :disabled="busy" @click="confirmReject">
            {{ busy ? '处理中…' : '确认拒绝' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PageHeader, DataTable, EmptyState, type Column } from "@/shell/components";
import {
  listRegistrations, approveRegistration, rejectRegistration,
  type RegistrationApplication, type RegScope,
} from "../api/admin";

// scope = "platform" (all tenants, platform console) or "tenant" (own tenant).
const props = withDefaults(defineProps<{ scope?: RegScope }>(), { scope: "tenant" });

const all = ref<RegistrationApplication[]>([]);
const loading = ref(true);

const tabs = [
  { key: "pending", label: "待审批" },
  { key: "approved", label: "已通过" },
  { key: "rejected", label: "已拒绝" },
] as const;
type TabKey = typeof tabs[number]["key"];
const tab = ref<TabKey>("pending");

const pending = computed(() => all.value.filter((a) => a.status === "pending"));
const rows = computed(() => all.value.filter((a) => a.status === tab.value));
const emptyTitle = computed(() => ({
  pending: "暂无待审批申请",
  approved: "暂无已通过申请",
  rejected: "暂无已拒绝申请",
}[tab.value]));

const columns: Column[] = [
  { key: "applicant",    label: "申请人",       width: 200 },
  { key: "organization", label: "目标单位",     width: 220 },
  { key: "applied_at",   label: "申请时间",     width: 160 },
  { key: "status",       label: "状态" },
  { key: "actions",      label: "操作",         width: 160, align: "right" },
];

async function refresh() {
  loading.value = true;
  try {
    const [pendingRows, approvedRows, rejectedRows] = await Promise.all([
      listRegistrations(props.scope, "pending"),
      listRegistrations(props.scope, "approved"),
      listRegistrations(props.scope, "rejected"),
    ]);
    all.value = [...pendingRows, ...approvedRows, ...rejectedRows];
  } finally { loading.value = false; }
}

onMounted(refresh);

// === Approve modal ===
const approving = ref<RegistrationApplication | null>(null);
const approveRole = ref<"tenant_member" | "tenant_admin">("tenant_member");

function openApprove(row: RegistrationApplication) {
  approving.value = row;
  approveRole.value = "tenant_member";
  actionError.value = "";
}

const busy = ref(false);
const actionError = ref("");

async function confirmApprove() {
  if (!approving.value) return;
  busy.value = true; actionError.value = "";
  try {
    await approveRegistration(props.scope, approving.value.id, approveRole.value);
    approving.value = null;
    await refresh();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "审批失败";
  } finally { busy.value = false; }
}

// === Reject modal ===
const rejecting = ref<RegistrationApplication | null>(null);
const rejectReason = ref("");

function openReject(row: RegistrationApplication) {
  rejecting.value = row;
  rejectReason.value = "";
  actionError.value = "";
}

async function confirmReject() {
  if (!rejecting.value) return;
  busy.value = true; actionError.value = "";
  try {
    await rejectRegistration(props.scope, rejecting.value.id, rejectReason.value);
    rejecting.value = null;
    await refresh();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "拒绝失败";
  } finally { busy.value = false; }
}

// === Helpers ===
function statusLabel(s: string) {
  return ({ pending: "待审批", approved: "已通过", rejected: "已拒绝" } as Record<string, string>)[s] ?? s;
}
function formatTime(iso: string) {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const m = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${m(d.getMonth()+1)}-${m(d.getDate())} ${m(d.getHours())}:${m(d.getMinutes())}`;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.tab-group { display: flex; gap: 4px; background: var(--surface-2); padding: 3px; border-radius: 8px; }
.tab {
  border: 0; background: transparent;
  padding: 5px 14px; border-radius: 6px;
  font-size: 12.5px; color: var(--text-3); cursor: pointer;
}
.tab.active { background: var(--surface); color: var(--text); box-shadow: var(--sh-1); font-weight: 500; }
.applicant-cell { display: flex; flex-direction: column; gap: 2px; }
.ap-name { font-weight: 600; color: var(--text); }
.ap-meta { font-size: 12px; color: var(--text-3); }
.ap-meta code {
  background: var(--surface-2);
  padding: 1px 5px;
  border-radius: 3px;
  font-family: var(--ff-mono);
  font-size: 11px;
}
.phone { margin-left: 6px; }
.org-cell { color: var(--text); font-size: 13px; }
.time { font-size: 12.5px; color: var(--text-2); font-family: var(--ff-mono); }
.status-tag {
  display: inline-block;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 600;
}
.status-pending { background: var(--warning-soft); color: var(--warning); }
.status-approved { background: var(--success-soft); color: var(--success); }
.status-rejected { background: var(--danger-soft); color: var(--danger); }
.status-detail { font-size: 11.5px; color: var(--text-3); margin-top: 3px; }
.muted { color: var(--text-3); }
.row-actions { display: flex; gap: 6px; justify-content: flex-end; }
.btn-sm { padding: 4px 12px; font-size: 12px; }

.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(13, 27, 46, .45);
  display: grid; place-items: center; z-index: 100;
  backdrop-filter: blur(3px);
}
.modal {
  background: var(--surface);
  border-radius: 14px;
  padding: 22px;
  width: min(440px, 92vw);
  box-shadow: var(--sh-4);
  display: flex; flex-direction: column; gap: 14px;
}
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-sub { font-size: 12.5px; color: var(--text-3); margin: -6px 0 4px; }
.modal-sub strong { color: var(--text); font-weight: 600; }
.field { display: flex; flex-direction: column; gap: 4px; }
.field .label { font-size: 12px; color: var(--text-2); }
.optional { color: var(--text-4); font-weight: 400; }
.textarea { resize: vertical; min-height: 60px; padding: 8px 10px; font-family: inherit; }
.form-error {
  font-size: 12.5px;
  color: var(--danger);
  background: var(--danger-soft);
  padding: 8px 10px;
  border-radius: 6px;
}
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
</style>
