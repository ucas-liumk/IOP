<template>
  <div class="bd-app">
    <header class="bd-head">
      <button class="bd-back" title="返回项目列表" @click="goBack">←</button>
      <input
        v-if="board"
        v-model="board.name"
        class="bd-name"
        @blur="saveName"
        @keyup.enter="saveName"
      />
      <span v-if="board && board.status === 'archived'" class="bd-arch-badge">已归档</span>
      <div class="bd-spacer"></div>
      <button v-if="board" class="btn btn-ghost" @click="addColumn">＋ 添加列</button>
    </header>

    <div v-if="loading" class="bd-state">加载中…</div>
    <div v-else-if="!board" class="bd-state">项目不存在</div>

    <div v-else class="bd-columns">
      <section
        v-for="col in board.columns || []" :key="col.id"
        class="bd-col"
        @dragover.prevent
        @drop="onDrop($event, col)"
      >
        <header class="bd-col-head">
          <input class="bd-col-name" :value="col.name" @change="renameColumn(col, ($event.target as HTMLInputElement).value)" />
          <span class="bd-col-count">{{ col.cards?.length || 0 }}</span>
          <button class="bd-col-del" title="删除该列" @click="removeColumn(col)">×</button>
        </header>

        <div class="bd-col-body">
          <article
            v-for="card in col.cards || []" :key="card.id"
            class="bd-card"
            :class="{ dragging: dragId === card.id }"
            draggable="true"
            @dragstart="onDragStart(card)"
            @dragend="dragId = null"
            @click="openCard(card)"
          >
            <span v-if="card.priority > 0" class="bd-pri" :style="{ background: priColor(card.priority) }">{{ priLabel(card.priority) }}</span>
            <div class="bd-card-title">{{ card.title }}</div>
            <div v-if="card.description" class="bd-card-desc">{{ card.description }}</div>
            <div class="bd-card-foot" v-if="card.due_date || card.assignee_id">
              <span v-if="card.due_date" class="bd-due" :class="{ overdue: isOverdue(card) }">📅 {{ fmtDue(card.due_date) }}</span>
              <span v-if="card.assignee_id" class="bd-assignee" :title="card.assignee_id">👤</span>
            </div>
          </article>

          <div v-if="addingTo === col.id" class="bd-add-card">
            <textarea
              v-model="newCardTitle"
              class="bd-add-input"
              placeholder="输入卡片标题，回车添加"
              autofocus
              @keyup.enter.prevent="confirmAddCard(col)"
              @keyup.esc="cancelAddCard"
            ></textarea>
            <div class="bd-add-actions">
              <button class="btn btn-primary btn-sm" @click="confirmAddCard(col)">添加</button>
              <button class="bd-add-cancel" @click="cancelAddCard">×</button>
            </div>
          </div>
          <button v-else class="bd-add-trigger" @click="startAddCard(col)">＋ 添加卡片</button>
        </div>
      </section>
    </div>

    <!-- card detail modal -->
    <div v-if="detail" class="bd-overlay" @click.self="closeDetail">
      <div class="bd-detail">
        <div class="bd-detail-head">
          <input v-model="detail.title" class="bd-d-title" @blur="saveCard('title')" @keyup.enter="saveCard('title')" />
          <button class="bd-detail-close" @click="closeDetail">×</button>
        </div>

        <label class="bd-d-label">描述</label>
        <textarea v-model="detail.description" class="bd-d-note" placeholder="补充细节…" @blur="saveCard('description')"></textarea>

        <div class="bd-d-row">
          <label class="bd-d-label sm">优先级</label>
          <select v-model.number="detail.priority" class="bd-d-input" @change="saveCard('priority')">
            <option :value="0">无</option>
            <option :value="1">低</option>
            <option :value="2">中</option>
            <option :value="3">高</option>
          </select>
        </div>
        <div class="bd-d-row">
          <label class="bd-d-label sm">截止</label>
          <input type="date" v-model="dueInput" class="bd-d-input" @change="saveDue" />
        </div>
        <div class="bd-d-row">
          <label class="bd-d-label sm">负责人</label>
          <input v-model="assigneeInput" class="bd-d-input" placeholder="成员 ID（可选）" @blur="saveAssignee" @keyup.enter="saveAssignee" />
        </div>
        <div class="bd-d-row">
          <label class="bd-d-label sm">移动到</label>
          <select :value="detail.column_id" class="bd-d-input" @change="moveTo(($event.target as HTMLSelectElement).value)">
            <option v-for="col in board?.columns || []" :key="col.id" :value="col.id">{{ col.name }}</option>
          </select>
        </div>

        <div class="bd-detail-foot">
          <button class="btn btn-ghost danger" @click="removeCard">删除卡片</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import * as api from "../api";
import type { Project, Column, Card } from "../api";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const route = useRoute();
const router = useRouter();
const notify = useNotification();
const { confirm } = useConfirm();

const projectId = route.params.id as string;
const board = ref<Project | null>(null);
const loading = ref(false);

