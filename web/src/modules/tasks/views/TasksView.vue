<template>
  <div class="tasks-app">
    <!-- Left: smart views + lists -->
    <aside class="tk-side">
      <div class="tk-side-group">
        <button
          v-for="v in smartViews" :key="v.key"
          class="tk-nav" :class="{ active: sel.type === 'view' && sel.view === v.key }"
          @click="selectView(v.key)"
        >
          <span class="tk-nav-ico" :style="{ color: v.color }">{{ v.icon }}</span>
          <span class="tk-nav-label">{{ v.label }}</span>
          <span v-if="counts[v.key]" class="tk-nav-count">{{ counts[v.key] }}</span>
        </button>
      </div>

      <div class="tk-side-head">
        <span>清单</span>
        <button class="tk-add-list" title="新建清单" @click="openNewList">＋</button>
      </div>
      <div class="tk-side-group">
        <button
          v-for="l in lists" :key="l.id"
          class="tk-nav" :class="{ active: sel.type === 'list' && sel.listId === l.id }"
          @click="selectList(l.id)"
        >
          <span class="tk-dot" :style="{ background: l.color || 'var(--text-4)' }"></span>
          <span class="tk-nav-label">{{ l.name }}</span>
          <span v-if="l.task_count - l.done_count > 0" class="tk-nav-count">{{ l.task_count - l.done_count }}</span>
        </button>
        <div v-if="lists.length === 0" class="tk-empty-list">还没有清单</div>
      </div>
    </aside>

    <!-- Center: task list -->
    <main class="tk-main">
      <header class="tk-main-head">
        <h2>{{ currentTitle }}</h2>
        <div v-if="sel.type === 'list' && currentList" class="tk-list-actions">
          <button class="tk-icon-btn" title="重命名" @click="openEditList(currentList)">✎</button>
          <button class="tk-icon-btn danger" title="删除清单" @click="removeList(currentList)">🗑</button>
        </div>
      </header>

      <div class="tk-quickadd" v-if="sel.view !== 'completed'">
        <span class="tk-qa-plus">＋</span>
        <input
          v-model="quickTitle"
          class="tk-qa-input"
          placeholder="添加任务，回车保存（支持 !1-!3 设置优先级）"
          @keyup.enter="quickAdd"
        />
      </div>

      <div v-if="loadingTasks" class="tk-loading">加载中…</div>
      <div v-else-if="tasks.length === 0" class="tk-empty">
        <div class="tk-empty-emoji">🎉</div>
        <div>{{ sel.view === 'completed' ? '还没有已完成的任务' : '没有任务，享受当下' }}</div>
      </div>

      <ul v-else class="tk-tasklist">
        <li
          v-for="t in tasks" :key="t.id"
          class="tk-task" :class="{ done: t.status === 'done', active: detail?.id === t.id }"
          @click="openDetail(t)"
        >
          <button class="tk-check" :class="{ checked: t.status === 'done' }"
                  @click.stop="toggleDone(t)" :aria-label="t.status === 'done' ? '标记未完成' : '标记完成'">
            <svg v-if="t.status === 'done'" width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="M5 12l5 5L20 7"/></svg>
          </button>
          <span v-if="t.priority > 0" class="tk-flag" :style="{ color: priColor(t.priority) }" :title="priLabel(t.priority)">⚑</span>
          <span class="tk-title">{{ t.title }}</span>
          <span class="tk-meta">
            <span v-for="tag in t.tags" :key="tag" class="tk-tag">#{{ tag }}</span>
            <span v-if="t.due_date" class="tk-due" :class="{ overdue: isOverdue(t) }">{{ fmtDue(t.due_date) }}</span>
            <span v-if="listColorOf(t)" class="tk-dot sm" :style="{ background: listColorOf(t) }"></span>
          </span>
        </li>
      </ul>
    </main>

    <!-- Right: detail panel -->
    <aside v-if="detail" class="tk-detail">
      <div class="tk-detail-head">
        <button class="tk-check lg" :class="{ checked: detail.status === 'done' }" @click="toggleDone(detail)">
          <svg v-if="detail.status === 'done'" width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3"><path d="M5 12l5 5L20 7"/></svg>
        </button>
        <button class="tk-detail-close" @click="detail = null">×</button>
      </div>

      <input v-model="detail.title" class="tk-d-title" @blur="saveDetail('title')" @keyup.enter="saveDetail('title')" />

      <div class="tk-d-row">
        <label class="tk-d-label">优先级</label>
        <select v-model.number="detail.priority" class="tk-d-input" @change="saveDetail('priority')">
          <option :value="0">无</option>
          <option :value="1">低</option>
          <option :value="2">中</option>
          <option :value="3">高</option>
        </select>
      </div>
      <div class="tk-d-row">
        <label class="tk-d-label">截止</label>
        <input type="date" v-model="dueInput" class="tk-d-input" @change="saveDue" />
      </div>
      <div class="tk-d-row">
        <label class="tk-d-label">清单</label>
        <select v-model="listInput" class="tk-d-input" @change="saveList">
          <option value="">（无清单）</option>
          <option v-for="l in lists" :key="l.id" :value="l.id">{{ l.name }}</option>
        </select>
      </div>
      <div class="tk-d-row">
        <label class="tk-d-label">标签</label>
        <input v-model="tagsInput" class="tk-d-input" placeholder="逗号分隔" @blur="saveTags" @keyup.enter="saveTags" />
      </div>

      <textarea v-model="detail.note" class="tk-d-note" placeholder="备注…" @blur="saveDetail('note')"></textarea>

      <!-- Subtasks -->
      <div class="tk-sub-head">子任务</div>
      <ul class="tk-sublist">
        <li v-for="st in detail.subtasks || []" :key="st.id" class="tk-sub" :class="{ done: st.status === 'done' }">
          <button class="tk-check sm" :class="{ checked: st.status === 'done' }" @click="toggleDone(st, true)"></button>
          <span class="tk-sub-title">{{ st.title }}</span>
          <button class="tk-icon-btn tiny danger" @click="removeTask(st, true)">×</button>
        </li>
      </ul>
      <div class="tk-quickadd sub">
        <span class="tk-qa-plus">＋</span>
        <input v-model="subTitle" class="tk-qa-input" placeholder="添加子任务" @keyup.enter="addSub" />
      </div>

      <div class="tk-detail-foot">
        <button class="btn btn-ghost danger" @click="removeTask(detail)">删除任务</button>
      </div>
    </aside>

    <!-- list create/edit modal -->
    <div v-if="listModal.open" class="tk-modal-overlay" @click.self="listModal.open = false">
      <div class="tk-modal">
        <h3>{{ listModal.id ? '重命名清单' : '新建清单' }}</h3>
        <label class="tk-d-label">名称</label>
        <input v-model="listModal.name" class="tk-d-input wide" autofocus @keyup.enter="saveList2" />
        <label class="tk-d-label">颜色</label>
        <div class="tk-colors">
          <button v-for="c in palette" :key="c" class="tk-color" :class="{ on: listModal.color === c }"
                  :style="{ background: c }" @click="listModal.color = c"></button>
        </div>
        <div class="tk-modal-foot">
          <button class="btn btn-ghost" @click="listModal.open = false">取消</button>
          <button class="btn btn-primary" @click="saveList2">{{ listModal.id ? '保存' : '创建' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import * as api from "../api/tasks";
import type { Task, TaskList, SmartView } from "../api/tasks";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();

const smartViews = [
  { key: "today" as SmartView, label: "今天", icon: "☀", color: "#e8920e" },
  { key: "next7" as SmartView, label: "最近 7 天", icon: "▤", color: "#1e5fd9" },
  { key: "all" as SmartView, label: "全部", icon: "≡", color: "#0fa8a3" },
  { key: "completed" as SmartView, label: "已完成", icon: "✓", color: "#1aa971" },
];
const palette = ["#1e5fd9", "#7c4ddb", "#0fa8a3", "#e8920e", "#1aa971", "#d63838", "#5b8bf5", "#9333ea"];

const lists = ref<TaskList[]>([]);
const tasks = ref<Task[]>([]);
const counts = ref<Record<string, number>>({});
const loadingTasks = ref(false);
const detail = ref<Task | null>(null);
const quickTitle = ref("");
const subTitle = ref("");

const sel = reactive<{ type: "view" | "list"; view: SmartView; listId: string }>({
  type: "view", view: "today", listId: "",
});

const currentList = computed(() => lists.value.find((l) => l.id === sel.listId) || null);
const currentTitle = computed(() => {
  if (sel.type === "list") return currentList.value?.name ?? "清单";
  return smartViews.find((v) => v.key === sel.view)?.label ?? "任务";
});

function priColor(p: number) { return ["", "#1aa971", "#e8920e", "#d63838"][p] || ""; }
function priLabel(p: number) { return ["无", "低", "中", "高"][p] || ""; }
function listColorOf(t: Task) { return lists.value.find((l) => l.id === t.list_id)?.color || ""; }
function isOverdue(t: Task) {
  if (!t.due_date || t.status === "done") return false;
  return new Date(t.due_date) < new Date(new Date().toDateString());
}
function fmtDue(iso: string) {
  const d = new Date(iso);
  const today = new Date(new Date().toDateString());
  const diff = Math.round((new Date(d.toDateString()).getTime() - today.getTime()) / 86400000);
  if (diff === 0) return "今天";
  if (diff === 1) return "明天";
  if (diff === -1) return "昨天";
  if (diff < 0) return `逾期 ${-diff} 天`;
  return `${d.getMonth() + 1}/${d.getDate()}`;
}

async function refreshLists() { lists.value = await api.listLists(); }
async function refreshCounts() { counts.value = await api.getCounts(); }
async function refreshTasks() {
  loadingTasks.value = true;
  try {
    if (sel.type === "list") tasks.value = await api.listTasks({ list_id: sel.listId });
    else tasks.value = await api.listTasks({ view: sel.view });
  } finally { loadingTasks.value = false; }
}
async function refreshAll() { await Promise.all([refreshLists(), refreshCounts(), refreshTasks()]); }

onMounted(refreshAll);

function selectView(v: SmartView) { sel.type = "view"; sel.view = v; sel.listId = ""; detail.value = null; refreshTasks(); }
function selectList(id: string) { sel.type = "list"; sel.listId = id; detail.value = null; refreshTasks(); }

async function quickAdd() {
  let title = quickTitle.value.trim();
  if (!title) return;
  let priority = 0;
  const m = title.match(/!([1-3])\b/);
  if (m) { priority = Number(m[1]); title = title.replace(/!([1-3])\b/, "").trim(); }
  const payload: any = { title, priority };
  if (sel.type === "list") payload.list_id = sel.listId;
  if (sel.type === "view" && (sel.view === "today" || sel.view === "next7")) {
    payload.due_date = new Date().toISOString().slice(0, 10);
  }
  try {
    await api.createTask(payload);
    quickTitle.value = "";
    await Promise.all([refreshTasks(), refreshCounts(), refreshLists()]);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "创建失败");
  }
}

async function toggleDone(t: Task, isSub = false) {
  try {
    const updated = t.status === "done" ? await api.reopenTask(t.id) : await api.completeTask(t.id);
    t.status = updated.status;
    t.completed_at = updated.completed_at;
    await Promise.all([refreshCounts(), refreshLists()]);
    if (!isSub) await refreshTasks();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  }
}

// ---- detail panel ----
const dueInput = ref("");
const listInput = ref("");
const tagsInput = ref("");

async function openDetail(t: Task) {
  try {
    const full = await api.getTask(t.id);
    detail.value = full;
    dueInput.value = full.due_date ? full.due_date.slice(0, 10) : "";
    listInput.value = full.list_id ?? "";
    tagsInput.value = (full.tags || []).join(", ");
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "加载任务失败"); }
}

