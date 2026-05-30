<template>
  <div class="lc-design">
    <!-- Left: form list -->
    <aside class="lc-side">
      <div class="lc-side-head">
        <span>我的表单</span>
        <button class="lc-add" title="新建表单" @click="newForm">＋</button>
      </div>
      <div class="lc-side-list">
        <button
          v-for="f in forms" :key="f.id"
          class="lc-side-item" :class="{ active: editing?.id === f.id }"
          @click="openForm(f.id)"
        >
          <span class="lc-side-ico">{{ f.icon || "📋" }}</span>
          <span class="lc-side-name">{{ f.name }}</span>
          <span class="lc-side-count">{{ f.entry_count }}</span>
        </button>
        <div v-if="forms.length === 0" class="lc-empty-side">还没有表单</div>
      </div>
    </aside>

    <!-- Right: designer -->
    <main class="lc-main" v-if="editing">
      <header class="lc-main-head">
        <div class="lc-title-row">
          <input v-model="editing.icon" class="lc-icon-input" maxlength="2" placeholder="📋" />
          <input v-model="editing.name" class="lc-name-input" placeholder="表单名称" />
        </div>
        <div class="lc-head-actions">
          <select v-model="editing.status" class="lc-status">
            <option value="active">启用</option>
            <option value="archived">归档</option>
          </select>
          <button class="btn btn-primary" :disabled="saving" @click="save">{{ saving ? "保存中…" : "保存表单" }}</button>
          <button v-if="editing.id" class="btn btn-ghost danger" @click="remove">删除</button>
        </div>
      </header>

      <div class="lc-code-row" v-if="!editing.id">
        <label>编码</label>
        <input v-model="editing.code" class="lc-code-input" placeholder="留空自动生成（小写字母/数字/-/_）" />
      </div>

      <div class="lc-fields-head">
        <span>字段（{{ editing.fields.length }}）</span>
        <div class="lc-add-types">
          <button v-for="t in fieldTypes" :key="t.type" class="lc-type-btn" @click="addField(t.type)">
            ＋{{ t.label }}
          </button>
        </div>
      </div>

      <div v-if="editing.fields.length === 0" class="lc-no-fields">
        点击上方按钮添加字段开始设计表单
      </div>

      <ul class="lc-fieldlist">
        <li v-for="(f, i) in editing.fields" :key="i" class="lc-field-card">
          <div class="lc-field-top">
            <span class="lc-field-badge">{{ typeLabel(f.type) }}</span>
            <div class="lc-field-order">
              <button class="lc-icon-btn" :disabled="i === 0" @click="move(i, -1)">↑</button>
              <button class="lc-icon-btn" :disabled="i === editing.fields.length - 1" @click="move(i, 1)">↓</button>
              <button class="lc-icon-btn danger" @click="removeField(i)">×</button>
            </div>
          </div>
          <div class="lc-field-body">
            <div class="lc-field-row">
              <label>标签</label>
              <input v-model="f.label" class="lc-fi" placeholder="字段标签" />
            </div>
            <div class="lc-field-row">
              <label>标识</label>
              <input v-model="f.key" class="lc-fi" placeholder="key（英文，留空自动）" />
            </div>
            <label class="lc-check">
              <input type="checkbox" v-model="f.required" /> 必填
            </label>
          </div>
          <div v-if="f.type === 'select'" class="lc-options">
            <label>选项（每行一个）</label>
            <textarea
              :value="(f.options || []).join('\n')"
              class="lc-opt-area"
              placeholder="选项A&#10;选项B"
              @input="setOptions(f, ($event.target as HTMLTextAreaElement).value)"
            ></textarea>
          </div>
        </li>
      </ul>
    </main>

    <main v-else class="lc-main lc-placeholder">
      <div class="lc-ph-emoji">🧩</div>
      <div>选择左侧表单进行编辑，或新建一个表单</div>
    </main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import * as api from "../api";
import type { FormDef, FormField, FieldType } from "../api";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();

