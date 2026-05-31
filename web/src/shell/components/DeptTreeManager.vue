<template>
  <div class="dtm">
    <div class="dtm-head">
      <slot name="head-left" />
      <div class="dtm-actions">
        <button class="btn btn-ghost btn-sm" :disabled="loading" @click="reload">
          <span v-if="loading" class="btn-spinner" aria-hidden="true" />刷新
        </button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" :disabled="exporting" @click="exportCsv">
          <span v-if="exporting" class="btn-spinner" aria-hidden="true" />{{ exporting ? '导出中…' : '导出' }}
        </button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="importOpen = true">导入</button>
        <button class="btn btn-primary btn-sm" v-perm="writePerm" @click="openCreate(null)">+ 新建组织</button>
      </div>
    </div>

    <div class="split">
      <!-- Left: department tree -->
      <article class="card tree-pane">
        <div class="pane-head-row">
          <div class="pane-head">组织架构 · {{ flatCount }} 个节点</div>
          <div class="tree-tools">
            <button class="tree-tool-btn" type="button" title="展开全部" @click="expandAll">展开全部</button>
            <button class="tree-tool-btn" type="button" title="收起全部" @click="collapseAll">收起全部</button>
          </div>
        </div>
        <div class="tree-search">
          <div class="search-wrap">
            <input class="input search" v-model="treeFilter" placeholder="搜索组织名称 / 编码" />
            <button v-if="treeFilter" class="search-clear" type="button" aria-label="清除搜索" @click="clearTreeFilter">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
            </button>
          </div>
          <select class="input status-filter" v-model="statusFilter">
            <option value="">全部状态</option>
            <option value="active">正常</option>
            <option value="disabled">停用</option>
          </select>
        </div>

        <!-- Loading: shimmer rows -->
        <div v-if="loading && tree.length === 0" class="tree-skeleton" role="status" aria-label="加载中">
          <SkeletonLoader :lines="7" :height="20" :last-short="false" />
        </div>

        <template v-else>
          <TreeView
            :nodes="tree"
            :selected-id="selectedId"
            :filter="''"
            :expand-signal="expandSignal"
            id-key="id"
            label-key="name"
            @select="select"
          >
            <template #label="{ node }">
              <span class="dept-node">
                {{ node.name }}
                <code v-if="node.org_code" class="node-code">{{ node.org_code }}</code>
                <span v-if="node.is_root" class="tag-root">根组织</span>
                <span v-if="node.status !== 'active'" class="tag-off">停用</span>
              </span>
            </template>
            <template #empty>
              <span v-if="hasTreeFilter">没有匹配结果 · 试试调整搜索或筛选</span>
              <span v-else>暂无部门</span>
            </template>
          </TreeView>

          <!-- No data at all (no filter active) → offer to create. -->
          <EmptyState
            v-if="tree.length === 0 && !hasTreeFilter"
            title="暂无部门"
            sub="新建第一个组织节点开始搭建层级"
            icon="◫"
          >
            <template #actions>
              <button class="btn btn-primary btn-sm" v-perm="writePerm" @click="openCreate(null)">+ 新建一个</button>
            </template>
          </EmptyState>
        </template>
      </article>

      <!-- Right: detail / form -->
      <article class="card detail-pane">
        <EmptyState v-if="!selected && mode === 'view'" title="从左侧选择组织 / 部门" sub="选择左侧任一节点查看或编辑，或新建下级组织。" icon="◫" />

        <template v-else>
          <div class="detail-head">
            <h2 class="detail-title">
              {{ mode === 'create' ? (form.parent_id ? '新建下级组织' : '新建组织') : form.name }}
            </h2>
            <div v-if="mode === 'view' && selected" class="detail-tools">
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openCreate(selected.id)">+ 下级</button>
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="startEdit">编辑</button>
              <button v-if="!selected.is_root" class="btn btn-ghost btn-sm" v-perm="writePerm" @click="toggleStatus(selected)">
                {{ selected.status === 'active' ? '禁用' : '启用' }}
              </button>
              <button v-if="!selected.is_root" class="btn btn-ghost btn-sm danger" v-perm="writePerm" @click="removeDept">删除</button>
            </div>
          </div>

          <!-- View mode -->
          <div v-if="mode === 'view' && selected" class="breadcrumb">
            <span v-for="(seg, i) in breadcrumb(selected)" :key="i" class="crumb">
              <span v-if="i > 0" class="crumb-sep">/</span>{{ seg }}
            </span>
          </div>
          <dl v-if="mode === 'view' && selected" class="info-grid">
            <div><dt>组织名称</dt><dd>{{ selected.name }}</dd></div>
            <div><dt>组织编码</dt><dd><code>{{ selected.org_code }}</code></dd></div>
            <div><dt>上级组织</dt><dd>{{ parentName(selected) }}</dd></div>
            <div><dt>组织类型</dt><dd>{{ orgTypeLabel(selected.org_type) }}</dd></div>
            <div><dt>排序</dt><dd>{{ selected.order_num }}</dd></div>
            <div><dt>负责人</dt><dd>{{ selected.leader || '—' }}</dd></div>
            <div><dt>负责人账号</dt><dd>{{ selected.leader_account || '—' }}</dd></div>
            <div><dt>电话</dt><dd>{{ selected.phone || '—' }}</dd></div>
            <div><dt>邮箱</dt><dd>{{ selected.email || '—' }}</dd></div>
            <div><dt>状态</dt><dd>
              <span class="badge" :class="selected.status === 'active' ? 'badge-success' : 'badge-neutral'">
                {{ selected.status === 'active' ? '正常' : '停用' }}
              </span>
            </dd></div>
            <div><dt>创建时间</dt><dd class="mono">{{ selected.created_at }}</dd></div>
            <div class="wide"><dt>组织路径</dt><dd>{{ selected.path || breadcrumb(selected).join('/') }}</dd></div>
            <div class="wide"><dt>备注</dt><dd>{{ selected.remark || '—' }}</dd></div>
          </dl>
          <section v-if="mode === 'view' && selected" class="children-panel">
            <div class="children-head">
              <span>下级组织</span>
              <span class="muted">{{ selectedChildren.length }} 个</span>
            </div>
            <div v-if="selectedChildren.length === 0" class="children-empty">暂无下级组织。</div>
            <div v-else class="children-scroll">
              <table class="children-table">
                <thead><tr><th>名称</th><th>编码</th><th>类型</th><th>状态</th><th class="ta-right">排序</th></tr></thead>
                <tbody>
                  <tr v-for="child in selectedChildren" :key="child.id" @click="select(child.id)">
                    <td>{{ child.name }}</td>
                    <td><code>{{ child.org_code }}</code></td>
                    <td>{{ orgTypeLabel(child.org_type) }}</td>
                    <td>{{ child.status === 'active' ? '正常' : '停用' }}</td>
                    <td class="ta-right tabular-nums">{{ child.order_num }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </section>

          <!-- Create / edit form -->
          <form v-else class="form" @submit.prevent="save">
            <label class="field">
              <span class="label">组织名称 *</span>
              <input
                class="input"
                :class="{ invalid: fieldErr.name }"
                v-model="form.name"
                required
                placeholder="例如：研发部"
                :aria-invalid="!!fieldErr.name"
                @blur="validateField('name')"
                @input="fieldErr.name && validateField('name')"
              />
              <span v-if="fieldErr.name" class="field-error" role="alert">{{ fieldErr.name }}</span>
            </label>
            <div class="form-row-2">
              <label class="field">
                <span class="label">组织编码 *</span>
                <input
                  class="input"
                  :class="{ invalid: fieldErr.org_code }"
                  v-model="form.org_code"
                  required
                  placeholder="例如：RD"
                  :aria-invalid="!!fieldErr.org_code"
                  @blur="validateField('org_code')"
                  @input="fieldErr.org_code && validateField('org_code')"
                />
                <span v-if="fieldErr.org_code" class="field-error" role="alert">{{ fieldErr.org_code }}</span>
              </label>
              <label class="field">
                <span class="label">组织类型</span>
                <select class="input" v-model="form.org_type">
                  <option value="unit">单位</option>
                  <option value="department">部门</option>
                  <option value="office">科室</option>
                  <option value="team">小组</option>
                </select>
              </label>
            </div>
            <label class="field">
              <span class="label">上级组织</span>
              <select class="input" v-model="form.parent_id" :disabled="mode === 'edit' && selected?.is_root">
                <option :value="null">（挂到根组织）</option>
                <option v-for="d in selectableParents" :key="d.id" :value="d.id">{{ indentName(d) }}</option>
              </select>
            </label>
            <div class="form-row-2">
              <label class="field">
                <span class="label">排序</span>
                <input class="input" type="number" v-model.number="form.order_num" />
              </label>
              <label class="field" v-if="mode === 'edit'">
                <span class="label">状态</span>
                <select class="input" v-model="form.status" :disabled="selected?.is_root">
                  <option value="active">正常</option>
                  <option value="disabled">停用</option>
                </select>
              </label>
            </div>
            <label class="field">
              <span class="label">负责人</span>
              <input class="input" v-model="form.leader" placeholder="可选" />
            </label>
            <div class="form-row-2">
              <label class="field">
                <span class="label">负责人账号</span>
                <input class="input" v-model="form.leader_account" placeholder="可选" />
              </label>
              <label class="field">
                <span class="label">电话</span>
                <input class="input" v-model="form.phone" placeholder="可选" />
              </label>
            </div>
            <div class="form-row-2">
              <label class="field">
                <span class="label">邮箱</span>
                <input class="input" v-model="form.email" placeholder="可选" />
              </label>
              <label class="field">
                <span class="label">备注</span>
                <input class="input" v-model="form.remark" placeholder="可选" />
              </label>
            </div>
            <div v-if="formError" class="form-error" role="alert">{{ formError }}</div>
            <div class="form-actions">
              <button class="btn btn-ghost" type="button" @click="cancelForm">取消</button>
              <button class="btn btn-primary" type="submit" :disabled="busy">
                <span v-if="busy" class="btn-spinner light" aria-hidden="true" />{{ busy ? '保存中…' : '保存' }}
              </button>
            </div>
          </form>
        </template>
      </article>
    </div>

    <ImportDialog
      v-model:open="importOpen"
      title="导入组织"
      :template-url="templateUrl"
      :import-url="importUrl"
      template-name="departments_template.xlsx"
      @done="onImportDone"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
// Import siblings directly (not via the ./index barrel) — the barrel re-exports
// this very component, so going through it would create a chunk-level circular
// dependency at build time.
import EmptyState from "./EmptyState.vue";
import TreeView from "./TreeView.vue";
import SkeletonLoader from "./SkeletonLoader.vue";
import ImportDialog, { type BulkResult } from "./ImportDialog.vue";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

// Shared department row / tree shapes. Kept local so the component does not
// couple to any one module's api typing (tenant + platform reuse this).
export interface DeptRow {
  id: string;
  tenant_id: string;
  name: string;
  org_code: string;
  parent_id?: string | null;
  org_type: string;
  order_num: number;
  leader?: string;
  leader_account?: string;
  phone?: string;
  email?: string;
  status: string;
  remark?: string;
  path?: string;
  is_root?: boolean;
  created_at: string;
}
export interface DeptTreeRow extends DeptRow {
  children?: DeptTreeRow[];
}
export interface DeptQuery {
  search?: string;
  status?: string;
  [key: string]: string | undefined;
}
export interface CreateDeptPayload {
  name: string; org_code: string; parent_id?: string | null; org_type?: string; order_num?: number;
  leader?: string; leader_account?: string; phone?: string; email?: string; status?: string; remark?: string;
}
export type UpdateDeptPatch = Partial<Pick<DeptRow, "name" | "org_code" | "parent_id" | "org_type" | "order_num" | "leader" | "leader_account" | "phone" | "email" | "status" | "remark">>;

// The api-adapter object decouples this component from how the endpoints are
// wired (tenant `/admin/depts*` vs platform `/platform/orgs/:tid/depts*`). The
// owning view supplies the concrete funcs; spreadsheet import/export use plain URLs
// (export via the adapter, import + template via ImportDialog's URL props).
export interface DeptApi {
  fetchTree(query?: DeptQuery): Promise<DeptTreeRow[]>;
  fetchFlat(query?: DeptQuery): Promise<DeptRow[]>;
  create(payload: CreateDeptPayload): Promise<DeptRow>;
  update(id: string, patch: UpdateDeptPatch): Promise<void>;
  setStatus(id: string, status: string, cascade?: boolean): Promise<void>;
  remove(id: string): Promise<void>;
  move(id: string, parentId: string | null): Promise<void>;
  exportCsv(query?: DeptQuery): Promise<void>;
}

const props = withDefaults(
  defineProps<{
    /** concrete dept endpoints (tenant or a specific org). */
    api: DeptApi;
    /** API path the ImportDialog fetches the template from. */
    templateUrl: string;
    /** API path the ImportDialog POSTs the multipart upload to (field "file"). */
    importUrl: string;
    /** button-level RBAC key for write actions (e.g. "dept:write" / "org:write"). */
    writePerm?: string;
  }>(),
  { writePerm: "dept:write" },
);

const notify = useNotification();
const { confirm } = useConfirm();

const tree = ref<DeptTreeRow[]>([]);
const flat = ref<DeptRow[]>([]);
const selectedId = ref<string | null>(null);
const mode = ref<"view" | "create" | "edit">("view");
const busy = ref(false);
const loading = ref(false);
const exporting = ref(false);
const formError = ref("");
// Per-field validation messages, shown below the field on blur / submit.
const fieldErr = reactive<{ name: string; org_code: string }>({ name: "", org_code: "" });
const treeFilter = ref("");
const statusFilter = ref("");
const importOpen = ref(false);
// Expand/collapse-all signal handed to TreeView (sign flips each click).
const expandSignal = ref(0);
let reloadTimer: number | undefined;

// True when a tree search / status filter is active (drives the "no match" copy).
const hasTreeFilter = computed(() => !!treeFilter.value.trim() || !!statusFilter.value);

const flatCount = computed(() => flat.value.length);
const selected = computed(() => flat.value.find((d) => d.id === selectedId.value) ?? null);
const selectedChildren = computed(() => selected.value ? flat.value.filter((d) => d.parent_id === selected.value!.id) : []);

const form = reactive({
  name: "", org_code: "", parent_id: null as string | null, org_type: "department", order_num: 0,
  leader: "", leader_account: "", phone: "", email: "", status: "active", remark: "",
});

// Parents selectable in the form: in edit mode, exclude self + descendants (no cycles).
const selectableParents = computed(() => {
  if (mode.value !== "edit" || !selected.value) return flat.value;
  const banned = new Set<string>([selected.value.id, ...descendantIds(selected.value.id)]);
  return flat.value.filter((d) => !banned.has(d.id));
});

function descendantIds(rootId: string): string[] {
  const childrenOf = new Map<string, string[]>();
  for (const d of flat.value) {
    if (d.parent_id) {
      const arr = childrenOf.get(d.parent_id) ?? [];
      arr.push(d.id);
      childrenOf.set(d.parent_id, arr);
    }
  }
  const out: string[] = [];
  const queue = [...(childrenOf.get(rootId) ?? [])];
  while (queue.length) {
    const cur = queue.shift()!;
    out.push(cur);
    queue.push(...(childrenOf.get(cur) ?? []));
  }
  return out;
}

onMounted(reload);
// Re-load whenever the bound api adapter changes (e.g. platform switches org).
// Reset transient view state so the previous org's selection/form does not leak.
watch(() => props.api, () => {
  selectedId.value = null;
  mode.value = "view";
  formError.value = "";
  treeFilter.value = "";
  statusFilter.value = "";
  reload();
});
watch([treeFilter, statusFilter], () => {
  if (reloadTimer) window.clearTimeout(reloadTimer);
  reloadTimer = window.setTimeout(() => reload(), 250);
});

async function reload() {
  const query = currentQuery();
  loading.value = true;
  try {
    [tree.value, flat.value] = await Promise.all([props.api.fetchTree(query), props.api.fetchFlat()]);
    if (selectedId.value && !flat.value.some((d) => d.id === selectedId.value)) {
      selectedId.value = null;
    }
  } finally {
    loading.value = false;
  }
}

function expandAll() { expandSignal.value = Math.abs(expandSignal.value) + 1; }
function collapseAll() { expandSignal.value = -(Math.abs(expandSignal.value) + 1); }
function clearTreeFilter() { treeFilter.value = ""; }

function currentQuery(): DeptQuery {
  return {
    search: treeFilter.value.trim() || undefined,
    status: statusFilter.value || undefined,
  };
}

function select(id: string) {
  selectedId.value = id;
  mode.value = "view";
  formError.value = "";
}

function parentName(d: DeptRow): string {
  if (d.is_root) return "—";
  if (!d.parent_id) return "根组织";
  return flat.value.find((x) => x.id === d.parent_id)?.name ?? "—";
}

function orgTypeLabel(t: string): string {
  return ({ unit: "单位", department: "部门", office: "科室", team: "小组" } as Record<string, string>)[t] ?? (t || "部门");
}

// Full ancestor → self name chain, used for the detail-panel breadcrumb.
function breadcrumb(d: DeptRow): string[] {
  const segs: string[] = [];
  let cur: DeptRow | undefined = d;
  let depth = 0;
  while (cur && depth < 20) {
    segs.unshift(cur.name);
    cur = cur.parent_id ? flat.value.find((x) => x.id === cur!.parent_id) : undefined;
    depth++;
  }
  return segs;
}

// Re-fetch the tree after a bulk import so newly created departments appear.
async function onImportDone(_r: BulkResult) {
  await reload();
}

// indented display name in the parent <select> for a sense of hierarchy.
function indentName(d: DeptRow): string {
  let depth = 0;
  let cur: DeptRow | undefined = d;
  while (cur?.parent_id) {
    cur = flat.value.find((x) => x.id === cur!.parent_id);
    depth++;
    if (depth > 20) break;
  }
  return "　".repeat(depth) + d.name;
}

function openCreate(parentId: string | null) {
  mode.value = "create";
  formError.value = "";
  fieldErr.name = ""; fieldErr.org_code = "";
  Object.assign(form, {
    name: "", org_code: "", parent_id: parentId, org_type: "department", order_num: 0,
    leader: "", leader_account: "", phone: "", email: "", status: "active", remark: "",
  });
}

function startEdit() {
  if (!selected.value) return;
  mode.value = "edit";
  formError.value = "";
  fieldErr.name = ""; fieldErr.org_code = "";
  Object.assign(form, {
    name: selected.value.name,
    org_code: selected.value.org_code,
    parent_id: selected.value.parent_id ?? null,
    org_type: selected.value.org_type || "department",
    order_num: selected.value.order_num,
    leader: selected.value.leader ?? "",
    leader_account: selected.value.leader_account ?? "",
    phone: selected.value.phone ?? "",
    email: selected.value.email ?? "",
    status: selected.value.status,
    remark: selected.value.remark ?? "",
  });
}

function cancelForm() {
  mode.value = "view";
  formError.value = "";
  fieldErr.name = ""; fieldErr.org_code = "";
}

function validateField(name: "name" | "org_code") {
  if (name === "name") fieldErr.name = form.name.trim() ? "" : "组织名称不能为空";
  if (name === "org_code") fieldErr.org_code = form.org_code.trim() ? "" : "组织编码不能为空";
}

async function save() {
  validateField("name");
  validateField("org_code");
  if (fieldErr.name || fieldErr.org_code) { formError.value = ""; return; }
  busy.value = true; formError.value = "";
  try {
    if (mode.value === "create") {
      const d = await props.api.create({
        name: form.name.trim(), org_code: form.org_code.trim(), parent_id: form.parent_id,
        org_type: form.org_type, order_num: form.order_num,
        leader: form.leader, leader_account: form.leader_account,
        phone: form.phone, email: form.email, status: form.status, remark: form.remark,
      });
      await reload();
      selectedId.value = d.id;
      mode.value = "view";
      notify.success("组织已创建");
    } else if (mode.value === "edit" && selected.value) {
      const id = selected.value.id;
      const origParent = selected.value.parent_id ?? null;
      const origStatus = selected.value.status;
      let cascade = false;
      if (form.status !== origStatus && form.status === "disabled" && descendantIds(id).length > 0) {
        cascade = await confirm({ title: "同步禁用", message: "该组织存在下级组织，是否同步禁用全部下级组织？取消则仅禁用当前组织。", danger: true });
      }
      const patch: UpdateDeptPatch = {
        name: form.name.trim(), org_code: form.org_code.trim(), org_type: form.org_type,
        order_num: form.order_num, leader: form.leader, leader_account: form.leader_account,
        phone: form.phone, email: form.email, remark: form.remark,
      };
      await props.api.update(id, patch);
      // Reparent separately via the dedicated move endpoint (cycle-checked server-side).
      if (!selected.value.is_root && (form.parent_id ?? null) !== origParent) {
        await props.api.move(id, form.parent_id);
      }
      if (!selected.value.is_root && form.status !== origStatus) {
        await props.api.setStatus(id, form.status, cascade);
      }
      await reload();
      mode.value = "view";
      notify.success("组织已更新");
    }
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}

async function removeDept() {
  if (!selected.value) return;
  if (selected.value.is_root) {
    notify.error("根组织不能删除");
    return;
  }
  const ok = await confirm({ title: "删除组织", message: `确认删除「${selected.value.name}」？存在下级组织或成员时无法删除。`, danger: true });
  if (!ok) return;
  try {
    await props.api.remove(selected.value.id);
    selectedId.value = null;
    mode.value = "view";
    await reload();
    notify.success("组织已删除");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

async function exportCsv() {
  if (exporting.value) return;
  exporting.value = true;
  try {
    await props.api.exportCsv(currentQuery());
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "导出失败");
  } finally {
    exporting.value = false;
  }
}

async function toggleStatus(d: DeptRow) {
  if (d.is_root) return;
  const next = d.status === "active" ? "disabled" : "active";
  let cascade = false;
  if (next === "disabled" && descendantIds(d.id).length > 0) {
    cascade = await confirm({ title: "同步禁用", message: "该组织存在下级组织，是否同步禁用全部下级组织？取消则仅禁用当前组织。", danger: true });
  }
  try {
    await props.api.setStatus(d.id, next, cascade);
    await reload();
    notify.success(next === "active" ? "组织已启用" : "组织已禁用");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "状态更新失败");
  }
}

// Let the owning view force a refresh (e.g. after external changes).
defineExpose({ reload });
</script>

<style scoped>
.dtm { display: flex; flex-direction: column; gap: 14px; }
.dtm-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.dtm-actions { display: flex; gap: 8px; margin-left: auto; }

.split { display: grid; grid-template-columns: 320px 1fr; gap: 16px; align-items: start; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.tree-pane { padding: 12px; }
.pane-head-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 10px; }
.tree-tools { display: flex; gap: 4px; padding: 0 4px 6px; }
.tree-tool-btn {
  border: 1px solid var(--border); background: var(--surface); color: var(--text-3);
  font-size: 11px; padding: 3px 7px; border-radius: var(--r-sm); cursor: pointer;
  transition: background .15s ease, color .15s ease, border-color .15s ease;
}
.tree-tool-btn:hover { background: var(--surface-2); color: var(--primary); border-color: var(--primary); }
.tree-search { padding: 0 4px 8px; display: grid; grid-template-columns: 1fr 104px; gap: 8px; }
.search-wrap { position: relative; min-width: 0; }
.tree-search .search { width: 100%; font-size: 13px; padding: 6px 28px 6px 10px; box-sizing: border-box; }
.search-clear {
  position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
  border: 0; background: transparent; color: var(--text-4); cursor: pointer;
  width: 18px; height: 18px; display: grid; place-items: center; border-radius: var(--r-sm);
  transition: color .15s ease, background .15s ease;
}
.search-clear:hover { color: var(--text-2); background: var(--surface-2); }
.status-filter { font-size: 12px; padding: 6px 8px; }
.tree-skeleton { padding: 8px; }
.breadcrumb { display: flex; flex-wrap: wrap; align-items: center; font-size: 12px; color: var(--text-3); margin-bottom: 14px; }
.crumb { display: inline-flex; align-items: center; }
.crumb-sep { margin: 0 6px; color: var(--text-4); }
.dept-node { display: inline-flex; align-items: center; gap: 6px; }
.node-code { font-size: 10.5px; color: var(--text-3); font-weight: 600; }
.tag-root { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--primary-soft); color: var(--primary); border-radius: 3px; }
.tag-off { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--bg-deep); color: var(--text-3); border-radius: 3px; }