async function saveDetail(field: "title" | "note" | "priority") {
  if (!detail.value) return;
  const d = detail.value;
  const patch: any = {};
  if (field === "title") { if (!d.title.trim()) return; patch.title = d.title.trim(); }
  if (field === "note") patch.note = d.note;
  if (field === "priority") patch.priority = d.priority;
  try {
    await api.updateTask(d.id, patch);
    await refreshTasks();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "保存失败"); }
}
async function saveDue() {
  if (!detail.value) return;
  try {
    await api.updateTask(detail.value.id, { due_date: dueInput.value });
    detail.value.due_date = dueInput.value || null;
    await Promise.all([refreshTasks(), refreshCounts()]);
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "保存失败"); }
}
async function saveList() {
  if (!detail.value) return;
  try {
    await api.updateTask(detail.value.id, { list_id: listInput.value });
    detail.value.list_id = listInput.value || null;
    await Promise.all([refreshTasks(), refreshLists()]);
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "保存失败"); }
}
async function saveTags() {
  if (!detail.value) return;
  const tags = tagsInput.value.split(/[,，]/).map((s) => s.trim()).filter(Boolean);
  try {
    await api.updateTask(detail.value.id, { tags });
    detail.value.tags = tags;
    await refreshTasks();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "保存失败"); }
}

