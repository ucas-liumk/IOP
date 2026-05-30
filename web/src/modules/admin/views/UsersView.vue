<template>
  <section class="admin-page">
    <PageHeader :title="'用户管理'" :sub="tabMode === 'org' ? `左选组织，右管其成员（按部门树筛选 / 导入导出 / 角色岗位） · 平台共 ${tenants.length} 家组织` : `全部平台账号 · 共 ${allUsersTotal} 个`">
      <template #actions>
        <div class="head-actions">
          <!-- Segmented control / tab switcher -->
          <div class="seg-ctrl">
            <button class="seg-btn" :class="{ active: tabMode === 'org' }" @click="tabMode = 'org'">按组织</button>
            <button class="seg-btn" :class="{ active: tabMode === 'all' }" @click="tabMode = 'all'">全部平台账号</button>
          </div>
          <button class="btn btn-ghost" @click="reload">刷新</button>
          <button class="btn btn-primary" v-perm="'user:write'" @click="openCreate">+ 新建用户</button>
        </div>
      </template>
    </PageHeader>

    <div v-if="actionError && !creating" class="page-error">
      {{ actionError }}
      <button class="page-error-close" @click="actionError = ''">×</button>
    </div>

    <!-- ===== TAB: 按组织 ===== -->
    <div v-if="tabMode === 'org'" class="two-pane">
      <!-- LEFT: organization (tenant) list -->
      <aside class="org-pane card">
        <div class="org-pane-head">
          <span class="org-pane-title">组织机构</span>
          <span class="org-count">{{ tenants.length }}</span>
        </div>
        <div class="org-search">
          <input class="input search-sm" v-model="orgFilter" placeholder="搜索组织" />
        </div>

        <div v-if="filteredTenants.length === 0 && !loading" class="org-empty">
          <EmptyState title="无匹配组织" sub="调整搜索或新建组织" icon="◫" />
        </div>

        <ul v-else class="org-list">
          <li
            v-for="t in filteredTenants"
            :key="t.id"
            class="org-row"
            :class="{ selected: t.id === selectedOrgId }"
            @click="selectOrg(t)"
          >
            <div class="t-logo" :style="{ background: colorFor(t.name) }">{{ t.name[0] }}</div>
            <div class="t-main">
              <div class="t-name">{{ t.name }}</div>
              <div class="t-meta">
                <code>{{ t.slug }}</code>
                <span class="status-tag" :class="`status-${t.status}`">
                  <span class="dot"></span>{{ statusLabel(t.status) }}
                </span>
              </div>
            </div>
          </li>
        </ul>
      </aside>

      <!-- RIGHT: member manager for the selected org -->
      <div class="member-pane">
        <EmptyState
          v-if="!selectedOrg"
          title="选择左侧组织以管理其成员"
          sub="点击左侧任一组织，在此按部门树筛选、导入导出、分配角色 / 岗位 / 部门"
          icon="◫"
        />
        <MemberManager
          v-else
          :key="selectedOrg.id"
          :api="orgApi!"
          :template-url="orgMemberTemplateUrl(selectedOrg.id)"
          :import-url="orgMemberImportUrl(selectedOrg.id)"
          write-perm="user:write"
        >
          <template #head-left>
            <div class="org-banner">
              <span class="org-banner-name">{{ selectedOrg.name }}</span>
              <span class="org-banner-slug">schema: <code class="mono">{{ selectedOrg.schema_name }}</code></span>
            </div>
          </template>
        </MemberManager>
      </div>
    </div>

    <!-- ===== TAB: 全部平台账号 ===== -->
    <div v-else class="all-users-pane card">
      <div class="all-users-toolbar">
        <input
          class="input search-md"
          v-model="allSearch"
          placeholder="搜索账户名 / 手机号…"
          @input="onAllSearchInput"
        />
        <span class="spacer" />
        <span class="user-count-hint">共 {{ allUsersTotal }} 个账号</span>
      </div>

      <div v-if="allLoading" class="all-loading">
        <LoadingSpinner />
        <span>加载中…</span>
      </div>

      <template v-else>
        <div v-if="allUsers.length === 0" class="all-empty">
          <EmptyState title="暂无平台账号" sub="可通过「新建用户」按钮创建" icon="◫" />
        </div>

        <table v-else class="all-table">
          <thead>
            <tr>
              <th>账户名</th>
              <th>手机</th>
              <th>状态</th>
              <th>创建时间</th>
              <th>最近登录</th>
              <th>操作</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="u in allUsers" :key="u.id">
              <td class="col-username">
                <span class="username-primary">{{ u.username || '—' }}</span>
                <span v-if="u.email" class="username-email">{{ u.email }}</span>
              </td>
              <td>{{ u.phone || '—' }}</td>
              <td>
                <span class="status-tag" :class="u.status === 'active' ? 'status-active' : 'status-suspended'">
                  <span class="dot"></span>{{ u.status === 'active' ? '启用' : '停用' }}
                </span>
              </td>
              <td class="col-time">{{ formatTime(u.created_at) }}</td>
              <td class="col-time">{{ u.last_login_at ? formatTime(u.last_login_at) : '—' }}</td>
              <td>
                <div class="row-actions">
                  <button class="btn btn-ghost btn-sm" v-perm="'user:write'" @click="openResetPwd(u)">重置密码</button>
                  <button
                    class="btn btn-ghost btn-sm"
                    :class="u.status === 'active' ? 'btn-danger-ghost' : ''"
                    v-perm="'user:write'"
                    @click="toggleUserStatus(u)"
                  >{{ u.status === 'active' ? '停用' : '启用' }}</button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>

        <Pagination
          v-if="allUsersTotal > 0"
          :total="allUsersTotal"
          :page="allPage"
          :pageSize="allPageSize"
          @update:page="onAllPageChange"
          @update:pageSize="onAllPageSizeChange"
        />
      </template>
    </div>

    <!-- Create platform user modal -->
    <div v-if="creating" class="modal-overlay" @click.self="closeCreate">
      <div class="modal">
        <h3>新建平台用户</h3>
        <p class="modal-sub">由管理员直接创建，跳过审批流程，账号立即可用。</p>
        <label class="field">
          <span class="label">用户名</span>
          <input class="input" v-model="form.username" type="text" required autofocus
                 placeholder="3-32 位字母/数字/-/_，以字母开头" />
        </label>
        <label class="field">
          <span class="label">真实姓名</span>
          <input class="input" v-model="form.real_name" type="text" required maxlength="32"
                 placeholder="例如：张三" />
        </label>
        <label class="field">
          <span class="label">手机号 <span class="optional">（可选）</span></span>
          <input class="input" v-model="form.phone" type="tel" maxlength="11"
                 inputmode="numeric" placeholder="11 位手机号" />
        </label>
        <label class="field">
          <span class="label">所属单位</span>
          <select class="input" v-model="form.organization_id" required>
            <option value="" disabled>请选择</option>
            <option v-for="o in tenants" :key="o.id" :value="o.id">{{ o.name }}</option>
          </select>
        </label>
        <label class="field">
          <span class="label">角色</span>
          <select class="input" v-model="form.role">
            <option value="tenant_member">租户成员</option>
            <option value="tenant_admin">租户管理员</option>
          </select>
        </label>
        <label class="field">
          <span class="label">初始密码</span>
          <input class="input" v-model="form.password" type="text" minlength="10"
                 placeholder="至少 10 位，含字母与数字" />
          <button type="button" class="btn-link" @click="form.password = randomPassword()">生成强密码</button>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeCreate">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmCreate">
            {{ busy ? '创建中…' : '确认创建' }}
          </button>
        </div>
      </div>
    </div>

    <!-- Reset password modal -->
    <div v-if="resetPwdUser" class="modal-overlay" @click.self="closeResetPwd">
      <div class="modal">
        <h3>重置密码</h3>
        <p class="modal-sub">为账号 <strong>{{ resetPwdUser.username || resetPwdUser.phone || resetPwdUser.id }}</strong> 设置新密码。</p>
        <label class="field">
          <span class="label">新密码</span>
          <input class="input" v-model="resetPwdValue" type="text" minlength="10"
                 placeholder="至少 10 位，含字母与数字" />
          <button type="button" class="btn-link" @click="resetPwdValue = randomPassword()">生成强密码</button>
        </label>
        <div v-if="actionError" class="form-error">{{ actionError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="closeResetPwd">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="confirmResetPwd">
            {{ busy ? '重置中…' : '确认重置' }}
          </button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { PageHeader, EmptyState, MemberManager, Pagination, LoadingSpinner, type MemberApi } from "@/shell/components";
import {
  listAllTenants, createPlatformUser,
  listPlatformUsersPaged, disablePlatformUser, enablePlatformUser, resetPlatformUserPassword,
  type PlatformTenant, type PlatformUser,
} from "../api/admin";
import {
  orgMemberApi, orgMemberTemplateUrl, orgMemberImportUrl,
} from "@/modules/platform/api/orgs";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();

// ==================== Shared state ====================
const tenants = ref<PlatformTenant[]>([]);
const loading = ref(false);
const busy = ref(false);
const actionError = ref("");

// Current tab: "org" | "all"
const tabMode = ref<"org" | "all">("org");

// ==================== 按组织 tab ====================
const orgFilter = ref("");
const selectedOrgId = ref<string | null>(null);
const selectedOrg = computed(() => tenants.value.find((t) => t.id === selectedOrgId.value) ?? null);
const orgApi = computed<MemberApi | null>(() => (selectedOrg.value ? orgMemberApi(selectedOrg.value.id) : null));

const filteredTenants = computed(() => {
  const q = orgFilter.value.trim().toLowerCase();
  if (!q) return tenants.value;
  return tenants.value.filter((t) => t.name.toLowerCase().includes(q) || t.slug.toLowerCase().includes(q));
});

function selectOrg(t: PlatformTenant) {
  selectedOrgId.value = t.id;
}

// ==================== 全部平台账号 tab ====================
const allUsers = ref<PlatformUser[]>([]);
const allUsersTotal = ref(0);
const allPage = ref(1);
const allPageSize = ref(20);
const allSearch = ref("");
const allLoading = ref(false);

let allSearchTimer: ReturnType<typeof setTimeout> | null = null;

function onAllSearchInput() {
  if (allSearchTimer) clearTimeout(allSearchTimer);
  allSearchTimer = setTimeout(() => {
    allPage.value = 1;
    loadAllUsers();
  }, 300);
}

function onAllPageChange(p: number) {
  allPage.value = p;
  loadAllUsers();
}

function onAllPageSizeChange(s: number) {
  allPageSize.value = s;
  allPage.value = 1;
  loadAllUsers();
}

async function loadAllUsers() {
  allLoading.value = true;
  try {
    const result = await listPlatformUsersPaged({
      page: allPage.value,
      pageSize: allPageSize.value,
      search: allSearch.value.trim() || undefined,
    });
    allUsers.value = result.data;
    allUsersTotal.value = result.total;
    allPage.value = result.page;
    allPageSize.value = result.pageSize;
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "加载用户列表失败");
  } finally {
    allLoading.value = false;
  }
}

// Load all-users list when switching to that tab.
watch(tabMode, (m) => {
  if (m === "all") loadAllUsers();
});

// ==================== Reset password ====================
const resetPwdUser = ref<PlatformUser | null>(null);
const resetPwdValue = ref("");

function openResetPwd(u: PlatformUser) {
  resetPwdUser.value = u;
  resetPwdValue.value = randomPassword();
  actionError.value = "";
}
function closeResetPwd() {
  resetPwdUser.value = null;
  resetPwdValue.value = "";
}
async function confirmResetPwd() {
  if (!resetPwdUser.value) return;
  if (!resetPwdValue.value || resetPwdValue.value.length < 10) {
    actionError.value = "密码至少 10 位";
    return;
  }
  busy.value = true;
  actionError.value = "";
  try {
    await resetPlatformUserPassword(resetPwdUser.value.id, resetPwdValue.value);
    notify.success("密码已重置");
    closeResetPwd();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "重置失败";
  } finally {
    busy.value = false;
  }
}

// ==================== Disable / Enable ====================
async function toggleUserStatus(u: PlatformUser) {
  const isActive = u.status === "active";
  const label = u.username || u.phone || u.id;
  const ok = await confirm({
    title: isActive ? "确认停用" : "确认启用",
    message: isActive
      ? `确定停用账号「${label}」？该账号将无法登录。`
      : `确定启用账号「${label}」？`,
    danger: isActive,
  });
  if (!ok) return;
  busy.value = true;
  try {
    if (isActive) {
      await disablePlatformUser(u.id);
    } else {
      await enablePlatformUser(u.id);
    }
    notify.success(isActive ? "账号已停用" : "账号已启用");
    // Refresh the list in place.
    await loadAllUsers();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "操作失败");
  } finally {
    busy.value = false;
  }
}

