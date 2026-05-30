<template>
  <div class="dtm">
    <div class="dtm-head">
      <slot name="head-left" />
      <div class="dtm-actions">
        <button class="btn btn-ghost btn-sm" @click="reload">刷新</button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="exportCsv">导出</button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="importOpen = true">导入</button>
        <button class="btn btn-primary btn-sm" v-perm="writePerm" @click="openCreate(null)">+ 新建部门</button>
      </div>
    </div>

    <div class="split">
      <!-- Left: department tree -->
      <article class="card tree-pane">
        <div class="pane-head">组织架构 · {{ flatCount }} 个节点</div>
        <div class="tree-search">
          <input class="input search" v-model="treeFilter" placeholder="搜索部门" />
        </div>
        <TreeView
          :nodes="tree"
          :selected-id="selectedId"
          :filter="treeFilter"
          id-key="id"
          label-key="name"
          @select="select"
        >
          <template #label="{ node }">
            <span class="dept-node">
              {{ node.name }}
              <span v-if="node.is_root" class="tag-root">根组织</span>
              <span v-if="node.status !== 'active'" class="tag-off">停用</span>
            </span>
          </template>
        </TreeView>
        <div v-if="tree.length === 0" class="tree-empty-hint">尚无组织节点。</div>
      </article>

      <!-- Right: detail / form -->
      <article class="card detail-pane">
        <EmptyState v-if="!selected && mode === 'view'" title="选择一个部门" sub="从左侧选择部门查看或编辑，或新建子部门。" icon="◫" />

        <template v-else>
          <div class="detail-head">
            <h2 class="detail-title">
              {{ mode === 'create' ? (form.parent_id ? '新建子部门' : '新建部门') : form.name }}
            </h2>
            <div v-if="mode === 'view' && selected" class="detail-tools">
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openCreate(selected.id)">+ 子部门</button>
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="startEdit">编辑</button>
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
            <div><dt>部门名称</dt><dd>{{ selected.name }}</dd></div>
            <div><dt>上级部门</dt><dd>{{ parentName(selected) }}</dd></div>
            <div><dt>排序</dt><dd>{{ selected.order_num }}</dd></div>
            <div><dt>负责人</dt><dd>{{ selected.leader || '—' }}</dd></div>
            <div><dt>电话</dt><dd>{{ selected.phone || '—' }}</dd></div>
            <div><dt>邮箱</dt><dd>{{ selected.email || '—' }}</dd></div>
            <div><dt>状态</dt><dd>
              <span class="badge" :class="selected.status === 'active' ? 'badge-success' : 'badge-neutral'">
                {{ selected.status === 'active' ? '正常' : '停用' }}
              </span>
            </dd></div>
            <div><dt>创建时间</dt><dd class="mono">{{ selected.created_at }}</dd></div>
          </dl>

          <!-- Create / edit form -->
          <form v-else class="form" @submit.prevent="save">
            <label class="field">
              <span class="label">部门名称 *</span>
              <input class="input" v-model="form.name" required placeholder="例如：研发部" />
            </label>
            <label class="field">
              <span class="label">上级部门</span>
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
                <span class="label">电话</span>
                <input class="input" v-model="form.phone" placeholder="可选" />
              </label>
              <label class="field">
                <span class="label">邮箱</span>
                <input class="input" v-model="form.email" placeholder="可选" />
              </label>
            </div>
            <div v-if="formError" class="form-error">{{ formError }}</div>
            <div class="form-actions">
              <button class="btn btn-ghost" type="button" @click="cancelForm">取消</button>
              <button class="btn btn-primary" type="submit" :disabled="busy">{{ busy ? '保存中…' : '保存' }}</button>
            </div>
          </form>
        </template>
      </article>
    </div>

    <ImportDialog
      v-model:open="importOpen"
      title="导入部门"
      :template-url="templateUrl"
      :import-url="importUrl"
      template-name="departments_template.csv"
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
import ImportDialog, { type BulkResult } from "./ImportDialog.vue";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

// Shared department row / tree shapes. Kept local so the component does not
// couple to any one module's api typing (tenant + platform reuse this).
export interface DeptRow {
  id: string;
  name: string;
  parent_id?: string | null;
  order_num: number;
  leader?: string;
  phone?: string;
  email?: string;
  status: string;
  is_root?: boolean;
  created_at: string;
}
export interface DeptTreeRow extends DeptRow {
  children?: DeptTreeRow[];
}
export interface CreateDeptPayload {
  name: string; parent_id?: string | null; order_num?: number;
  leader?: string; phone?: string; email?: string;
}
export type UpdateDeptPatch = Partial<Pick<DeptRow, "name" | "order_num" | "leader" | "phone" | "email" | "status">>;

// The api-adapter object decouples this component from how the endpoints are
// wired (tenant `/admin/depts*` vs platform `/platform/orgs/:tid/depts*`). The
// owning view supplies the concrete funcs; CSV import/export use plain URLs
// (export via the adapter, import + template via ImportDialog's URL props).
export interface DeptApi {
  fetchTree(): Promise<DeptTreeRow[]>;
  fetchFlat(): Promise<DeptRow[]>;
  create(payload: CreateDeptPayload): Promise<DeptRow>;
  update(id: string, patch: UpdateDeptPatch): Promise<void>;
  remove(id: string): Promise<void>;
  move(id: string, parentId: string | null): Promise<void>;
  exportCsv(): Promise<void>;
}

