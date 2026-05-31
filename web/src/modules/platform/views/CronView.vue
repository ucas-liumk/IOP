<template>
  <section class="admin-page">
    <PageHeader title="定时任务" :sub="`平台调度任务 · 共 ${jobs.length} 个`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button class="btn btn-primary" v-perm="'job:manage'" @click="openCreate">+ 新建任务</button>
      </template>
    </PageHeader>

    <DataTable :columns="columns" :rows="jobs" row-key="id" empty-text="暂无定时任务 · 点击右上角新建">
      <template #cell-name="{ row }">
        <span class="j-name">{{ row.name }}</span>
      </template>
      <template #cell-cron_expr="{ row }">
        <code>{{ row.cron_expr || '—' }}</code>
      </template>
      <template #cell-handler="{ row }">
        <code class="muted">{{ row.handler }}</code>
      </template>
      <template #cell-status="{ row }">
        <span class="badge" :class="row.status === 'enabled' ? 'on' : 'off'">
          {{ row.status === 'enabled' ? '启用' : '停用' }}
        </span>
      </template>
      <template #cell-last_run_at="{ row }">
        <span class="time">{{ fmt(row.last_run_at) }}</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="row-actions">
          <button class="link-btn ok" v-perm="'job:manage'" @click="run(row)">立即执行</button>
          <button class="link-btn" @click="openRuns(row)">执行记录</button>
          <button class="link-btn" v-perm="'job:manage'" @click="openEdit(row)">编辑</button>
          <button class="link-btn danger" v-perm="'job:manage'" @click="doDelete(row)">删除</button>
        </div>
      </template>
    </DataTable>

    <!-- Create / edit modal -->
    <div v-if="editor" class="modal-overlay" @click.self="editor = false">
      <div class="modal">
        <h3>{{ editing ? '编辑任务' : '新建任务' }}</h3>
        <label class="field">
          <span class="label">名称 *</span>
          <input class="input" v-model="form.name" placeholder="任务名称" />
        </label>
        <label class="field">
          <span class="label">Cron 表达式</span>
          <input class="input mono" v-model="form.cron_expr" placeholder="例如 0 */5 * * * *" />
        </label>
        <label class="field">
          <span class="label">Handler</span>
          <select class="input" v-model="form.handler">
            <option value="noop">noop（空操作）</option>
            <option value="echo">echo（回显）</option>
          </select>
        </label>
        <label class="field">
          <span class="label">状态</span>
          <select class="input" v-model="form.status">
            <option value="enabled">启用</option>
            <option value="disabled">停用</option>
          </select>
        </label>
        <div v-if="formError" class="form-error">{{ formError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="editor = false">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="submit">{{ editing ? '保存' : '创建' }}</button>
        </div>
      </div>
    </div>

    <!-- Runs drawer -->
    <div v-if="runsJob" class="drawer-overlay" @click.self="runsJob = null">
      <aside class="drawer">
        <header class="drawer-head">
          <h3>执行记录 · {{ runsJob.name }}</h3>
          <button class="x" @click="runsJob = null">✕</button>
        </header>
        <table class="runs-table">
          <thead>
            <tr><th>开始</th><th>结束</th><th>状态</th><th>详情</th></tr>
          </thead>
          <tbody>
            <tr v-for="r in runs" :key="r.id">
              <td class="time">{{ fmt(r.started_at) }}</td>
              <td class="time">{{ fmt(r.finished_at) }}</td>
              <td><span class="badge" :class="runClass(r.status)">{{ r.status }}</span></td>
              <td class="rd">{{ r.detail || '—' }}</td>
            </tr>
            <tr v-if="runs.length === 0">
              <td colspan="4" class="empty-cell">{{ runsLoading ? '加载中…' : '暂无执行记录' }}</td>
            </tr>
          </tbody>
        </table>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, DataTable, type Column } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  listJobs, createJob, updateJob, deleteJob, runJobNow, listJobRuns,
  type PlatformJob, type PlatformJobRun,
} from "../api/system";

const notify = useNotification();
const { confirm } = useConfirm();

const columns: Column[] = [
  { key: "name", label: "名称" },
  { key: "cron_expr", label: "Cron", width: "160px" },
  { key: "handler", label: "Handler", width: "120px" },
  { key: "status", label: "状态", width: "80px" },
  { key: "last_run_at", label: "上次执行", width: "170px" },
  { key: "actions", label: "操作", width: "260px" },
];

const jobs = ref<PlatformJob[]>([]);

const editor = ref(false);
const editing = ref<PlatformJob | null>(null);
const busy = ref(false);
const formError = ref("");
const form = reactive({ name: "", cron_expr: "", handler: "noop", status: "enabled" });

