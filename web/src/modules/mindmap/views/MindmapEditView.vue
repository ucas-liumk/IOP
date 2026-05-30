<template>
  <section class="mm-edit">
    <header class="mm-toolbar">
      <button class="mm-tb-back" title="返回列表" @click="goBack">←</button>
      <input
        v-model="title"
        class="mm-title-input"
        placeholder="导图标题"
        @blur="onTitleBlur"
        @keyup.enter="($event.target as HTMLInputElement).blur()"
      />

      <div class="mm-tb-group" v-if="ready">
        <button class="mm-tb-btn" title="插入子节点 (Tab)" @click="cmd('INSERT_CHILD_NODE')">+子节点</button>
        <button class="mm-tb-btn" title="插入同级节点 (Enter)" @click="cmd('INSERT_NODE')">+同级</button>
        <button class="mm-tb-btn" title="删除节点 (Delete)" @click="cmd('REMOVE_NODE')">删除</button>
        <span class="mm-tb-sep"></span>
        <button class="mm-tb-btn" title="撤销" @click="cmd('BACK')">↶</button>
        <button class="mm-tb-btn" title="重做" @click="cmd('FORWARD')">↷</button>
        <span class="mm-tb-sep"></span>
        <button class="mm-tb-btn" title="适应画布" @click="fit">适应</button>
      </div>

      <div class="mm-tb-spacer"></div>
      <span class="mm-dirty" v-if="dirty">未保存</span>
      <button class="btn btn-primary" :disabled="saving || !ready" @click="save">
        {{ saving ? "保存中…" : "保存" }}
      </button>
    </header>

    <div v-if="loading" class="mm-edit-state">加载中…</div>
    <div v-else-if="loadError" class="mm-edit-state mm-edit-error">{{ loadError }}</div>

    <!-- simple-mind-map mounts on this container; it needs a non-zero size. -->
    <div ref="canvas" class="mm-canvas" v-show="!loading && !loadError"></div>
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from "vue";
import { useRoute, useRouter, onBeforeRouteLeave } from "vue-router";
// NOTE: requires the npm package "simple-mind-map" to be installed.
// The integration agent runs `npm install simple-mind-map`; do not install here.
// Docs: https://github.com/wanglin2/mind-map — `new MindMap({ el, data })`,
// `getData()` / `setData()` round-trip the {data:{text},children:[]} node tree.
// @ts-ignore — types ship with the package once installed.
import MindMap from "simple-mind-map";
import { getMap, updateMap, type MindNode } from "../api";
import { useNotification } from "@/shell/notify";

const route = useRoute();
const router = useRouter();
const notify = useNotification();

const id = String(route.params.id);
const canvas = ref<HTMLDivElement | null>(null);
const title = ref("");
const loading = ref(true);
const loadError = ref("");
const ready = ref(false);
const saving = ref(false);
const dirty = ref(false);

let mind: any = null;

async function init() {
  loading.value = true;
  try {
    const m = await getMap(id);
    title.value = m.title;
    const data: MindNode = m.data ?? { data: { text: m.title }, children: [] };
    loading.value = false;
    // Wait a tick so v-show reveals the container with a real size before mount.
    requestAnimationFrame(() => mount(data));
  } catch (e: any) {
    loading.value = false;
    loadError.value = e.response?.data?.error?.message ?? "加载失败";
  }
}

function mount(data: MindNode) {
  if (!canvas.value) return;
  try {
    // simple-mind-map applies its own defaults; its bundled .d.ts incorrectly
    // marks every constructor option as required, so cast to bypass it.
    mind = new MindMap({
      el: canvas.value,
      data,
      layout: "logicalStructure",
      theme: "default",
    } as any);
    mind.on("data_change", () => {
      dirty.value = true;
    });
    ready.value = true;
  } catch (e: any) {
    loadError.value =
      "思维导图编辑器初始化失败，请确认已安装 simple-mind-map 依赖。" + (e?.message ? `（${e.message}）` : "");
  }
}

function cmd(command: string, ...args: unknown[]) {
  if (mind) mind.execCommand(command, ...args);
}

function fit() {
  if (mind && typeof mind.fit === "function") mind.fit();
}

async function save() {
  if (!mind) return;
  saving.value = true;
  try {
    const data: MindNode = mind.getData(); // node tree only: {data:{text},children:[]}
    await updateMap(id, { title: title.value.trim() || "未命名导图", data });
    dirty.value = false;
    notify.success("已保存");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  } finally {
    saving.value = false;
  }
}

async function onTitleBlur() {
  const t = title.value.trim();
  if (!t) {
    title.value = "未命名导图";
  }
  dirty.value = true;
}

function goBack() {
  router.push("/mindmap");
}

onBeforeRouteLeave(() => {
  if (dirty.value && !confirm("有未保存的修改，确认离开？")) return false;
  return true;
});

onMounted(init);
onBeforeUnmount(() => {
  if (mind && typeof mind.destroy === "function") mind.destroy();
  mind = null;
});
</script>

<style scoped>
.mm-edit {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
}
.mm-toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 16px;
  border-bottom: 1px solid var(--border, #e5e7eb);
  background: var(--surface, #fff);
  flex: none;
}
.mm-tb-back {
  border: none;
  background: transparent;
  font-size: 20px;
  cursor: pointer;
  padding: 0 6px;
  color: var(--text-2);
}
.mm-title-input {
  border: 1px solid transparent;
  border-radius: 6px;
  padding: 6px 10px;
  font-size: 16px;
  font-weight: 600;
  min-width: 160px;
  max-width: 280px;
}
.mm-title-input:hover,
.mm-title-input:focus {
  border-color: var(--border, #e5e7eb);
  outline: none;
}
.mm-tb-group {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-left: 12px;
}
.mm-tb-btn {
  border: 1px solid var(--border, #e5e7eb);
  background: var(--surface, #fff);
  border-radius: 6px;
  padding: 5px 10px;
  font-size: 13px;
  cursor: pointer;
}
.mm-tb-btn:hover {
  background: #f3f4f6;
}
.mm-tb-sep {
  width: 1px;
  height: 18px;
  background: var(--border, #e5e7eb);
  margin: 0 4px;
}
.mm-tb-spacer {
  flex: 1;
}
.mm-dirty {
  font-size: 12px;
  color: #f59e0b;
  margin-right: 6px;
}
.mm-edit-state {
  padding: 48px;
  text-align: center;
  color: var(--text-3);
}
.mm-edit-error {
  color: #ef4444;
}
.mm-canvas {
  flex: 1;
  min-height: 0;
  width: 100%;
  background: #fafafa;
  overflow: hidden;
}
</style>