const props = withDefaults(
  defineProps<{
    /** concrete dept endpoints (tenant or a specific org). */
    api: DeptApi;
    /** API path the ImportDialog fetches the template CSV from. */
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
const formError = ref("");
const treeFilter = ref("");
const importOpen = ref(false);

const flatCount = computed(() => flat.value.length);
const selected = computed(() => flat.value.find((d) => d.id === selectedId.value) ?? null);

const form = reactive({
  name: "", parent_id: null as string | null, order_num: 0,
  leader: "", phone: "", email: "", status: "active",
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
  reload();
});

async function reload() {
  [tree.value, flat.value] = await Promise.all([props.api.fetchTree(), props.api.fetchFlat()]);
  if (selectedId.value && !flat.value.some((d) => d.id === selectedId.value)) {
    selectedId.value = null;
  }
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
  Object.assign(form, { name: "", parent_id: parentId, order_num: 0, leader: "", phone: "", email: "", status: "active" });
}

function startEdit() {
  if (!selected.value) return;
  mode.value = "edit";
  formError.value = "";
  Object.assign(form, {
    name: selected.value.name,
    parent_id: selected.value.parent_id ?? null,
    order_num: selected.value.order_num,
    leader: selected.value.leader ?? "",
    phone: selected.value.phone ?? "",
    email: selected.value.email ?? "",
    status: selected.value.status,
  });
}

function cancelForm() {
  mode.value = "view";
  formError.value = "";
}

async function save() {
  if (!form.name.trim()) { formError.value = "部门名称不能为空"; return; }
  busy.value = true; formError.value = "";
  try {
    if (mode.value === "create") {
      const d = await props.api.create({
        name: form.name.trim(), parent_id: form.parent_id, order_num: form.order_num,
        leader: form.leader, phone: form.phone, email: form.email,
      });
      await reload();
      selectedId.value = d.id;
      mode.value = "view";
      notify.success("部门已创建");
    } else if (mode.value === "edit" && selected.value) {
      const id = selected.value.id;
      const origParent = selected.value.parent_id ?? null;
      const patch: UpdateDeptPatch = {
        name: form.name.trim(), order_num: form.order_num,
        leader: form.leader, phone: form.phone, email: form.email,
      };
      if (!selected.value.is_root) patch.status = form.status;
      await props.api.update(id, patch);
      // Reparent separately via the dedicated move endpoint (cycle-checked server-side).
      if (!selected.value.is_root && (form.parent_id ?? null) !== origParent) {
        await props.api.move(id, form.parent_id);
      }
      await reload();
      mode.value = "view";
      notify.success("部门已更新");
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
  const ok = await confirm({ title: "删除部门", message: `确认删除「${selected.value.name}」？子部门或在职成员存在时无法删除。`, danger: true });
  if (!ok) return;
  try {
    await props.api.remove(selected.value.id);
    selectedId.value = null;
    mode.value = "view";
    await reload();
    notify.success("部门已删除");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

async function exportCsv() {
  try {
    await props.api.exportCsv();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "导出失败");
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
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 10px; }
.tree-search { padding: 0 4px 8px; }
.tree-search .search { width: 100%; font-size: 13px; padding: 6px 10px; box-sizing: border-box; }
.tree-empty-hint { color: var(--text-4); font-size: 12.5px; padding: 12px 8px; }
.breadcrumb { display: flex; flex-wrap: wrap; align-items: center; font-size: 12px; color: var(--text-3); margin-bottom: 14px; }
.crumb { display: inline-flex; align-items: center; }
.crumb-sep { margin: 0 6px; color: var(--text-4); }
.dept-node { display: inline-flex; align-items: center; gap: 6px; }
.tag-root { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--primary-soft); color: var(--primary); border-radius: 3px; }
.tag-off { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--bg-deep); color: var(--text-3); border-radius: 3px; }

.detail-pane { padding: 20px 22px; min-height: 320px; }
.detail-head { display: flex; justify-content: space-between; align-items: center; margin-bottom: 18px; }
.detail-title { font-size: 17px; font-weight: 600; }
.detail-tools { display: flex; gap: 6px; }

.info-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 16px 24px; }
.info-grid dt { font-size: 11.5px; color: var(--text-3); margin-bottom: 3px; }
.info-grid dd { font-size: 13.5px; color: var(--text); }
.mono { font-family: var(--ff-mono); font-size: 12.5px; color: var(--text-2); }

.form { display: flex; flex-direction: column; gap: 14px; max-width: 520px; }
.form-row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 5px; }
.label { font-size: 12px; font-weight: 500; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.form-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.badge-success { background: var(--success-soft); color: var(--success); }
.badge-neutral { background: var(--bg-deep); color: var(--text-3); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-ghost { background: var(--surface); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }
</style>