.detail-pane { padding: 20px 22px; min-height: 320px; }
.detail-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.detail-title { font-size: 17px; font-weight: 600; }
.detail-tools { display: flex; gap: 6px; }

.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px 24px; }
.info-grid .wide { grid-column: 1 / -1; }
.info-grid dt { font-size: 11.5px; color: var(--text-3); margin-bottom: 3px; }
.info-grid dd { font-size: 13.5px; color: var(--text); }
.mono { font-family: var(--ff-mono); font-size: 12.5px; color: var(--text-2); }

.children-panel { margin-top: 22px; border-top: 1px solid var(--border); padding-top: 14px; }
.children-head { display: flex; align-items: center; justify-content: space-between; font-size: 12px; font-weight: 700; color: var(--text-2); margin-bottom: 8px; }
.muted { color: var(--text-3); font-weight: 500; }
.children-empty { font-size: 12.5px; color: var(--text-4); padding: 8px 0; }
.children-table { width: 100%; border-collapse: collapse; font-size: 12.5px; }
.children-scroll { overflow-x: auto; }
.children-table th { text-align: left; color: var(--text-3); font-weight: 600; padding: 7px 8px; border-bottom: 1px solid var(--border); }
.children-table td { padding: 8px; border-bottom: 1px solid var(--border-soft); color: var(--text-2); }
.children-table .ta-right { text-align: right; }
.children-table .tabular-nums { font-variant-numeric: tabular-nums; }
.children-table tbody tr { cursor: pointer; transition: background .15s ease; }
.children-table tbody tr:hover { background: var(--surface-2); }

