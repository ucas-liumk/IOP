<template>
  <section class="mm-list">
    <header class="page-header">
      <div>
        <h1 class="page-title">思维导图</h1>
        <div class="page-subtitle">在线脑图 · 节点树 · 画布编辑（仿 ProcessOn / GitMind / 百度脑图）</div>
      </div>
      <button class="btn btn-primary" @click="openCreate">+ 新建导图</button>
    </header>

    <div v-if="loading" class="mm-state">加载中…</div>
    <div v-else-if="maps.length === 0" class="mm-state mm-empty">
      <div class="mm-empty-emoji">🧠</div>
      <div>还没有思维导图，点击右上角「新建导图」开始</div>
    </div>

    <ul v-else class="mm-grid">
      <li v-for="m in maps" :key="m.id" class="mm-card" @click="open(m.id)">
        <div class="mm-thumb">
          <svg viewBox="0 0 24 24" width="40" height="40" fill="currentColor">
            <path d="M18 16.08c-.76 0-1.44.3-1.96.77L8.91 12.7c.05-.23.09-.46.09-.7s-.04-.47-.09-.7l7.05-4.11c.54.5 1.25.81 2.04.81 1.66 0 3-1.34 3-3s-1.34-3-3-3-3 1.34-3 3c0 .24.04.47.09.7L8.04 9.81C7.5 9.31 6.79 9 6 9c-1.66 0-3 1.34-3 3s1.34 3 3 3c.79 0 1.5-.31 2.04-.81l7.12 4.16c-.05.21-.08.43-.08.65 0 1.61 1.31 2.92 2.92 2.92s2.92-1.31 2.92-2.92-1.31-2.92-2.92-2.92z"/>
          </svg>
        </div>
        <div class="mm-card-body">
          <div class="mm-card-title" :title="m.title">{{ m.title }}</div>
          <div class="mm-card-meta">更新于 {{ fmtTime(m.updated_at) }}</div>
        </div>
        <button class="mm-del" title="删除" @click.stop="remove(m)">🗑</button>
      </li>
    </ul>

    <!-- Create dialog -->
    <div v-if="showCreate" class="mm-mask" @click.self="showCreate = false">
      <div class="mm-dialog">
        <h3 class="mm-dialog-title">新建思维导图</h3>
        <input
          ref="titleInput"
          v-model="newTitle"
          class="input"
          placeholder="导图标题"
          @keyup.enter="create"
        />
        <div class="mm-dialog-actions">
          <button class="btn" @click="showCreate = false">取消</button>
          <button class="btn btn-primary" :disabled="creating" @click="create">
            {{ creating ? "创建中…" : "创建并编辑" }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { nextTick, onMounted, ref } from "vue";
import { useRouter } from "vue-router";
import { listMaps, createMap, deleteMap, type Mindmap } from "../api";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const router = useRouter();
const notify = useNotification();
const { confirm } = useConfirm();

const maps = ref<Mindmap[]>([]);
const loading = ref(true);

const showCreate = ref(false);
const newTitle = ref("");
const creating = ref(false);
const titleInput = ref<HTMLInputElement | null>(null);

async function reload() {
  loading.value = true;
  try {
    maps.value = await listMaps();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载失败");
  } finally {
    loading.value = false;
  }
}

function open(id: string) {
  router.push(`/mindmap/${id}`);
}

function openCreate() {
  newTitle.value = "未命名导图";
  showCreate.value = true;
  nextTick(() => titleInput.value?.select());
}

async function create() {
  const title = newTitle.value.trim();
  if (!title) {
    notify.warning("请输入标题");
    return;
  }
  creating.value = true;
  try {
    const m = await createMap(title);
    showCreate.value = false;
    router.push(`/mindmap/${m.id}`);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "创建失败");
  } finally {
    creating.value = false;
  }
}

async function remove(m: Mindmap) {
  if (!(await confirm({ title: "删除思维导图", message: `确认删除「${m.title}」？`, danger: true }))) return;
  try {
    await deleteMap(m.id);
    maps.value = maps.value.filter((x) => x.id !== m.id);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

function fmtTime(s: string): string {
  const d = new Date(s);
  if (isNaN(d.getTime())) return s;
  return d.toLocaleString();
}

onMounted(reload);
</script>

<style scoped>
.mm-list {
  padding: 24px;
}
.mm-state {
  padding: 48px;
  text-align: center;
  color: var(--text-3);
}
.mm-empty-emoji {
  font-size: 40px;
  margin-bottom: 12px;
}
.mm-grid {
  list-style: none;
  margin: 0;
  padding: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: 16px;
}
.mm-card {
  position: relative;
  display: flex;
  flex-direction: column;
  border: 1px solid var(--border, #e5e7eb);
  border-radius: 10px;
  overflow: hidden;
  cursor: pointer;
  background: var(--surface, #fff);
  transition: box-shadow 0.15s, transform 0.15s;
}
.mm-card:hover {
  box-shadow: 0 6px 20px rgba(0, 0, 0, 0.1);
  transform: translateY(-2px);
}
.mm-thumb {
  height: 110px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #eef2ff, #e0e7ff);
  color: #6366f1;
}
.mm-card-body {
  padding: 12px 14px;
}
.mm-card-title {
  font-weight: 600;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mm-card-meta {
  margin-top: 4px;
  font-size: 12px;
  color: var(--text-3);
}
.mm-del {
  position: absolute;
  top: 8px;
  right: 8px;
  border: none;
  background: rgba(255, 255, 255, 0.85);
  border-radius: 6px;
  width: 28px;
  height: 28px;
  cursor: pointer;
  opacity: 0;
  transition: opacity 0.15s;
}
.mm-card:hover .mm-del {
  opacity: 1;
}
.mm-mask {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.mm-dialog {
  background: var(--surface, #fff);
  border-radius: 12px;
  padding: 24px;
  width: 360px;
  box-shadow: 0 12px 40px rgba(0, 0, 0, 0.25);
}
.mm-dialog-title {
  margin: 0 0 16px;
  font-size: 16px;
}
.mm-dialog-actions {
  margin-top: 18px;
  display: flex;
  justify-content: flex-end;
  gap: 10px;
}
</style>
