<template>
  <div class="role-editor">
    <div class="editor-head">
      <div>
        <h3 class="re-title">
          {{ role.name }}
          <code class="re-code">{{ role.code }}</code>
          <span v-if="role.built_in" class="tag-builtin">内置</span>
        </h3>
        <p class="re-sub">勾选菜单授予对应权限 · 设置数据范围</p>
      </div>
      <button class="btn btn-primary btn-sm" :disabled="busy" @click="saveScope">{{ busy ? '保存中…' : '保存设置' }}</button>
    </div>

    <div class="editor-body">
      <!-- Menu permission tree -->
      <section class="re-section">
        <div class="re-section-head">菜单权限</div>
        <div class="menu-tree-box">
          <TreeView
            :nodes="menus"
            :checked-ids="checkedKeys"
            checkbox
            id-key="key"
            label-key="title"
            @check="onCheck"
          >
            <template #label="{ node }">
              <span class="menu-node" :class="{ dir: node.type === 'dir' }">
                {{ node.title }}
                <code v-if="node.perm" class="perm-tag">{{ node.perm }}</code>
                <span v-else class="dir-tag">目录</span>
              </span>
            </template>
          </TreeView>
          <div v-if="menus.length === 0" class="empty-hint">无可分配的菜单。</div>
        </div>
      </section>

      <!-- Data scope -->
      <section class="re-section">
        <div class="re-section-head">数据范围</div>
        <div class="scope-options">
          <label v-for="opt in scopeOptions" :key="opt.value" class="scope-opt">
            <input type="radio" :value="opt.value" v-model="dataScope" />
            <span>{{ opt.label }}</span>
          </label>
        </div>
        <div v-if="dataScope === 'custom'" class="custom-depts">
          <div class="cd-head">选择部门</div>
          <ul class="check-list">
            <li v-for="d in depts" :key="d.id">
              <label>
                <input type="checkbox" :value="d.id" v-model="customDeptIds" />
                <span>{{ d.name }}</span>
              </label>
            </li>
            <li v-if="depts.length === 0" class="muted">尚无部门。</li>
          </ul>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { TreeView } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import {
  addPolicy, removePolicy, updateRole,
  type Role, type MenuNode, type Dept, type DataScope, type PolicyRule,
} from "../api/admin";

const props = defineProps<{
  role: Role;
  menus: MenuNode[];
  depts: Dept[];
}>();

const emit = defineEmits<{ (e: "changed"): void }>();

const notify = useNotification();
const busy = ref(false);

const scopeOptions: { value: DataScope; label: string }[] = [
  { value: "all", label: "全部数据" },
  { value: "dept", label: "本部门" },
  { value: "dept_and_sub", label: "本部门及以下" },
  { value: "self", label: "仅本人" },
  { value: "custom", label: "自定义部门" },
];

const dataScope = ref<DataScope>(props.role.data_scope ?? "all");
const customDeptIds = ref<string[]>([...(props.role.dept_ids ?? [])]);

watch(
  () => props.role,
  (r) => {
    dataScope.value = r.data_scope ?? "all";
    customDeptIds.value = [...(r.dept_ids ?? [])];
  },
);

// --- Menu checkbox <-> policy mapping ---
// A menu node is "checked" when the role's policies satisfy its perm
// (resource:action). Wildcards ("*:*", "res:*") count as satisfying.
const grantedSet = computed(() => {
  const set = new Set<string>();
  for (const p of (props.role.policies ?? []) as PolicyRule[]) {
    if (p.effect === "allow") set.add(`${p.resource}:${p.action}`);
  }
  return set;
});

function permSatisfied(perm: string): boolean {
  if (!perm) return false;
  const [needRes, needAct] = splitPerm(perm);
  for (const g of grantedSet.value) {
    const [gRes, gAct] = splitPerm(g);
    if ((gRes === "*" || gRes === needRes) && (gAct === "*" || gAct === needAct)) return true;
  }
  return false;
}
function splitPerm(p: string): [string, string] {
  const i = p.lastIndexOf(":");
  if (i < 0) return [p, "*"];
  return [p.slice(0, i), p.slice(i + 1)];
}

// Keys of nodes whose perm is currently satisfied (drives the checkboxes).
const checkedKeys = computed(() => {
  const keys: string[] = [];
  walk(props.menus, (n) => {
    if (n.perm && permSatisfied(n.perm)) keys.push(n.key);
  });
  return keys;
});

