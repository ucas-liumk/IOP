<template>
  <div class="mm">
    <div class="mm-head">
      <slot name="head-left">
        <div class="mm-title">用户管理 · 共 {{ total }} 名成员{{ activeDeptName ? ' · ' + activeDeptName : '' }}</div>
      </slot>
      <div class="mm-actions">
        <input class="input search" v-model="search" placeholder="搜索姓名 / 账户名 / 手机" @keyup.enter="search1" />
        <button class="btn btn-ghost btn-sm" @click="search1">搜索</button>
        <button class="btn btn-ghost btn-sm" @click="reload">刷新</button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="exportCsv">导出</button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="importOpen = true">导入</button>
      </div>
    </div>

    <div class="split">
      <!-- Left: department tree filter -->
      <article class="card tree-pane">
        <div class="pane-head">按部门筛选</div>
        <div class="tree-search">
          <input class="input search-sm" v-model="deptFilter" placeholder="搜索部门" />
        </div>
        <div class="filter-all" :class="{ active: !activeDeptId }" @click="filterDept(null)">全部成员</div>
        <label class="subtree-toggle">
          <input type="checkbox" v-model="subtree" @change="search1" /> 含子部门
        </label>
        <TreeView
          :nodes="deptTree"
          :selected-id="activeDeptId"
          :filter="deptFilter"
          id-key="id"
          label-key="name"
          @select="filterDept"
        />
      </article>

      <!-- Right: members table -->
      <div class="members-pane">
        <DataTable :columns="columns" :rows="members" rowKey="member_id" :loading="loading" emptyText="暂无成员">
          <template #cell-member="{ row }">
            <div class="member-cell" :class="{ off: row.status !== 'active' }">
              <div class="avatar" :style="{ background: avatarColor(row.display_name || row.username) }">{{ (row.display_name || row.username || '?')[0] }}</div>
              <div>
                <div class="name">{{ row.display_name || row.username }}</div>
                <div class="sub-id">
                  <code class="acct">{{ row.username || '—' }}</code>
                  <span v-if="row.phone" class="phone">· {{ row.phone }}</span>
                </div>
              </div>
            </div>
          </template>
          <template #cell-department="{ row }">
            <span class="chip-dept">{{ row.department || '未分配' }}</span>
          </template>
          <template #cell-posts="{ row }">
            <span v-if="!row.posts?.length" class="muted">—</span>
            <span v-for="p in row.posts" :key="p.post_id" class="chip-post">{{ p.name }}</span>
          </template>
          <template #cell-status="{ row }">
            <span class="badge" :class="row.status === 'active' ? 'badge-success' : 'badge-neutral'">
              {{ row.status === 'active' ? '正常' : '已禁用' }}
            </span>
          </template>
          <template #cell-actions="{ row }">
            <div class="row-actions">
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openRoles(row)">角色</button>
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openPosts(row)">岗位</button>
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openDept(row)">部门</button>
              <button v-if="api.resetPassword" class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openReset(row)">重置密码</button>
              <button v-if="api.setDisabled" class="btn btn-ghost btn-sm" :class="{ danger: row.status === 'active' }" v-perm="writePerm" @click="toggleStatus(row)">
                {{ row.status === 'active' ? '禁用' : '启用' }}
              </button>
            </div>
          </template>
        </DataTable>
        <Pagination
          v-model:page="page"
          v-model:page-size="pageSize"
          :total="total"
          @update:page="reload"
          @update:page-size="reload"
        />
      </div>
    </div>

    <ImportDialog
      v-model:open="importOpen"
      title="导入成员"
      :template-url="templateUrl"
      :import-url="importUrl"
      template-name="members_template.csv"
      @done="onImportDone"
    />

    <!-- Roles modal -->
    <div v-if="rolesFor" class="modal-overlay" @click.self="rolesFor = null">
      <div class="modal">
        <h3>分配角色 · {{ rolesFor.display_name || rolesFor.username }}</h3>
        <p class="modal-sub">勾选授予，取消勾选撤销。</p>
        <ul class="check-list">
          <li v-for="r in allRoles" :key="r.id">
            <label>
              <input type="checkbox" :checked="memberRoleIds.has(r.id)" :disabled="busy" @change="toggleRole(r, ($event.target as HTMLInputElement).checked)" />
              <span class="r-name">{{ r.name }}</span>
              <code class="r-code">{{ r.code }}</code>
              <span v-if="r.built_in" class="tag-builtin">内置</span>
            </label>
          </li>
          <li v-if="allRoles.length === 0" class="muted">尚无角色，请先在「角色管理」创建。</li>
        </ul>
        <div class="modal-actions">
          <button class="btn btn-primary" @click="closeRoles">完成</button>
        </div>
      </div>
    </div>

    <!-- Posts modal -->
    <div v-if="postsFor" class="modal-overlay" @click.self="postsFor = null">
      <div class="modal">
        <h3>分配岗位 · {{ postsFor.display_name || postsFor.username }}</h3>
        <p class="modal-sub">勾选授予，取消勾选移除。</p>
        <ul class="check-list">
          <li v-for="p in allPosts" :key="p.id">
            <label>
              <input type="checkbox" :checked="memberPostIds.has(p.id)" :disabled="busy" @change="togglePost(p, ($event.target as HTMLInputElement).checked)" />
              <span class="r-name">{{ p.name }}</span>
              <code class="r-code">{{ p.code }}</code>
            </label>
          </li>
          <li v-if="allPosts.length === 0" class="muted">尚无岗位，请先在「岗位管理」创建。</li>
        </ul>
        <div class="modal-actions">
          <button class="btn btn-primary" @click="closePosts">完成</button>
        </div>
      </div>
    </div>

    <!-- Dept modal -->
    <div v-if="deptFor" class="modal-overlay" @click.self="deptFor = null">
      <div class="modal">
        <h3>设置部门 · {{ deptFor.display_name || deptFor.username }}</h3>
        <label class="field">
          <span class="label">所属部门</span>
          <select class="input" v-model="deptChoice">
            <option value="">（未分配）</option>
            <option v-for="d in deptFlat" :key="d.id" :value="d.id">{{ indentName(d) }}</option>
          </select>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="deptFor = null">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="saveDept">保存</button>
        </div>
      </div>
    </div>

    <!-- Reset password modal -->
    <div v-if="resetFor" class="modal-overlay" @click.self="closeReset">
      <div class="modal">
        <h3>重置密码 · {{ resetFor.display_name || resetFor.username }}</h3>
        <label class="field">
          <span class="label">新密码</span>
          <input class="input" v-model="newPassword" type="text" minlength="10" placeholder="至少 10 位，含字母与数字" />
          <button type="button" class="btn-link" @click="newPassword = randomPassword()">生成强密码</button>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeReset">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmReset">{{ busy ? '处理中…' : '确认重置' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from "vue";
// Import siblings directly (not via the ./index barrel) — the barrel re-exports
// this very component, so going through it would create a chunk-level circular
// dependency at build time (mirrors DeptTreeManager).
import DataTable from "./DataTable.vue";
import Pagination from "./Pagination.vue";
import TreeView from "./TreeView.vue";
import ImportDialog, { type BulkResult } from "./ImportDialog.vue";
import type { Column } from "./types";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

// Shared member / role / post / dept shapes. Kept local so the component does not
// couple to any one module's api typing (tenant `/admin/*` + platform
// `/platform/orgs/:tid/*` both reuse this).
export interface MemberPostRow { post_id: string; code: string; name: string }
export interface MemberRow {
  member_id: string;
  platform_user_id: string;
  // username is the primary login identity; email is optional and often NULL,
  // so it is NOT shown as the identifier.
  username: string;
  display_name: string;
  email?: string;
  department: string;
  dept_id?: string | null;
  title?: string;
  phone?: string;
  status: string;
  joined_at?: string;
  posts?: MemberPostRow[];
}
export interface RoleRow { id: string; code: string; name: string; built_in?: boolean }
export interface PostRow { id: string; code: string; name: string }
export interface MemberDeptRow { id: string; name: string; parent_id?: string | null }
export interface MemberDeptTreeRow extends MemberDeptRow { children?: MemberDeptTreeRow[] }

export interface MemberListParams {
  page: number;
  pageSize: number;
  search?: string;
  deptId?: string | null;
  subtree?: boolean;
}
export interface MemberPage {
  data: MemberRow[];
  total: number;
  page: number;
  pageSize: number;
}

// The api-adapter object decouples this component from how the endpoints are
// wired (tenant `/admin/members*` vs platform `/platform/orgs/:tid/members*`).
// The owning view supplies the concrete funcs; CSV import/export use plain URLs
// (export via the adapter, import + template via ImportDialog's URL props).
// `setDisabled` and `resetPassword` are optional — when absent the matching row
// action button is hidden (e.g. the platform org console may resolve reset via
// the separate platform-user endpoint, supplied by the view).
export interface MemberApi {
  listMembers(p: MemberListParams): Promise<MemberPage>;
  fetchDeptTree(): Promise<MemberDeptTreeRow[]>;
  fetchDeptFlat(): Promise<MemberDeptRow[]>;
  setDept(memberId: string, deptId: string | null): Promise<void>;
  assignPost(memberId: string, postId: string): Promise<void>;
  removePost(memberId: string, postId: string): Promise<void>;
  listRoles(): Promise<RoleRow[]>;
  memberRoles(member: MemberRow): Promise<RoleRow[]>;
  grantRole(member: MemberRow, code: string): Promise<void>;
  revokeRole(member: MemberRow, roleId: string): Promise<void>;
  listPosts(): Promise<PostRow[]>;
  exportCsv(): Promise<void>;
  setDisabled?(member: MemberRow, disabled: boolean): Promise<void>;
  resetPassword?(member: MemberRow, newPassword: string): Promise<void>;
}

const props = withDefaults(
  defineProps<{
    /** concrete member endpoints (tenant or a specific org). */
    api: MemberApi;
    /** API path the ImportDialog fetches the template CSV from. */
    templateUrl: string;
    /** API path the ImportDialog POSTs the multipart upload to (field "file"). */
    importUrl: string;
    /** button-level RBAC key for write actions (e.g. "member:write" / "user:write"). */
    writePerm?: string;
  }>(),
  { writePerm: "member:write" },
);

const notify = useNotification();
const { confirm } = useConfirm();

const members = ref<MemberRow[]>([]);
const search = ref("");
const subtree = ref(false);
const busy = ref(false);
const loading = ref(false);
const actionError = ref("");
const importOpen = ref(false);

// Server-side pagination state.
const page = ref(1);
const pageSize = ref(20);
const total = ref(0);

const deptTree = ref<MemberDeptTreeRow[]>([]);
const deptFlat = ref<MemberDeptRow[]>([]);
const deptFilter = ref("");
const activeDeptId = ref<string | null>(null);

const activeDeptName = computed(() => deptFlat.value.find((d) => d.id === activeDeptId.value)?.name ?? "");

const columns: Column[] = [
  { key: "member",     label: "成员",   width: 260 },
  { key: "department", label: "部门",   width: 130 },
  { key: "posts",      label: "岗位" },
  { key: "status",     label: "状态",   width: 90 },
  { key: "actions",    label: "操作",   width: 360, align: "right" },
];

onMounted(loadAll);
// Re-load everything whenever the bound api adapter changes (e.g. platform
// switches org). Reset transient state so the previous org's data does not leak.
watch(() => props.api, () => {
  search.value = "";
  subtree.value = false;
  activeDeptId.value = null;
  deptFilter.value = "";
  page.value = 1;
  members.value = [];
  deptTree.value = [];
  deptFlat.value = [];
  total.value = 0;
  allRoles.value = [];
  allPosts.value = [];
  loadAll();
});

async function loadAll() {
  [deptTree.value, deptFlat.value] = await Promise.all([props.api.fetchDeptTree(), props.api.fetchDeptFlat()]);
  await reload();
}

async function reload() {
  loading.value = true;
  try {
    const res = await props.api.listMembers({
      page: page.value,
      pageSize: pageSize.value,
      search: search.value.trim(),
      deptId: activeDeptId.value,
      subtree: subtree.value,
    });
    members.value = res.data;
    total.value = res.total;
  } finally {
    loading.value = false;
  }
}

// Reset to page 1 whenever a filter changes (search / dept / subtree).
function search1() {
  page.value = 1;
  reload();
}
function filterDept(id: string | null) {
  activeDeptId.value = id;
  page.value = 1;
  reload();
}

async function onImportDone(_r: BulkResult) {
  page.value = 1;
  await reload();
}

async function toggleStatus(m: MemberRow) {
  if (!props.api.setDisabled) return;
  if (m.status === "active") {
    const ok = await confirm({ title: "禁用成员", message: `确认禁用 ${m.display_name || m.username}？`, danger: true });
    if (!ok) return;
  }
  try {
    await props.api.setDisabled(m, m.status === "active");
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "操作失败"); }
}

// === Roles ===
const rolesFor = ref<MemberRow | null>(null);
const allRoles = ref<RoleRow[]>([]);
const memberRoleIds = ref<Set<string>>(new Set());

async function openRoles(m: MemberRow) {
  rolesFor.value = m;
  actionError.value = "";
  if (allRoles.value.length === 0) allRoles.value = await props.api.listRoles();
  const mr = await props.api.memberRoles(m);
  memberRoleIds.value = new Set(mr.map((r) => r.id));
}
async function toggleRole(r: RoleRow, checked: boolean) {
  if (!rolesFor.value) return;
  busy.value = true;
  try {
    if (checked) {
      await props.api.grantRole(rolesFor.value, r.code);
      memberRoleIds.value.add(r.id);
    } else {
      await props.api.revokeRole(rolesFor.value, r.id);
      memberRoleIds.value.delete(r.id);
    }
    memberRoleIds.value = new Set(memberRoleIds.value);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  } finally { busy.value = false; }
}
async function closeRoles() { rolesFor.value = null; await reload(); }

// === Posts ===
const postsFor = ref<MemberRow | null>(null);
const allPosts = ref<PostRow[]>([]);
const memberPostIds = ref<Set<string>>(new Set());

async function openPosts(m: MemberRow) {
  postsFor.value = m;
  actionError.value = "";
  if (allPosts.value.length === 0) allPosts.value = await props.api.listPosts();
  memberPostIds.value = new Set((m.posts ?? []).map((p) => p.post_id));
}
async function togglePost(p: PostRow, checked: boolean) {
  if (!postsFor.value) return;
  busy.value = true;
  try {
    if (checked) {
      await props.api.assignPost(postsFor.value.member_id, p.id);
      memberPostIds.value.add(p.id);
    } else {
      await props.api.removePost(postsFor.value.member_id, p.id);
      memberPostIds.value.delete(p.id);
    }
    memberPostIds.value = new Set(memberPostIds.value);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  } finally { busy.value = false; }
}
async function closePosts() { postsFor.value = null; await reload(); }

// === Dept ===
const deptFor = ref<MemberRow | null>(null);
const deptChoice = ref<string>("");
function openDept(m: MemberRow) {
  deptFor.value = m;
  deptChoice.value = m.dept_id ?? "";
  actionError.value = "";
}
async function saveDept() {
  if (!deptFor.value) return;
  busy.value = true; actionError.value = "";
  try {
    await props.api.setDept(deptFor.value.member_id, deptChoice.value || null);
    deptFor.value = null;
    await reload();
    notify.success("部门已更新");
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
}

// === Reset password ===
const resetFor = ref<MemberRow | null>(null);
const newPassword = ref("");
function openReset(m: MemberRow) {
  resetFor.value = m;
  newPassword.value = randomPassword();
  actionError.value = "";
}
function closeReset() { resetFor.value = null; newPassword.value = ""; actionError.value = ""; }
async function confirmReset() {
  if (!resetFor.value || !props.api.resetPassword) return;
  busy.value = true; actionError.value = "";
  try {
    await props.api.resetPassword(resetFor.value, newPassword.value);
    notify.success(`密码已重置为：${newPassword.value}（请妥善记录并通知用户）`, 8000);
    closeReset();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "重置失败（需平台管理员权限）";
  } finally { busy.value = false; }
}

async function exportCsv() {
  try {
    await props.api.exportCsv();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "导出失败");
  }
}

// === Helpers ===
// indented display name in the dept <select> for a sense of hierarchy.
function indentName(d: MemberDeptRow): string {
  let depth = 0;
  let cur: MemberDeptRow | undefined = d;
  while (cur?.parent_id) {
    cur = deptFlat.value.find((x) => x.id === cur!.parent_id);
    depth++;
    if (depth > 20) break;
  }
  return "　".repeat(depth) + d.name;
}
function avatarColor(name: string) {
  const seed = (name || "?").split("").reduce((s, c) => s + c.charCodeAt(0), 0);
  const palette = [
    "linear-gradient(135deg,#1e5fd9,#4a85ee)",
    "linear-gradient(135deg,#7c4ddb,#5a2db5)",
    "linear-gradient(135deg,#0fa8a3,#0a7e7a)",
    "linear-gradient(135deg,#e8920e,#b86d05)",
    "linear-gradient(135deg,#1aa971,#0e7b51)",
  ];
  return palette[seed % palette.length];
}
function randPick(set: string): string {
  const max = 256 - (256 % set.length);
  const buf = new Uint8Array(1);
  for (;;) { crypto.getRandomValues(buf); if (buf[0] < max) return set[buf[0] % set.length]; }
}
function randomPassword(): string {
  const letters = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ";
  const digits = "23456789";
  const symbols = "!@#$%";
  const all = letters + digits + symbols;
  const chars: string[] = [randPick(letters), randPick(digits), randPick(symbols)];
  for (let i = 0; i < 9; i++) chars.push(randPick(all));
  for (let i = chars.length - 1; i > 0; i--) {
    const j = cryptoIndex(i + 1);
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  return chars.join("");
}
function cryptoIndex(n: number): number {
  const max = 256 - (256 % n);
  const buf = new Uint8Array(1);
  for (;;) { crypto.getRandomValues(buf); if (buf[0] < max) return buf[0] % n; }
}

// Let the owning view force a refresh (e.g. after external changes).
defineExpose({ reload });
</script>

<style scoped>
.mm { display: flex; flex-direction: column; gap: 14px; }
.mm-head { display: flex; align-items: center; justify-content: space-between; gap: 12px; flex-wrap: wrap; }
.mm-title { font-size: 13px; color: var(--text-2); }
.mm-actions { display: flex; align-items: center; gap: 8px; margin-left: auto; flex-wrap: wrap; }
.search { width: 220px; font-size: 13px; padding: 6px 10px; }

.split { display: grid; grid-template-columns: 260px 1fr; gap: 16px; align-items: start; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.tree-pane { padding: 12px; }
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 8px; }
.tree-search { padding: 0 4px 8px; }
.tree-search .search-sm { width: 100%; font-size: 13px; padding: 6px 10px; box-sizing: border-box; border: 1px solid var(--border-strong); border-radius: 6px; background: var(--surface); }
.tree-search .search-sm:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.filter-all { padding: 6px 8px; font-size: 13px; border-radius: 6px; cursor: pointer; color: var(--text-2); }
.filter-all:hover { background: var(--surface-2); }
.filter-all.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.subtree-toggle { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-3); padding: 6px 8px; }
.members-pane { min-width: 0; }

.member-cell { display: flex; gap: 10px; align-items: center; }
.member-cell.off { opacity: .55; }
.avatar { width: 32px; height: 32px; border-radius: 50%; color: white; font-weight: 600; font-size: 12px; display: grid; place-items: center; }
.name { font-weight: 600; font-size: 13.5px; }
.sub-id { font-size: 11.5px; color: var(--text-3); margin-top: 2px; display: flex; align-items: center; gap: 4px; }
.acct { font-family: var(--ff-mono); font-size: 11px; color: var(--text-2); background: var(--surface-2); padding: 1px 5px; border-radius: 3px; }
.phone { color: var(--text-3); }
.chip-dept { display: inline-block; padding: 2px 8px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 999px; font-size: 11.5px; }
.chip-post { display: inline-block; padding: 2px 8px; margin: 0 4px 2px 0; background: var(--primary-soft); color: var(--primary); border-radius: 999px; font-size: 11.5px; font-weight: 600; }
.muted { color: var(--text-4); }
.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.badge-success { background: var(--success-soft); color: var(--success); }
.badge-neutral { background: var(--bg-deep); color: var(--text-3); }

.row-actions { display: flex; gap: 4px; justify-content: flex-end; flex-wrap: wrap; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-ghost { background: var(--surface); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }
.btn-link { border: 0; background: none; color: var(--primary); font-size: 11.5px; cursor: pointer; text-align: left; padding: 4px 0 0; }

.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(440px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-sub { font-size: 12.5px; color: var(--text-3); margin: -6px 0 4px; }
.check-list { list-style: none; margin: 0; padding: 0; max-height: 320px; overflow-y: auto; display: flex; flex-direction: column; gap: 2px; }
.check-list li { padding: 2px 0; }
.check-list label { display: flex; align-items: center; gap: 8px; padding: 7px 8px; border-radius: 6px; cursor: pointer; font-size: 13px; }
.check-list label:hover { background: var(--surface-2); }
.check-list input { accent-color: var(--primary); }
.r-name { font-weight: 500; }
.r-code { font-family: var(--ff-mono); font-size: 11.5px; color: var(--text-3); padding: 1px 6px; background: var(--surface-2); border-radius: 4px; }
.tag-builtin { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--purple-soft); color: var(--purple); border-radius: 3px; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
</style>
