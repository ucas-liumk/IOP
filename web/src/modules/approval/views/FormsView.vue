<template>
  <section class="admin-page">
    <PageHeader title="审批模板" sub="自定义表单字段 + 审批流程（多级审批 / 抄送 / 会签 或签）">
      <template #actions>
        <button class="btn btn-ghost" @click="load">刷新</button>
        <button class="btn btn-primary" v-perm="'approval.form:manage'" @click="openCreate">+ 新建模板</button>
      </template>
    </PageHeader>

    <div v-if="loading" class="ap-loading">加载中…</div>
    <EmptyState v-else-if="forms.length === 0" title="还没有审批模板" sub="新建一个模板供成员发起审批" icon="▤" />

    <ul v-else class="ap-tpl-list">
      <li v-for="f in forms" :key="f.id" class="ap-tpl-row card">
        <div class="ap-tpl-info">
          <div class="ap-tpl-name">
            {{ f.name }}
            <span class="badge" :class="f.status === 'active' ? 'badge-success' : 'badge-info'">
              {{ f.status === "active" ? "启用" : "停用" }}
            </span>
          </div>
          <div class="ap-tpl-meta">{{ f.fields.length }} 个字段 · {{ f.flow.length }} 个流程节点 · {{ f.description || "无描述" }}</div>
        </div>
        <div class="ap-tpl-ops" v-perm="'approval.form:manage'">
          <button class="btn btn-ghost btn-sm" @click="openEdit(f)">编辑</button>
          <button class="btn btn-ghost btn-sm danger" @click="remove(f)">删除</button>
        </div>
      </li>
    </ul>

    <!-- Editor drawer -->
    <div v-if="editing" class="ap-drawer-mask" @click.self="editing = null">
      <div class="ap-drawer card">
        <header class="ap-drawer-head">
          <h3>{{ draft.id ? "编辑模板" : "新建模板" }}</h3>
          <button class="ap-close" @click="editing = null">×</button>
        </header>

        <div class="ap-drawer-body">
          <div class="ap-edit-grid">
            <label class="ap-edit-field">
              <span class="ap-label">模板名称 *</span>
              <input v-model="draft.name" class="input" placeholder="如 请假申请" />
            </label>
            <label class="ap-edit-field">
              <span class="ap-label">编码</span>
              <input v-model="draft.code" class="input" placeholder="leave" />
            </label>
            <label class="ap-edit-field">
              <span class="ap-label">状态</span>
              <select v-model="draft.status" class="input">
                <option value="active">启用</option>
                <option value="disabled">停用</option>
              </select>
            </label>
            <label class="ap-edit-field ap-span2">
              <span class="ap-label">描述</span>
              <input v-model="draft.description" class="input" placeholder="模板说明" />
            </label>
          </div>

          <!-- Field editor -->
          <div class="ap-section">
            <div class="ap-section-head">
              <span>表单字段</span>
              <button class="btn btn-ghost btn-sm" @click="addField">+ 添加字段</button>
            </div>
            <div v-if="draft.fields.length === 0" class="ap-hint">暂无字段，点击「添加字段」</div>
            <div v-for="(fld, i) in draft.fields" :key="i" class="ap-item">
              <input v-model="fld.label" class="input ap-item-label" placeholder="字段名" @input="syncKey(fld)" />
              <select v-model="fld.type" class="input ap-item-type">
                <option value="text">单行文本</option>
                <option value="textarea">多行文本</option>
                <option value="number">数字</option>
                <option value="date">日期</option>
                <option value="select">下拉选择</option>
                <option value="radio">单选</option>
              </select>
              <input
                v-if="fld.type === 'select' || fld.type === 'radio'"
                :value="fld.options.join(',')"
                class="input ap-item-opts"
                placeholder="选项,逗号分隔"
                @input="setOptions(fld, ($event.target as HTMLInputElement).value)"
              />
              <label class="ap-item-req"><input type="checkbox" v-model="fld.required" /> 必填</label>
              <button class="ap-item-del" @click="draft.fields.splice(i, 1)">×</button>
            </div>
          </div>

          <!-- Flow editor -->
          <div class="ap-section">
            <div class="ap-section-head">
              <span>审批流程</span>
              <button class="btn btn-ghost btn-sm" @click="addNode">+ 添加节点</button>
            </div>
            <div v-if="draft.flow.length === 0" class="ap-hint">暂无节点（无审批节点的模板将自动通过）</div>
            <div v-for="(n, i) in draft.flow" :key="i" class="ap-item ap-node">
              <span class="ap-node-idx">{{ i + 1 }}</span>
              <select v-model="n.type" class="input ap-item-type">
                <option value="approve">审批</option>
                <option value="cc">抄送</option>
              </select>
              <select v-model="n.assignee_type" class="input ap-item-type">
                <option value="user">指定成员</option>
                <option value="role">按角色</option>
                <option value="dept_leader">部门负责人</option>
              </select>
              <select v-if="n.assignee_type === 'user'" v-model="n.assignee_id" class="input ap-item-opts">
                <option value="">选择成员</option>
                <option v-for="m in members" :key="m.id" :value="m.id">{{ m.name }}{{ m.department ? ` (${m.department})` : "" }}</option>
              </select>
              <input v-else-if="n.assignee_type === 'role'" v-model="n.role_code" class="input ap-item-opts" placeholder="角色编码 如 tenant_admin" />
              <span v-else class="ap-node-fixed">发起人所在部门负责人</span>
              <select v-if="n.type === 'approve'" v-model="n.mode" class="input ap-item-mode">
                <option value="or">或签</option>
                <option value="and">会签</option>
              </select>
              <button class="ap-item-del" @click="draft.flow.splice(i, 1)">×</button>
            </div>
          </div>
        </div>

        <footer class="ap-drawer-foot">
          <button class="btn btn-ghost" @click="editing = null">取消</button>
          <button class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? "保存中…" : "保存" }}</button>
        </footer>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, EmptyState } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import * as api from "../api";
