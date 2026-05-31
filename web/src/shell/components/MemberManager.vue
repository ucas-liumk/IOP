<template>
  <div class="mm">
    <div class="mm-head">
      <slot name="head-left">
        <div class="mm-title">用户管理 · 共 {{ total }} 名成员{{ activeDeptName ? ' · ' + activeDeptName : '' }}</div>
      </slot>
      <div class="mm-actions">
        <div class="search-wrap">
          <input class="input search" v-model="search" placeholder="搜索姓名 / 账户名 / 手机 / 邮箱" @keyup.enter="search1" @input="onSearchInput" />
          <button v-if="search" class="search-clear" type="button" aria-label="清除搜索" @click="clearSearch">
            <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
          </button>
        </div>
        <select class="input status-select" v-model="statusFilter" @change="search1">
          <option value="">全部状态</option>
          <option value="active">正常</option>
          <option value="disabled">已禁用</option>
        </select>
        <button class="btn btn-ghost btn-sm" @click="search1">搜索</button>
        <button v-if="api.createMember" class="btn btn-primary btn-sm" v-perm="writePerm" @click="openCreate">新增</button>
        <button class="btn btn-ghost btn-sm" :disabled="loading" @click="reload">
          <span v-if="loading" class="btn-spinner" aria-hidden="true" />刷新
        </button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" :disabled="exporting" @click="exportCsv">
          <span v-if="exporting" class="btn-spinner" aria-hidden="true" />{{ exporting ? '导出中…' : '导出' }}
        </button>
        <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="importOpen = true">导入</button>
      </div>
    </div>

    <div class="split">
      <!-- Left: department tree filter -->
      <article class="card tree-pane">
        <div class="pane-head-row">
          <div class="pane-head">按部门筛选</div>
          <div class="tree-tools">
            <button class="tree-tool-btn" type="button" title="展开全部" @click="expandDeptAll">展开</button>
            <button class="tree-tool-btn" type="button" title="收起全部" @click="collapseDeptAll">收起</button>
          </div>
        </div>
        <div class="tree-search">
          <div class="search-wrap">
            <input class="input search-sm" v-model="deptFilter" placeholder="搜索部门" />
            <button v-if="deptFilter" class="search-clear" type="button" aria-label="清除搜索" @click="deptFilter = ''">
              <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round"><line x1="18" y1="6" x2="6" y2="18" /><line x1="6" y1="6" x2="18" y2="18" /></svg>
            </button>
          </div>
        </div>
        <div class="filter-all" :class="{ active: !activeDeptId }" @click="filterDept(null)">全部成员</div>
        <label class="subtree-toggle">
          <input type="checkbox" v-model="subtree" @change="search1" /> 含子部门
        </label>

        <div v-if="deptLoading && deptTree.length === 0" class="tree-skeleton" role="status" aria-label="加载中">
          <SkeletonLoader :lines="6" :height="20" :last-short="false" />
        </div>
        <TreeView
          v-else
          :nodes="deptTree"
          :selected-id="activeDeptId"
          :filter="deptFilter"
          :expand-signal="deptExpandSignal"
          id-key="id"
          label-key="name"
          @select="filterDept"
        >
          <template #empty>
            <span v-if="deptFilter">没有匹配结果 · 试试调整搜索</span>
            <span v-else>暂无部门</span>
          </template>
        </TreeView>
      </article>

      <!-- Right: members table -->
      <div class="members-pane">
        <!-- Bulk action bar -->
        <Transition name="bulk-bar">
          <div v-if="bulkEnabled && selectedCount > 0" class="bulk-bar">
            <span class="bulk-count">已选 {{ selectedCount }} 项</span>
            <span class="bulk-spacer" />
            <span v-if="bulkBusy" class="bulk-progress">处理中 {{ bulkProgress }} / {{ selectedCount }}…</span>
            <template v-else>
              <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="bulkSetDisabled(false)">批量启用</button>
              <button class="btn btn-ghost btn-sm danger" v-perm="writePerm" @click="bulkSetDisabled(true)">批量停用</button>
              <button class="btn btn-ghost btn-sm" @click="clearSelection">取消选择</button>
            </template>
          </div>
        </Transition>

        <DataTable :columns="columns" :rows="members" rowKey="member_id" :loading="loading">
          <template v-if="bulkEnabled" #head-select>
            <input
              type="checkbox"
              class="row-check"
              :checked="allRowsSelected"
              :indeterminate.prop="someRowsSelected"
              :disabled="members.length === 0"
              aria-label="全选"
              @change="toggleSelectAll(($event.target as HTMLInputElement).checked)"
            />
          </template>
          <template v-if="bulkEnabled" #cell-select="{ row }">
            <input
              type="checkbox"
              class="row-check"
              :checked="selectedIds.has(row.member_id)"
              @change="toggleRowSelect(row.member_id, ($event.target as HTMLInputElement).checked)"
            />
          </template>
          <template #empty>
            <EmptyState
              v-if="hasMemberFilter"
              title="没有匹配结果"
              sub="试试调整搜索或筛选条件"
              icon="◌"
            />
            <EmptyState
              v-else
              title="暂无成员"
              sub="新建或导入成员以开始"
              icon="◫"
            >
              <template #actions>
                <button v-if="api.createMember" class="btn btn-primary btn-sm" v-perm="writePerm" @click="openCreate">新增</button>
                <button class="btn btn-ghost btn-sm" v-perm="writePerm" @click="importOpen = true">导入</button>
              </template>
            </EmptyState>
          </template>
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
            <span class="chip-dept">{{ row.dept_path || row.department || '未分配' }}</span>
          </template>
          <template #cell-contact="{ row }">
            <div class="contact-cell">
              <span>{{ row.phone || '—' }}</span>
              <span v-if="row.email" class="muted">{{ row.email }}</span>
            </div>
          </template>
          <template #cell-posts="{ row }">
            <span v-if="!row.posts?.length" class="muted">—</span>
            <span v-for="p in row.posts" :key="p.post_id" class="chip-post">{{ p.name }}</span>
          </template>
          <template #cell-roles="{ row }">
            <span v-if="!row.roles?.length" class="muted">—</span>
            <span v-for="r in row.roles" :key="r.role_id" class="chip-role">{{ r.name }}</span>
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
              <button v-if="api.updateMember" class="btn btn-ghost btn-sm" v-perm="writePerm" @click="openEdit(row)">编辑</button>
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
      template-name="members_template.xlsx"
      @done="onImportDone"
    />

    <!-- Create / edit modal -->
    <div v-if="editorOpen" class="modal-overlay" @click.self="closeEditor">
      <div class="modal member-editor">
        <h3>{{ editorMode === 'create' ? '新增用户' : '编辑用户' }}</h3>
        <div class="form-grid">
          <label v-if="editorMode === 'create'" class="field">
            <span class="label">用户名 *</span>
            <input
              class="input"
              :class="{ invalid: memberFieldErr.username }"
              v-model="memberForm.username"
              placeholder="zhangsan"
              :aria-invalid="!!memberFieldErr.username"
              @blur="validateMemberField('username')"
              @input="memberFieldErr.username && validateMemberField('username')"
            />
            <span v-if="memberFieldErr.username" class="field-error" role="alert">{{ memberFieldErr.username }}</span>
          </label>
          <label class="field">
            <span class="label">姓名 *</span>
            <input
              class="input"
              :class="{ invalid: memberFieldErr.display_name }"
              v-model="memberForm.display_name"
              placeholder="张三"
              :aria-invalid="!!memberFieldErr.display_name"
              @blur="validateMemberField('display_name')"
              @input="memberFieldErr.display_name && validateMemberField('display_name')"
            />
            <span v-if="memberFieldErr.display_name" class="field-error" role="alert">{{ memberFieldErr.display_name }}</span>
          </label>
          <label class="field">
            <span class="label">手机号</span>
            <input class="input" v-model="memberForm.phone" inputmode="numeric" placeholder="13800000000" />
          </label>
          <label class="field">
            <span class="label">邮箱</span>
            <input class="input" v-model="memberForm.email" type="email" placeholder="name@example.com" />
          </label>
          <label class="field">
            <span class="label">性别</span>
            <select class="input" v-model="memberForm.gender">
              <option value="">未设置</option>
              <option value="male">男</option>
              <option value="female">女</option>
              <option value="other">其他</option>
            </select>
          </label>
          <label class="field">
            <span class="label">所属组织 *</span>
            <select
              class="input"
              :class="{ invalid: memberFieldErr.dept_id }"
              v-model="memberForm.dept_id"
              :aria-invalid="!!memberFieldErr.dept_id"
              @change="validateMemberField('dept_id')"
              @blur="validateMemberField('dept_id')"
            >
              <option value="" disabled>请选择组织</option>
              <option v-for="d in deptFlat" :key="d.id" :value="d.id">{{ indentName(d) }}</option>
            </select>
            <span v-if="memberFieldErr.dept_id" class="field-error" role="alert">{{ memberFieldErr.dept_id }}</span>
          </label>
          <label class="field">
            <span class="label">岗位名称</span>
            <input class="input" v-model="memberForm.title" placeholder="岗位 / 职务" />
          </label>
          <label class="field">
            <span class="label">状态</span>
            <select class="input" v-model="memberForm.status">
              <option value="active">正常</option>
              <option value="disabled">禁用</option>
            </select>
          </label>
          <label v-if="editorMode === 'create'" class="field">
            <span class="label">角色</span>
            <select class="input" v-model="memberForm.role_code">
              <option v-for="r in allRoles" :key="r.id" :value="r.code">{{ r.name }}</option>
            </select>
          </label>
          <label v-if="editorMode === 'create'" class="field">
            <span class="label">岗位编码</span>
            <select class="input" v-model="memberForm.post_id">
              <option value="">不分配</option>
              <option v-for="p in allPosts" :key="p.id" :value="p.id">{{ p.name }}</option>
            </select>
          </label>
          <label v-if="editorMode === 'create'" class="field">
            <span class="label">初始密码</span>
            <input class="input" v-model="memberForm.password" type="text" />
            <button type="button" class="btn-link" @click="memberForm.password = randomPassword()">生成强密码</button>
          </label>
          <label class="field field-wide">
            <span class="label">备注</span>
            <textarea class="input textarea" v-model="memberForm.remark" rows="3" />
          </label>
        </div>
        <div v-if="actionError" class="form-error" role="alert">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeEditor">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="saveMember">
            <span v-if="busy" class="btn-spinner light" aria-hidden="true" />{{ busy ? '保存中…' : '保存' }}
          </button>
        </div>
      </div>
    </div>

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
        <div v-if="actionError" class="form-error" role="alert">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="deptFor = null">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="saveDept">
            <span v-if="busy" class="btn-spinner light" aria-hidden="true" />{{ busy ? '保存中…' : '保存' }}
          </button>
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
        <div v-if="actionError" class="form-error" role="alert">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeReset">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmReset">
            <span v-if="busy" class="btn-spinner light" aria-hidden="true" />{{ busy ? '处理中…' : '确认重置' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
// Import siblings directly (not via the ./index barrel) — the barrel re-exports
// this very component, so going through it would create a chunk-level circular
// dependency at build time (mirrors DeptTreeManager).
import DataTable from "./DataTable.vue";
import Pagination from "./Pagination.vue";
import TreeView from "./TreeView.vue";
import SkeletonLoader from "./SkeletonLoader.vue";
import EmptyState from "./EmptyState.vue";
import ImportDialog, { type BulkResult } from "./ImportDialog.vue";
import type { Column } from "./types";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

// Shared member / role / post / dept shapes. Kept local so the component does not
// couple to any one module's api typing (tenant `/admin/*` + platform
// `/platform/orgs/:tid/*` both reuse this).
export interface MemberPostRow { post_id: string; code: string; name: string }
export interface MemberRoleRow { role_id: string; code: string; name: string }
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
  dept_code?: string;
  dept_path?: string;
  gender?: string;
  title?: string;
  phone?: string;
  status: string;
  remark?: string;
  joined_at?: string;
  posts?: MemberPostRow[];
  roles?: MemberRoleRow[];
}
export interface RoleRow { id: string; code: string; name: string; built_in?: boolean }
export interface PostRow { id: string; code: string; name: string }
export interface MemberDeptRow { id: string; name: string; parent_id?: string | null }
export interface MemberDeptTreeRow extends MemberDeptRow { children?: MemberDeptTreeRow[] }

export interface MemberListParams {
  page: number;
  pageSize: number;
  search?: string;
  status?: string;
  deptId?: string | null;
  subtree?: boolean;
  ids?: string[];
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
  createMember?(payload: {
    username: string; display_name: string; phone?: string; email?: string; gender?: string;
    dept_id: string; title?: string; role_codes?: string[]; post_ids?: string[];
    status?: string; password?: string; remark?: string;
  }): Promise<MemberRow>;
  updateMember?(memberId: string, patch: Partial<MemberRow> & { dept_id?: string | null }): Promise<void>;
  exportCsv(p: MemberListParams): Promise<void>;
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
const statusFilter = ref("");
const subtree = ref(false);
const busy = ref(false);
const loading = ref(false);
const deptLoading = ref(false);
const exporting = ref(false);
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
// Expand/collapse-all signal for the dept filter tree.
const deptExpandSignal = ref(0);

const activeDeptName = computed(() => deptFlat.value.find((d) => d.id === activeDeptId.value)?.name ?? "");
// Distinguish "no members at all" from "search/filter returned nothing".
const hasMemberFilter = computed(() => !!search.value.trim() || !!statusFilter.value || !!activeDeptId.value);

// Debounced free-text search (~250ms) — still supports Enter / the search button.
let searchTimer: ReturnType<typeof setTimeout> | null = null;
function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(() => search1(), 250);
}
function clearSearch() {
  search.value = "";
  search1();
}
function expandDeptAll() { deptExpandSignal.value = Math.abs(deptExpandSignal.value) + 1; }
function collapseDeptAll() { deptExpandSignal.value = -(Math.abs(deptExpandSignal.value) + 1); }

// Bulk multi-select is only offered when the bulk-capable endpoint exists.
const bulkEnabled = computed(() => !!props.api.setDisabled);

const columns = computed<Column[]>(() => {
  const base: Column[] = [
    { key: "member",     label: "成员",   width: 260 },
    { key: "department", label: "组织",   width: 190 },
    { key: "contact",    label: "联系方式", width: 170 },
    { key: "posts",      label: "岗位" },
    { key: "roles",      label: "角色" },
    { key: "status",     label: "状态",   width: 90 },
    { key: "actions",    label: "操作",   width: 420, align: "right" },
  ];
  return bulkEnabled.value ? [{ key: "select", label: "", width: 40, align: "center" }, ...base] : base;
});

// === Bulk selection ===
const selectedIds = ref<Set<string>>(new Set());
const selectedCount = computed(() => selectedIds.value.size);
const allRowsSelected = computed(() => members.value.length > 0 && members.value.every((m) => selectedIds.value.has(m.member_id)));
const someRowsSelected = computed(() => selectedCount.value > 0 && !allRowsSelected.value);
const bulkBusy = ref(false);
const bulkProgress = ref(0);

function toggleRowSelect(id: string, on: boolean) {
  const next = new Set(selectedIds.value);
  if (on) next.add(id); else next.delete(id);
  selectedIds.value = next;
}
function toggleSelectAll(on: boolean) {
  selectedIds.value = on ? new Set(members.value.map((m) => m.member_id)) : new Set();
}
function clearSelection() { selectedIds.value = new Set(); }

// Bulk enable/disable loops the existing per-row setDisabled API client-side.
async function bulkSetDisabled(disabled: boolean) {
  if (!props.api.setDisabled || selectedIds.value.size === 0) return;
  const targets = members.value.filter((m) => selectedIds.value.has(m.member_id) && (m.status === "active") === disabled);
  if (targets.length === 0) {
    notify.warning(disabled ? "所选成员均已停用" : "所选成员均已启用");
    return;
  }
  const ok = await confirm({
    title: disabled ? "批量停用" : "批量启用",
    message: `确认${disabled ? "停用" : "启用"} ${targets.length} 名成员？`,
    danger: disabled,
  });
  if (!ok) return;
  bulkBusy.value = true;
  bulkProgress.value = 0;
  let okCount = 0;
  let failCount = 0;
  for (const m of targets) {
    try {
      await props.api.setDisabled(m, disabled);
      okCount++;
    } catch {
      failCount++;
    }
    bulkProgress.value++;
  }
  bulkBusy.value = false;
  if (failCount === 0) notify.success(`已${disabled ? "停用" : "启用"} ${okCount} 名成员`);
  else notify.warning(`完成：成功 ${okCount}，失败 ${failCount}`);
  clearSelection();
  await reload();
}

onMounted(loadAll);
// Re-load everything whenever the bound api adapter changes (e.g. platform
// switches org). Reset transient state so the previous org's data does not leak.
watch(() => props.api, () => {
  search.value = "";
  statusFilter.value = "";
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
  selectedIds.value = new Set();
  loadAll();
});

async function loadAll() {
  deptLoading.value = true;
  try {
    [deptTree.value, deptFlat.value] = await Promise.all([props.api.fetchDeptTree(), props.api.fetchDeptFlat()]);
  } finally {
    deptLoading.value = false;
  }
  await reload();
}

async function reload() {
  loading.value = true;
  try {
    const res = await props.api.listMembers({
      page: page.value,
      pageSize: pageSize.value,
      search: search.value.trim(),
      status: statusFilter.value,
      deptId: activeDeptId.value,
      subtree: subtree.value,
    });
    members.value = res.data;
    total.value = res.total;
    // Drop selections for rows no longer present.
    const present = new Set(members.value.map((m) => m.member_id));
    selectedIds.value = new Set([...selectedIds.value].filter((id) => present.has(id)));
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

type EditorMode = "create" | "edit";
const editorOpen = ref(false);
const editorMode = ref<EditorMode>("create");
const editorMember = ref<MemberRow | null>(null);
// Inline per-field validation for the editor modal.
const memberFieldErr = reactive<{ username: string; display_name: string; dept_id: string }>({ username: "", display_name: "", dept_id: "" });
function validateMemberField(field: "username" | "display_name" | "dept_id") {
  const f = memberForm.value;
  if (field === "username") memberFieldErr.username = editorMode.value === "create" && !f.username.trim() ? "用户名不能为空" : "";
  if (field === "display_name") memberFieldErr.display_name = f.display_name.trim() ? "" : "姓名不能为空";
  if (field === "dept_id") memberFieldErr.dept_id = f.dept_id ? "" : "请选择所属组织";
}
function resetMemberFieldErr() { memberFieldErr.username = ""; memberFieldErr.display_name = ""; memberFieldErr.dept_id = ""; }
const memberForm = ref({
  username: "",
  display_name: "",
  phone: "",
  email: "",
  gender: "",
  dept_id: "",
  title: "",
  role_code: "tenant_member",
  post_id: "",
  status: "active",
  password: "",
  remark: "",
});

async function ensurePickers() {
  const tasks: Promise<any>[] = [];
  if (allRoles.value.length === 0) tasks.push(props.api.listRoles().then((r) => { allRoles.value = r; }));
  if (allPosts.value.length === 0) tasks.push(props.api.listPosts().then((p) => { allPosts.value = p; }));
  if (tasks.length) await Promise.all(tasks);
}

async function openCreate() {
  if (!props.api.createMember) return;
  await ensurePickers();
  editorMode.value = "create";
  editorMember.value = null;
  memberForm.value = {
    username: "",
    display_name: "",
    phone: "",
    email: "",
    gender: "",
    dept_id: activeDeptId.value || deptFlat.value.find((d) => !d.parent_id)?.id || deptFlat.value[0]?.id || "",
    title: "",
    role_code: allRoles.value.find((r) => r.code === "tenant_member")?.code || allRoles.value[0]?.code || "tenant_member",
    post_id: "",
    status: "active",
    password: randomPassword(),
    remark: "",
  };
  actionError.value = "";
  resetMemberFieldErr();
  editorOpen.value = true;
}

async function openEdit(m: MemberRow) {
  if (!props.api.updateMember) return;
  editorMode.value = "edit";
  editorMember.value = m;
  memberForm.value = {
    username: m.username || "",
    display_name: m.display_name || "",
    phone: m.phone || "",
    email: m.email || "",
    gender: m.gender || "",
    dept_id: m.dept_id || "",
    title: m.title || "",
    role_code: m.roles?.[0]?.code || "tenant_member",
    post_id: "",
    status: m.status || "active",
    password: "",
    remark: m.remark || "",
  };
  actionError.value = "";
  resetMemberFieldErr();
  editorOpen.value = true;
}

function closeEditor() {
  editorOpen.value = false;
  editorMember.value = null;
  actionError.value = "";
  resetMemberFieldErr();
}

async function saveMember() {
  const f = memberForm.value;
  validateMemberField("username");
  validateMemberField("display_name");
  validateMemberField("dept_id");
  if (memberFieldErr.username || memberFieldErr.display_name || memberFieldErr.dept_id) { actionError.value = ""; return; }
  busy.value = true; actionError.value = "";
  try {
    if (editorMode.value === "create") {
      if (!props.api.createMember) return;
      await props.api.createMember({
        username: f.username.trim(),
        display_name: f.display_name.trim(),
        phone: f.phone.trim(),
        email: f.email.trim(),
        gender: f.gender,
        dept_id: f.dept_id,
        title: f.title.trim(),
        role_codes: f.role_code ? [f.role_code] : ["tenant_member"],
        post_ids: f.post_id ? [f.post_id] : [],
        status: f.status,
        password: f.password,
        remark: f.remark.trim(),
      });
      notify.success("用户已创建");
    } else if (editorMember.value && props.api.updateMember) {
      await props.api.updateMember(editorMember.value.member_id, {
        display_name: f.display_name.trim(),
        phone: f.phone.trim(),
        email: f.email.trim(),
        gender: f.gender,
        dept_id: f.dept_id,
        title: f.title.trim(),
        status: f.status,
        remark: f.remark.trim(),
      });
      notify.success("用户已更新");
    }
    closeEditor();
    await reload();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "保存失败";
  } finally { busy.value = false; }
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
  if (exporting.value) return;
  exporting.value = true;
  try {
    await props.api.exportCsv({
      page: page.value,
      pageSize: pageSize.value,
      search: search.value.trim(),
      status: statusFilter.value,
      deptId: activeDeptId.value,
      subtree: subtree.value,
    });
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "导出失败");
  } finally {
    exporting.value = false;
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
.search-wrap { position: relative; }
.search { width: 220px; font-size: 13px; padding: 6px 28px 6px 10px; box-sizing: border-box; }
.search-clear {
  position: absolute; right: 6px; top: 50%; transform: translateY(-50%);
  border: 0; background: transparent; color: var(--text-4); cursor: pointer;
  width: 18px; height: 18px; display: grid; place-items: center; border-radius: var(--r-sm);
  transition: color .15s ease, background .15s ease;
}
.search-clear:hover { color: var(--text-2); background: var(--surface-2); }
.status-select { width: 104px; font-size: 12px; padding: 5px 8px; }

.split { display: grid; grid-template-columns: 260px 1fr; gap: 16px; align-items: start; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.tree-pane { padding: 12px; }
.pane-head-row { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 8px; }
.tree-tools { display: flex; gap: 4px; padding: 0 4px 6px; }
.tree-tool-btn {
  border: 1px solid var(--border); background: var(--surface); color: var(--text-3);
  font-size: 11px; padding: 3px 7px; border-radius: var(--r-sm); cursor: pointer;
  transition: background .15s ease, color .15s ease, border-color .15s ease;
}
.tree-tool-btn:hover { background: var(--surface-2); color: var(--primary); border-color: var(--primary); }
.tree-search { padding: 0 4px 8px; }
.tree-search .search-wrap { position: relative; }
.tree-search .search-sm { width: 100%; font-size: 13px; padding: 6px 28px 6px 10px; box-sizing: border-box; border: 1px solid var(--border-strong); border-radius: 6px; background: var(--surface); }
.tree-search .search-sm:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.tree-skeleton { padding: 8px; }
.filter-all { padding: 6px 8px; font-size: 13px; border-radius: 6px; cursor: pointer; color: var(--text-2); transition: background .15s ease, color .15s ease; }
.filter-all:hover { background: var(--surface-2); }
.filter-all.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.subtree-toggle { display: flex; align-items: center; gap: 6px; font-size: 12px; color: var(--text-3); padding: 6px 8px; }
.members-pane { min-width: 0; }

/* Bulk action bar */
.bulk-bar {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 12px; margin-bottom: 10px;
  background: var(--primary-soft); border: 1px solid var(--primary);
  border-radius: var(--r-md);
}
.bulk-count { font-size: 12.5px; font-weight: 600; color: var(--primary); }
.bulk-spacer { flex: 1; }
.bulk-progress { font-size: 12.5px; color: var(--primary); font-variant-numeric: tabular-nums; }
.row-check { width: 15px; height: 15px; cursor: pointer; accent-color: var(--primary); }

.member-cell { display: flex; gap: 10px; align-items: center; }
.member-cell.off { opacity: .55; }
.avatar { width: 32px; height: 32px; border-radius: 50%; color: white; font-weight: 600; font-size: 12px; display: grid; place-items: center; }
.name { font-weight: 600; font-size: 13.5px; }
.sub-id { font-size: 11.5px; color: var(--text-3); margin-top: 2px; display: flex; align-items: center; gap: 4px; }
.acct { font-family: var(--ff-mono); font-size: 11px; color: var(--text-2); background: var(--surface-2); padding: 1px 5px; border-radius: 3px; }
.phone { color: var(--text-3); }
.chip-dept { display: inline-block; padding: 2px 8px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 999px; font-size: 11.5px; }
.chip-post { display: inline-block; padding: 2px 8px; margin: 0 4px 2px 0; background: var(--primary-soft); color: var(--primary); border-radius: 999px; font-size: 11.5px; font-weight: 600; }
.chip-role { display: inline-block; padding: 2px 8px; margin: 0 4px 2px 0; background: var(--surface-2); color: var(--text-2); border: 1px solid var(--border); border-radius: 999px; font-size: 11.5px; font-weight: 600; }
.contact-cell { display: flex; flex-direction: column; gap: 2px; font-size: 12px; }
.muted { color: var(--text-4); }
.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.badge-success { background: var(--success-soft); color: var(--success); }
.badge-neutral { background: var(--bg-deep); color: var(--text-3); }

.row-actions { display: flex; gap: 4px; justify-content: flex-end; flex-wrap: wrap; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; display: inline-flex; align-items: center; justify-content: center; transition: background .15s ease, border-color .15s ease, color .15s ease; }
.btn:hover { background: var(--bg); }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-ghost { background: var(--surface); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }
.btn-link { border: 0; background: none; color: var(--primary); font-size: 11.5px; cursor: pointer; text-align: left; padding: 4px 0 0; }

/* Inline button spinner */
.btn-spinner {
  display: inline-block; width: 12px; height: 12px; margin-right: 6px; vertical-align: -1px;
  border: 2px solid var(--border-strong); border-top-color: var(--primary);
  border-radius: 50%; animation: mm-spin .6s linear infinite;
}
.btn-spinner.light { border-color: color-mix(in srgb, currentColor 45%, transparent); border-top-color: currentColor; }
@keyframes mm-spin { to { transform: rotate(360deg); } }

.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(440px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.member-editor { width: min(720px, 94vw); max-height: 88vh; overflow-y: auto; }
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
.form-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.field-wide { grid-column: 1 / -1; }
.textarea { resize: vertical; min-height: 74px; }
.label { font-size: 12px; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); transition: border-color .15s ease, box-shadow .15s ease; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input.invalid { border-color: var(--danger); }
.input.invalid:focus { outline-color: var(--danger-soft); }
.field-error { font-size: 11.5px; color: var(--danger); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

/* Bulk bar enter/leave */
.bulk-bar-enter-active, .bulk-bar-leave-active { transition: opacity .2s ease, transform .2s ease; }
.bulk-bar-enter-from, .bulk-bar-leave-to { opacity: 0; transform: translateY(-6px); }

@media (max-width: 900px) {
  .split { grid-template-columns: 1fr; }
  .mm-head { flex-wrap: wrap; }
  .mm-actions { margin-left: 0; }
  .search { width: 100%; }
  .search-wrap { flex: 1; min-width: 160px; }
}
@media (max-width: 760px) {
  .form-grid { grid-template-columns: 1fr; }
}
@media (prefers-reduced-motion: reduce) {
  .btn, .input, .tree-tool-btn, .search-clear, .filter-all { transition: none; }
  .btn-spinner { animation-duration: 1.2s; }
  .bulk-bar-enter-active, .bulk-bar-leave-active { transition: none; }
  .bulk-bar-enter-from, .bulk-bar-leave-to { opacity: 1; transform: none; }
}
</style>