async function addSub() {
  if (!detail.value || !subTitle.value.trim()) return;
  try {
    await api.createTask({ title: subTitle.value.trim(), parent_id: detail.value.id });
    subTitle.value = "";
    detail.value = await api.getTask(detail.value.id);
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "添加子任务失败"); }
}

async function removeTask(t: Task, isSub = false) {
  if (!(await confirm({ title: "删除任务", message: `确认删除「${t.title}」？`, danger: true }))) return;
  await api.deleteTask(t.id);
  if (isSub && detail.value) { detail.value = await api.getTask(detail.value.id); }
  else { detail.value = null; await Promise.all([refreshTasks(), refreshCounts(), refreshLists()]); }
}

// ---- list modal ----
const listModal = reactive<{ open: boolean; id: string; name: string; color: string }>({
  open: false, id: "", name: "", color: palette[0],
});
function openNewList() { listModal.open = true; listModal.id = ""; listModal.name = ""; listModal.color = palette[0]; }
function openEditList(l: TaskList) { listModal.open = true; listModal.id = l.id; listModal.name = l.name; listModal.color = l.color || palette[0]; }
async function saveList2() {
  if (!listModal.name.trim()) { notify.warning("请输入清单名称"); return; }
  try {
    if (listModal.id) {
      await api.updateList(listModal.id, { name: listModal.name.trim(), color: listModal.color });
    } else {
      await api.createList(listModal.name.trim(), listModal.color);
    }
    listModal.open = false;
    await refreshLists();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "保存失败"); }
}
async function removeList(l: TaskList) {
  if (!(await confirm({ title: "删除清单", message: `删除清单「${l.name}」将同时删除其中所有任务，确认？`, danger: true }))) return;
  await api.deleteList(l.id);
  selectView("today");
  await refreshLists();
}
</script>

