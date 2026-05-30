<template>
  <section>
    <header class="page-head">
      <div>
        <h1>角色权限</h1>
        <p class="sub">3 个内置角色 + 自定义角色 · 策略支持 resource:action 通配</p>
      </div>
      <button class="btn btn-primary" @click="showCreate = !showCreate">+ 新建角色</button>
    </header>

    <form v-if="showCreate" class="card create-form" @submit.prevent="create">
      <div class="form-row">
        <label class="field">
          <span class="label">编码 (唯一)</span>
          <input class="input" v-model="newRole.code" required pattern="[a-z_]+" placeholder="例如：editor" />
        </label>
        <label class="field">
          <span class="label">显示名称</span>
          <input class="input" v-model="newRole.name" required placeholder="例如：编辑者" />
        </label>
        <button class="btn btn-primary" :disabled="saving">创建</button>
        <button class="btn" type="button" @click="showCreate = false">取消</button>
      </div>
    </form>

    <div class="roles-grid">
      <article v-for="r in roles" :key="r.id" class="role-card">
        <div class="role-head">
          <div class="role-info">
            <div class="role-title">
              {{ r.name }}
              <span class="role-code">{{ r.code }}</span>
              <span v-if="r.built_in" class="tag-builtin">内置</span>
            </div>
            <div class="role-meta">{{ r.member_count }} 名成员</div>
          </div>
          <button v-if="!r.built_in" class="link-btn warn" @click="del(r.id)">删除</button>
        </div>
        <div class="policies">
          <span v-if="!r.policies?.length" class="no-pol">{{ r.code === 'platform_admin' || r.code === 'tenant_admin' ? '★ 内置超级权限' : '尚无策略' }}</span>
          <span v-for="p in (r.policies ?? [])" :key="`${p.resource}:${p.action}`" class="pol-chip">
            {{ p.resource }}<span class="sep">:</span>{{ p.action }}
            <button class="rm" @click="removePol(r.id, p.resource, p.action)">×</button>
          </span>
        </div>
        <div v-if="!r.built_in" class="add-pol-row">
          <select class="input input-sm" v-model="newPol[r.id].resource">
            <option value="">选择资源</option>
            <option v-for="res in Object.keys(perms.by_resource)" :key="res" :value="res">{{ res }}</option>
          </select>
          <select class="input input-sm" v-model="newPol[r.id].action" :disabled="!newPol[r.id].resource">
            <option value="">选择动作</option>
            <option v-for="p in (perms.by_resource[newPol[r.id].resource] ?? [])" :key="p.action" :value="p.action">
              {{ p.action }} · {{ p.label }}
            </option>
            <option value="*">★ * (全部)</option>
          </select>
          <button class="btn btn-sm" @click="addPol(r.id)" :disabled="!newPol[r.id].resource || !newPol[r.id].action">+ 加策略</button>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from "vue";
import { listRoles, createRole, deleteRole, addPolicy, removePolicy, type Role } from "../api/admin";
import { getPermissionRegistry, type PermissionRegistry } from "@/shell/appcenter/appstore";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();

const roles = ref<Role[]>([]);
const perms = ref<PermissionRegistry>({ permissions: [], by_resource: {} });
const showCreate = ref(false);
const saving = ref(false);
const newRole = reactive({ code: "", name: "" });
const newPol = reactive<Record<string, { resource: string; action: string }>>({});

onMounted(async () => {
  await reload();
  try { perms.value = await getPermissionRegistry(); } catch {}
});
watch(roles, (rs) => rs.forEach((r) => (newPol[r.id] ??= { resource: "", action: "" })));

async function reload() { roles.value = await listRoles(); }
async function create() {
  saving.value = true;
  try {
    await createRole(newRole.code, newRole.name);
    newRole.code = ""; newRole.name = ""; showCreate.value = false;
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "创建失败"); }
  finally { saving.value = false; }
}
async function del(id: string) {
  if (!(await confirm({ title: "确认", message: "确定删除该角色？", danger: true }))) return;
  try { await deleteRole(id); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "删除失败"); }
}
async function addPol(roleId: string) {
  const p = newPol[roleId];
  if (!p.resource || !p.action) return;
  try {
    await addPolicy(roleId, p.resource, p.action);
    p.resource = ""; p.action = "";
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "添加失败"); }
}
async function removePol(roleId: string, resource: string, action: string) {
  try { await removePolicy(roleId, resource, action); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "移除失败"); }
}
</script>

<style scoped>
.page-head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-sm { padding: 5px 10px; font-size: 12px; }

.create-form { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 16px; margin-bottom: 16px; }
.form-row { display: grid; grid-template-columns: 1fr 2fr auto auto; gap: 10px; align-items: end; }
.field { display: flex; flex-direction: column; gap: 4px; }
.label { font-size: 12px; font-weight: 500; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; }
.input-sm { padding: 5px 9px; font-size: 12px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }

.roles-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(380px, 1fr)); gap: 14px; }
.role-card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px 18px;
}
.role-head { display: flex; justify-content: space-between; align-items: flex-start; }
.role-title { font-size: 15px; font-weight: 600; display: flex; align-items: center; gap: 8px; }
.role-code { font-family: var(--ff-mono); font-size: 11.5px; color: var(--text-3); padding: 2px 7px; background: var(--surface-2); border-radius: 4px; }
.tag-builtin { font-size: 10px; font-weight: 700; padding: 1px 5px; background: var(--purple-soft); color: var(--purple); border-radius: 3px; }
.role-meta { font-size: 11.5px; color: var(--text-3); margin-top: 4px; }

.policies {
  margin-top: 14px;
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  min-height: 36px;
  padding: 10px;
  background: var(--surface-2);
  border-radius: 8px;
}
.no-pol { color: var(--text-3); font-size: 12.5px; font-style: italic; }
.pol-chip {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 3px 5px 3px 9px;
  background: var(--primary-soft);
  color: var(--primary);
  border-radius: 5px;
  font-family: var(--ff-mono);
  font-size: 11.5px;
  font-weight: 600;
}
.pol-chip .sep { color: rgba(30, 95, 217, .5); }
.pol-chip .rm {
  background: transparent;
  border: 0;
  color: var(--primary);
  font-size: 13px;
  cursor: pointer;
  padding: 0 3px;
  border-radius: 3px;
}
.pol-chip .rm:hover { background: rgba(30, 95, 217, .15); }

.add-pol-row {
  margin-top: 10px;
  display: grid;
  grid-template-columns: 1fr 1fr auto;
  gap: 6px;
}

.link-btn { background: transparent; border: 0; font-size: 12.5px; color: var(--primary); cursor: pointer; padding: 4px 8px; border-radius: 4px; }
.link-btn.warn { color: var(--danger); }
.link-btn.warn:hover { background: var(--danger-soft); }
</style>
