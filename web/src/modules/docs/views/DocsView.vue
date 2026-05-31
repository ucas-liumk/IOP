<template>
  <div class="docs-app">
    <!-- Left: folder/doc tree -->
    <aside class="dk-side">
      <header class="dk-side-head">
        <span>知识库</span>
        <div class="dk-side-actions">
          <button class="dk-icon-btn" title="新建目录" @click="createRoot('folder')">📁＋</button>
          <button class="dk-icon-btn" title="新建文档" @click="createRoot('doc')">📄＋</button>
        </div>
      </header>

      <div class="dk-search">
        <input v-model="filter" class="dk-search-input" placeholder="搜索文档…" />
      </div>

      <div class="dk-tree-wrap">
        <TreeView
          :nodes="tree"
          :selected-id="selectedId"
          :filter="filter"
          id-key="id"
          label-key="title"
          children-key="children"
          :default-expanded="true"
          @select="onSelect"
        >
          <template #label="{ node }">
            <span class="dk-node-label">
              <span class="dk-node-ico">{{ node.type === 'folder' ? '📁' : '📄' }}</span>
              <span class="dk-node-title">{{ node.title }}</span>
            </span>
          </template>
          <template #suffix="{ node }">
            <button v-if="node.type === 'folder'" class="dk-row-btn" title="在此新建文档" @click="createChild(node, 'doc')">＋</button>
            <button v-if="node.type === 'folder'" class="dk-row-btn" title="在此新建子目录" @click="createChild(node, 'folder')">📁</button>
            <button class="dk-row-btn" title="重命名" @click="renameNodePrompt(node)">✎</button>
            <button class="dk-row-btn danger" title="删除" @click="removeNode(node)">🗑</button>
          </template>
          <template #empty>还没有文档，点击右上角新建</template>
        </TreeView>
      </div>
    </aside>

    <!-- Right: editor -->
    <main class="dk-main">
      <div v-if="!current" class="dk-blank">
        <div class="dk-blank-emoji">📖</div>
        <div>{{ tree.length ? '从左侧选择一篇文档开始' : '新建你的第一篇文档' }}</div>
      </div>

      <div v-else-if="current.type === 'folder'" class="dk-blank">
        <div class="dk-blank-emoji">📁</div>
        <div>「{{ current.title }}」是一个目录</div>
        <div class="dk-blank-actions">
          <button class="dk-btn" @click="createChild(current, 'doc')">＋ 在此新建文档</button>
        </div>
      </div>

      <template v-else>
        <header class="dk-editor-head">
          <input v-model="title" class="dk-title-input" placeholder="无标题文档" />
          <div class="dk-editor-tools">
            <span class="dk-saved" :class="{ dirty }">{{ dirty ? '未保存' : '已保存' }}</span>
            <button class="dk-mode-btn" :class="{ active: mode === 'rich' }" @click="mode = 'rich'">富文本</button>
            <button class="dk-mode-btn" :class="{ active: mode === 'md' }" @click="mode = 'md'">源码</button>
            <button class="dk-btn primary" :disabled="saving || !dirty" @click="save">{{ saving ? '保存中…' : '保存' }}</button>
          </div>
        </header>

        <!-- Rich-text toolbar (rich mode only) -->
        <div v-if="mode === 'rich'" class="dk-toolbar">
          <button class="dk-tb" title="加粗" @mousedown.prevent="exec('bold')"><b>B</b></button>
          <button class="dk-tb" title="斜体" @mousedown.prevent="exec('italic')"><i>I</i></button>
          <button class="dk-tb" title="下划线" @mousedown.prevent="exec('underline')"><u>U</u></button>
          <span class="dk-tb-sep" />
          <button class="dk-tb" title="标题" @mousedown.prevent="exec('formatBlock', 'H2')">H2</button>
          <button class="dk-tb" title="正文" @mousedown.prevent="exec('formatBlock', 'P')">¶</button>
          <span class="dk-tb-sep" />
          <button class="dk-tb" title="无序列表" @mousedown.prevent="exec('insertUnorderedList')">•</button>
          <button class="dk-tb" title="有序列表" @mousedown.prevent="exec('insertOrderedList')">1.</button>
          <button class="dk-tb" title="引用" @mousedown.prevent="exec('formatBlock', 'BLOCKQUOTE')">❝</button>
          <button class="dk-tb" title="代码块" @mousedown.prevent="exec('formatBlock', 'PRE')">{ }</button>
          <span class="dk-tb-sep" />
          <button class="dk-tb" title="清除格式" @mousedown.prevent="exec('removeFormat')">✕</button>
        </div>

        <div class="dk-editor-body">
          <!-- Rich-text: contenteditable -->
          <div
            v-show="mode === 'rich'"
            ref="editorEl"
            class="dk-rich"
            contenteditable="true"
            @input="onRichInput"
          ></div>

          <!-- Source: textarea (markdown / html) -->
          <textarea
            v-show="mode === 'md'"
            v-model="content"
            class="dk-source"
            placeholder="在这里输入内容（支持 HTML / Markdown）…"
            @input="dirty = true"
          ></textarea>
        </div>
      </template>
    </main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, nextTick, watch } from "vue";
