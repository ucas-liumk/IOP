<template>
  <ul class="tree" :class="{ 'tree-root': depth === 0 }">
    <template v-for="node in nodes" :key="keyOf(node)">
      <li v-if="visible(node)" class="tree-li">
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

        <Transition name="tree-expand">
          <TreeView
            v-if="hasChildren(node) && isOpen(node)"
            :nodes="childrenOf(node)"
            :selected-id="selectedId"
            :checked-ids="checkedIds"
            :checkbox="checkbox"
            :cascade="cascade"
            :filter="filter"
            :id-key="idKey"
            :label-key="labelKey"
            :children-key="childrenKey"
            :depth="depth + 1"
            :default-expanded="defaultExpanded"
            :expand-signal="expandSignal"
            @select="$emit('select', $event)"
            @check="(id, val) => $emit('check', id, val)"
            @check-set="(ids) => $emit('check-set', ids)"
          >
            <template v-if="$slots.label" #label="childProps">
              <slot name="label" :node="childProps.node" />
            </template>
            <template v-if="$slots.suffix" #suffix="childProps">
              <slot name="suffix" :node="childProps.node" />
            </template>
          </TreeView>
        </Transition>
      </li>
    </template>
    <li v-if="depth === 0 && !nodes.some((n) => visible(n))" class="tree-empty">
      <slot name="empty">{{ filter ? "无匹配项" : "暂无数据" }}</slot>
    </li>
  </ul>
</template>

<script setup lang="ts">
import { reactive, computed, watch } from "vue";

// Generic, recursive tree renderer. Works with any node shape via the *-key
// props. Supports single-select mode (emit "select") and checkbox mode
// (controlled via :checked-ids + emit "check"). Expand/collapse is internal.
//
// Two checkbox flavours:
//   - default: each checkbox emits "check" (id, value) for the node alone.
//     The parent owns cascade logic (back-compat with RoleEditor et al.).
//   - cascade: toggling a node also toggles all descendants; parents show
//     indeterminate when some-but-not-all descendants are checked. The full
//     resulting checked-id set is emitted via "check-set".
//
// A `filter` string hides non-matching nodes while keeping ancestors of any
// match and auto-revealing matches.
const props = withDefaults(
  defineProps<{
    nodes: any[];
    selectedId?: string | null;
    /** checkbox mode: set of checked node ids (controlled by parent). */
    checkedIds?: string[];
    checkbox?: boolean;
    /** cascade checkbox mode: toggling a node toggles its whole subtree. */
    cascade?: boolean;
    /** case-insensitive label filter; hides non-matches, keeps ancestors. */
    filter?: string;
    idKey?: string;
    labelKey?: string;
    childrenKey?: string;
    depth?: number;
    /** expand every node by default on first render. */
    defaultExpanded?: boolean;
    /**
     * Imperative expand/collapse-all signal. Bumping this with a positive id
     * forces every node open; a negative id forces every node closed. The sign
     * is what matters (not the magnitude) — the parent flips it on each click.
     */
    expandSignal?: number;
  }>(),
  {
    selectedId: null,
    checkedIds: () => [],
    checkbox: false,
    cascade: false,
    filter: "",
    idKey: "id",
    labelKey: "name",
    childrenKey: "children",
    depth: 0,
    defaultExpanded: true,
    expandSignal: 0,
  },
);

const emit = defineEmits<{
  (e: "select", id: string): void;
  (e: "check", id: string, value: boolean): void;
  /** cascade mode only: the complete set of checked ids after a toggle. */
  (e: "check-set", ids: string[]): void;
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

// --- filtering ---
const filterLc = computed(() => props.filter.trim().toLowerCase());
function selfMatches(n: any): boolean {
  if (!filterLc.value) return true;
  return String(labelOf(n) ?? "").toLowerCase().includes(filterLc.value);
}
// A subtree is relevant if the node itself matches or any descendant does.
function subtreeMatches(n: any): boolean {
  if (selfMatches(n)) return true;
  return childrenOf(n).some(subtreeMatches);
}
// Visible = no active filter, or this node's subtree contains a match.
function visible(n: any): boolean {
  if (!filterLc.value) return true;
  return subtreeMatches(n);
}

// Per-key open state. Defaults follow `defaultExpanded`.
const openState = reactive<Record<string, boolean>>({});
// An expand/collapse-all override: while set, every node ignores its per-key
// state and follows this flag. Cleared the moment the user toggles a single node.
const forceOpen = computed(() => (props.expandSignal > 0 ? true : props.expandSignal < 0 ? false : null));
const overrideAll = reactive<{ value: boolean | null }>({ value: null });
watch(
  () => props.expandSignal,
  () => { overrideAll.value = forceOpen.value; },
);
function isOpen(n: any): boolean {
  // With an active filter, auto-expand branches that contain matches so the
  // matching descendants are revealed regardless of manual collapse state.
  if (filterLc.value && childrenOf(n).some(subtreeMatches)) return true;
  const k = keyOf(n);
  if (openState[k] === undefined && overrideAll.value !== null) return overrideAll.value;
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

// Collect a node + all descendant ids (used for cascade toggling).
function subtreeIds(n: any): string[] {
  const ids = [keyOf(n)];
  for (const c of childrenOf(n)) ids.push(...subtreeIds(c));
  return ids;
}

function onCheck(n: any, value: boolean) {
  if (props.cascade) {
    // Toggle the entire subtree and emit the resulting full checked-id set.
    const set = new Set(props.checkedIds);
    const ids = subtreeIds(n);
    for (const id of ids) {
      if (value) set.add(id);
      else set.delete(id);
    }
    emit("check-set", Array.from(set));
    return;
  }
  // Default: per-node emit (parent owns any cascade behaviour).
  emit("check", keyOf(n), value);
}

// --- checkbox mode helpers (parent owns the checked set) ---
const checkedSet = () => new Set(props.checkedIds);
function isChecked(n: any): boolean {
  return checkedSet().has(keyOf(n));
}
// indeterminate: some-but-not-all descendants checked.
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
  if (props.cascade) {
    // In cascade mode a node is checked iff every descendant is checked, so
    // indeterminate = some (but not all) descendants checked.
    return on > 0 && on < total;
  }
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

/* Animated expand/collapse for child sub-trees. */
.tree-expand-enter-active,
.tree-expand-leave-active {
  transition: opacity .18s ease, transform .18s ease;
  transform-origin: top;
  overflow: hidden;
}
.tree-expand-enter-from,
.tree-expand-leave-to {
  opacity: 0;
  transform: scaleY(.96) translateY(-2px);
}
@media (prefers-reduced-motion: reduce) {
  .twisty { transition: none; }
  .tree-expand-enter-active,
  .tree-expand-leave-active { transition: none; }
  .tree-expand-enter-from,
  .tree-expand-leave-to { opacity: 1; transform: none; }
}
</style>
