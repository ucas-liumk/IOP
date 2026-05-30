<template>
  <section class="admin-page">
    <PageHeader title="用户管理" :sub="`左选组织，右管其成员（按部门树筛选 / 导入导出 / 角色岗位） · 平台共 ${tenants.length} 家组织`">
      <template #actions>
        <div class="head-actions">
          <button class="btn btn-ghost" @click="reload">刷新</button>
          <button class="btn btn-primary" v-perm="'user:write'" @click="openCreate">+ 新建用户</button>
        </div>
      </template>
    </PageHeader>

    <div v-if="actionError && !creating" class="page-error">
      {{ actionError }}
      <button class="page-error-close" @click="actionError = ''">×</button>
    </div>

    <div class="two-pane">
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
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { PageHeader, EmptyState, MemberManager, type MemberApi } from "@/shell/components";
import {
  listAllTenants, createPlatformUser, type PlatformTenant,
} from "../api/admin";
import {
  orgMemberApi, orgMemberTemplateUrl, orgMemberImportUrl,
} from "@/modules/platform/api/orgs";

const tenants = ref<PlatformTenant[]>([]);
const orgFilter = ref("");
const loading = ref(false);
const busy = ref(false);
const actionError = ref("");

// Selected org → drives the right-hand member manager.
const selectedOrgId = ref<string | null>(null);
const selectedOrg = computed(() => tenants.value.find((t) => t.id === selectedOrgId.value) ?? null);
// New adapter per selected org; <MemberManager :key> remounts on org change.
const orgApi = computed<MemberApi | null>(() => (selectedOrg.value ? orgMemberApi(selectedOrg.value.id) : null));

const filteredTenants = computed(() => {
  const q = orgFilter.value.trim().toLowerCase();
  if (!q) return tenants.value;
  return tenants.value.filter((t) => t.name.toLowerCase().includes(q) || t.slug.toLowerCase().includes(q));
});

function selectOrg(t: PlatformTenant) {
  selectedOrgId.value = t.id;
}

onMounted(reload);
async function reload() {
  loading.value = true;
  try {
    tenants.value = await listAllTenants();
    if (selectedOrgId.value && !tenants.value.some((t) => t.id === selectedOrgId.value)) {
      selectedOrgId.value = null;
    }
  } finally { loading.value = false; }
}

// === Create platform user modal ===
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
    // If the new user joined the currently-selected org, remount the manager to refresh.
    if (form.organization_id === selectedOrgId.value) {
      const cur = selectedOrgId.value;
      selectedOrgId.value = null;
      await nextTickReselect(cur);
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
.head-actions { display: flex; gap: 8px; }
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close { border: 0; background: transparent; color: inherit; font-size: 18px; line-height: 1; cursor: pointer; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }

/* two-pane shell: org list | member manager */
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

/* modal */
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
