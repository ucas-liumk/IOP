<template>
  <section class="rbac">
    <div v-if="pageError" class="page-error">{{ pageError }}</div>
    <PageHeader title="权限管理" sub="平台角色 · 用户 → 角色 →（菜单权限 + 数据范围）">
      <template #actions>
        <div class="head-actions">
          <button class="btn btn-primary" @click="showCreate = !showCreate">+ 新建角色</button>
        </div>
      </template>
    </PageHeader>

    <form v-if="showCreate" class="card create-form" @submit.prevent="create">
      <input class="input" v-model="newRole.code" required pattern="[a-z][a-z0-9_-]*" placeholder="编码，如 ops_admin" />
      <input class="input" v-model="newRole.name" required placeholder="显示名称" />
      <button class="btn btn-primary" :disabled="saving">创建</button>
      <button class="btn" type="button" @click="showCreate = false">取消</button>
    </form>

    <div class="roles-grid">
      <article v-for="r in roles" :key="r.id" class="role-card">
        <div class="role-head">
          <div class="role-title">
            {{ r.name }} <span class="role-code">{{ r.code }}</span>
            <span v-if="r.built_in" class="tag-builtin">内置</span>
          </div>
          <button v-if="!r.built_in" class="link-btn warn" @click="del(r.id)">删除</button>
        </div>
        <div class="role-meta">{{ r.member_count }} 名成员</div>
        <div class="policies">
          <span v-if="r.code === 'super_admin'" class="no-pol">★ 全权</span>
          <span v-else-if="!r.policies?.length" class="no-pol">尚无权限</span>
          <span v-for="p in (r.policies ?? [])" :key="`${p.resource}:${p.action}`" class="pol-chip">
            {{ p.resource }}<span class="sep">/</span>{{ p.action }}
            <button v-if="!r.built_in" class="rm" @click="removePol(r.id, p.resource, p.action)">×</button>
          </span>
        </div>
        <div v-if="!r.built_in" class="add-pol-row">
          <select class="input input-sm" v-model="newPol[r.id].key">
            <option value="">添加权限点…</option>
            <optgroup v-for="(list, dom) in byDomain" :key="dom" :label="dom">
              <option v-for="p in list" :key="`${p.resource}/${p.action}`" :value="`${p.resource}/${p.action}`">
                {{ p.label }} ({{ p.resource }}/{{ p.action }}){{ p.is_high_risk ? ' ⚠' : '' }}
              </option>
            </optgroup>
          </select>
          <button class="btn btn-sm" @click="addPol(r.id)" :disabled="!newPol[r.id].key">+ 加</button>
        </div>

        <div class="members">
          <div class="members-head">成员 · {{ r.members?.length || 0 }}</div>
          <div class="member-chips">
            <span v-for="uid in (r.members ?? [])" :key="uid" class="member-chip">
              {{ userName(uid) }}
              <button class="rm" @click="revokeMember(r, uid)">×</button>
            </span>
            <span v-if="!r.members?.length" class="no-pol">尚无成员</span>
          </div>
          <div class="add-member-row">
            <select class="input input-sm" v-model="newMember[r.id]">
              <option value="">添加成员…</option>
              <option v-for="u in assignableUsers(r)" :key="u.id" :value="u.id">{{ u.username }}</option>
            </select>
            <button class="btn btn-sm" @click="addMember(r)" :disabled="!newMember[r.id]">+ 加</button>
          </div>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { PageHeader } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";
import {
  getRbacMe, listPlatformRoles, listPlatformPermissions, createPlatformRole,
  deletePlatformRole, addPlatformPolicy, removePlatformPolicy,
  grantPlatformRole, revokePlatformRole,
  type PlatformRole, type PlatformPermission, type RbacMe,
} from "../api/rbac";
import { listPlatformUsers, type PlatformUser } from "@/modules/admin/api/admin";

const notify = useNotification();
const { confirm } = useConfirm();

const roles = ref<PlatformRole[]>([]);
const perms = ref<PlatformPermission[]>([]);
const me = ref<RbacMe>({ roles: [], permissions: [], is_super_admin: false });
const users = ref<PlatformUser[]>([]);
const showCreate = ref(false);
const saving = ref(false);
const newRole = reactive({ code: "", name: "" });
const newPol = reactive<Record<string, { key: string }>>({});
const newMember = reactive<Record<string, string>>({});
const pageError = ref("");

const byDomain = computed(() => {
  const m: Record<string, PlatformPermission[]> = {};
  for (const p of perms.value) (m[p.domain] ??= []).push(p);
  return m;
});
const userMap = computed(() => Object.fromEntries(users.value.map((u) => [u.id, u.username || u.id])));
function userName(uid: string) { return userMap.value[uid] ?? uid; }
function assignableUsers(r: PlatformRole) {
  const have = new Set(r.members ?? []);
  return users.value.filter((u) => !have.has(u.id));
}