const fieldTypes: { type: FieldType; label: string }[] = [
  { type: "text", label: "单行文本" },
  { type: "textarea", label: "多行文本" },
  { type: "number", label: "数字" },
  { type: "date", label: "日期" },
  { type: "select", label: "下拉选择" },
  { type: "checkbox", label: "勾选" },
  { type: "money", label: "金额" },
  { type: "phone", label: "电话" },
];
function typeLabel(t: string) {
  return fieldTypes.find((x) => x.type === t)?.label ?? t;
}

interface EditState {
  id: string;
  code: string;
  name: string;
  icon: string;
  status: "active" | "archived";
  fields: FormField[];
}

const forms = ref<FormDef[]>([]);
const editing = ref<EditState | null>(null);
const saving = ref(false);

async function refresh() {
  forms.value = await api.listForms(true);
}
onMounted(refresh);

function newForm() {
  editing.value = { id: "", code: "", name: "", icon: "📋", status: "active", fields: [] };
}

async function openForm(id: string) {
  try {
    const f = await api.getForm(id);
    editing.value = {
      id: f.id, code: f.code, name: f.name, icon: f.icon, status: f.status,
      fields: f.fields.map((x) => ({ ...x, options: x.options ? [...x.options] : [] })),
    };
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载表单失败");
  }
}

let fieldSeq = 0;
function addField(type: FieldType) {
  if (!editing.value) return;
  fieldSeq += 1;
  editing.value.fields.push({
    key: "",
    label: typeLabel(type),
    type,
    required: false,
    options: type === "select" ? ["选项A", "选项B"] : [],
  });
}
function removeField(i: number) {
  editing.value?.fields.splice(i, 1);
}
function move(i: number, dir: number) {
  if (!editing.value) return;
  const arr = editing.value.fields;
  const j = i + dir;
  if (j < 0 || j >= arr.length) return;
  [arr[i], arr[j]] = [arr[j], arr[i]];
}
function setOptions(f: FormField, raw: string) {
  f.options = raw.split("\n").map((s) => s.trim()).filter(Boolean);
}

async function save() {
  if (!editing.value) return;
  const e = editing.value;
  if (!e.name.trim()) {
    notify.warning("请输入表单名称");
    return;
  }
  saving.value = true;
  try {
    const payload: api.SaveFormPayload = {
      name: e.name.trim(),
      icon: e.icon,
      status: e.status,
      fields: e.fields,
    };
    if (!e.id) payload.code = e.code.trim() || undefined;
    const saved = e.id ? await api.updateForm(e.id, payload) : await api.createForm(payload);
    notify.success("已保存");
    await refresh();
    await openForm(saved.id);
  } catch (err: any) {
    notify.error(err.response?.data?.error?.message ?? "保存失败");
  } finally {
    saving.value = false;
  }
}