// ==================== Lifecycle ====================
onMounted(reload);
async function reload() {
  loading.value = true;
  try {
    tenants.value = await listAllTenants();
    if (selectedOrgId.value && !tenants.value.some((t) => t.id === selectedOrgId.value)) {
      selectedOrgId.value = null;
    }
    if (tabMode.value === "all") {
      await loadAllUsers();
    }
  } finally { loading.value = false; }
}

// ==================== Create platform user modal ====================
const creating = ref(false);
const form = reactive({
  username: "", real_name: "", phone: "",
  organization_id: "", role: "tenant_member" as "tenant_member" | "tenant_admin",
  password: "",
});

function openCreate() {
  creating.value = true;
  form.username = "";
  form.real_name = "";
  form.phone = "";
  form.organization_id = selectedOrg.value?.id ?? tenants.value[0]?.id ?? "";
  form.role = "tenant_member";
  form.password = randomPassword();
  actionError.value = "";
}
function closeCreate() { creating.value = false; }

async function confirmCreate() {
  if (!form.organization_id) { actionError.value = "请选择所属单位"; return; }
  busy.value = true; actionError.value = "";
  try {
    await createPlatformUser({ ...form });
    creating.value = false;
    notify.success("用户已创建");
    // If the new user joined the currently-selected org, remount the manager to refresh.
    if (form.organization_id === selectedOrgId.value) {
      const cur = selectedOrgId.value;
      selectedOrgId.value = null;
      await nextTickReselect(cur);
    }
    // Also refresh the flat list if it's been loaded.
    if (tabMode.value === "all" || allUsersTotal.value > 0) {
      await loadAllUsers();
    }
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "创建失败";
  } finally { busy.value = false; }
}

