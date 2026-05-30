<template>
  <section class="admin-page">
    <PageHeader title="参数配置" :sub="`平台级键值参数（public.platform_setting）· 共 ${params.length} 项 · 值为 JSON`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button class="btn btn-primary" v-perm="'param:manage'" @click="openCreate">+ 新建参数</button>
      </template>
    </PageHeader>

    <DataTable :columns="columns" :rows="params" row-key="key" empty-text="暂无参数 · 点击右上角新建">
      <template #cell-key="{ row }">
        <code class="k">{{ row.key }}</code>
      </template>
      <template #cell-value="{ row }">
        <span class="v-preview">{{ valuePreview(row.value) }}</span>
      </template>
      <template #cell-updated_at="{ row }">
        <span class="time">{{ fmt(row.updated_at) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="row-actions">
          <button class="link-btn" v-perm="'param:manage'" @click="openEdit(row)">编辑</button>
          <button class="link-btn danger" v-perm="'param:manage'" @click="doDelete(row)">删除</button>
        </div>
      </template>
    </DataTable>

    <!-- Create / edit modal -->
    <div v-if="editor" class="modal-overlay" @click.self="editor = false">
      <div class="modal modal-lg">
        <h3>{{ editing ? '编辑参数' : '新建参数' }}</h3>
        <label class="field">
          <span class="label">键 (key) *</span>
          <input class="input" v-model="form.key" :disabled="editing" placeholder="例如 site.maintenance_mode" />
        </label>
        <label class="field">
          <span class="label">值 (JSON) *</span>
          <textarea
            class="input textarea mono"
            v-model="form.value"
            rows="10"
            placeholder='例如 true / 42 / "text" / {"a":1} / ["x","y"]'
          ></textarea>
        </label>
        <div class="hint">值必须是合法 JSON。字符串需加引号，例如 <code>"abc"</code>；布尔/数字/对象/数组直接书写。</div>
        <div v-if="formError" class="form-error">{{ formError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="editor = false">取消</button>
          <button class="btn btn-ghost" type="button" @click="prettify">格式化</button>
          <button class="btn btn-primary" :disabled="busy" @click="submit">{{ editing ? '保存' : '创建' }}</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, DataTable, type Column } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import { listParams, upsertParam, deleteParam, type PlatformParam } from "../api/system";

const notify = useNotification();
const { confirm } = useConfirm();

const columns: Column[] = [
  { key: "key", label: "键", width: "280px" },
  { key: "value", label: "值 (JSON)" },
  { key: "updated_at", label: "更新时间", width: "180px" },
  { key: "actions", label: "操作", width: "140px" },
];

const params = ref<PlatformParam[]>([]);
const editor = ref(false);
const editing = ref(false);
const busy = ref(false);
const formError = ref("");
const form = reactive({ key: "", value: "" });

onMounted(reload);

async function reload() {
  params.value = await listParams();
}

function valuePreview(v: any): string {
  const s = JSON.stringify(v);
  if (s == null) return "null";
  return s.length > 120 ? s.slice(0, 120) + "…" : s;
}

function fmt(s: string): string {
  return (s || "").slice(0, 19).replace("T", " ");
}

function openCreate() {
  editing.value = false;
  editor.value = true;
  formError.value = "";
  form.key = "";
  form.value = "";
}

function openEdit(p: PlatformParam) {
  editing.value = true;
  editor.value = true;
  formError.value = "";
  form.key = p.key;
  form.value = JSON.stringify(p.value, null, 2);
}

function prettify() {
  try {
    form.value = JSON.stringify(JSON.parse(form.value), null, 2);
    formError.value = "";
  } catch {
    formError.value = "当前内容不是合法 JSON，无法格式化";
  }
}

async function submit() {
  const key = form.key.trim();
  if (!key) { formError.value = "键不能为空"; return; }
  let parsed: any;
  try {
    parsed = form.value.trim() === "" ? null : JSON.parse(form.value);
  } catch {
    formError.value = "值必须是合法 JSON（字符串请加引号）";
    return;
  }
  busy.value = true; formError.value = "";
  try {
    await upsertParam(key, parsed);
    notify.success(editing.value ? "参数已更新" : "参数已创建");
    editor.value = false;
    await reload();
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}

async function doDelete(p: PlatformParam) {
  const ok = await confirm({ title: "删除参数", message: `确认删除参数「${p.key}」？此操作不可恢复。`, danger: true });
  if (!ok) return;
  try {
    await deleteParam(p.key);
    notify.success("已删除");
    await reload();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.k { background: var(--surface-2); padding: 2px 7px; border-radius: 4px; font-family: var(--ff-mono); font-size: 12px; color: var(--primary); }
.v-preview { font-family: var(--ff-mono); font-size: 12px; color: var(--text-2); word-break: break-all; }
.time { font-family: var(--ff-mono); font-size: 12px; color: var(--text-3); }

.row-actions { display: flex; gap: 2px; }
.link-btn { background: transparent; border: 0; font-size: 12px; color: var(--primary); cursor: pointer; padding: 3px 7px; border-radius: 4px; }
.link-btn:hover { background: var(--primary-soft); }
.link-btn.danger { color: var(--danger); }
.link-btn.danger:hover { background: var(--danger-soft); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }

.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(420px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal-lg { width: min(620px, 94vw); }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); font-family: inherit; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input:disabled { background: var(--surface-2); color: var(--text-3); }
.textarea { resize: vertical; line-height: 1.6; }
.mono { font-family: var(--ff-mono); font-size: 12.5px; }
.hint { font-size: 11.5px; color: var(--text-3); line-height: 1.5; }
.hint code { background: var(--surface-2); padding: 1px 5px; border-radius: 3px; font-family: var(--ff-mono); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
</style>
