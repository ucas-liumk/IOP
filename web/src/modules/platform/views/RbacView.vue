<template>
  <section class="admin-page">
    <PageHeader title="角色权限" :sub="tab === 'platform' ? `平台角色 ${roles.length} 个` : `全部角色 ${allRoles.length} 个`">
      <template #actions>
        <select class="tenant-select" v-model="allFilters.tenant_id" @change="reloadAll">
          <option value="">全部租户</option>
          <option v-for="t in tenants" :key="t.id" :value="t.id">{{ t.name }}</option>
        </select>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button v-if="tab === 'platform'" class="btn btn-primary" v-perm="'role:manage'" @click="openCreate">+ 新建平台角色</button>
      </template>
    </PageHeader>

    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'platform' }" @click="tab = 'platform'">平台角色</button>
      <button class="tab" :class="{ active: tab === 'all' }" @click="tab = 'all'">全部角色</button>
    </div>

    <div v-if="pageError" class="page-error">
      {{ pageError }}
      <button class="page-error-close" @click="pageError = ''">×</button>
    </div>

    <template v-if="tab === 'platform'">
      <div class="split">
        <article class="card list-pane">
          <div class="pane-head">平台角色</div>
          <div class="filters">
            <input class="input input-sm" v-model="platformFilters.q" placeholder="名称 / 编码" @keyup.enter="reloadPlatform" />
            <select class="input input-sm" v-model="platformFilters.status" @change="reloadPlatform">
              <option value="">全部状态</option>
              <option value="active">正常</option>
              <option value="disabled">停用</option>
            </select>
          </div>
          <ul class="role-list">
            <li
              v-for="r in roles"
              :key="r.id"
              class="role-item"
              :class="{ active: r.id === selectedId }"
              @click="select(r.id)"
            >
              <div class="ri-main">
                <span class="ri-name">{{ r.name }}</span>
                <code class="ri-code">{{ r.code }}</code>
              </div>
              <div class="ri-meta">
                <span class="status-badge" :class="r.status === 'active' ? 'ok' : 'off'">
                  {{ r.status === 'active' ? '正常' : '停用' }}
                </span>
                <span v-if="r.built_in" class="tag-builtin">内置</span>
                <span class="ri-count">{{ r.member_count }} 人</span>
              </div>
            </li>
            <li v-if="roles.length === 0" class="muted">尚无角色</li>
          </ul>
        </article>

        <article class="card editor-pane">
          <EmptyState v-if="!selectedRole" title="选择一个角色" sub="从左侧选择平台角色编辑其菜单权限与成员。" icon="◷" />
          <template v-else>
            <PlatformRoleEditor
              :key="selectedRole.id"
              :role="selectedRole"
              :menus="menus"
              :users="users"
              @changed="reloadPlatform"
            />
            <div v-if="!selectedRole.built_in" class="danger-zone">
              <button class="btn btn-ghost btn-sm danger" v-perm="'role:manage'" @click="removeRole">删除该角色</button>
            </div>
          </template>
        </article>
      </div>
    </template>

    <article v-else class="card all-pane">
      <div class="table-tools">
        <input class="input input-sm" v-model="allFilters.q" placeholder="名称 / 编码" @keyup.enter="reloadAll" />
        <select class="input input-sm" v-model="allFilters.role_type" @change="reloadAll">
          <option value="">全部类型</option>
          <option value="platform">平台角色</option>
          <option value="tenant">租户角色</option>
        </select>
        <select class="input input-sm" v-model="allFilters.status" @change="reloadAll">
          <option value="">全部状态</option>
          <option value="active">正常</option>
          <option value="disabled">停用</option>
        </select>
      </div>
      <table class="role-table">
        <thead>
          <tr>
            <th>角色</th>
            <th>类型</th>
            <th>租户</th>
            <th>数据范围</th>
            <th>状态</th>
            <th>用户数</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in allRoles" :key="r.id">
            <td>
              <div class="table-role">
                <span>{{ r.name }}</span>
                <code>{{ r.code }}</code>
                <em v-if="r.built_in">内置</em>
              </div>
            </td>
            <td>{{ r.role_type === 'platform' ? '平台角色' : '租户角色' }}</td>
            <td>{{ tenantName(r.tenant_id) }}</td>
            <td>{{ scopeLabel(r.data_scope) }}</td>
            <td>
              <span class="status-badge" :class="r.status === 'active' ? 'ok' : 'off'">
                {{ r.status === 'active' ? '正常' : '停用' }}
              </span>
            </td>
            <td>{{ r.member_count }}</td>
          </tr>
          <tr v-if="allRoles.length === 0">
            <td colspan="6" class="empty-cell">没有匹配的角色</td>
          </tr>
        </tbody>
      </table>
    </article>

    <div v-if="creating" class="modal-overlay" @click.self="creating = false">
      <div class="modal">
        <h3>新建平台角色</h3>
        <div class="modal-grid">
          <label class="field">
            <span class="label">编码 *</span>
            <input class="input" v-model="form.code" pattern="[a-z][a-z0-9_-]*" placeholder="例如：ops_admin" />
          </label>
          <label class="field">
            <span class="label">名称 *</span>
            <input class="input" v-model="form.name" placeholder="例如：运维管理员" />
          </label>
          <label class="field">
            <span class="label">状态</span>
            <select class="input" v-model="form.status">
              <option value="active">正常</option>
              <option value="disabled">停用</option>
            </select>
          </label>
          <label class="field">
            <span class="label">排序</span>
            <input class="input" type="number" v-model.number="form.order_num" />
          </label>
        </div>
        <label class="field">
          <span class="label">备注</span>
          <textarea class="input textarea" v-model="form.remark" rows="3" />
        </label>
        <div v-if="formError" class="form-error">{{ formError }}</div>
        <div class="modal-actions">
          <button class="btn btn-ghost" @click="creating = false">取消</button>
          <button class="btn btn-primary" :disabled="busy" @click="create">创建</button>
        </div>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { PageHeader, EmptyState } from "@/shell/components";
