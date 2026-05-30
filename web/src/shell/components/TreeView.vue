<template>
  <ul class="tree" :class="{ 'tree-root': depth === 0 }">
    <li v-for="node in nodes" :key="keyOf(node)" class="tree-li">
      <div
        class="tree-row"
        :class="{ selected: !checkbox && selectedId === keyOf(node) }"
        :style="{ paddingLeft: depth * 16 + 8 + 'px' }"
        @click="onRowClick(node)"
      >
        <button
          v-if="hasChildren(node)"
          class="twisty"
          :class="{ open: isOpen(node) }"
          type="button"
          @click.stop="toggle(node)"
          :aria-label="isOpen(node) ? '折叠' : '展开'"
        >
          <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3">
            <polyline points="9 6 15 12 9 18" />
          </svg>
        </button>
        <span v-else class="twisty-spacer" />

        <input
          v-if="checkbox"
          type="checkbox"
          class="tree-check"
          :checked="isChecked(node)"
          :indeterminate="isIndeterminate(node)"
          @click.stop
          @change="onCheck(node, ($event.target as HTMLInputElement).checked)"
        />

        <span class="tree-label">
          <slot name="label" :node="node">{{ labelOf(node) }}</slot>
        </span>

        <span v-if="$slots.suffix" class="tree-suffix" @click.stop>
          <slot name="suffix" :node="node" />
        </span>
      </div>

      <TreeView
        v-if="hasChildren(node) && isOpen(node)"
        :nodes="childrenOf(node)"
        :selected-id="selectedId"
        :checked-ids="checkedIds"
        :checkbox="checkbox"
        :id-key="idKey"
        :label-key="labelKey"
        :children-key="childrenKey"
        :depth="depth + 1"
        :default-expanded="defaultExpanded"
        @select="$emit('select', $event)"
        @check="(id, val) => $emit('check', id, val)"
      >
        <template v-if="$slots.label" #label="childProps">
          <slot name="label" :node="childProps.node" />
        </template>
        <template v-if="$slots.suffix" #suffix="childProps">
          <slot name="suffix" :node="childProps.node" />
        </template>
      </TreeView>
    </li>
    <li v-if="nodes.length === 0 && depth === 0" class="tree-empty">
      <slot name="empty">暂无数据</slot>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { reactive } from "vue";

// Generic, recursive tree renderer. Works with any node shape via the *-key
// props. Supports single-select mode (emit "select") and checkbox mode
// (controlled via :checked-ids + emit "check"). Expand/collapse is internal.
const props = withDefaults(
  defineProps<{
    nodes: any[];
    selectedId?: string | null;
    /** checkbox mode: set of checked node ids (controlled by parent). */
    checkedIds?: string[];
    checkbox?: boolean;
    idKey?: string;
    labelKey?: string;
    childrenKey?: string;
    depth?: number;
    /** expand every node by default on first render. */
    defaultExpanded?: boolean;
  }>(),
  {
    selectedId: null,
    checkedIds: () => [],
    checkbox: false,
    idKey: "id",
    labelKey: "name",
    childrenKey: "children",
    depth: 0,
    defaultExpanded: true,
  },
);

const emit = defineEmits<{
  (e: "select", id: string): void;
  (e: "check", id: string, value: boolean): void;
}>();

// Explicit slot typing — required so the self-recursive <TreeView> use below can
// infer the forwarded slot-prop types instead of failing with TS7022.
defineSlots<{
  label?(props: { node: any }): any;
  suffix?(props: { node: any }): any;
  empty?(): any;
}>();

const keyOf = (n: any): string => String(n[props.idKey]);
const labelOf = (n: any): string => n[props.labelKey];
const childrenOf = (n: any): any[] => n[props.childrenKey] ?? [];
const hasChildren = (n: any): boolean => childrenOf(n).length > 0;

// Per-key open state. Defaults follow `defaultExpanded`.
const openState = reactive<Record<string, boolean>>({});
function isOpen(n: any): boolean {
  const k = keyOf(n);
  return openState[k] ?? props.defaultExpanded;
}
function toggle(n: any) {
  const k = keyOf(n);
  openState[k] = !isOpen(n);
}

function onRowClick(n: any) {
  if (props.checkbox) return;
  emit("select", keyOf(n));
}
function onCheck(n: any, value: boolean) {
  emit("check", keyOf(n), value);
}

// --- checkbox mode helpers (parent owns the checked set) ---
const checkedSet = () => new Set(props.checkedIds);
function isChecked(n: any): boolean {
  return checkedSet().has(keyOf(n));
}
// indeterminate: some-but-not-all descendants checked (purely visual).
function isIndeterminate(n: any): boolean {
  if (!hasChildren(n)) return false;
  const set = checkedSet();
  let total = 0;
  let on = 0;
  const walk = (node: any) => {
    for (const c of childrenOf(node)) {
      total++;
      if (set.has(keyOf(c))) on++;
      walk(c);
    }
  };
  walk(n);
  if (total === 0) return false;
  return on > 0 && on < total + (isChecked(n) ? 1 : 0) && !(on === total && isChecked(n));
}
</script>

<script lang="ts">
// Named export so it can recurse on itself in <template>.
export default { name: "TreeView" };
</script>

<style scoped>
.tree { list-style: none; margin: 0; padding: 0; }
.tree-root { font-size: 13px; }
.tree-li { user-select: none; }
.tree-row {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px;
  border-radius: 6px;
  cursor: default;
  color: var(--text-2);
  min-height: 30px;
}
.tree-row:hover { background: var(--surface-2); }
.tree-row.selected { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.twisty {
  border: 0;
  background: transparent;
  color: var(--text-3);
  width: 16px; height: 16px;
  display: grid; place-items: center;
  cursor: pointer;
  border-radius: 4px;
  flex-shrink: 0;
  transition: transform .12s;
}
.twisty.open { transform: rotate(90deg); }
.twisty:hover { background: var(--bg); color: var(--text); }
.twisty-spacer { width: 16px; height: 16px; flex-shrink: 0; }
.tree-check { width: 14px; height: 14px; cursor: pointer; flex-shrink: 0; accent-color: var(--primary); }
.tree-label { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tree-suffix { display: flex; align-items: center; gap: 4px; flex-shrink: 0; }
.tree-empty { color: var(--text-4); font-size: 12.5px; padding: 16px 8px; text-align: center; }
</style>
