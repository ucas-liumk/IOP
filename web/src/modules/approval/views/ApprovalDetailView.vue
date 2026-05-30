<template>
  <section class="admin-page">
    <PageHeader :title="ins ? ins.form_name : '审批详情'" :sub="ins ? `发起人 ${ins.initiator_name || '—'} · ${fmt(ins.created_at)}` : ''">
      <template #actions>
        <RouterLink class="btn btn-ghost" to="/approval/mine">← 返回</RouterLink>
      </template>
    </PageHeader>

    <div v-if="loading" class="ap-loading">加载中…</div>
    <EmptyState v-else-if="!ins" title="审批不存在" sub="可能已被撤回或删除" icon="!" />

    <div v-else class="ap-detail">
      <!-- Left: form data -->
      <div class="card card-pad ap-data">
        <div class="ap-data-head">
          <span class="badge" :class="statusBadge(ins.status)">{{ statusLabel(ins.status) }}</span>
        </div>
        <dl class="ap-fields">
          <template v-for="f in ins.fields" :key="f.key">
            <dt>{{ f.label }}</dt>
            <dd>{{ display(ins.data[f.key]) }}</dd>
          </template>
          <div v-if="ins.fields.length === 0" class="ap-empty-fields">无表单字段</div>
        </dl>

        <!-- Action bar (only if I have a pending approve task) -->
        <div v-if="myPendingTask" class="ap-actions">
          <textarea v-model="comment" class="input ap-comment" placeholder="审批意见（可选）"></textarea>
          <div class="ap-action-btns">
            <button class="btn btn-danger" :disabled="acting" @click="decide('reject')">拒绝</button>
            <button class="btn btn-primary" :disabled="acting" @click="decide('approve')">同意</button>
          </div>
        </div>
        <div v-else-if="myCcTask && myCcTask.status === 'pending'" class="ap-actions">
          <button class="btn btn-primary" :disabled="acting" @click="decide('read')">标记已阅</button>
        </div>
        <div v-if="canCancel" class="ap-actions">
          <button class="btn btn-ghost" :disabled="acting" @click="doCancel">撤回申请</button>
        </div>
      </div>

      <!-- Right: timeline -->
      <div class="card card-pad ap-timeline">
        <h3 class="ap-tl-title">审批流程</h3>
        <ol class="ap-tl">
          <li class="ap-tl-item">
            <span class="ap-tl-dot done"></span>
            <div class="ap-tl-body">
              <div class="ap-tl-name">{{ ins.initiator_name || "发起人" }} <span class="ap-tl-role">发起</span></div>
              <div class="ap-tl-time">{{ fmt(ins.created_at) }}</div>
            </div>
          </li>
          <li v-for="t in ins.tasks" :key="t.id" class="ap-tl-item">
            <span class="ap-tl-dot" :class="taskDot(t.status)"></span>
            <div class="ap-tl-body">
              <div class="ap-tl-name">
                {{ t.assignee_name || "审批人" }}
                <span class="ap-tl-role">{{ t.type === "cc" ? "抄送" : "审批" }}</span>
                <span class="badge" :class="taskBadge(t.status)">{{ taskLabel(t.status) }}</span>
              </div>
              <div v-if="t.comment" class="ap-tl-comment">“{{ t.comment }}”</div>
              <div class="ap-tl-time">{{ t.acted_at ? fmt(t.acted_at) : "待处理" }}</div>
            </div>
          </li>
        </ol>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useRoute } from "vue-router";
import { PageHeader, EmptyState } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import * as api from "../api";
import type { Instance, Task } from "../api";

const route = useRoute();
const notify = useNotification();
const { confirm } = useConfirm();

const ins = ref<Instance | null>(null);
const loading = ref(false);
const acting = ref(false);
const comment = ref("");

// Action visibility uses server-computed, viewer-relative flags (task.mine /
// instance.can_cancel) — the client never needs the caller's member id.
const myPendingTask = computed<Task | null>(() => {
  if (!ins.value) return null;
  return ins.value.tasks?.find((t) => t.type === "approve" && t.status === "pending" && t.mine) ?? null;
});
const myCcTask = computed<Task | null>(() => {
  if (!ins.value) return null;
  return ins.value.tasks?.find((t) => t.type === "cc" && t.mine) ?? null;
});
const canCancel = computed(() => !!ins.value?.can_cancel);

async function load() {
  loading.value = true;
  try {
    ins.value = await api.getInstance(String(route.params.id));
  } catch {
    ins.value = null;
  } finally {
    loading.value = false;
  }
}

