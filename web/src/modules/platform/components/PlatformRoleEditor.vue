<template>
  <div class="role-editor">
    <div class="editor-head">
      <div>
        <h3 class="re-title">
          {{ role.name }}
          <code class="re-code">{{ role.code }}</code>
          <span v-if="role.built_in" class="tag-builtin">🔒 内置</span>
        </h3>
        <p class="re-sub">勾选菜单授予对应平台权限 · 管理角色成员（平台角色无数据范围）</p>
      </div>
      <button
        v-if="role.code !== 'super_admin'"
        class="btn btn-primary btn-sm"
        :disabled="busy || locked || !dirty"
        @click="save"
      >
        {{ busy ? '保存中…' : '保存权限' }}
      </button>
    </div>

    <div v-if="locked && role.code !== 'super_admin'" class="builtin-note">🔒 内置角色不可修改，权限已锁定。</div>

    <div class="editor-body">
      <!-- Menu permission tree -->
      <section class="re-section" :class="{ locked }">
        <div class="re-section-head">
          <span>菜单权限</span>
          <div v-if="role.code !== 'super_admin'" class="tree-tools">
            <button type="button" class="link-btn" :disabled="locked" @click="checkAll(true)">全选</button>
            <button type="button" class="link-btn" :disabled="locked" @click="checkAll(false)">取消全选</button>
            <span class="tool-sep">|</span>
            <button type="button" class="link-btn" @click="expandAll = true">展开</button>
            <button type="button" class="link-btn" @click="expandAll = false">收起</button>
          </div>
        </div>
        <div v-if="role.code === 'super_admin'" class="super-note">
          ★ 超级管理员拥有全部平台权限（通配策略），无需逐项勾选。
        </div>
        <div v-else class="menu-tree-box">
          <TreeView
            :key="treeKey"
            :nodes="menus"
            :checked-ids="checkedKeys"
            checkbox
            cascade
            :default-expanded="expandAll"
            id-key="key"
            label-key="title"
            @check-set="onCheckSet"
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

      <!-- Members -->
      <section class="re-section">
        <div class="re-section-head">角色成员 · {{ role.members?.length || 0 }}</div>
        <div class="member-chips">
          <span v-for="uid in (role.members ?? [])" :key="uid" class="member-chip">
            {{ userName(uid) }}
            <button v-perm="'authz:grant'" class="rm" @click="revoke(uid)">×</button>
          </span>
          <span v-if="!role.members?.length" class="no-members">尚无成员</span>
        </div>
        <div class="add-member-row" v-perm="'authz:grant'">
          <select class="input input-sm" v-model="addUid">
            <option value="">添加成员…</option>
            <option v-for="u in assignableUsers" :key="u.id" :value="u.id">{{ u.username || u.id }}</option>
          </select>
          <button class="btn btn-sm" :disabled="!addUid || busy" @click="add">+ 加入</button>
        </div>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from "vue";
import { TreeView } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  platformBatchPolicy,
  grantPlatformRole, revokePlatformRole,
  type PlatformRole, type PolicyRule, type PolicyChange,
} from "../api/rbac";
import type { MenuTreeNode } from "../api/menus";
import type { PlatformUser } from "@/modules/admin/api/admin";

const props = defineProps<{
  role: PlatformRole;
  menus: MenuTreeNode[];
  users: PlatformUser[];
}>();

const emit = defineEmits<{ (e: "changed"): void }>();

const notify = useNotification();
const { confirm } = useConfirm();
const busy = ref(false);
const addUid = ref("");
const expandAll = ref(true);
const treeKey = ref(0);

// Built-in (non-super) platform roles are locked from policy edits.
const locked = computed(() => props.role.built_in);