import { TreeView } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  getTree, getDoc, createNode, saveDoc, renameNode, deleteNode,
  type DocNode, type NodeType,
} from "../api";

const notify = useNotification();
const { confirm } = useConfirm();

const tree = reactive<DocNode[]>([]);
const selectedId = ref<string | null>(null);
const current = ref<DocNode | null>(null);
const filter = ref("");

const title = ref("");
const content = ref("");
const dirty = ref(false);
const saving = ref(false);
const mode = ref<"rich" | "md">("rich");
const editorEl = ref<HTMLDivElement | null>(null);

function errMsg(e: any, fallback: string): string {
  return e?.response?.data?.error?.message ?? fallback;
}

async function loadTree() {
  try {
    const t = await getTree();
    tree.splice(0, tree.length, ...t);
  } catch (e: any) {
    notify.error(errMsg(e, "加载知识库失败"));
  }
}

async function onSelect(id: string) {
  if (current.value && current.value.type === "doc" && dirty.value) {
    const ok = await confirm({ message: "当前文档未保存，确定离开吗？", danger: true, confirmText: "离开" });
    if (!ok) return;
  }
  selectedId.value = id;
  const node = findNode(tree, id);
  if (!node) return;
  if (node.type === "folder") {
    current.value = node;
    return;
  }
  try {
    const doc = await getDoc(id);
    current.value = doc;
    title.value = doc.title;
    content.value = doc.content ?? "";
    dirty.value = false;
    await nextTick();
    if (editorEl.value) editorEl.value.innerHTML = content.value;
  } catch (e: any) {
    notify.error(errMsg(e, "加载文档失败"));
  }
}

function findNode(nodes: DocNode[], id: string): DocNode | null {
  for (const n of nodes) {
    if (n.id === id) return n;
    if (n.children?.length) {
      const f = findNode(n.children, id);
      if (f) return f;
    }
  }
  return null;
}

function exec(cmd: string, arg?: string) {
  document.execCommand(cmd, false, arg);
  onRichInput();
  editorEl.value?.focus();
}

function onRichInput() {
  if (editorEl.value) {
    content.value = editorEl.value.innerHTML;
    dirty.value = true;
  }
}

// Keep the contenteditable in sync when switching back from source mode.
watch(mode, async (m) => {
  await nextTick();
  if (m === "rich" && editorEl.value) editorEl.value.innerHTML = content.value;
});

async function save() {
  if (!current.value || current.value.type !== "doc") return;
  saving.value = true;
  try {
    const t = title.value.trim() || "无标题文档";
    const updated = await saveDoc(current.value.id, content.value, t);
    current.value = updated;
    title.value = updated.title;
    dirty.value = false;
    const node = findNode(tree, updated.id);
    if (node) node.title = updated.title;
    notify.success("已保存");
  } catch (e: any) {
    notify.error(errMsg(e, "保存失败"));
  } finally {
    saving.value = false;
  }
}

async function createRoot(type: NodeType) {
  await doCreate(type, undefined);
}

async function createChild(parent: DocNode, type: NodeType) {
  await doCreate(type, parent.id);
}

async function doCreate(type: NodeType, parentId?: string) {
  const def = type === "folder" ? "新目录" : "新文档";
  const name = window.prompt(type === "folder" ? "目录名称" : "文档标题", def);
  if (name === null) return;
  const t = name.trim() || def;
  try {
    const node = await createNode({ title: t, type, parent_id: parentId, content: "" });
    await loadTree();
    if (type === "doc") await onSelect(node.id);
    else selectedId.value = node.id;
    notify.success("已创建");
  } catch (e: any) {
    notify.error(errMsg(e, "创建失败"));
  }
}

async function renameNodePrompt(node: DocNode) {
  const name = window.prompt("重命名", node.title);
  if (name === null) return;
  const t = name.trim();
  if (!t || t === node.title) return;
  try {
    await renameNode(node.id, t);
    node.title = t;
    if (current.value?.id === node.id) {
      title.value = t;
      current.value = { ...current.value, title: t };
    }
    notify.success("已重命名");
  } catch (e: any) {
    notify.error(errMsg(e, "重命名失败"));
  }
}

async function removeNode(node: DocNode) {
  const isFolder = node.type === "folder";
  const ok = await confirm({
    title: "删除",
    message: isFolder
      ? `确定删除目录「${node.title}」及其下所有内容吗？此操作不可恢复。`
      : `确定删除文档「${node.title}」吗？此操作不可恢复。`,
    danger: true,
    confirmText: "删除",
  });
  if (!ok) return;
  try {
    await deleteNode(node.id);
    if (current.value && (current.value.id === node.id || isDescendantOf(current.value.id, node))) {
      current.value = null;
      selectedId.value = null;
    }
    await loadTree();
    notify.success("已删除");
  } catch (e: any) {
    notify.error(errMsg(e, "删除失败"));
  }
}

function isDescendantOf(id: string, ancestor: DocNode): boolean {
  if (!ancestor.children?.length) return false;
  return !!findNode(ancestor.children, id);
}

loadTree();
</script>

