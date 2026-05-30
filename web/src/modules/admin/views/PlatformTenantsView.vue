<template>
  <section class="admin-page">
    <PageHeader title="组织机构" :sub="`平台共 ${tenants.length} 家组织 · 仅平台管理员可见`">
      <template #actions>
        <div class="head-actions">
          <button class="btn btn-ghost" @click="reload">刷新</button>
          <button class="btn btn-primary" v-perm="'org:write'" @click="openCreate">+ 新建组织</button>
        </div>
      </template>
    </PageHeader>

    <div v-if="actionError && !creating" class="page-error">
      {{ actionError }}
      <button class="page-error-close" @click="actionError = ''">×</button>
    </div>

    <DataTable :columns="columns" :rows="tenants" rowKey="id">
      <template #cell-org="{ row }">
        <div class="tenant-cell">
          <div class="t-logo" :style="{ background: colorFor(row.name) }">{{ row.name[0] }}</div>
          <div>
            <div class="t-name">{{ row.name }}</div>
            <div class="t-meta">
              <code>{{ row.slug }}</code>
              <span class="schema">schema: <code class="mono">{{ row.schema_name }}</code></span>
            </div>
          </div>
        </div>
      </template>
      <template #cell-status="{ row }">
        <span class="status-tag" :class="`status-${row.status}`">
          <span class="dot"></span>{{ statusLabel(row.status) }}
        </span>
      </template>
      <template #cell-created_at="{ row }">
        <span class="time">{{ formatTime(row.created_at) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="row-actions">
          <button v-if="row.status === 'active'" v-perm="'org:write'" class="btn btn-ghost btn-sm danger" @click="suspend(row)">暂停</button>
          <button v-else-if="row.status === 'suspended'" v-perm="'org:write'" class="btn btn-ghost btn-sm" @click="resume(row)">恢复</button>
          <span v-else class="muted">—</span>
        </div>
      </template>
    </DataTable>

    <EmptyState v-if="tenants.length === 0 && !loading" title="尚无组织机构" sub="点击右上「+ 新建组织」开通" />

    <!-- Create modal -->
    <div v-if="creating" class="modal-overlay" @click.self="closeCreate">
      <div class="modal">
        <h3>新建组织机构</h3>
        <p class="modal-sub">将创建一个独立的租户 schema（PG 数据隔离）。</p>
        <label class="field">
          <span class="label">组织名称</span>
          <input class="input" v-model="form.name" type="text" required autofocus maxlength="40"
                 placeholder="例如：演示一公司" />
        </label>
        <label class="field">
          <span class="label">标识 (slug)</span>
          <input class="input" v-model="form.slug" type="text" required pattern="^[a-z][a-z0-9_-]{1,30}[a-z0-9]$"
                 placeholder="3-32 位小写字母/数字/-/_，以字母开头" />
          <span class="field-hint">用于 schema 名 (tenant_xxx) 和接口标识，创建后不可修改</span>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeCreate">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmCreate">
            {{ busy ? '创建中…' : '确认创建' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, DataTable, EmptyState, type Column } from "@/shell/components";
import { client } from "@/api/client";
import {
  listAllTenants, suspendTenant, resumeTenant, type PlatformTenant,
} from "../api/admin";
import { useConfirm } from "@/shell/confirm";

const { confirm } = useConfirm();

const tenants = ref<PlatformTenant[]>([]);
const loading = ref(false);
const busy = ref(false);
const actionError = ref("");

const columns: Column[] = [
  { key: "org",        label: "组织",      width: 360 },
  { key: "status",     label: "状态",      width: 100 },
  { key: "created_at", label: "创建时间",  width: 160 },
  { key: "actions",    label: "操作",      width: 140, align: "right" },
];

onMounted(reload);
async function reload() {
  loading.value = true;
  try { tenants.value = await listAllTenants(); }
  finally { loading.value = false; }
}

// === Create modal ===
const creating = ref(false);
const form = reactive({ name: "", slug: "" });

function openCreate() {
  creating.value = true;
  form.name = "";
  form.slug = "";
  actionError.value = "";
}
function closeCreate() { creating.value = false; }

async function confirmCreate() {
  if (!form.name.trim() || !form.slug.trim()) {
    actionError.value = "组织名称和 slug 必填"; return;
  }
  busy.value = true; actionError.value = "";
  try {
    await client.post("/tenants", { name: form.name.trim(), slug: form.slug.trim() });
    creating.value = false;
    await reload();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "创建失败";
  } finally { busy.value = false; }
}

async function suspend(t: PlatformTenant) {
  if (!(await confirm({ title: "确认", message: `确定暂停组织 "${t.name}"？该组织成员将无法登录。`, danger: true }))) return;
  actionError.value = "";
  try {
    await suspendTenant(t.id);
    await reload();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "暂停失败";
  }
}
async function resume(t: PlatformTenant) {
  actionError.value = "";
  try {
    await resumeTenant(t.id);
    await reload();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "恢复失败";
  }
}

function statusLabel(s: string) {
  return ({ active: "运行中", suspended: "已暂停", closed: "已关闭" } as Record<string, string>)[s] ?? s;
}
function formatTime(iso?: string): string {
  if (!iso) return "—";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const m = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${m(d.getMonth()+1)}-${m(d.getDate())} ${m(d.getHours())}:${m(d.getMinutes())}`;
}
function colorFor(name: string) {
  const seed = (name || "?").split("").reduce((s, c) => s + c.charCodeAt(0), 0);
  const palette = [
    "linear-gradient(135deg,#1e5fd9,#4a85ee)",
    "linear-gradient(135deg,#7c4ddb,#5a2db5)",
    "linear-gradient(135deg,#0fa8a3,#0a7e7a)",
    "linear-gradient(135deg,#e8920e,#b86d05)",
    "linear-gradient(135deg,#1aa971,#0e7b51)",
  ];
  return palette[seed % palette.length];
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.head-actions { display: flex; gap: 8px; }
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close {
  border: 0; background: transparent; color: inherit;
  font-size: 18px; line-height: 1; cursor: pointer;
}

.tenant-cell { display: flex; gap: 10px; align-items: center; }
.t-logo {
  width: 32px; height: 32px;
  border-radius: 7px;
  color: white; font-weight: 700;
  display: grid; place-items: center;
  font-size: 14px;
  flex-shrink: 0;
}
.t-name { font-size: 13.5px; font-weight: 600; color: var(--text); }
.t-meta { font-size: 11.5px; color: var(--text-3); margin-top: 2px; display: flex; gap: 10px; }
.schema { color: var(--text-4); }
code {
  background: var(--surface-2);
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--ff-mono);
  font-size: 11px;
}
code.mono { color: var(--text-2); }

.status-tag {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 11.5px;
  font-weight: 600;
}
.status-tag .dot { width: 5px; height: 5px; background: currentColor; border-radius: 50%; }
.status-active { background: var(--success-soft); color: var(--success); }
.status-suspended { background: var(--warning-soft); color: var(--warning); }
.status-closed { background: var(--danger-soft); color: var(--danger); }

.time { font-size: 12.5px; color: var(--text-2); font-family: var(--ff-mono); }
.row-actions { display: flex; gap: 4px; justify-content: flex-end; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger { color: var(--warning); }
.btn-sm.danger:hover { background: var(--warning-soft); }
.muted { color: var(--text-4); }

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
  display: flex; flex-direction: column; gap: 12px;
}
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-sub { font-size: 12.5px; color: var(--text-3); margin: -6px 0 4px; }
.field { display: flex; flex-direction: column; gap: 4px; }
.field .label { font-size: 12px; color: var(--text-2); }
.field-hint { margin-top: 2px; font-size: 11px; color: var(--text-3); }
.form-error {
  font-size: 12.5px;
  color: var(--danger);
  background: var(--danger-soft);
  padding: 8px 10px;
  border-radius: 6px;
}
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
</style>