.form { display: flex; flex-direction: column; gap: 14px; max-width: 520px; }
.form-row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 5px; }
.label { font-size: 12px; font-weight: 500; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); transition: border-color .15s ease, box-shadow .15s ease; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input.invalid { border-color: var(--danger); }
.input.invalid:focus { outline-color: var(--danger-soft); }
.field-error { font-size: 11.5px; color: var(--danger); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.form-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.badge-success { background: var(--success-soft); color: var(--success); }
.badge-neutral { background: var(--bg-deep); color: var(--text-3); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; display: inline-flex; align-items: center; justify-content: center; transition: background .15s ease, border-color .15s ease, color .15s ease; }
.btn:hover { background: var(--bg); }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-ghost { background: var(--surface); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }

/* Inline button spinner */
.btn-spinner {
  display: inline-block; width: 12px; height: 12px; margin-right: 6px; vertical-align: -1px;
  border: 2px solid var(--border-strong); border-top-color: var(--primary);
  border-radius: 50%; animation: dtm-spin .6s linear infinite;
}
.btn-spinner.light { border-color: color-mix(in srgb, currentColor 45%, transparent); border-top-color: currentColor; }
@keyframes dtm-spin { to { transform: rotate(360deg); } }

@media (max-width: 900px) {
  .split { grid-template-columns: 1fr; }
  .dtm-head { flex-wrap: wrap; }
}
@media (prefers-reduced-motion: reduce) {
  .btn, .input, .tree-tool-btn, .search-clear, .children-table tbody tr { transition: none; }
  .btn-spinner { animation-duration: 1.2s; }
}
</style>