function splitPerm(p: string): [string, string] {
  const i = p.lastIndexOf(":");
  if (i < 0) return [p, "*"];
  return [p.slice(0, i), p.slice(i + 1)];
}
function permSatisfied(perm: string, granted: Set<string>): boolean {
  if (!perm) return false;
  const [needRes, needAct] = splitPerm(perm);
  for (const g of granted) {
    const [gRes, gAct] = splitPerm(g);
    if ((gRes === "*" || gRes === needRes) && (gAct === "*" || gAct === needAct)) return true;
  }
  return false;
}

// Initial checked keys derived from the role's current policies (server truth).
// Satisfied perm-less nodes are pushed too, to keep cascade visuals consistent.
function computeInitialChecked(): string[] {
  const granted = new Set<string>();
  for (const p of (props.role.policies ?? []) as PolicyRule[]) {
    if (p.effect === "allow") granted.add(`${p.resource}:${p.action}`);
  }
  const keys: string[] = [];
  const visit = (n: MenuTreeNode): boolean => {
    const children = n.children ?? [];
    if (!children.length) {
      const ok = !n.perm || permSatisfied(n.perm, granted);
      if (ok) keys.push(n.key);
      return ok;
    }
    const childResults = children.map(visit);
    const ownOk = !n.perm || permSatisfied(n.perm, granted);
    const checked = ownOk && childResults.every(Boolean);
    if (checked) keys.push(n.key);
    return checked;
  };
  for (const n of props.menus) visit(n);
  return keys;
}

const checkedKeys = ref<string[]>([]);
const initialKeys = ref<Set<string>>(new Set());

function reset() {
  const init = computeInitialChecked();
  checkedKeys.value = [...init];
  initialKeys.value = new Set(init);
}
reset();
watch(() => props.role, reset);
watch(() => props.menus, () => { if (props.menus.length) reset(); });
watch(expandAll, () => { treeKey.value++; });

function onCheckSet(ids: string[]) {
  if (locked.value) return;
  checkedKeys.value = ids;
}

function allKeys(): string[] {
  const keys: string[] = [];
  walk(props.menus, (n) => keys.push(n.key));
  return keys;
}
function checkAll(value: boolean) {
  if (locked.value) return;
  checkedKeys.value = value ? allKeys() : [];
}

const dirty = computed(() => {
  const cur = new Set(checkedKeys.value);
  if (cur.size !== initialKeys.value.size) return true;
  for (const k of cur) if (!initialKeys.value.has(k)) return true;
  return false;
});

function permsForKeys(keys: Set<string>): Set<string> {
  const perms = new Set<string>();
  walk(props.menus, (n) => { if (keys.has(n.key) && n.perm) perms.add(n.perm); });
  return perms;
}

const userMap = computed(() => Object.fromEntries(props.users.map((u) => [u.id, u.username || u.id])));
function userName(uid: string) { return userMap.value[uid] ?? uid; }
const assignableUsers = computed(() => {
  const have = new Set(props.role.members ?? []);
  return props.users.filter((u) => !have.has(u.id));
});

async function save() {
  if (locked.value) return;
  busy.value = true;
  try {
    const beforePerms = permsForKeys(initialKeys.value);
    const afterPerms = permsForKeys(new Set(checkedKeys.value));
    const add: PolicyChange[] = [];
    const remove: PolicyChange[] = [];
    for (const p of afterPerms) {
      if (!beforePerms.has(p)) { const [r, a] = splitPerm(p); add.push({ resource: r, action: a }); }
    }
    for (const p of beforePerms) {
      if (!afterPerms.has(p)) { const [r, a] = splitPerm(p); remove.push({ resource: r, action: a }); }
    }
    if (add.length || remove.length) {
      await platformBatchPolicy(props.role.id, { add, remove });
    }
    notify.success("角色权限已保存");
    emit("changed");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  } finally { busy.value = false; }
}