import PlatformRoleEditor from "../components/PlatformRoleEditor.vue";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  listPlatformRoles, createPlatformRole, deletePlatformRole, listAllRoles,
  type PlatformRole, type RoleSummary, type RoleStatus,
} from "../api/rbac";
import { getPlatformMenuTree, type MenuTreeNode } from "../api/menus";
import { listPlatformUsers, listAllTenants, type PlatformUser, type PlatformTenant } from "@/modules/admin/api/admin";

const notify = useNotification();
const { confirm } = useConfirm();

const tab = ref<"platform" | "all">("platform");
const roles = ref<PlatformRole[]>([]);
const allRoles = ref<RoleSummary[]>([]);
const menus = ref<MenuTreeNode[]>([]);
const users = ref<PlatformUser[]>([]);
const tenants = ref<PlatformTenant[]>([]);
const selectedId = ref<string | null>(null);
const pageError = ref("");

const platformFilters = reactive({ q: "", status: "" });
const allFilters = reactive({ q: "", status: "", role_type: "", tenant_id: "" });

const selectedRole = computed(() => roles.value.find((r) => r.id === selectedId.value) ?? null);
const tenantMap = computed(() => Object.fromEntries(tenants.value.map((t) => [t.id, t.name])));

const creating = ref(false);
const busy = ref(false);
const formError = ref("");
const form = reactive({
  code: "",
  name: "",
  status: "active" as RoleStatus,
  order_num: 0,
  remark: "",
});

onMounted(async () => {
  [menus.value, users.value, tenants.value] = await Promise.all([
    getPlatformMenuTree().catch(() => []),
    listPlatformUsers("").catch(() => []),
    listAllTenants().catch(() => []),
  ]);
  await Promise.all([reloadPlatform(), reloadAll()]);
});

watch(tab, async (v) => {
  if (v === "all") await reloadAll();
});

function reload() {
  return tab.value === "platform" ? reloadPlatform() : reloadAll();
}

async function reloadPlatform() {
  try {
    roles.value = await listPlatformRoles({
      q: platformFilters.q.trim() || undefined,
      status: platformFilters.status || undefined,
    });
    pageError.value = "";
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "加载失败，请检查权限或重试";
    return;
  }
  if (selectedId.value && !roles.value.some((r) => r.id === selectedId.value)) {
    selectedId.value = null;
  } else if (!selectedId.value && roles.value.length > 0) {
    selectedId.value = roles.value[0].id;
  }
}

async function reloadAll() {
  try {
    allRoles.value = await listAllRoles({
      q: allFilters.q.trim() || undefined,
      status: allFilters.status || undefined,
      role_type: allFilters.role_type || undefined,
      tenant_id: allFilters.tenant_id || undefined,
    });
    pageError.value = "";
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "加载失败，请检查权限或重试";
  }
}

function select(id: string) { selectedId.value = id; }

function openCreate() {
  creating.value = true;
  formError.value = "";
  form.code = "";
  form.name = "";
  form.status = "active";
  form.order_num = 0;
  form.remark = "";
}

