<template>
  <section class="admin-page">
    <PageHeader title="通知公告" :sub="`平台范围公告 · 共 ${notices.length} 条 · 草稿/已发布/已撤回`">
      <template #actions>
        <select class="select" v-model="statusFilter" @change="reload">
          <option value="">全部状态</option>
          <option value="draft">草稿</option>
          <option value="published">已发布</option>
          <option value="withdrawn">已撤回</option>
        </select>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button class="btn btn-primary" v-perm="'notice:manage'" @click="openCreate">+ 新建公告</button>
      </template>
    </PageHeader>

    <DataTable :columns="columns" :rows="notices" row-key="id" empty-text="暂无公告 · 点击右上角新建">
      <template #cell-title="{ row }">
        <span class="n-title">{{ row.title }}</span>
      </template>
      <template #cell-type="{ row }">
        <span class="tag">{{ typeLabel(row.type) }}</span>
      </template>
      <template #cell-status="{ row }">
        <span class="badge" :class="statusClass(row.status)">{{ statusLabel(row.status) }}</span>
      </template>
      <template #cell-created_at="{ row }">
        <span class="time">{{ fmt(row.created_at) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="row-actions">
          <button class="link-btn" @click="openView(row)">查看</button>
          <button class="link-btn" v-perm="'notice:manage'" @click="openEdit(row)">编辑</button>
          <button
            v-if="row.status !== 'published'"
            class="link-btn ok" v-perm="'notice:manage'" @click="doPublish(row)"
          >发布</button>
          <button
            v-else
            class="link-btn warn" v-perm="'notice:manage'" @click="doWithdraw(row)"
          >撤回</button>
          <button class="link-btn danger" v-perm="'notice:manage'" @click="doDelete(row)">删除</button>
        </div>
      </template>
    </DataTable>

    <!-- Create / edit modal -->
    <div v-if="editor" class="modal-overlay" @click.self="editor = false">
      <div class="modal modal-lg">
        <h3>{{ editing ? '编辑公告' : '新建公告' }}</h3>
        <label class="field">
          <span class="label">标题 *</span>
          <input class="input" v-model="form.title" placeholder="公告标题" />
        </label>
        <label class="field">
          <span class="label">类型</span>
          <select class="input" v-model="form.type">
            <option value="notice">通知</option>
            <option value="announcement">公告</option>
          </select>
        </label>
        <label class="field">
          <span class="label">内容</span>
          <textarea class="input textarea" v-model="form.content" rows="8" placeholder="公告正文…"></textarea>
        </label>
        <div v-if="formError" class="form-error">{{ formError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="editor = false">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="submit">{{ editing ? '保存' : '创建' }}</button>
        </div>
      </div>
    </div>

    <!-- View drawer -->
    <div v-if="viewing" class="modal-overlay" @click.self="viewing = null">
      <div class="modal modal-lg">
        <div class="view-head">
          <h3>{{ viewing.title }}</h3>
          <span class="badge" :class="statusClass(viewing.status)">{{ statusLabel(viewing.status) }}</span>
        </div>
        <p class="view-meta">{{ typeLabel(viewing.type) }} · {{ fmt(viewing.created_at) }}</p>
        <pre class="view-content">{{ viewing.content || '（无正文）' }}</pre>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="viewing = null">关闭</button>
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
import {
  listNotices, createNotice, updateNotice, deleteNotice, publishNotice, withdrawNotice,
  type PlatformNotice, type PlatformNoticeStatus,
} from "../api/system";

const notify = useNotification();
const { confirm } = useConfirm();

const columns: Column[] = [
  { key: "title", label: "标题" },
  { key: "type", label: "类型", width: "100px" },
  { key: "status", label: "状态", width: "90px" },
  { key: "created_at", label: "创建时间", width: "170px" },
  { key: "actions", label: "操作", width: "260px" },
];

const notices = ref<PlatformNotice[]>([]);
const statusFilter = ref<PlatformNoticeStatus>("");

const editor = ref(false);
const editing = ref<PlatformNotice | null>(null);
const viewing = ref<PlatformNotice | null>(null);
const busy = ref(false);
const formError = ref("");
const form = reactive({ title: "", type: "notice", content: "" });

onMounted(reload);

async function reload() {
  notices.value = await listNotices(statusFilter.value);
}

function typeLabel(t: string): string {
  return t === "announcement" ? "公告" : "通知";
}
function statusLabel(s: string): string {
  if (s === "published") return "已发布";
  if (s === "withdrawn") return "已撤回";
  return "草稿";
}
function statusClass(s: string): string {
  if (s === "published") return "on";
  if (s === "withdrawn") return "warn";
  return "off";
}
function fmt(s: string): string {
  return (s || "").slice(0, 19).replace("T", " ");
}

function openCreate() {
  editing.value = null;
  editor.value = true;
  formError.value = "";
  form.title = ""; form.type = "notice"; form.content = "";
}

function openEdit(n: PlatformNotice) {
  editing.value = n;
  editor.value = true;
  formError.value = "";
  form.title = n.title; form.type = n.type || "notice"; form.content = n.content;
}

function openView(n: PlatformNotice) { viewing.value = n; }

async function submit() {
  if (!form.title.trim()) { formError.value = "标题不能为空"; return; }
  busy.value = true; formError.value = "";
  try {
    if (editing.value) {
      await updateNotice(editing.value.id, { title: form.title.trim(), type: form.type, content: form.content });
      notify.success("公告已更新");
    } else {
      await createNotice({ title: form.title.trim(), type: form.type, content: form.content });
      notify.success("公告已创建（草稿）");
    }
    editor.value = false;
    await reload();
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}

async function doPublish(n: PlatformNotice) {
  const ok = await confirm({ title: "发布公告", message: `确认发布「${n.title}」？发布后全平台用户可见。` });
  if (!ok) return;
  try { await publishNotice(n.id); notify.success("已发布"); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "发布失败"); }
}

async function doWithdraw(n: PlatformNotice) {
  const ok = await confirm({ title: "撤回公告", message: `确认撤回「${n.title}」？将退回草稿/撤回状态。` });
  if (!ok) return;
  try { await withdrawNotice(n.id); notify.success("已撤回"); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "撤回失败"); }
}

async function doDelete(n: PlatformNotice) {
  const ok = await confirm({ title: "删除公告", message: `确认删除「${n.title}」？此操作不可恢复。`, danger: true });
  if (!ok) return;
  try { await deleteNotice(n.id); notify.success("已删除"); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "删除失败"); }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.n-title { font-weight: 600; color: var(--text); }
.time { font-family: var(--ff-mono); font-size: 12px; color: var(--text-3); }
.tag { font-size: 11.5px; padding: 2px 8px; border-radius: 4px; background: var(--surface-2); color: var(--text-2); }
.badge { font-size: 11px; font-weight: 600; padding: 2px 9px; border-radius: 999px; }
.badge.on { background: var(--success-soft); color: var(--success); }
.badge.off { background: var(--surface-2); color: var(--text-3); }
.badge.warn { background: var(--warning-soft); color: var(--warning); }

.row-actions { display: flex; gap: 2px; flex-wrap: wrap; }
.link-btn { background: transparent; border: 0; font-size: 12px; color: var(--primary); cursor: pointer; padding: 3px 7px; border-radius: 4px; }
.link-btn:hover { background: var(--primary-soft); }
.link-btn.ok { color: var(--success); }
.link-btn.ok:hover { background: var(--success-soft); }
.link-btn.warn { color: var(--warning); }
.link-btn.warn:hover { background: var(--warning-soft); }
.link-btn.danger { color: var(--danger); }
.link-btn.danger:hover { background: var(--danger-soft); }

.select { padding: 6px 10px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }

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
.textarea { resize: vertical; line-height: 1.6; }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

.view-head { display: flex; align-items: center; gap: 10px; }
.view-meta { font-size: 12px; color: var(--text-3); margin: -4px 0 4px; }
.view-content {
  background: var(--bg-deep); padding: 14px; border-radius: 8px; margin: 0;
  font-family: inherit; font-size: 13px; line-height: 1.7; color: var(--text);
  white-space: pre-wrap; word-break: break-word; max-height: 50vh; overflow: auto;
}
</style>