onMounted(reload);

async function reload() {
  try {
    me.value = await getRbacMe();
    roles.value = await listPlatformRoles();
    pageError.value = "";
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "加载失败，请检查权限或重试";
    return;
  }
  roles.value.forEach((r) => { newPol[r.id] ??= { key: "" }; if (!(r.id in newMember)) newMember[r.id] = ""; });
  try { perms.value = await listPlatformPermissions(); } catch {}
  try { users.value = await listPlatformUsers(""); } catch {}
}

async function create() {
  saving.value = true;
  try {
    await createPlatformRole(newRole.code, newRole.name);
    newRole.code = ""; newRole.name = ""; showCreate.value = false;
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "创建失败"); }
  finally { saving.value = false; }
}
async function del(id: string) {
  if (!(await confirm({ title: "确认", message: "确定删除该角色？", danger: true }))) return;
  try { await deletePlatformRole(id); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "删除失败"); }
}
async function addPol(id: string) {
  const key = newPol[id].key;
  if (!key) return;
  const [resource, action] = key.split("/");
  try { await addPlatformPolicy(id, resource, action); newPol[id].key = ""; await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "添加失败"); }
}
async function removePol(id: string, resource: string, action: string) {
  if (!(await confirm({ title: "移除权限", message: "确认移除该权限点？", danger: true }))) return;
  try { await removePlatformPolicy(id, resource, action); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "移除失败"); }
}
async function addMember(r: PlatformRole) {
  const uid = newMember[r.id];
  if (!uid) return;
  try { await grantPlatformRole(r.id, uid); newMember[r.id] = ""; await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "添加成员失败"); }
}
async function revokeMember(r: PlatformRole, uid: string) {
  if (!(await confirm({ title: "移除成员", message: "确认移除该成员？", danger: true }))) return;
  try { await revokePlatformRole(r.id, uid); await reload(); }
  catch (e: any) { notify.error(e.response?.data?.error?.message ?? "移除失败"); }
}
</script>

<style scoped>
.rbac { display: flex; flex-direction: column; gap: var(--sp-5); }
.page-error { background: var(--danger-soft, #fde8e8); color: var(--danger); border: 1px solid var(--danger); border-radius: 8px; padding: 10px 14px; font-size: 13px; }
.head-actions { display: flex; gap: 8px; }
.create-form { display: flex; gap: 10px; align-items: center; padding: 14px 16px; }
.roles-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(340px, 1fr)); gap: 14px; }
.role-card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 16px 18px; box-shadow: var(--sh-1); }
.role-head { display: flex; justify-content: space-between; align-items: center; }
.role-title { font-weight: 600; display: flex; align-items: center; gap: 6px; }
.role-code { font-size: 11px; color: var(--text-3); background: var(--surface-2); border-radius: 4px; padding: 1px 6px; font-family: var(--ff-mono); }
.tag-builtin { font-size: 10px; color: var(--primary); background: var(--primary-soft); border-radius: 3px; padding: 1px 5px; }
.role-meta { font-size: 12px; color: var(--text-3); margin: 4px 0 10px; }
.policies { display: flex; flex-wrap: wrap; gap: 6px; min-height: 24px; }
.pol-chip { font-size: 11.5px; background: var(--surface-2); border: 1px solid var(--border); border-radius: 5px; padding: 2px 6px; display: inline-flex; align-items: center; gap: 3px; }
.pol-chip .sep { color: var(--text-4); }
.pol-chip .rm { border: 0; background: none; color: var(--text-3); cursor: pointer; }
.no-pol { font-size: 12px; color: var(--text-3); }
.add-pol-row { display: flex; gap: 8px; margin-top: 12px; }
.input-sm { font-size: 12px; padding: 4px 8px; }
.link-btn.warn { color: var(--danger); background: none; border: 0; cursor: pointer; font-size: 12px; }
.members { margin-top: 12px; border-top: 1px solid var(--border); padding-top: 10px; }
.members-head { font-size: 11px; color: var(--text-3); margin-bottom: 6px; }
.member-chips { display: flex; flex-wrap: wrap; gap: 6px; min-height: 22px; }
.member-chip { font-size: 11.5px; background: var(--primary-soft); color: var(--primary); border-radius: 5px; padding: 2px 6px; display: inline-flex; align-items: center; gap: 3px; }
.member-chip .rm { border: 0; background: none; color: var(--primary); cursor: pointer; }
.add-member-row { display: flex; gap: 8px; margin-top: 8px; }
</style>