// Collect the (node + descendants) perms so checking a dir cascades to children.
function collectPerms(node: MenuNode): string[] {
  const perms: string[] = [];
  const visit = (n: MenuNode) => {
    if (n.perm) perms.push(n.perm);
    for (const c of n.children ?? []) visit(c);
  };
  visit(node);
  return Array.from(new Set(perms));
}

function findNode(key: string): MenuNode | null {
  let found: MenuNode | null = null;
  walk(props.menus, (n) => { if (n.key === key) found = n; });
  return found;
}

async function onCheck(key: string, checked: boolean) {
  const node = findNode(key);
  if (!node) return;
  const perms = collectPerms(node);
  if (perms.length === 0) {
    notify.warning("该目录节点没有可授予的权限");
    return;
  }
  busy.value = true;
  try {
    for (const perm of perms) {
      const [res, act] = splitPerm(perm);
      if (checked) await addPolicy(props.role.id, res, act);
      else await removePolicy(props.role.id, res, act);
    }
    emit("changed"); // parent reloads roles so policies (and checkboxes) refresh
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "权限更新失败");
  } finally { busy.value = false; }
}

async function saveScope() {
  busy.value = true;
  try {
    await updateRole(props.role.id, {
      data_scope: dataScope.value,
      dept_ids: dataScope.value === "custom" ? customDeptIds.value : [],
    });
    notify.success("数据范围已保存");
    emit("changed");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  } finally { busy.value = false; }
}

function walk(nodes: MenuNode[], fn: (n: MenuNode) => void) {
  for (const n of nodes) {
    fn(n);
    if (n.children?.length) walk(n.children, fn);
  }
}
</script>

<style scoped>
.role-editor { display: flex; flex-direction: column; gap: 16px; }
.editor-head { display: flex; justify-content: space-between; align-items: flex-start; }
.re-title { font-size: 16px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
.re-code { font-family: var(--ff-mono); font-size: 11.5px; color: var(--text-3); padding: 2px 7px; background: var(--surface-2); border-radius: 4px; }
.tag-builtin { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--purple-soft); color: var(--purple); border-radius: 3px; }
.re-sub { font-size: 12.5px; color: var(--text-3); margin-top: 4px; }

.editor-body { display: grid; grid-template-columns: 1.4fr 1fr; gap: 18px; align-items: start; }
.re-section { display: flex; flex-direction: column; gap: 8px; }
.re-section-head { font-size: 12px; font-weight: 600; color: var(--text-2); text-transform: uppercase; letter-spacing: .4px; }
.menu-tree-box { border: 1px solid var(--border); border-radius: 8px; padding: 8px; max-height: 440px; overflow-y: auto; background: var(--surface); }
.empty-hint { color: var(--text-4); font-size: 12.5px; padding: 12px; }
.menu-node { display: inline-flex; align-items: center; gap: 6px; }
.menu-node.dir { font-weight: 600; }
.perm-tag { font-family: var(--ff-mono); font-size: 10.5px; color: var(--primary); background: var(--primary-soft); padding: 1px 5px; border-radius: 3px; }
.dir-tag { font-size: 10px; color: var(--text-4); }

.scope-options { display: flex; flex-direction: column; gap: 4px; }
.scope-opt { display: flex; align-items: center; gap: 8px; font-size: 13px; padding: 6px 8px; border-radius: 6px; cursor: pointer; }
.scope-opt:hover { background: var(--surface-2); }
.scope-opt input { accent-color: var(--primary); }
.custom-depts { margin-top: 8px; border: 1px solid var(--border); border-radius: 8px; padding: 8px; }
.cd-head { font-size: 11.5px; color: var(--text-3); margin-bottom: 4px; }
.check-list { list-style: none; margin: 0; padding: 0; max-height: 220px; overflow-y: auto; }
.check-list label { display: flex; align-items: center; gap: 8px; font-size: 13px; padding: 5px 6px; border-radius: 6px; cursor: pointer; }
.check-list label:hover { background: var(--surface-2); }
.check-list input { accent-color: var(--primary); }
.muted { color: var(--text-4); font-size: 12.5px; padding: 6px; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-sm { padding: 5px 12px; font-size: 12.5px; }
</style>
