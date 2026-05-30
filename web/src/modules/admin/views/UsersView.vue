<template>
  <section class="admin-page">
    <PageHeader title="用户管理" :sub="`平台用户 · 共 ${users.length} 个${search ? ' · 搜索: ' + search : ''}`">
      <template #actions>
        <div class="head-actions">
          <input class="input search" v-model="search" placeholder="搜索用户名 / 手机 / 邮箱" @keyup.enter="refresh" />
          <button class="btn btn-ghost" @click="refresh">刷新</button>
          <button class="btn btn-primary" v-perm="'user:write'" @click="openCreate">+ 新建用户</button>
        </div>
      </template>
    </PageHeader>

    <div v-if="pageError" class="page-error">
      {{ pageError }}
      <button class="page-error-close" @click="pageError = ''">×</button>
    </div>

    <DataTable :columns="columns" :rows="users" rowKey="id">
      <template #cell-account="{ row }">
        <div class="user-cell">
          <div class="u-avatar">{{ initialsOf(row) }}</div>
          <div>
            <div class="u-name">
              <code>{{ row.username || '—' }}</code>
              <span v-if="row.status === 'disabled'" class="tag-disabled">已停用</span>
            </div>
            <div class="u-meta">
              <span v-if="row.phone">📱 {{ row.phone }}</span>
              <span v-if="row.email">✉️ {{ row.email }}</span>
              <span v-if="!row.phone && !row.email" class="muted">无联系方式</span>
            </div>
          </div>
        </div>
      </template>
      <template #cell-created_at="{ row }">
        <span class="time">{{ formatTime(row.created_at) }}</span>
      </template>
      <template #cell-last_login_at="{ row }">
        <span v-if="row.last_login_at" class="time">{{ formatTime(row.last_login_at) }}</span>
        <span v-else class="muted">从未登录</span>
      </template>
      <template #cell-actions="{ row }">
        <div class="row-actions">
          <button class="btn btn-ghost btn-sm" v-perm="'user:write'" @click="openReset(row)">重置密码</button>
          <button v-if="row.status === 'active'" v-perm="'user:write'" class="btn btn-ghost btn-sm danger" @click="toggleStatus(row)">停用</button>
          <button v-else v-perm="'user:write'" class="btn btn-ghost btn-sm" @click="toggleStatus(row)">启用</button>
        </div>
      </template>
    </DataTable>

    <EmptyState v-if="users.length === 0 && !loading" title="没有匹配的用户" sub="" />

    <!-- Create modal -->
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
            <option v-for="o in orgs" :key="o.id" :value="o.id">{{ o.name }}</option>
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
    <div v-if="resetting" class="modal-overlay" @click.self="closeReset">
      <div class="modal">
        <h3>重置 {{ resetting.username }} 的密码</h3>

        <template v-if="!resetDone">
          <label class="field">
            <span class="label">新密码</span>
            <input class="input" v-model="newPassword" type="text" minlength="10"
                   placeholder="至少 10 位，含字母与数字" />
            <button type="button" class="btn-link" @click="newPassword = randomPassword()">生成强密码</button>
          </label>
          <div v-if="actionError" class="form-error">{{ actionError }}</div>
          <div class="modal-actions">
            <button class="btn btn-ghost" @click="closeReset">取消</button>
            <button class="btn btn-primary" :disabled="busy" @click="confirmReset">
              {{ busy ? '处理中…' : '确认重置' }}
            </button>
          </div>
        </template>

        <template v-else>
          <p class="modal-sub">密码已重置成功，请妥善记录并通知用户（关闭后将无法再次查看）。</p>
          <div class="pw-reveal">
            <code class="pw-text">{{ newPassword }}</code>
            <button type="button" class="btn btn-ghost btn-sm" @click="copyNewPassword">复制</button>
          </div>
          <div class="modal-actions">
            <button class="btn btn-primary" @click="closeReset">完成</button>
          </div>
        </template>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader, DataTable, EmptyState, type Column } from "@/shell/components";