<style scoped>
.docs-app {
  display: flex;
  height: 100%;
  min-height: 0;
  background: var(--bg);
}

/* ---- Left tree ---- */
.dk-side {
  width: 280px;
  flex-shrink: 0;
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: var(--surface);
}
.dk-side-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 14px 10px;
  font-weight: 700;
  color: var(--text);
}
.dk-side-actions { display: flex; gap: 4px; }
.dk-icon-btn {
  border: 1px solid var(--border);
  background: var(--surface);
  border-radius: 6px;
  padding: 3px 7px;
  font-size: 12px;
  cursor: pointer;
  color: var(--text-2);
}
.dk-icon-btn:hover { background: var(--surface-2); }
.dk-search { padding: 0 12px 8px; }
.dk-search-input {
  width: 100%;
  border: 1px solid var(--border);
  border-radius: 7px;
  padding: 6px 10px;
  font-size: 13px;
  background: var(--bg);
  color: var(--text);
  box-sizing: border-box;
}
.dk-tree-wrap { flex: 1; overflow-y: auto; padding: 4px 6px 16px; min-height: 0; }
.dk-node-label { display: inline-flex; align-items: center; gap: 6px; min-width: 0; }
.dk-node-ico { flex-shrink: 0; font-size: 13px; }
.dk-node-title { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dk-row-btn {
  border: 0;
  background: transparent;
  cursor: pointer;
  font-size: 12px;
  opacity: 0;
  padding: 2px 4px;
  border-radius: 4px;
  color: var(--text-3);
}
.tree-row:hover .dk-row-btn { opacity: 1; }
.dk-row-btn:hover { background: var(--bg); color: var(--text); }
.dk-row-btn.danger:hover { color: var(--danger, #e5484d); }

/* ---- Right editor ---- */
.dk-main { flex: 1; display: flex; flex-direction: column; min-width: 0; min-height: 0; }
.dk-blank {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: var(--text-3);
}
.dk-blank-emoji { font-size: 48px; }
.dk-blank-actions { margin-top: 8px; }

.dk-editor-head {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px 28px 8px;
  border-bottom: 1px solid var(--border);
}
.dk-title-input {
  flex: 1;
  border: 0;
  outline: none;
  font-size: 24px;
  font-weight: 700;
  background: transparent;
  color: var(--text);
}
.dk-editor-tools { display: flex; align-items: center; gap: 8px; }
.dk-saved { font-size: 12px; color: var(--text-4); }
.dk-saved.dirty { color: var(--warning, #f5a623); }
.dk-mode-btn {
  border: 1px solid var(--border);
  background: var(--surface);
  border-radius: 6px;
  padding: 4px 10px;
  font-size: 12px;
  cursor: pointer;
  color: var(--text-2);
}
.dk-mode-btn.active { background: var(--primary-soft); color: var(--primary); border-color: var(--primary); }

.dk-btn {
  border: 1px solid var(--border);
  background: var(--surface);
  border-radius: 7px;
  padding: 6px 14px;
  font-size: 13px;
  cursor: pointer;
  color: var(--text);
}
.dk-btn:hover { background: var(--surface-2); }
.dk-btn.primary { background: var(--primary); color: #fff; border-color: var(--primary); }
.dk-btn.primary:disabled { opacity: .5; cursor: not-allowed; }

.dk-toolbar {
  display: flex;
  align-items: center;
  gap: 2px;
  padding: 8px 28px;
  border-bottom: 1px solid var(--border);
  flex-wrap: wrap;
}
.dk-tb {
  border: 1px solid transparent;
  background: transparent;
  border-radius: 6px;
  min-width: 30px;
  height: 30px;
  cursor: pointer;
  color: var(--text-2);
  font-size: 13px;
}
.dk-tb:hover { background: var(--surface-2); border-color: var(--border); }
.dk-tb-sep { width: 1px; height: 18px; background: var(--border); margin: 0 6px; }

.dk-editor-body { flex: 1; overflow-y: auto; min-height: 0; }
.dk-rich {
  min-height: 100%;
  padding: 24px 32px;
  outline: none;
  font-size: 15px;
  line-height: 1.75;
  color: var(--text);
}
.dk-rich:empty::before {
  content: "开始撰写…";
  color: var(--text-4);
}
.dk-rich :deep(h1), .dk-rich :deep(h2), .dk-rich :deep(h3) { font-weight: 700; margin: .8em 0 .4em; }
.dk-rich :deep(blockquote) {
  border-left: 3px solid var(--border);
  margin: .6em 0;
  padding: .2em 1em;
  color: var(--text-2);
}
.dk-rich :deep(pre) {
  background: var(--surface-2);
  padding: 12px 14px;
  border-radius: 8px;
  overflow-x: auto;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
}
.dk-rich :deep(ul), .dk-rich :deep(ol) { padding-left: 1.6em; margin: .4em 0; }
.dk-source {
  width: 100%;
  height: 100%;
  min-height: 60vh;
  box-sizing: border-box;
  border: 0;
  outline: none;
  resize: none;
  padding: 24px 32px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13.5px;
  line-height: 1.7;
  background: var(--bg);
  color: var(--text);
}
</style>
