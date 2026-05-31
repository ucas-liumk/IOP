<template>
  <section class="admin-page">
    <PageHeader title="岗位管理" :sub="`共 ${posts.length} 个岗位`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button class="btn btn-primary" v-perm="'post:write'" @click="openCreate">+ 新建岗位</button>
      </template>
    </PageHeader>

    <DataTable :columns="columns" :rows="posts" rowKey="id" emptyText="暂无岗位">
      <template #cell-code="{ row }"><code>{{ row.code }}</code></template>
      <template #cell-status="{ row }">
        <span class="badge" :class="row.status === 'active' ? 'badge-success' : 'badge-neutral'">
          {{ row.status === 'active' ? '正常' : '停用' }}
        </span>
      </template>
      <template #cell-created_at="{ row }"><span class="mono">{{ row.created_at }}</span></template>
      <template #cell-actions="{ row }">
        <div class="row-actions">
          <button class="btn btn-ghost btn-sm" v-perm="'post:write'" @click="openEdit(row)">编辑</button>
          <button class="btn btn-ghost btn-sm" v-perm="'post:write'" @click="toggleStatus(row)">
            {{ row.status === 'active' ? '停用' : '启用' }}
          </button>
          <button class="btn btn-ghost btn-sm danger" v-perm="'post:write'" @click="remove(row)">删除</button>
        </div>
      </template>
    </DataTable>

    <!-- Create / edit modal -->
    <div v-if="editing" class="modal-overlay" @click.self="close">
      <div class="modal">
        <h3>{{ isNew ? '新建岗位' : '编辑岗位' }}</h3>
        <label class="field">
          <span class="label">岗位编码 *</span>
          <input class="input" v-model="form.code" :disabled="!isNew" placeholder="例如：dev" />
          <span v-if="!isNew" class="hint">编码创建后不可修改</span>
        </label>
        <label class="field">
          <span class="label">岗位名称 *</span>
          <input class="input" v-model="form.name" placeholder="例如：研发工程师" />
        </label>
        <label class="field">
          <span class="label">排序</span>
          <input class="input" type="number" v-model.number="form.order_num" />
        </label>
        <div v-if="formError" class="form-error">{{ formError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="close">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="save">{{ busy ? '保存中…' : '保存' }}</button>
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
import { listPosts, createPost, updatePost, deletePost, type Post } from "../api/admin";

const notify = useNotification();
const { confirm } = useConfirm();

const posts = ref<Post[]>([]);
const editing = ref<Post | null | "new">(null);
const busy = ref(false);
const formError = ref("");
const form = reactive({ code: "", name: "", order_num: 0 });

const isNew = ref(false);

const columns: Column[] = [
  { key: "code",       label: "编码",     width: 160 },
  { key: "name",       label: "名称" },
  { key: "order_num",  label: "排序",     width: 90 },
  { key: "status",     label: "状态",     width: 90 },
  { key: "created_at", label: "创建时间", width: 170 },
  { key: "actions",    label: "操作",     width: 200, align: "right" },
];

onMounted(reload);
async function reload() { posts.value = await listPosts(); }

function openCreate() {
  isNew.value = true;
  editing.value = "new";
  formError.value = "";
  Object.assign(form, { code: "", name: "", order_num: 0 });
}
function openEdit(p: Post) {
  isNew.value = false;
  editing.value = p;
  formError.value = "";
  Object.assign(form, { code: p.code, name: p.name, order_num: p.order_num });
}
function close() { editing.value = null; formError.value = ""; }

async function save() {
  if (!form.name.trim() || (isNew.value && !form.code.trim())) {
    formError.value = "编码与名称不能为空"; return;
  }
  busy.value = true; formError.value = "";
  try {
    if (isNew.value) {
      await createPost({ code: form.code.trim(), name: form.name.trim(), order_num: form.order_num });
      notify.success("岗位已创建");
    } else if (editing.value && editing.value !== "new") {
      await updatePost(editing.value.id, { name: form.name.trim(), order_num: form.order_num });
      notify.success("岗位已更新");
    }
    close();
    await reload();
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}

async function toggleStatus(p: Post) {
  try {
    await updatePost(p.id, { status: p.status === "active" ? "disabled" : "active" });
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "操作失败"); }
}

async function remove(p: Post) {
  const ok = await confirm({ title: "删除岗位", message: `确认删除岗位「${p.name}」？已分配成员时无法删除。`, danger: true });
  if (!ok) return;
  try {
    await deletePost(p.id);
    await reload();
    notify.success("岗位已删除");
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "删除失败"); }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.row-actions { display: flex; gap: 4px; justify-content: flex-end; }
code { background: var(--surface-2); padding: 1px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 12px; color: var(--text-2); }
.mono { font-family: var(--ff-mono); font-size: 12.5px; color: var(--text-2); }
.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.badge-success { background: var(--success-soft); color: var(--success); }
.badge-neutral { background: var(--bg-deep); color: var(--text-3); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }

.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(420px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; color: var(--text-2); }
.hint { font-size: 11px; color: var(--text-4); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input:disabled { background: var(--surface-2); color: var(--text-3); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
</style>