async function create() {
  if (!form.code.trim() || !form.name.trim()) { formError.value = "编码与名称不能为空"; return; }
  busy.value = true; formError.value = "";
  try {
    const created = await createPlatformRole({
      code: form.code.trim(),
      name: form.name.trim(),
      status: form.status,
      order_num: form.order_num,
      remark: form.remark.trim(),
    });
    creating.value = false;
    await reloadPlatform();
    selectedId.value = roles.value.find((r) => r.id === created.id)?.id ?? roles.value.find((r) => r.code === form.code.trim())?.id ?? null;
    notify.success("角色已创建");
  } catch (e: any) {
    formError.value = e.response?.data?.error?.message ?? "创建失败";
  } finally { busy.value = false; }
}

async function removeRole() {
  if (!selectedRole.value) return;
  const ok = await confirm({ title: "删除角色", message: `确认删除角色「${selectedRole.value.name}」？`, danger: true });
  if (!ok) return;
  try {
    await deletePlatformRole(selectedRole.value.id);
    selectedId.value = null;
    await reloadPlatform();
    await reloadAll();
    notify.success("角色已删除");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}

function tenantName(id?: string) {
  if (!id) return "平台";
  return tenantMap.value[id] ?? id;
}
function scopeLabel(scope: string) {
  const labels: Record<string, string> = {
    all: "全部数据",
    tenant: "本租户数据",
    dept: "本组织数据",
    dept_and_sub: "本组织及下级",
    self: "仅本人",
    custom: "自定义组织",
  };
  return labels[scope] ?? scope;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.tenant-select { padding: 7px 10px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); min-width: 160px; }
.tabs { display: inline-flex; gap: 2px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 8px; padding: 3px; width: max-content; }
.tab { border: 0; background: transparent; border-radius: 6px; padding: 6px 12px; font-size: 13px; cursor: pointer; color: var(--text-2); }
.tab.active { background: var(--surface); color: var(--text); box-shadow: var(--sh-1); font-weight: 600; }
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close { border: 0; background: transparent; color: inherit; font-size: 18px; line-height: 1; cursor: pointer; }
.split { display: grid; grid-template-columns: 300px 1fr; gap: 16px; align-items: start; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.list-pane { padding: 12px; }
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 8px; }
.filters, .table-tools { display: grid; grid-template-columns: 1fr 94px; gap: 8px; padding: 0 4px 10px; }
.table-tools { grid-template-columns: minmax(180px, 1fr) 130px 120px; padding: 0 0 12px; }
.role-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }
.role-item { display: flex; justify-content: space-between; align-items: center; gap: 10px; padding: 9px 10px; border-radius: 7px; cursor: pointer; }
.role-item:hover { background: var(--surface-2); }
.role-item.active { background: var(--primary-soft); }
.ri-main { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.ri-name { font-size: 13.5px; font-weight: 600; color: var(--text); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.role-item.active .ri-name { color: var(--primary); }
.ri-code { font-family: var(--ff-mono); font-size: 11px; color: var(--text-3); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ri-meta { display: flex; align-items: center; gap: 6px; flex-shrink: 0; }
.ri-count { font-size: 11px; color: var(--text-3); }
.tag-builtin, .status-badge { font-size: 10px; font-weight: 700; padding: 1px 5px; border-radius: 3px; white-space: nowrap; }
.tag-builtin { background: var(--purple-soft); color: var(--purple); }
.status-badge.ok { background: var(--success-soft); color: var(--success); }
.status-badge.off { background: var(--surface-2); color: var(--text-3); }
.muted { color: var(--text-4); font-size: 12.5px; padding: 8px; }

.editor-pane { padding: 20px 22px; min-height: 360px; }
.danger-zone { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border-soft); }
.all-pane { padding: 16px; }
.role-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.role-table th, .role-table td { text-align: left; padding: 10px 8px; border-bottom: 1px solid var(--border-soft); vertical-align: top; }
.role-table th { color: var(--text-3); font-size: 11.5px; font-weight: 600; text-transform: uppercase; letter-spacing: .4px; }
.table-role { display: flex; flex-direction: column; gap: 2px; }
.table-role span { font-weight: 600; }
.table-role code { font-family: var(--ff-mono); font-size: 11px; color: var(--text-3); }
.table-role em { width: max-content; font-style: normal; font-size: 10px; font-weight: 700; color: var(--purple); background: var(--purple-soft); padding: 1px 5px; border-radius: 3px; }
.empty-cell { text-align: center; color: var(--text-4); padding: 28px 8px; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }

.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); min-width: 0; }
.input-sm { font-size: 12px; padding: 6px 8px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.textarea { resize: vertical; min-height: 68px; }
.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 12px; padding: 22px; width: min(560px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; color: var(--text-2); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
@media (max-width: 900px) {
  .split { grid-template-columns: 1fr; }
  .table-tools { grid-template-columns: 1fr; }
  .modal-grid { grid-template-columns: 1fr; }
}
</style>