async function add() {
  if (!addUid.value) return;
  busy.value = true;
  try {
    await grantPlatformRole(props.role.id, addUid.value);
    addUid.value = "";
    emit("changed");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "添加成员失败");
  } finally { busy.value = false; }
}
async function revoke(uid: string) {
  if (!(await confirm({ title: "移除成员", message: "确认移除该成员？", danger: true }))) return;
  busy.value = true;
  try {
    await revokePlatformRole(props.role.id, uid);
    emit("changed");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "移除失败");
  } finally { busy.value = false; }
}

function walk(nodes: MenuTreeNode[], fn: (n: MenuTreeNode) => void) {
  for (const n of nodes) {
    fn(n);
    if (n.children?.length) walk(n.children, fn);
  }
}
</script>

<style scoped>
.role-editor { display: flex; flex-direction: column; gap: 16px; }
.editor-head { display: flex; justify-content: space-between; align-items: flex-start; }
.re-title { font-size: 16px; font-weight: 600; display: flex; align-items: center; gap: 8px; margin: 0; }
.re-code { font-family: var(--ff-mono); font-size: 11.5px; color: var(--text-3); padding: 2px 7px; background: var(--surface-2); border-radius: 4px; }
.tag-builtin { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--purple-soft); color: var(--purple); border-radius: 3px; }
.re-sub { font-size: 12.5px; color: var(--text-3); margin-top: 4px; }

.builtin-note { font-size: 12.5px; color: var(--text-3); background: var(--surface-2); border: 1px solid var(--border); border-radius: 8px; padding: 10px 12px; }
.editor-body { display: grid; grid-template-columns: 1.4fr 1fr; gap: 18px; align-items: start; }
.re-section { display: flex; flex-direction: column; gap: 8px; }
.re-section.locked { opacity: .7; }
.re-section-head { font-size: 12px; font-weight: 600; color: var(--text-2); text-transform: uppercase; letter-spacing: .4px; display: flex; align-items: center; justify-content: space-between; }
.tree-tools { display: flex; align-items: center; gap: 6px; text-transform: none; letter-spacing: 0; }
.link-btn { border: 0; background: none; color: var(--primary); font-size: 11.5px; cursor: pointer; padding: 2px 4px; }
.link-btn:hover:not(:disabled) { text-decoration: underline; }
.link-btn:disabled { color: var(--text-4); cursor: not-allowed; }
.tool-sep { color: var(--border-strong); font-size: 11px; }
.super-note { font-size: 12.5px; color: var(--primary); background: var(--primary-soft); border-radius: 8px; padding: 12px 14px; line-height: 1.5; }
.menu-tree-box { border: 1px solid var(--border); border-radius: 8px; padding: 8px; max-height: 440px; overflow-y: auto; background: var(--surface); }
.empty-hint { color: var(--text-4); font-size: 12.5px; padding: 12px; }
.menu-node { display: inline-flex; align-items: center; gap: 6px; }
.menu-node.dir { font-weight: 600; }
.perm-tag { font-family: var(--ff-mono); font-size: 10.5px; color: var(--primary); background: var(--primary-soft); padding: 1px 5px; border-radius: 3px; }
.dir-tag { font-size: 10px; color: var(--text-4); }

.member-chips { display: flex; flex-wrap: wrap; gap: 6px; min-height: 24px; }
.member-chip { font-size: 11.5px; background: var(--primary-soft); color: var(--primary); border-radius: 5px; padding: 2px 6px; display: inline-flex; align-items: center; gap: 3px; }
.member-chip .rm { border: 0; background: none; color: var(--primary); cursor: pointer; font-size: 13px; line-height: 1; }
.no-members { font-size: 12.5px; color: var(--text-3); }
.add-member-row { display: flex; gap: 8px; margin-top: 4px; }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input-sm { font-size: 12px; padding: 5px 8px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover:not(:disabled) { background: var(--primary-hover); }
.btn-primary:disabled { opacity: .5; cursor: not-allowed; }
.btn-sm { padding: 5px 12px; font-size: 12.5px; }
</style>