import { useNotification } from "@/shell/notify";
import { useConfirm } from "@/shell/confirm";

const notify = useNotification();
const { confirm } = useConfirm();
import {
  listPlatformUsers, createPlatformUser,
  disablePlatformUser, enablePlatformUser, resetPlatformUserPassword,
  listAllTenants, type PlatformTenant,
  type PlatformUser,
} from "../api/admin";

const users = ref<PlatformUser[]>([]);
const orgs = ref<PlatformTenant[]>([]);
const search = ref("");
const loading = ref(false);
const busy = ref(false);
const actionError = ref("");
const pageError = ref(""); // row-level action errors (shown as a banner)

const columns: Column[] = [
  { key: "account",       label: "账号",       width: 320 },
  { key: "created_at",    label: "创建时间",   width: 160 },
  { key: "last_login_at", label: "最近登录",   width: 160 },
  { key: "actions",       label: "操作",       width: 220, align: "right" },
];

async function refresh() {
  loading.value = true;
  try {
    users.value = await listPlatformUsers(search.value.trim());
  } finally { loading.value = false; }
}

onMounted(async () => {
  try { orgs.value = await listAllTenants(); } catch {}
  await refresh();
});

// === Create modal ===
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
  form.organization_id = orgs.value[0]?.id ?? "";
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
    await refresh();
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "创建失败";
  } finally { busy.value = false; }
}

// === Status toggle ===
async function toggleStatus(u: PlatformUser) {
  pageError.value = "";
  try {
    if (u.status === "active") {
      if (!(await confirm({ title: "确认", message: `确认停用 ${u.username}？停用后该账号无法登录。`, danger: true }))) return;
      await disablePlatformUser(u.id);
    } else {
      await enablePlatformUser(u.id);
    }
    await refresh();
  } catch (e: any) {
    pageError.value = e.response?.data?.error?.message ?? "操作失败";
  }
}

// === Reset password ===
const resetting = ref<PlatformUser | null>(null);
const newPassword = ref("");

function openReset(u: PlatformUser) {
  resetting.value = u;
  newPassword.value = randomPassword();
  resetDone.value = false;
  actionError.value = "";
}

const resetDone = ref(false); // keeps the new password visible after a reset

async function confirmReset() {
  if (!resetting.value) return;
  busy.value = true; actionError.value = "";
  try {
    await resetPlatformUserPassword(resetting.value.id, newPassword.value);
    // Keep the password visible (admin must relay it) and also surface a toast.
    resetDone.value = true;
    notify.success(`密码已重置为：${newPassword.value}（请妥善记录并通知用户）`, 8000);
  } catch (e: any) {
    actionError.value = e.response?.data?.error?.message ?? "重置失败";
  } finally { busy.value = false; }
}

async function copyNewPassword() {
  try {
    await navigator.clipboard.writeText(newPassword.value);
    notify.success("已复制到剪贴板");
  } catch {
    notify.error("复制失败，请手动选择密码文本");
  }
}

function closeReset() {
  resetting.value = null;
  resetDone.value = false;
  newPassword.value = "";
}