const runsJob = ref<PlatformJob | null>(null);
const runs = ref<PlatformJobRun[]>([]);
const runsLoading = ref(false);

onMounted(reload);

async function reload() {
  jobs.value = await listJobs();
}

function fmt(s?: string | null): string {
  return s ? s.slice(0, 19).replace("T", " ") : "—";
}
function runClass(s: string): string {
  if (s === "success") return "on";
  if (s === "failed") return "fail";
  return "running";
}

function openCreate() {
  editing.value = null;
  editor.value = true;
  formError.value = "";
  form.name = ""; form.cron_expr = ""; form.handler = "noop"; form.status = "enabled";
}

function openEdit(j: PlatformJob) {
  editing.value = j;
  editor.value = true;
  formError.value = "";
  form.name = j.name; form.cron_expr = j.cron_expr; form.handler = j.handler || "noop"; form.status = j.status || "enabled";
}

async function submit() {
  if (!form.name.trim()) { formError.value = "名称不能为空"; return; }
  busy.value = true; formError.value = "";
  try {
    if (editing.value) {
      await updateJob(editing.value.id, { name: form.name.trim(), cron_expr: form.cron_expr, handler: form.handler, status: form.status });
      notify.success("任务已更新");
    } else {
      await createJob({ name: form.name.trim(), cron_expr: form.cron_expr, handler: form.handler, status: form.status });
      notify.success("任务已创建");
    }
    editor.value = false;
    await reload();
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}

async function doDelete(j: PlatformJob) {
  const ok = await confirm({ title: "删除任务", message: `确认删除任务「${j.name}」？其执行记录将一并删除。`, danger: true });
  if (!ok) return;
  try { await deleteJob(j.id); notify.success("已删除"); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "删除失败"); }
}

async function run(j: PlatformJob) {
  const ok = await confirm({ title: "立即执行", message: `确认立即执行任务「${j.name}」？` });
  if (!ok) return;
  try {
    const r = await runJobNow(j.id);
    notify.success(`已执行 · ${r?.status ?? "完成"}`);
    await reload();
    if (runsJob.value?.id === j.id) await loadRuns(j.id);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "执行失败");
  }
}

async function openRuns(j: PlatformJob) {
  runsJob.value = j;
  await loadRuns(j.id);
}
async function loadRuns(id: string) {
  runsLoading.value = true;
  try { runs.value = await listJobRuns(id); }
  finally { runsLoading.value = false; }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.j-name { font-weight: 600; color: var(--text); }
.time { font-family: var(--ff-mono); font-size: 12px; color: var(--text-3); }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; color: var(--primary); }
code.muted { color: var(--text-3); }
.badge { font-size: 11px; font-weight: 600; padding: 2px 9px; border-radius: 999px; }
.badge.on { background: var(--success-soft); color: var(--success); }
.badge.off { background: var(--surface-2); color: var(--text-3); }
.badge.fail { background: var(--danger-soft); color: var(--danger); }
.badge.running { background: var(--warning-soft); color: var(--warning); }

.row-actions { display: flex; gap: 2px; flex-wrap: wrap; }
.link-btn { background: transparent; border: 0; font-size: 12px; color: var(--primary); cursor: pointer; padding: 3px 7px; border-radius: 4px; }
.link-btn:hover { background: var(--primary-soft); }
.link-btn.ok { color: var(--success); }
.link-btn.ok:hover { background: var(--success-soft); }
.link-btn.danger { color: var(--danger); }
.link-btn.danger:hover { background: var(--danger-soft); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }

.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(460px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); font-family: inherit; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.mono { font-family: var(--ff-mono); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

.drawer-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.4); z-index: 110; display: flex; justify-content: flex-end; backdrop-filter: blur(2px); }
.drawer { width: min(640px, 96vw); height: 100%; background: var(--surface); box-shadow: var(--sh-4); padding: 20px 22px; overflow: auto; display: flex; flex-direction: column; gap: 16px; }
.drawer-head { display: flex; justify-content: space-between; align-items: center; }
.drawer-head h3 { font-size: 15px; font-weight: 600; margin: 0; }
.x { background: transparent; border: 0; font-size: 16px; color: var(--text-3); cursor: pointer; }
.runs-table { width: 100%; border-collapse: collapse; }
.runs-table th { text-align: left; font-size: 11px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 8px 10px; border-bottom: 1px solid var(--border); }
.runs-table td { padding: 9px 10px; font-size: 12.5px; border-bottom: 1px solid var(--border-soft); vertical-align: top; }
.rd { color: var(--text-2); word-break: break-all; }
.empty-cell { text-align: center; color: var(--text-4); padding: 30px 0; }
</style>