// Reselect on the next microtask so the keyed MemberManager fully remounts.
async function nextTickReselect(id: string | null) {
  await Promise.resolve();
  selectedOrgId.value = id;
}

// ==================== Helpers ====================
function statusLabel(s: string) {
  return ({ active: "运行中", suspended: "已暂停", closed: "已关闭" } as Record<string, string>)[s] ?? s;
}
function colorFor(name: string) {
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
function formatTime(s: string): string {
  if (!s) return "—";
  const d = new Date(s);
  if (isNaN(d.getTime())) return s;
  return d.toLocaleString("zh-CN", { year: "numeric", month: "2-digit", day: "2-digit", hour: "2-digit", minute: "2-digit" });
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
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.head-actions { display: flex; gap: 8px; align-items: center; }
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close { border: 0; background: transparent; color: inherit; font-size: 18px; line-height: 1; cursor: pointer; }

/* Segmented control */
.seg-ctrl {
  display: inline-flex;
  border: 1px solid var(--border);
  border-radius: 8px;
  overflow: hidden;
  background: var(--surface-2);
}
.seg-btn {
  border: 0;
  background: transparent;
  padding: 5px 14px;
  font-size: 13px;
  cursor: pointer;
  color: var(--text-2);
  transition: background .12s, color .12s;
  white-space: nowrap;
}
.seg-btn:hover { background: var(--surface); color: var(--text); }
.seg-btn.active { background: var(--surface); color: var(--primary); font-weight: 600; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }

/* ====== 按组织 tab ====== */
.two-pane { display: grid; grid-template-columns: 320px 1fr; gap: 16px; align-items: start; }

/* LEFT pane */
.org-pane { padding: 8px; display: flex; flex-direction: column; max-height: calc(100vh - 200px); }
.org-pane-head {
  display: flex; align-items: center; gap: 8px;
  padding: 8px 10px 8px;
  font-size: 11.5px; font-weight: 600; color: var(--text-3);
  text-transform: uppercase; letter-spacing: .5px;
}
.org-count { margin-left: auto; background: var(--surface-2); color: var(--text-2); font-size: 11px; font-weight: 700; padding: 1px 8px; border-radius: 999px; }
.org-search { padding: 0 6px 8px; }
.org-search .search-sm { width: 100%; font-size: 13px; padding: 6px 10px; box-sizing: border-box; border: 1px solid var(--border-strong); border-radius: 6px; background: var(--surface); }
.org-search .search-sm:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.org-empty { padding: 8px 0; }
.org-list { list-style: none; margin: 0; padding: 0; overflow: auto; }
.org-row {
  display: flex; align-items: center; gap: 10px;
  padding: 9px 10px; border-radius: 9px; cursor: pointer;
  border: 1px solid transparent;
  transition: background .12s, border-color .12s;
}
.org-row:hover { background: var(--surface-2); }
.org-row.selected { background: var(--primary-soft); border-color: var(--primary); }
.t-logo { width: 32px; height: 32px; border-radius: 7px; color: white; font-weight: 700; display: grid; place-items: center; font-size: 14px; flex-shrink: 0; }
.t-main { flex: 1; min-width: 0; }
.t-name { font-size: 13.5px; font-weight: 600; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.t-meta { font-size: 11.5px; color: var(--text-3); margin-top: 3px; display: flex; gap: 8px; align-items: center; }
code { background: var(--surface-2); padding: 1px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11px; }
code.mono { color: var(--text-2); }
.org-row.selected code { background: var(--surface); }

.status-tag { display: inline-flex; align-items: center; gap: 5px; padding: 1px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.status-tag .dot { width: 5px; height: 5px; background: currentColor; border-radius: 50%; }
.status-active { background: var(--success-soft); color: var(--success); }
.status-suspended { background: var(--warning-soft); color: var(--warning); }
.status-closed { background: var(--danger-soft); color: var(--danger); }

/* RIGHT pane */
.member-pane { min-width: 0; }
.org-banner { display: flex; flex-direction: column; gap: 2px; }
.org-banner-name { font-size: 15px; font-weight: 600; color: var(--text); }
.org-banner-slug { font-size: 11.5px; color: var(--text-3); }

/* ====== 全部平台账号 tab ====== */
.all-users-pane { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.all-users-toolbar {
  display: flex; align-items: center; gap: 10px;
  flex-wrap: wrap;
}
.search-md {
  width: 260px; font-size: 13px; padding: 6px 10px;
  border: 1px solid var(--border-strong); border-radius: 6px; background: var(--surface);
}
.search-md:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.spacer { flex: 1; }
.user-count-hint { font-size: 12px; color: var(--text-3); white-space: nowrap; }
.all-loading { display: flex; align-items: center; gap: 8px; color: var(--text-3); font-size: 13px; padding: 20px 0; }
.all-empty { padding: 8px 0; }

.all-table {
  width: 100%; border-collapse: collapse; font-size: 13px;
}
.all-table th {
  text-align: left; padding: 8px 10px;
  font-size: 11.5px; font-weight: 600; color: var(--text-3);
  border-bottom: 1px solid var(--border); white-space: nowrap;
}
.all-table td {
  padding: 10px 10px; border-bottom: 1px solid var(--border);
  color: var(--text); vertical-align: middle;
}
.all-table tr:last-child td { border-bottom: 0; }
.all-table tr:hover td { background: var(--surface-2); }

.col-username { min-width: 120px; }
.username-primary { font-weight: 600; display: block; }
.username-email { font-size: 11.5px; color: var(--text-3); display: block; margin-top: 1px; }
.col-time { font-size: 12px; color: var(--text-3); white-space: nowrap; }

.row-actions { display: flex; gap: 6px; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-danger-ghost { color: var(--danger); border-color: var(--danger-soft); }
.btn-danger-ghost:hover { background: var(--danger-soft); }

/* ====== modal ====== */
.modal-overlay { position: fixed; inset: 0; background: rgba(13, 27, 46, .45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 14px; padding: 22px; width: min(440px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-sub { font-size: 12.5px; color: var(--text-3); margin: -6px 0 4px; }
.field { display: flex; flex-direction: column; gap: 4px; }
.field .label { font-size: 12px; color: var(--text-2); }
.optional { color: var(--text-4); font-weight: 400; }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.btn-link { border: 0; background: none; color: var(--primary); font-size: 11.5px; cursor: pointer; text-align: left; padding: 4px 0 0; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-ghost { background: var(--surface); }
</style>