// === Helpers ===
function initialsOf(u: PlatformUser): string {
  const s = u.username || u.email || u.phone || "?";
  return s.slice(0, 2).toUpperCase();
}
function formatTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return iso;
  const m = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${m(d.getMonth()+1)}-${m(d.getDate())} ${m(d.getHours())}:${m(d.getMinutes())}`;
}
// Cryptographically-strong password generator. Uses crypto.getRandomValues with
// rejection sampling (no modulo bias) and a crypto Fisher-Yates shuffle — these are
// real account credentials, so the non-crypto PRNG must not be used.
function randPick(set: string): string {
  const max = 256 - (256 % set.length); // reject the biased tail
  const buf = new Uint8Array(1);
  for (;;) {
    crypto.getRandomValues(buf);
    if (buf[0] < max) return set[buf[0] % set.length];
  }
}
function randomPassword(): string {
  const letters = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ"; // no l/I/O/0 confusables
  const digits = "23456789";
  const symbols = "!@#$%";
  const all = letters + digits + symbols;
  const chars: string[] = [
    randPick(letters), // guarantee ≥1 letter
    randPick(digits),  // guarantee ≥1 digit
    randPick(symbols),
  ];
  for (let i = 0; i < 9; i++) chars.push(randPick(all)); // 12 chars total
  // Fisher-Yates with crypto-drawn indices
  for (let i = chars.length - 1; i > 0; i--) {
    const j = cryptoIndex(i + 1);
    [chars[i], chars[j]] = [chars[j], chars[i]];
  }
  return chars.join("");
}
function cryptoIndex(n: number): number {
  const max = 256 - (256 % n);
  const buf = new Uint8Array(1);
  for (;;) {
    crypto.getRandomValues(buf);
    if (buf[0] < max) return buf[0] % n;
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.head-actions { display: flex; align-items: center; gap: 8px; }
.page-error {
  display: flex; align-items: center; justify-content: space-between;
  background: var(--danger-soft); color: var(--danger);
  font-size: 13px; padding: 10px 14px; border-radius: 8px;
}
.page-error-close {
  border: 0; background: transparent; color: inherit;
  font-size: 18px; line-height: 1; cursor: pointer;
}
.search {
  width: 240px;
  font-size: 13px;
  padding: 6px 10px;
}

.user-cell { display: flex; align-items: center; gap: 10px; }
.u-avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary), #5b8bf5);
  color: #fff;
  display: grid; place-items: center;
  font-size: 12px;
  font-weight: 700;
}
.u-name {
  display: flex; align-items: center; gap: 8px;
  font-size: 13.5px;
  color: var(--text);
}
.u-name code {
  background: var(--surface-2);
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--ff-mono);
  font-size: 12px;
}
.u-meta {
  font-size: 11.5px;
  color: var(--text-3);
  display: flex; gap: 10px;
  margin-top: 2px;
}
.muted { color: var(--text-4); }
.time { font-size: 12.5px; color: var(--text-2); font-family: var(--ff-mono); }
.tag-disabled {
  font-size: 10.5px;
  background: var(--danger-soft);
  color: var(--danger);
  padding: 1px 6px;
  border-radius: 4px;
  font-weight: 600;
}
.row-actions { display: flex; gap: 4px; justify-content: flex-end; }
.btn-sm { padding: 4px 10px; font-size: 12px; }
.btn-sm.danger { color: var(--danger); }
.btn-sm.danger:hover { background: var(--danger-soft); }
.btn-link {
  border: 0;
  background: none;
  color: var(--primary);
  font-size: 11.5px;
  cursor: pointer;
  text-align: left;
  padding: 4px 0 0;
}

.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(13, 27, 46, .45);
  display: grid; place-items: center; z-index: 100;
  backdrop-filter: blur(3px);
}
.modal {
  background: var(--surface);
  border-radius: 14px;
  padding: 22px;
  width: min(440px, 92vw);
  box-shadow: var(--sh-4);
  display: flex; flex-direction: column; gap: 12px;
}
.modal h3 { font-size: 16px; font-weight: 600; margin: 0; }
.modal-sub { font-size: 12.5px; color: var(--text-3); margin: -6px 0 4px; }
.field { display: flex; flex-direction: column; gap: 4px; }
.field .label { font-size: 12px; color: var(--text-2); }
.optional { color: var(--text-4); font-weight: 400; }
.form-error {
  font-size: 12.5px;
  color: var(--danger);
  background: var(--danger-soft);
  padding: 8px 10px;
  border-radius: 6px;
}
.modal-actions { display: flex; gap: 8px; justify-content: flex-end; margin-top: 4px; }
.pw-reveal {
  display: flex; align-items: center; gap: 8px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: var(--r-sm);
  padding: 10px 12px;
}
.pw-text {
  flex: 1;
  font-family: var(--ff-mono);
  font-size: 15px;
  letter-spacing: 1px;
  color: var(--text);
  word-break: break-all;
  user-select: all;
}
</style>