const dragId = ref<string | null>(null);
const draggedCard = ref<Card | null>(null);

const addingTo = ref<string | null>(null);
const newCardTitle = ref("");

const detail = ref<Card | null>(null);
const dueInput = ref("");
const assigneeInput = ref("");

function priColor(p: number) { return ["", "#1aa971", "#e8920e", "#d63838"][p] || "transparent"; }
function priLabel(p: number) { return ["", "低", "中", "高"][p] || ""; }
function isOverdue(c: Card) {
  if (!c.due_date) return false;
  return new Date(c.due_date) < new Date(new Date().toDateString());
}
function fmtDue(iso: string) {
  const d = new Date(iso);
  return `${d.getMonth() + 1}/${d.getDate()}`;
}

async function refresh() {
  loading.value = true;
  try {
    board.value = await api.getBoard(projectId);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载看板失败");
    board.value = null;
  } finally {
    loading.value = false;
  }
}
onMounted(refresh);

function goBack() { router.push("/project"); }

async function saveName() {
  if (!board.value || !board.value.name.trim()) return;
  try {
    await api.updateProject(projectId, { name: board.value.name.trim() });
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  }
}

// ---- columns ----
async function addColumn() {
  try {
    await api.createColumn(projectId, "新列");
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "添加列失败");
  }
}
async function renameColumn(col: Column, name: string) {
  const n = name.trim();
  if (!n || n === col.name) return;
  try {
    await api.updateColumn(col.id, { name: n });
    col.name = n;
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "重命名失败");
  }
}
async function removeColumn(col: Column) {
  if (!(await confirm({ title: "删除列", message: `删除「${col.name}」将同时删除其中所有卡片，确认？`, danger: true }))) return;
  try {
    await api.deleteColumn(col.id);
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

// ---- cards: add ----
function startAddCard(col: Column) { addingTo.value = col.id; newCardTitle.value = ""; }
function cancelAddCard() { addingTo.value = null; newCardTitle.value = ""; }
async function confirmAddCard(col: Column) {
  const title = newCardTitle.value.trim();
  if (!title) { cancelAddCard(); return; }
  try {
    await api.createCard(projectId, { column_id: col.id, title });
    newCardTitle.value = "";
    await refresh();
    addingTo.value = col.id; // keep open for rapid entry
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "添加卡片失败");
  }
}

// ---- cards: drag & drop move ----
function onDragStart(card: Card) {
  dragId.value = card.id;
  draggedCard.value = card;
}
async function onDrop(_e: DragEvent, col: Column) {
  const card = draggedCard.value;
  dragId.value = null;
  draggedCard.value = null;
  if (!card || card.column_id === col.id) return;
  const order = col.cards?.length ?? 0;
  try {
    await api.moveCard(card.id, col.id, order);
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "移动失败");
  }
}

// ---- card detail ----
async function openCard(card: Card) {
  try {
    const full = await api.getCard(card.id);
    detail.value = full;
    dueInput.value = full.due_date ? full.due_date.slice(0, 10) : "";
    assigneeInput.value = full.assignee_id ?? "";
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载卡片失败");
  }
}
function closeDetail() { detail.value = null; }