async function decide(action: "approve" | "reject" | "read") {
  const task = action === "read" ? myCcTask.value : myPendingTask.value;
  if (!task) return;
  if (action === "reject") {
    const ok = await confirm({ title: "拒绝审批", message: "确认拒绝该申请？流程将终止。", danger: true, confirmText: "拒绝" });
    if (!ok) return;
  }
  acting.value = true;
  try {
    await api.act(task.id, action, comment.value);
    notify.success(action === "approve" ? "已同意" : action === "reject" ? "已拒绝" : "已阅");
    comment.value = "";
    await load();
  } catch (e: unknown) {
    notify.error(errMsg(e) || "操作失败");
  } finally {
    acting.value = false;
  }
}

async function doCancel() {
  const ok = await confirm({ title: "撤回申请", message: "确认撤回该审批申请？", danger: true, confirmText: "撤回" });
  if (!ok) return;
  acting.value = true;
  try {
    await api.cancelInstance(String(route.params.id));
    notify.success("已撤回");
    await load();
  } catch (e: unknown) {
    notify.error(errMsg(e) || "撤回失败");
  } finally {
    acting.value = false;
  }
}

function display(v: unknown): string {
  if (v === null || v === undefined || v === "") return "—";
  if (Array.isArray(v)) return v.join("、");
  return String(v);
}

const statusLabels: Record<string, string> = {
  pending: "审批中", approved: "已通过", rejected: "已拒绝", canceled: "已撤回",
};
function statusLabel(s: string): string { return statusLabels[s] ?? s; }
function statusBadge(s: string): string {
  return s === "approved" ? "badge-success" : s === "rejected" ? "badge-danger" : s === "canceled" ? "badge-info" : "badge-warning";
}

const taskLabels: Record<string, string> = {
  pending: "待处理", approved: "已同意", rejected: "已拒绝", read: "已阅", canceled: "已跳过",
};
function taskLabel(s: string): string { return taskLabels[s] ?? s; }
function taskBadge(s: string): string {
  return s === "approved" ? "badge-success" : s === "rejected" ? "badge-danger" : s === "read" ? "badge-teal" : s === "canceled" ? "badge-info" : "badge-warning";
}
function taskDot(s: string): string {
  return s === "approved" || s === "read" ? "done" : s === "rejected" ? "rejected" : s === "pending" ? "pending" : "skipped";
}

function fmt(s: string): string {
  if (!s) return "";
  return new Date(s).toLocaleString("zh-CN", { month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
}
function errMsg(e: unknown): string {
  return (e as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message ?? "";
}

onMounted(load);
</script>

<style scoped>
.ap-loading { padding: var(--sp-6); color: var(--text-3); }
.ap-detail { display: grid; grid-template-columns: 1.4fr 1fr; gap: var(--sp-4); align-items: start; }
@media (max-width: 900px) { .ap-detail { grid-template-columns: 1fr; } }

.ap-data-head { margin-bottom: var(--sp-4); }
.ap-fields { display: grid; grid-template-columns: 120px 1fr; gap: var(--sp-3) var(--sp-4); margin: 0; }
.ap-fields dt { color: var(--text-3); font-size: 13px; }
.ap-fields dd { margin: 0; color: var(--text); }
.ap-empty-fields { color: var(--text-3); }

.ap-actions { margin-top: var(--sp-5); padding-top: var(--sp-4); border-top: 1px solid var(--border-soft); }
.ap-comment { width: 100%; min-height: 64px; resize: vertical; margin-bottom: var(--sp-3); }
.ap-action-btns { display: flex; justify-content: flex-end; gap: var(--sp-3); }

.ap-tl-title { margin: 0 0 var(--sp-4); font-size: 15px; }
.ap-tl { list-style: none; margin: 0; padding: 0; }
.ap-tl-item { display: flex; gap: var(--sp-3); padding-bottom: var(--sp-4); position: relative; }
.ap-tl-item:not(:last-child)::before {
  content: ""; position: absolute; left: 5px; top: 16px; bottom: 0; width: 2px; background: var(--border-soft);
}
.ap-tl-dot { width: 12px; height: 12px; border-radius: 50%; flex-shrink: 0; margin-top: 3px; background: var(--text-4); z-index: 1; }
.ap-tl-dot.done { background: var(--success); }
.ap-tl-dot.rejected { background: var(--danger); }
.ap-tl-dot.pending { background: var(--warning); }
.ap-tl-dot.skipped { background: var(--text-4); }
.ap-tl-body { min-width: 0; }
.ap-tl-name { display: flex; align-items: center; gap: var(--sp-2); font-weight: 600; color: var(--text); flex-wrap: wrap; }
.ap-tl-role { font-size: 12px; font-weight: 400; color: var(--text-3); }
.ap-tl-comment { font-size: 13px; color: var(--text-2); margin-top: 4px; }
.ap-tl-time { font-size: 12px; color: var(--text-4); margin-top: 2px; }
</style>