<style scoped>
.tasks-app {
  display: grid;
  grid-template-columns: 220px 1fr auto;
  height: calc(100vh - 56px);
  margin: -22px -28px -40px -28px;
  background: var(--bg);
}

/* left sidebar */
.tk-side {
  background: var(--surface);
  border-right: 1px solid var(--border);
  padding: 16px 10px;
  overflow-y: auto;
}
.tk-side-group { display: flex; flex-direction: column; gap: 2px; margin-bottom: 14px; }
.tk-side-head {
  display: flex; align-items: center; justify-content: space-between;
  font-size: 11px; font-weight: 700; color: var(--text-4);
  text-transform: uppercase; letter-spacing: .6px;
  padding: 4px 10px; margin-top: 4px;
}
.tk-add-list { border: 0; background: transparent; color: var(--text-3); font-size: 16px; cursor: pointer; line-height: 1; }
.tk-add-list:hover { color: var(--primary); }
.tk-nav {
  display: flex; align-items: center; gap: 9px;
  padding: 7px 10px; border: 0; background: transparent;
  border-radius: 8px; cursor: pointer; font-size: 13px; color: var(--text-2);
  text-align: left; width: 100%;
}
.tk-nav:hover { background: var(--bg); }
.tk-nav.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.tk-nav-ico { width: 16px; text-align: center; font-size: 14px; }
.tk-nav-label { flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tk-nav-count { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 999px; padding: 0 7px; }
.tk-nav.active .tk-nav-count { background: rgba(255,255,255,.5); }
.tk-dot { width: 9px; height: 9px; border-radius: 50%; flex-shrink: 0; }
.tk-dot.sm { width: 7px; height: 7px; }
.tk-empty-list { font-size: 12px; color: var(--text-4); padding: 6px 10px; }

/* center */
.tk-main { display: flex; flex-direction: column; padding: 20px 24px; overflow-y: auto; min-width: 0; }
.tk-main-head { display: flex; align-items: center; gap: 10px; margin-bottom: 14px; }
.tk-main-head h2 { font-size: 20px; font-weight: 700; color: var(--text); margin: 0; }
.tk-list-actions { display: flex; gap: 4px; }
.tk-icon-btn { border: 0; background: transparent; cursor: pointer; color: var(--text-3); font-size: 13px; padding: 4px 6px; border-radius: 6px; }
.tk-icon-btn:hover { background: var(--surface-2); color: var(--text); }
.tk-icon-btn.danger:hover { color: var(--danger); background: var(--danger-soft); }
.tk-icon-btn.tiny { font-size: 14px; padding: 0 4px; }

.tk-quickadd {
  display: flex; align-items: center; gap: 8px;
  background: var(--surface); border: 1px solid var(--border);
  border-radius: 10px; padding: 10px 14px; margin-bottom: 14px;
}
.tk-quickadd.sub { margin: 8px 0 0; padding: 8px 10px; }
.tk-qa-plus { color: var(--primary); font-size: 16px; }
.tk-qa-input { flex: 1; border: 0; background: transparent; outline: none; font-size: 13.5px; color: var(--text); }

.tk-loading, .tk-empty { color: var(--text-3); text-align: center; padding: 40px 0; font-size: 13px; }
.tk-empty-emoji { font-size: 34px; margin-bottom: 8px; }

.tk-tasklist { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }
.tk-task {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px; border-radius: 9px; cursor: pointer;
  background: var(--surface); border: 1px solid transparent;
}
.tk-task:hover { border-color: var(--border); }
.tk-task.active { border-color: var(--primary); box-shadow: 0 0 0 2px var(--primary-soft); }
.tk-task.done .tk-title { color: var(--text-4); text-decoration: line-through; }
.tk-check {
  width: 18px; height: 18px; border-radius: 50%;
  border: 1.5px solid var(--border-strong); background: var(--surface);
  cursor: pointer; flex-shrink: 0; display: grid; place-items: center; color: #fff; padding: 0;
}
.tk-check.lg { width: 22px; height: 22px; }
.tk-check.sm { width: 15px; height: 15px; }
.tk-check.checked { background: var(--success); border-color: var(--success); }
.tk-flag { font-size: 13px; }
.tk-title { flex: 1; min-width: 0; font-size: 13.5px; color: var(--text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.tk-meta { display: flex; align-items: center; gap: 7px; flex-shrink: 0; }
.tk-tag { font-size: 11px; color: var(--primary); background: var(--primary-soft); padding: 1px 6px; border-radius: 4px; }
.tk-due { font-size: 11.5px; color: var(--text-3); }
.tk-due.overdue { color: var(--danger); font-weight: 600; }

/* detail */
.tk-detail {
  width: 340px; background: var(--surface); border-left: 1px solid var(--border);
  padding: 16px 18px; overflow-y: auto; display: flex; flex-direction: column; gap: 12px;
}
.tk-detail-head { display: flex; align-items: center; justify-content: space-between; }
.tk-detail-close { border: 0; background: transparent; font-size: 22px; line-height: 1; color: var(--text-3); cursor: pointer; }
.tk-d-title {
  border: 0; border-bottom: 1px solid transparent; font-size: 16px; font-weight: 600;
  color: var(--text); outline: none; background: transparent; padding: 4px 0;
}
.tk-d-title:focus { border-bottom-color: var(--primary); }
.tk-d-row { display: flex; align-items: center; gap: 10px; }
.tk-d-label { font-size: 12px; color: var(--text-3); width: 48px; flex-shrink: 0; }
.tk-d-input {
  flex: 1; border: 1px solid var(--border); border-radius: 7px; padding: 6px 9px;
  font-size: 13px; background: var(--surface); color: var(--text); outline: none;
}
.tk-d-input.wide { width: 100%; }
.tk-d-input:focus { border-color: var(--primary); }
.tk-d-note {
  border: 1px solid var(--border); border-radius: 8px; padding: 8px 10px;
  font-size: 13px; min-height: 70px; resize: vertical; font-family: inherit;
  background: var(--surface); color: var(--text); outline: none;
}
.tk-d-note:focus { border-color: var(--primary); }
.tk-sub-head { font-size: 12px; font-weight: 600; color: var(--text-3); margin-top: 4px; }
.tk-sublist { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 4px; }
.tk-sub { display: flex; align-items: center; gap: 8px; font-size: 13px; }
.tk-sub.done .tk-sub-title { color: var(--text-4); text-decoration: line-through; }
.tk-sub-title { flex: 1; min-width: 0; color: var(--text); }
.tk-detail-foot { margin-top: auto; padding-top: 10px; border-top: 1px solid var(--border); }
.btn.danger { color: var(--danger); }

/* list modal */
.tk-modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.tk-modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(380px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 10px; }
.tk-modal h3 { font-size: 16px; font-weight: 600; margin: 0 0 4px; }
.tk-colors { display: flex; gap: 8px; flex-wrap: wrap; }
.tk-color { width: 24px; height: 24px; border-radius: 50%; border: 2px solid transparent; cursor: pointer; }
.tk-color.on { border-color: var(--text); box-shadow: 0 0 0 2px var(--surface) inset; }
.tk-modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 6px; }
</style>