import type { Form, Field, FlowNode, MemberRef, FormPayload } from "../api";

const notify = useNotification();
const { confirm } = useConfirm();

const forms = ref<Form[]>([]);
const members = ref<MemberRef[]>([]);
const loading = ref(false);
const saving = ref(false);
const editing = ref<Form | "new" | null>(null);

const draft = reactive<{
  id: string;
  code: string;
  name: string;
  description: string;
  status: "active" | "disabled";
  fields: Field[];
  flow: FlowNode[];
}>({ id: "", code: "", name: "", description: "", status: "active", fields: [], flow: [] });

async function load() {
  loading.value = true;
  try {
    forms.value = await api.listForms(true);
  } catch {
    notify.error("加载失败");
  } finally {
    loading.value = false;
  }
}

async function loadMembers() {
  try {
    members.value = await api.listMembers();
  } catch {
    members.value = [];
  }
}

function reset() {
  draft.id = "";
  draft.code = "";
  draft.name = "";
  draft.description = "";
  draft.status = "active";
  draft.fields = [];
  draft.flow = [];
}

function openCreate() {
  reset();
  editing.value = "new";
}

function openEdit(f: Form) {
  draft.id = f.id;
  draft.code = f.code;
  draft.name = f.name;
  draft.description = f.description;
  draft.status = f.status;
  draft.fields = JSON.parse(JSON.stringify(f.fields));
  draft.flow = JSON.parse(JSON.stringify(f.flow));
  editing.value = f;
}

function addField() {
  draft.fields.push({ key: `f${Date.now()}`, label: "", type: "text", required: false, options: [] });
}
function syncKey(fld: Field) {
  if (!fld.key) fld.key = `f${Date.now()}`;
}
function setOptions(fld: Field, raw: string) {
  fld.options = raw.split(",").map((s) => s.trim()).filter(Boolean);
}

function addNode() {
  draft.flow.push({ type: "approve", assignee_type: "user", assignee_id: "", role_code: "", mode: "or" });
}