async function saveCard(field: "title" | "description" | "priority") {
  if (!detail.value) return;
  const d = detail.value;
  const patch: any = {};
  if (field === "title") { if (!d.title.trim()) return; patch.title = d.title.trim(); }
  if (field === "description") patch.description = d.description;
  if (field === "priority") patch.priority = d.priority;
  try {
    await api.updateCard(d.id, patch);
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  }
}
async function saveDue() {
  if (!detail.value) return;
  try {
    await api.updateCard(detail.value.id, { due_date: dueInput.value });
    detail.value.due_date = dueInput.value || null;
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  }
}
async function saveAssignee() {
  if (!detail.value) return;
  try {
    await api.updateCard(detail.value.id, { assignee_id: assigneeInput.value.trim() });
    detail.value.assignee_id = assigneeInput.value.trim() || null;
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  }
}
async function moveTo(columnId: string) {
  if (!detail.value || columnId === detail.value.column_id) return;
  try {
    const updated = await api.moveCard(detail.value.id, columnId, 9999);
    detail.value = updated;
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "移动失败");
  }
}
async function removeCard() {
  if (!detail.value) return;
  if (!(await confirm({ title: "删除卡片", message: `确认删除「${detail.value.title}」？`, danger: true }))) return;
  try {
    await api.deleteCard(detail.value.id);
    detail.value = null;
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}
</script>

<style scoped>
.bd-app {
  display: flex; flex-direction: column;
  height: calc(100vh - 56px);
  margin: -22px -28px -40px -28px;
  background: var(--bg);
}
.bd-head {
  display: flex; align-items: center; gap: 10px;
  padding: 14px 24px; border-bottom: 1px solid var(--border); background: var(--surface); flex-shrink: 0;
}
.bd-back { border: 0; background: var(--surface-2); border-radius: 8px; width: 30px; height: 30px; cursor: pointer; font-size: 16px; color: var(--text-2); }
.bd-back:hover { background: var(--bg); color: var(--text); }
.bd-name { border: 0; border-bottom: 1px solid transparent; font-size: 18px; font-weight: 700; color: var(--text); background: transparent; outline: none; padding: 2px 0; min-width: 120px; }
.bd-name:focus { border-bottom-color: var(--primary); }
.bd-arch-badge { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 999px; padding: 2px 8px; }
.bd-spacer { flex: 1; }

.bd-state { color: var(--text-3); text-align: center; padding: 60px 0; }

.bd-columns {
  flex: 1; display: flex; gap: 14px; padding: 18px 24px;
  overflow-x: auto; align-items: flex-start;
}
.bd-col {
  flex: 0 0 280px; max-width: 280px; background: var(--surface);
  border: 1px solid var(--border); border-radius: 12px;
  display: flex; flex-direction: column; max-height: 100%;
}
.bd-col-head { display: flex; align-items: center; gap: 6px; padding: 10px 12px; border-bottom: 1px solid var(--border); }
.bd-col-name { flex: 1; min-width: 0; border: 0; background: transparent; font-size: 14px; font-weight: 600; color: var(--text); outline: none; padding: 2px 4px; border-radius: 6px; }
.bd-col-name:hover, .bd-col-name:focus { background: var(--bg); }
.bd-col-count { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 999px; padding: 1px 7px; }
.bd-col-del { border: 0; background: transparent; color: var(--text-4); cursor: pointer; font-size: 18px; line-height: 1; }
.bd-col-del:hover { color: var(--danger); }
.bd-col-body { padding: 10px; overflow-y: auto; display: flex; flex-direction: column; gap: 8px; }

.bd-card {
  background: var(--bg); border: 1px solid var(--border); border-radius: 9px;
  padding: 10px 11px; cursor: pointer; position: relative; transition: box-shadow .12s, border-color .12s;
}
.bd-card:hover { box-shadow: var(--sh-1); border-color: var(--border-strong); }
.bd-card.dragging { opacity: .4; }
.bd-pri { display: inline-block; font-size: 10px; color: #fff; border-radius: 4px; padding: 1px 6px; margin-bottom: 5px; }
.bd-card-title { font-size: 13.5px; color: var(--text); line-height: 1.35; }
.bd-card-desc { font-size: 12px; color: var(--text-3); margin-top: 4px; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.bd-card-foot { display: flex; align-items: center; gap: 8px; margin-top: 7px; }
.bd-due { font-size: 11px; color: var(--text-3); }
.bd-due.overdue { color: var(--danger); font-weight: 600; }
.bd-assignee { font-size: 12px; }

.bd-add-trigger { border: 1px dashed var(--border); background: transparent; color: var(--text-3); border-radius: 8px; padding: 7px; cursor: pointer; font-size: 12.5px; }
.bd-add-trigger:hover { border-color: var(--primary); color: var(--primary); }
.bd-add-card { display: flex; flex-direction: column; gap: 6px; }
.bd-add-input { border: 1px solid var(--primary); border-radius: 8px; padding: 8px; font-size: 13px; font-family: inherit; resize: vertical; min-height: 48px; outline: none; background: var(--surface); color: var(--text); }
.bd-add-actions { display: flex; align-items: center; gap: 6px; }
.btn-sm { padding: 4px 12px; font-size: 12.5px; }
.bd-add-cancel { border: 0; background: transparent; color: var(--text-3); font-size: 18px; cursor: pointer; line-height: 1; }

.bd-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.bd-detail { background: var(--surface); border-radius: 14px; padding: 20px 22px; width: min(440px, 94vw); max-height: 86vh; overflow-y: auto; box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 10px; }
.bd-detail-head { display: flex; align-items: flex-start; gap: 8px; }
.bd-d-title { flex: 1; border: 0; border-bottom: 1px solid transparent; font-size: 16px; font-weight: 600; color: var(--text); background: transparent; outline: none; padding: 4px 0; }
.bd-d-title:focus { border-bottom-color: var(--primary); }
.bd-detail-close { border: 0; background: transparent; font-size: 22px; line-height: 1; color: var(--text-3); cursor: pointer; }
.bd-d-label { font-size: 12px; color: var(--text-3); }
.bd-d-label.sm { width: 48px; flex-shrink: 0; }
.bd-d-row { display: flex; align-items: center; gap: 10px; }
.bd-d-input { flex: 1; border: 1px solid var(--border); border-radius: 7px; padding: 6px 9px; font-size: 13px; background: var(--surface); color: var(--text); outline: none; }
.bd-d-input:focus { border-color: var(--primary); }
.bd-d-note { border: 1px solid var(--border); border-radius: 8px; padding: 8px 10px; font-size: 13px; min-height: 70px; resize: vertical; font-family: inherit; background: var(--surface); color: var(--text); outline: none; }
.bd-d-note:focus { border-color: var(--primary); }
.bd-detail-foot { margin-top: 6px; padding-top: 10px; border-top: 1px solid var(--border); }
.btn.danger { color: var(--danger); }
</style>
