<template>
  <section class="admin-page">
    <PageHeader title="组织管理" :sub="`左选组织，右管其层级架构（部门 / 处室） · 平台共 ${tenants.length} 家组织`">
      <template #actions>
        <div class="head-actions">
          <select class="tenant-select" v-model="selectedOrgId">
            <option :value="null">选择租户</option>
            <option v-for="t in tenants" :key="t.id" :value="t.id">{{ t.name }}</option>
          </select>
          <button class="btn btn-ghost" @click="reload">刷新</button>
          <button class="btn btn-primary" v-perm="'org:write'" @click="openCreate">+ 新建组织</button>
        </div>
      </template>
    </PageHeader>

    <div v-if="actionError && !creating" class="page-error">
      {{ actionError }}
      <button class="page-error-close" @click="actionError = ''">×</button>
    </div>

    <div class="two-pane">
      <!-- LEFT: organization (tenant) list -->
      <aside class="org-pane card">
        <div class="org-pane-head">
          <span class="org-pane-title">组织机构</span>
          <span class="org-count">{{ tenants.length }}</span>
        </div>

        <!-- Loading skeleton -->
        <div v-if="loading && tenants.length === 0" class="org-skeleton" role="status" aria-label="加载中">
          <SkeletonLoader :lines="6" :height="40" :last-short="false" />
        </div>

        <div v-else-if="tenants.length === 0" class="org-empty">
          <EmptyState title="尚无组织机构" sub="开通第一个组织开始" icon="◫">
            <template #actions>
              <button class="btn btn-primary btn-sm" v-perm="'org:write'" @click="openCreate">+ 新建组织</button>
            </template>
          </EmptyState>
        </div>

        <ul v-else class="org-list">
          <li
            v-for="t in tenants"
            :key="t.id"
            class="org-row"
            :class="{ selected: t.id === selectedOrgId }"
            @click="selectOrg(t)"
          >
            <div class="t-logo" :style="{ background: colorFor(t.name) }">{{ t.name[0] }}</div>
            <div class="t-main">
              <div class="t-name">{{ t.name }}</div>
              <div class="t-meta">
                <code>{{ t.slug }}</code>
                <span class="status-tag" :class="`status-${t.status}`">
                  <span class="dot"></span>{{ statusLabel(t.status) }}
                </span>
              </div>
            </div>
            <div class="row-actions" @click.stop>
              <button v-if="t.status === 'active'" v-perm="'org:write'" class="btn btn-ghost btn-sm danger" :disabled="rowBusy.has(t.id)" @click="suspend(t)">
                <span v-if="rowBusy.has(t.id)" class="btn-spinner" aria-hidden="true" />停用
              </button>
              <button v-else-if="t.status === 'suspended'" v-perm="'org:write'" class="btn btn-ghost btn-sm" :disabled="rowBusy.has(t.id)" @click="resume(t)">
                <span v-if="rowBusy.has(t.id)" class="btn-spinner" aria-hidden="true" />恢复
              </button>
              <span v-else class="muted">—</span>
            </div>
          </li>
        </ul>
      </aside>

      <!-- RIGHT: dept-tree manager for the selected org -->
      <div class="dept-pane">
        <EmptyState
          v-if="!selectedOrg"
          title="选择左侧组织以管理其组织架构"
          sub="点击左侧任一组织，在此查看与编辑其部门 / 处室层级"
          icon="◫"
        />
        <DeptTreeManager
          v-else
          :key="selectedOrg.id"
          :api="orgApi!"
          :template-url="orgDeptTemplateUrl(selectedOrg.id)"
          :import-url="orgDeptImportUrl(selectedOrg.id)"
          write-perm="org:write"
        >
          <template #head-left>
            <div class="org-banner">
              <span class="org-banner-name">{{ selectedOrg.name }}</span>
              <span class="org-banner-slug">schema: <code class="mono">{{ selectedOrg.schema_name }}</code></span>
            </div>
          </template>
        </DeptTreeManager>
      </div>
    </div>

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
        <div v-if="actionError" class="form-error" role="alert">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeCreate">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmCreate">
            <span v-if="busy" class="btn-spinner light" aria-hidden="true" />{{ busy ? '创建中…' : '确认创建' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { PageHeader, EmptyState, SkeletonLoader, DeptTreeManager, type DeptApi } from "@/shell/components";
import { client } from "@/api/client";
import {
  listAllTenants, suspendTenant, resumeTenant, type PlatformTenant,
} from "../api/admin";
import {
  orgDeptApi, orgDeptTemplateUrl, orgDeptImportUrl,
} from "@/modules/platform/api/orgs";
import { useConfirm } from "@/shell/confirm";

const { confirm } = useConfirm();

const tenants = ref<PlatformTenant[]>([]);
const loading = ref(false);
const busy = ref(false);
const actionError = ref("");
// Per-row in-flight status toggles (suspend / resume) keyed by tenant id.
const rowBusy = ref<Set<string>>(new Set());
function setRowBusy(id: string, on: boolean) {
  const next = new Set(rowBusy.value);
  if (on) next.add(id); else next.delete(id);
  rowBusy.value = next;
}

// Selected org → drives the right-hand dept manager.
const selectedOrgId = ref<string | null>(null);
const selectedOrg = computed(() => tenants.value.find((t) => t.id === selectedOrgId.value) ?? null);
// New adapter per selected org; <DeptTreeManager :key> remounts on org change.
const orgApi = computed<DeptApi | null>(() => (selectedOrg.value ? orgDeptApi(selectedOrg.value.id) : null));

function selectOrg(t: PlatformTenant) {
  selectedOrgId.value = t.id;
}

onMounted(reload);
async function reload() {
  loading.value = true;
  try {
    tenants.value = await listAllTenants();
    // Drop a stale selection if the org disappeared.
    if (selectedOrgId.value && !tenants.value.some((t) => t.id === selectedOrgId.value)) {
      selectedOrgId.value = null;
    }
  } finally { loading.value = false; }
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
  if (!(await confirm({ title: "确认", message: `确定停用组织 "${t.name}"？该组织成员将无法登录。`, danger: true }))) return;
  actionError.value = "";
  setRowBusy(t.id, true);
  try {
    await suspendTenant(t.id);
    await reload();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "停用失败";
  } finally {
    setRowBusy(t.id, false);
  }
}
async function resume(t: PlatformTenant) {
  actionError.value = "";
  setRowBusy(t.id, true);
  try {
    await resumeTenant(t.id);
    await reload();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "恢复失败";
  } finally {
    setRowBusy(t.id, false);
  }
}

function statusLabel(s: string) {
  return ({ active: "运行中", suspended: "已暂停", closed: "已关闭" } as Record<string, string>)[s] ?? s;
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
.head-actions { display: flex; gap: 8px; align-items: center; }
.tenant-select {
  min-width: 220px; max-width: 280px;
  padding: 7px 10px; border: 1px solid var(--border-strong);
  border-radius: 7px; background: var(--surface); color: var(--text);
  font-size: 13px;
}
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close {
  border: 0; background: transparent; color: inherit;
  font-size: 18px; line-height: 1; cursor: pointer;
}

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }

/* two-pane shell: org list | dept manager */
.two-pane { display: grid; grid-template-columns: 340px 1fr; gap: 16px; align-items: start; }

/* LEFT pane */
.org-pane { padding: 8px; display: flex; flex-direction: column; max-height: calc(100vh - 220px); }
.org-pane-head {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px 10px;
  font-size: 11.5px; font-weight: 600; color: var(--text-3);
  text-transform: uppercase; letter-spacing: .5px;
}
.org-count {
  margin-left: auto; background: var(--surface-2); color: var(--text-2);
  font-size: 11px; font-weight: 700; padding: 1px 8px; border-radius: 999px;
}
.org-empty { padding: 8px 0; }
.org-skeleton { padding: 8px 6px; }
.org-list { list-style: none; margin: 0; padding: 0; overflow: auto; }
.org-row {
  display: flex; align-items: center; gap: 10px;
  padding: 9px 10px; border-radius: 9px; cursor: pointer;
  border: 1px solid transparent;
  border-left: 3px solid transparent;
  transition: background .15s ease, border-color .15s ease;
}
.org-row:hover { background: var(--surface-2); }
.org-row.selected { background: var(--primary-soft); border-color: var(--primary); border-left-color: var(--primary); }
.t-logo {
  width: 32px; height: 32px; border-radius: 7px;
  color: white; font-weight: 700;
  display: grid; place-items: center; font-size: 14px; flex-shrink: 0;
}
.t-main { flex: 1; min-width: 0; }
.t-name { font-size: 13.5px; font-weight: 600; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.t-meta { font-size: 11.5px; color: var(--text-3); margin-top: 3px; display: flex; gap: 8px; align-items: center; }
code {
  background: var(--surface-2);
  padding: 1px 6px; border-radius: 3px;
  font-family: var(--ff-mono); font-size: 11px;
}
code.mono { color: var(--text-2); }
.org-row.selected code { background: var(--surface); }

.status-tag {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 1px 8px; border-radius: 999px;
  font-size: 11px; font-weight: 600;
}
.status-tag .dot { width: 5px; height: 5px; background: currentColor; border-radius: 50%; }
.status-active { background: var(--success-soft); color: var(--success); }
.status-suspended { background: var(--warning-soft); color: var(--warning); }
.status-closed { background: var(--danger-soft); color: var(--danger); }

.row-actions { display: flex; gap: 4px; flex-shrink: 0; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger { color: var(--warning); }
.btn-sm.danger:hover { background: var(--warning-soft); }
.muted { color: var(--text-4); }

/* RIGHT pane */
.dept-pane { min-width: 0; }
.org-banner { display: flex; flex-direction: column; gap: 2px; }
.org-banner-name { font-size: 15px; font-weight: 600; color: var(--text); }
.org-banner-slug { font-size: 11.5px; color: var(--text-3); }

/* modal */
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
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.form-error {
  font-size: 12.5px;
  color: var(--danger);
  background: var(--danger-soft);
  padding: 8px 10px;
  border-radius: 6px;
}
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; display: inline-flex; align-items: center; justify-content: center; transition: background .15s ease, border-color .15s ease, color .15s ease; }
.btn:hover { background: var(--bg); }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-ghost { background: var(--surface); }

/* Inline button spinner */
.btn-spinner {
  display: inline-block; width: 11px; height: 11px; margin-right: 5px; vertical-align: -1px;
  border: 2px solid var(--border-strong); border-top-color: var(--primary);
  border-radius: 50%; animation: ptv-spin .6s linear infinite;
}
.btn-spinner.light { border-color: color-mix(in srgb, currentColor 45%, transparent); border-top-color: currentColor; }
@keyframes ptv-spin { to { transform: rotate(360deg); } }

@media (max-width: 900px) {
  .two-pane { grid-template-columns: 1fr; }
  .org-pane { max-height: none; }
  .head-actions { flex-wrap: wrap; }
}
@media (prefers-reduced-motion: reduce) {
  .org-row, .btn { transition: none; }
  .btn-spinner { animation-duration: 1.2s; }
}
</style>