async function save() {
  if (!draft.name.trim()) {
    notify.error("请填写模板名称");
    return;
  }
  // basic per-node validation
  for (const n of draft.flow) {
    if (n.assignee_type === "user" && !n.assignee_id) {
      notify.error("请为每个「指定成员」节点选择成员");
      return;
    }
    if (n.assignee_type === "role" && !n.role_code.trim()) {
      notify.error("请为每个「按角色」节点填写角色编码");
      return;
    }
  }
  const payload: FormPayload = {
    code: draft.code,
    name: draft.name,
    description: draft.description,
    status: draft.status,
    fields: draft.fields.map((f) => ({ ...f, options: f.options ?? [] })),
    flow: draft.flow,
  };
  saving.value = true;
  try {
    if (draft.id) await api.updateForm(draft.id, payload);
    else await api.createForm(payload);
    notify.success("已保存");
    editing.value = null;
    await load();
  } catch (e: unknown) {
    notify.error(errMsg(e) || "保存失败");
  } finally {
    saving.value = false;
  }
}

async function remove(f: Form) {
  const ok = await confirm({ title: "删除模板", message: `确认删除「${f.name}」？`, danger: true, confirmText: "删除" });
  if (!ok) return;
  try {
    await api.deleteForm(f.id);
    notify.success("已删除");
    await load();
  } catch (e: unknown) {
    notify.error(errMsg(e) || "删除失败");
  }
}

function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message ?? "";
}

onMounted(() => {
  void load();
  void loadMembers();
});
</script>

<style scoped>
.ap-loading { padding: var(--sp-6); color: var(--text-3); }
.ap-tpl-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: var(--sp-3); }
.ap-tpl-row { display: flex; align-items: center; justify-content: space-between; padding: var(--sp-4) var(--sp-5); }
.ap-tpl-name { font-weight: 600; color: var(--text); display: flex; align-items: center; gap: var(--sp-2); }
.ap-tpl-meta { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.ap-tpl-ops { display: flex; gap: var(--sp-2); }

.ap-drawer-mask {
  position: fixed; inset: 0; background: rgba(0, 0, 0, 0.35);
  display: flex; justify-content: flex-end; z-index: 100;
}
.ap-drawer { width: min(720px, 100%); height: 100%; border-radius: 0; display: flex; flex-direction: column; }
.ap-drawer-head { display: flex; align-items: center; justify-content: space-between; padding: var(--sp-4) var(--sp-5); border-bottom: 1px solid var(--border-soft); }
.ap-drawer-head h3 { margin: 0; font-size: 16px; }
.ap-close { border: none; background: none; font-size: 22px; cursor: pointer; color: var(--text-3); }
.ap-drawer-body { flex: 1; overflow-y: auto; padding: var(--sp-5); }
.ap-drawer-foot { display: flex; justify-content: flex-end; gap: var(--sp-3); padding: var(--sp-4) var(--sp-5); border-top: 1px solid var(--border-soft); }

.ap-edit-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-3); }
.ap-edit-field { display: flex; flex-direction: column; gap: 6px; }
.ap-span2 { grid-column: 1 / -1; }
.ap-label { font-size: 13px; font-weight: 600; color: var(--text-2); }

.ap-section { margin-top: var(--sp-5); }
.ap-section-head { display: flex; align-items: center; justify-content: space-between; font-weight: 600; color: var(--text-2); margin-bottom: var(--sp-3); }
.ap-hint { font-size: 13px; color: var(--text-3); padding: var(--sp-3) 0; }
.ap-item { display: flex; align-items: center; gap: var(--sp-2); margin-bottom: var(--sp-2); }
.ap-item-label { flex: 1.2; }
.ap-item-type { width: 120px; }
.ap-item-opts { flex: 1.4; }
.ap-item-mode { width: 90px; }
.ap-item-req { display: inline-flex; align-items: center; gap: 4px; font-size: 13px; white-space: nowrap; }
.ap-item-del { border: none; background: none; font-size: 18px; cursor: pointer; color: var(--text-4); }
.ap-node-idx { width: 22px; height: 22px; border-radius: 50%; display: grid; place-items: center; font-size: 12px; background: var(--primary-soft); color: var(--primary); flex-shrink: 0; }
.ap-node-fixed { flex: 1.4; font-size: 13px; color: var(--text-3); }
</style>
