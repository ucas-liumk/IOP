<template>
  <section class="admin-page">
    <PageHeader title="角色管理" :sub="`共 ${roles.length} 个租户角色 · 菜单权限 + 数据范围`">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
        <button class="btn btn-primary" v-perm="'role:write'" @click="openCreate">+ 新建角色</button>
      </template>
    </PageHeader>

    <div class="split">
      <article class="card list-pane">
        <div class="pane-head">角色列表</div>
        <div class="filters">
          <input class="input input-sm" v-model="filters.q" placeholder="名称 / 编码" @keyup.enter="reload" />
          <select class="input input-sm" v-model="filters.status" @change="reload">
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
        <EmptyState v-if="!selectedRole" title="选择一个角色" sub="从左侧选择角色编辑其菜单权限与数据范围。" icon="◷" />
        <template v-else>
          <RoleEditor
            :key="selectedRole.id"
            :role="selectedRole"
            :menus="menus"
            :depts="depts"
            @changed="reload"
          />
          <div v-if="!selectedRole.built_in" class="danger-zone">
            <button class="btn btn-ghost btn-sm danger" v-perm="'role:write'" @click="removeRole">删除该角色</button>
          </div>
        </template>
      </article>
    </div>

    <div v-if="creating" class="modal-overlay" @click.self="creating = false">
      <div class="modal">
        <h3>新建租户角色</h3>
        <div class="modal-grid">
          <label class="field">
            <span class="label">编码 *</span>
            <input class="input" v-model="form.code" pattern="[a-z][a-z0-9_-]*" placeholder="例如：dept_admin" />
          </label>
          <label class="field">
            <span class="label">名称 *</span>
            <input class="input" v-model="form.name" placeholder="例如：部门管理员" />
          </label>
          <label class="field">
            <span class="label">角色类型</span>
            <select class="input" v-model="form.role_type" disabled>
              <option value="tenant">租户角色</option>
            </select>
          </label>
          <label class="field">
            <span class="label">状态</span>
            <select class="input" v-model="form.status">
              <option value="active">正常</option>
              <option value="disabled">停用</option>
            </select>
          </label>
          <label class="field">
            <span class="label">数据范围</span>
            <select class="input" v-model="form.data_scope">
              <option value="tenant">本租户数据</option>
              <option value="dept">本组织数据</option>
              <option value="dept_and_sub">本组织及下级组织</option>
              <option value="self">仅本人数据</option>
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
import { computed, onMounted, reactive, ref } from "vue";
import { PageHeader, EmptyState } from "@/shell/components";
import RoleEditor from "../components/RoleEditor.vue";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  listRoles, createRole, deleteRole,
  getTenantMenuTree, listDepts,
  type Role, type MenuNode, type Dept, type DataScope, type RoleStatus,
} from "../api/admin";

const notify = useNotification();
const { confirm } = useConfirm();

const roles = ref<Role[]>([]);
const menus = ref<MenuNode[]>([]);
const depts = ref<Dept[]>([]);
const selectedId = ref<string | null>(null);
const filters = reactive({ q: "", status: "" });

const selectedRole = computed(() => roles.value.find((r) => r.id === selectedId.value) ?? null);

const creating = ref(false);
const busy = ref(false);
const formError = ref("");
const form = reactive({
  code: "",
  name: "",
  role_type: "tenant" as const,
  status: "active" as RoleStatus,
  data_scope: "tenant" as DataScope,
  order_num: 0,
  remark: "",
});

onMounted(async () => {
  [menus.value, depts.value] = await Promise.all([
    getTenantMenuTree().catch(() => []),
    listDepts().catch(() => []),
  ]);
  await reload();
});

async function reload() {
  roles.value = await listRoles({
    q: filters.q.trim() || undefined,
    status: filters.status || undefined,
    role_type: "tenant",
  });
  if (selectedId.value && !roles.value.some((r) => r.id === selectedId.value)) {
    selectedId.value = null;
  } else if (!selectedId.value && roles.value.length > 0) {
    selectedId.value = roles.value[0].id;
  }
}

function select(id: string) { selectedId.value = id; }

function openCreate() {
  creating.value = true;
  formError.value = "";
  form.code = "";
  form.name = "";
  form.role_type = "tenant";
  form.status = "active";
  form.data_scope = "tenant";
  form.order_num = 0;
  form.remark = "";
}

async function create() {
  if (!form.code.trim() || !form.name.trim()) { formError.value = "编码与名称不能为空"; return; }
  busy.value = true; formError.value = "";
  try {
    const r = await createRole({
      code: form.code.trim(),
      name: form.name.trim(),
      role_type: "tenant",
      status: form.status,
      data_scope: form.data_scope,
      order_num: form.order_num,
      remark: form.remark.trim(),
    });
    creating.value = false;
    await reload();
    selectedId.value = r.id;
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
    await deleteRole(selectedRole.value.id);
    selectedId.value = null;
    await reload();
    notify.success("角色已删除");
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "删除失败");
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.split { display: grid; grid-template-columns: 300px 1fr; gap: 16px; align-items: start; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.list-pane { padding: 12px; }
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 8px; }
.filters { display: grid; grid-template-columns: 1fr 94px; gap: 8px; padding: 0 4px 10px; }
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
.tag-builtin, .status-badge { font-size: 10px; font-weight: 700; padding: 1px 5px; border-radius: 3px; }
.tag-builtin { background: var(--purple-soft); color: var(--purple); }
.status-badge.ok { background: var(--success-soft); color: var(--success); }
.status-badge.off { background: var(--surface-2); color: var(--text-3); }
.muted { color: var(--text-4); font-size: 12.5px; padding: 8px; }

.editor-pane { padding: 20px 22px; min-height: 360px; }
.danger-zone { margin-top: 20px; padding-top: 16px; border-top: 1px solid var(--border-soft); }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger, .danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }

.modal-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.45); display: grid; place-items: center; z-index: 100; backdrop-filter: blur(3px); }
.modal { background: var(--surface); border-radius: 12px; padding: 22px; width: min(560px, 92vw); box-shadow: var(--sh-4); display: flex; flex-direction: column; gap: 12px; }
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-grid { display: grid; grid-template-columns: repeat(2, minmax(0, 1fr)); gap: 12px; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); min-width: 0; }
.input-sm { font-size: 12px; padding: 6px 8px; }
.textarea { resize: vertical; min-height: 68px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input:disabled { background: var(--surface-2); color: var(--text-3); }
.form-error { font-size: 12.5px; color: var(--danger); background: var(--danger-soft); padding: 8px 10px; border-radius: 6px; }
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
@media (max-width: 860px) {
  .split { grid-template-columns: 1fr; }
  .modal-grid { grid-template-columns: 1fr; }
}
</style>
