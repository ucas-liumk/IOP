<template>
  <div class="pj-app">
    <header class="pj-head">
      <div>
        <h2 class="pj-title">项目管理</h2>
        <p class="pj-sub">看板式协作 · 拖拽流转任务卡片</p>
      </div>
      <button class="btn btn-primary" @click="openNew">＋ 新建项目</button>
    </header>

    <div v-if="loading" class="pj-state">加载中…</div>
    <div v-else-if="projects.length === 0" class="pj-empty">
      <div class="pj-empty-emoji">📋</div>
      <div class="pj-empty-text">还没有项目，点击右上角新建一个看板</div>
    </div>

    <div v-else class="pj-grid">
      <article
        v-for="p in projects" :key="p.id"
        class="pj-card" :class="{ archived: p.status === 'archived' }"
        @click="openBoard(p.id)"
      >
        <div class="pj-card-top">
          <h3 class="pj-card-name">{{ p.name }}</h3>
          <span v-if="p.status === 'archived'" class="pj-badge">已归档</span>
        </div>
        <p class="pj-card-desc">{{ p.description || "暂无描述" }}</p>
        <div class="pj-card-foot">
          <span class="pj-chip">{{ p.card_count }} 张卡片</span>
          <span class="pj-time">{{ fmtDate(p.created_at) }}</span>
        </div>
        <div class="pj-card-actions" @click.stop>
          <button class="pj-mini" :title="p.status === 'archived' ? '恢复' : '归档'" @click="toggleArchive(p)">
            {{ p.status === 'archived' ? '↺' : '📥' }}
          </button>
          <button class="pj-mini danger" title="删除" @click="removeProject(p)">🗑</button>
        </div>
      </article>
    </div>

    <!-- create modal -->
    <div v-if="modal.open" class="pj-overlay" @click.self="modal.open = false">
      <div class="pj-modal">
        <h3>新建项目</h3>
        <label class="pj-label">项目名称</label>
        <input v-model="modal.name" class="pj-input" placeholder="例如：Q3 产品迭代" autofocus @keyup.enter="save" />
        <label class="pj-label">描述（可选）</label>
        <textarea v-model="modal.description" class="pj-textarea" placeholder="一句话说明这个项目的目标"></textarea>
        <div class="pj-modal-foot">
          <button class="btn btn-ghost" @click="modal.open = false">取消</button>
          <button class="btn btn-primary" @click="save">创建</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { useRouter } from "vue-router";
import * as api from "../api";
import type { Project } from "../api";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const router = useRouter();
const notify = useNotification();
const { confirm } = useConfirm();

const projects = ref<Project[]>([]);
const loading = ref(false);

const modal = reactive<{ open: boolean; name: string; description: string }>({
  open: false, name: "", description: "",
});

function fmtDate(iso: string) {
  const d = new Date(iso);
  return `${d.getFullYear()}/${d.getMonth() + 1}/${d.getDate()}`;
}

async function refresh() {
  loading.value = true;
  try {
    projects.value = await api.listProjects();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载项目失败");
  } finally {
    loading.value = false;
  }
}
onMounted(refresh);

function openNew() {
  modal.open = true;
  modal.name = "";
  modal.description = "";
}

async function save() {
  if (!modal.name.trim()) {
    notify.warning("请输入项目名称");
    return;
  }
  try {
    const p = await api.createProject({ name: modal.name.trim(), description: modal.description.trim() });
    modal.open = false;
    openBoard(p.id);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "创建失败");
  }
}

function openBoard(id: string) {
  router.push(`/project/${id}`);
}

async function toggleArchive(p: Project) {
  const next = p.status === "archived" ? "active" : "archived";
  try {
    await api.updateProject(p.id, { status: next });
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  }
}

async function removeProject(p: Project) {
  if (!(await confirm({ title: "删除项目", message: `删除「${p.name}」将同时删除其所有看板列与卡片，确认？`, danger: true }))) return;
  try {
    await api.deleteProject(p.id);
    await refresh();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}
</script>

<style scoped>
.pj-app { padding: 4px 2px 40px; }
.pj-head { display: flex; align-items: flex-start; justify-content: space-between; margin-bottom: 20px; }
.pj-title { font-size: 20px; font-weight: 700; color: var(--text); margin: 0; }
.pj-sub { font-size: 13px; color: var(--text-3); margin: 4px 0 0; }

.pj-state, .pj-empty { color: var(--text-3); text-align: center; padding: 60px 0; }
.pj-empty-emoji { font-size: 40px; margin-bottom: 10px; }
.pj-empty-text { font-size: 13.5px; }

.pj-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 16px; }
.pj-card {
  position: relative; background: var(--surface); border: 1px solid var(--border);
  border-radius: 12px; padding: 16px 18px; cursor: pointer; transition: box-shadow .15s, transform .1s;
  display: flex; flex-direction: column; gap: 8px; min-height: 130px;
}
.pj-card:hover { box-shadow: var(--sh-2); transform: translateY(-1px); border-color: var(--border-strong); }
.pj-card.archived { opacity: .68; }
.pj-card-top { display: flex; align-items: center; gap: 8px; }
.pj-card-name { font-size: 15.5px; font-weight: 600; color: var(--text); margin: 0; flex: 1; min-width: 0; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.pj-badge { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 999px; padding: 1px 8px; flex-shrink: 0; }
.pj-card-desc { font-size: 13px; color: var(--text-3); margin: 0; flex: 1; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.pj-card-foot { display: flex; align-items: center; justify-content: space-between; margin-top: 2px; }
.pj-chip { font-size: 11.5px; color: var(--primary); background: var(--primary-soft); border-radius: 6px; padding: 2px 8px; }
.pj-time { font-size: 11.5px; color: var(--text-4); }
.pj-card-actions { position: absolute; top: 10px; right: 10px; display: none; gap: 4px; }
.pj-card:hover .pj-card-actions { display: flex; }
.pj-mini { border: 0; background: var(--surface-2); border-radius: 6px; width: 26px; height: 26px; cursor: pointer; font-size: 13px; color: var(--text-3); }
.pj-mini:hover { background: var(--bg); color: var(--text); }
.pj-mini.danger:hover { color: var(--danger); background: var(--danger-soft); }

.pj-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.pj-modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(420px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 8px; }
.pj-modal h3 { font-size: 16px; font-weight: 600; margin: 0 0 6px; }
.pj-label { font-size: 12px; color: var(--text-3); margin-top: 4px; }
.pj-input, .pj-textarea {
  border: 1px solid var(--border); border-radius: 8px; padding: 8px 10px;
  font-size: 13.5px; background: var(--surface); color: var(--text); outline: none; font-family: inherit;
}
.pj-input:focus, .pj-textarea:focus { border-color: var(--primary); }
.pj-textarea { min-height: 64px; resize: vertical; }
.pj-modal-foot { display: flex; justify-content: flex-end; gap: 8px; margin-top: 10px; }
</style>