async function remove() {
  if (!editing.value?.id) return;
  if (!(await confirm({ title: "删除表单", message: `删除「${editing.value.name}」将同时删除其所有数据，确认？`, danger: true }))) return;
  try {
    await api.deleteForm(editing.value.id);
    editing.value = null;
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}
</script>

<style scoped>
.lc-design {
  display: grid;
  grid-template-columns: 240px 1fr;
  height: calc(100vh - 56px);
  margin: -22px -28px -40px -28px;
  background: var(--bg);
}
.lc-side { background: var(--surface); border-right: 1px solid var(--border); padding: 16px 10px; overflow-y: auto; }
.lc-side-head {
  display: flex; align-items: center; justify-content: space-between;
  font-size: 11px; font-weight: 700; color: var(--text-4);
  text-transform: uppercase; letter-spacing: .6px; padding: 4px 10px 10px;
}
.lc-add { border: 0; background: transparent; color: var(--text-3); font-size: 16px; cursor: pointer; line-height: 1; }
.lc-add:hover { color: var(--primary); }
.lc-side-list { display: flex; flex-direction: column; gap: 2px; }
.lc-side-item {
  display: flex; align-items: center; gap: 9px; padding: 8px 10px; border: 0;
  background: transparent; border-radius: 8px; cursor: pointer; font-size: 13px;
  color: var(--text-2); text-align: left; width: 100%;
}
.lc-side-item:hover { background: var(--bg); }
.lc-side-item.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.lc-side-ico { width: 18px; text-align: center; }
.lc-side-name { flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.lc-side-count { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 999px; padding: 0 7px; }
.lc-empty-side { font-size: 12px; color: var(--text-4); padding: 6px 10px; }

.lc-main { padding: 22px 26px; overflow-y: auto; display: flex; flex-direction: column; gap: 16px; }
.lc-placeholder { align-items: center; justify-content: center; color: var(--text-3); }
.lc-ph-emoji { font-size: 40px; margin-bottom: 8px; }

.lc-main-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.lc-title-row { display: flex; align-items: center; gap: 8px; }
.lc-icon-input {
  width: 44px; height: 38px; text-align: center; font-size: 18px;
  border: 1px solid var(--border); border-radius: 8px; background: var(--surface); outline: none;
}
.lc-name-input {
  font-size: 18px; font-weight: 700; color: var(--text); border: 0; border-bottom: 1px solid transparent;
  background: transparent; outline: none; padding: 4px 0; min-width: 220px;
}
.lc-name-input:focus { border-bottom-color: var(--primary); }
.lc-head-actions { display: flex; align-items: center; gap: 8px; }
.lc-status { border: 1px solid var(--border); border-radius: 7px; padding: 7px 9px; font-size: 13px; background: var(--surface); color: var(--text); }

.lc-code-row { display: flex; align-items: center; gap: 10px; font-size: 13px; }
.lc-code-row label { color: var(--text-3); width: 40px; }
.lc-code-input { flex: 1; max-width: 360px; border: 1px solid var(--border); border-radius: 7px; padding: 7px 9px; font-size: 13px; background: var(--surface); color: var(--text); outline: none; }

.lc-fields-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; flex-wrap: wrap; font-size: 12px; font-weight: 700; color: var(--text-3); }
.lc-add-types { display: flex; flex-wrap: wrap; gap: 6px; }
.lc-type-btn { border: 1px dashed var(--border-strong); background: var(--surface); color: var(--text-2); border-radius: 7px; padding: 5px 9px; font-size: 12px; cursor: pointer; }
.lc-type-btn:hover { border-color: var(--primary); color: var(--primary); }
.lc-no-fields { color: var(--text-4); font-size: 13px; padding: 24px 0; text-align: center; border: 1px dashed var(--border); border-radius: 10px; }

.lc-fieldlist { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.lc-field-card { background: var(--surface); border: 1px solid var(--border); border-radius: 10px; padding: 12px 14px; }
.lc-field-top { display: flex; align-items: center; justify-content: space-between; margin-bottom: 8px; }
.lc-field-badge { font-size: 11px; font-weight: 600; color: var(--primary); background: var(--primary-soft); padding: 2px 8px; border-radius: 5px; }
.lc-field-order { display: flex; gap: 2px; }
.lc-icon-btn { border: 0; background: transparent; cursor: pointer; color: var(--text-3); font-size: 14px; padding: 3px 7px; border-radius: 6px; }
.lc-icon-btn:hover:not(:disabled) { background: var(--surface-2); color: var(--text); }
.lc-icon-btn:disabled { opacity: .35; cursor: not-allowed; }
.lc-icon-btn.danger:hover { color: var(--danger); background: var(--danger-soft); }
.lc-field-body { display: flex; align-items: center; gap: 14px; flex-wrap: wrap; }
.lc-field-row { display: flex; align-items: center; gap: 7px; }
.lc-field-row label { font-size: 12px; color: var(--text-3); }
.lc-fi { border: 1px solid var(--border); border-radius: 7px; padding: 6px 9px; font-size: 13px; background: var(--surface); color: var(--text); outline: none; }
.lc-fi:focus { border-color: var(--primary); }
.lc-check { font-size: 12.5px; color: var(--text-2); display: flex; align-items: center; gap: 5px; cursor: pointer; }
.lc-options { margin-top: 10px; }
.lc-options label { font-size: 12px; color: var(--text-3); display: block; margin-bottom: 4px; }
.lc-opt-area { width: 100%; min-height: 60px; border: 1px solid var(--border); border-radius: 8px; padding: 7px 9px; font-size: 13px; font-family: inherit; resize: vertical; background: var(--surface); color: var(--text); outline: none; }
.lc-opt-area:focus { border-color: var(--primary); }
.btn.danger { color: var(--danger); }
</style>
