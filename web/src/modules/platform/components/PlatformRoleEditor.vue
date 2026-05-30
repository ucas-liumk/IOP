<template>
  <div class="role-editor">
    <div class="editor-head">
      <div>
        <h3 class="re-title">
          {{ role.name }}
          <code class="re-code">{{ role.code }}</code>
          <span v-if="role.built_in" class="tag-builtin">内置</span>
        </h3>
        <p class="re-sub">勾选菜单授予对应平台权限 · 管理角色成员（平台角色无数据范围）</p>
      </div>
    </div>

    <div class="editor-body">
      <!-- Menu permission tree -->
      <section class="re-section">
        <div class="re-section-head">菜单权限</div>
        <div v-if="role.code === 'super_admin'" class="super-note">
          ★ 超级管理员拥有全部平台权限（通配策略），无需逐项勾选。
        </div>
        <div v-else class="menu-tree-box">
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
import { computed, ref } from "vue";
import { TreeView } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  addPlatformPolicy, removePlatformPolicy,
  grantPlatformRole, revokePlatformRole,
  type PlatformRole, type PolicyRule,
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

// --- Menu checkbox <-> platform policy mapping ---
// A node is "checked" when the role's policies satisfy its perm
// (resource:action). Wildcards ("*:*", "res:*") count as satisfying.
const grantedSet = computed(() => {
  const set = new Set<string>();
  for (const p of (props.role.policies ?? []) as PolicyRule[]) {
    if (p.effect === "allow") set.add(`${p.resource}:${p.action}`);
  }
  return set;
});

function splitPerm(p: string): [string, string] {
  const i = p.lastIndexOf(":");
  if (i < 0) return [p, "*"];
  return [p.slice(0, i), p.slice(i + 1)];
}
function permSatisfied(perm: string): boolean {
  if (!perm) return false;
  const [needRes, needAct] = splitPerm(perm);
  for (const g of grantedSet.value) {
    const [gRes, gAct] = splitPerm(g);
    if ((gRes === "*" || gRes === needRes) && (gAct === "*" || gAct === needAct)) return true;
  }
  return false;
}

// Keys of nodes whose perm is currently satisfied (drives the checkboxes).
const checkedKeys = computed(() => {
  const keys: string[] = [];
  walk(props.menus, (n) => { if (n.perm && permSatisfied(n.perm)) keys.push(n.key); });
  return keys;
});

const userMap = computed(() => Object.fromEntries(props.users.map((u) => [u.id, u.username || u.id])));
function userName(uid: string) { return userMap.value[uid] ?? uid; }
const assignableUsers = computed(() => {
  const have = new Set(props.role.members ?? []);
  return props.users.filter((u) => !have.has(u.id));
});

// Collect node + descendants perms so checking a dir cascades to children.
function collectPerms(node: MenuTreeNode): string[] {
  const perms: string[] = [];
  const visit = (n: MenuTreeNode) => {
    if (n.perm) perms.push(n.perm);
    for (const c of n.children ?? []) visit(c);
  };
  visit(node);
  return Array.from(new Set(perms));
}
function findNode(key: string): MenuTreeNode | null {
  let found: MenuTreeNode | null = null;
  walk(props.menus, (n) => { if (n.key === key) found = n; });
  return found;
}

async function onCheck(key: string, checked: boolean) {
  const node = findNode(key);
  if (!node) return;
  const perms = collectPerms(node);
  if (perms.length === 0) { notify.warning("该目录节点没有可授予的权限"); return; }
  busy.value = true;
  try {
    for (const perm of perms) {
      const [res, act] = splitPerm(perm);
      if (checked) await addPlatformPolicy(props.role.id, res, act);
      else await removePlatformPolicy(props.role.id, res, act);
    }
    emit("changed");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "权限更新失败");
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

.editor-body { display: grid; grid-template-columns: 1.4fr 1fr; gap: 18px; align-items: start; }
.re-section { display: flex; flex-direction: column; gap: 8px; }
.re-section-head { font-size: 12px; font-weight: 600; color: var(--text-2); text-transform: uppercase; letter-spacing: .4px; }
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
.btn-sm { padding: 5px 12px; font-size: 12.5px; }
</style>
