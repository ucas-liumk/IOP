<template>
  <section>
    <header class="page-head">
      <div>
        <h1>成员管理</h1>
        <p class="sub">共 {{ members.length }} 名成员 · 演示账号支持编辑姓名/部门/职位/电话</p>
      </div>
      <div class="head-actions">
        <input v-model="search" class="input search-input" placeholder="搜索姓名/邮箱/部门" @keyup.enter="reload" />
        <button class="btn btn-primary">+ 邀请成员</button>
      </div>
    </header>

    <article class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 200px">成员</th>
            <th>邮箱</th>
            <th style="width: 110px">部门</th>
            <th style="width: 110px">职位</th>
            <th style="width: 80px">状态</th>
            <th style="width: 120px">加入时间</th>
            <th style="width: 140px"></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="m in members" :key="m.member_id" :class="{ disabled: m.status === 'disabled' }">
            <td>
              <div class="member-cell">
                <div class="avatar" :style="{ background: avatarColor(m.display_name) }">{{ m.display_name[0] }}</div>
                <div>
                  <div class="name">{{ m.display_name }}</div>
                  <div class="email-tiny">{{ m.email }}</div>
                </div>
              </div>
            </td>
            <td><code>{{ m.email }}</code></td>
            <td><span class="chip-dept">{{ m.department || '—' }}</span></td>
            <td class="muted">{{ m.title || '—' }}</td>
            <td>
              <span class="badge" :class="m.status === 'active' ? 'badge-success' : 'badge-neutral'">
                {{ m.status === 'active' ? '正常' : '已禁用' }}
              </span>
            </td>
            <td class="time">{{ m.joined_at.slice(0, 10) }}</td>
            <td class="actions">
              <button class="link-btn" @click="openEdit(m)">编辑</button>
              <button class="link-btn warn" @click="toggleStatus(m)">{{ m.status === 'active' ? '禁用' : '启用' }}</button>
            </td>
          </tr>
          <tr v-if="members.length === 0"><td colspan="7" class="empty">暂无成员</td></tr>
        </tbody>
      </table>
    </article>

    <!-- Edit Modal -->
    <div v-if="editing" class="modal-overlay" @click.self="editing = null">
      <div class="modal">
        <header class="modal-head">
          <h3>编辑成员</h3>
          <button class="close-btn" @click="editing = null">×</button>
        </header>
        <div class="modal-body">
          <label class="field">
            <span class="label">姓名</span>
            <input class="input" v-model="form.display_name" required />
          </label>
          <label class="field">
            <span class="label">部门</span>
            <input class="input" v-model="form.department" placeholder="例如：工程部" />
          </label>
          <label class="field">
            <span class="label">职位</span>
            <input class="input" v-model="form.title" placeholder="例如：高级工程师" />
          </label>
          <label class="field">
            <span class="label">电话</span>
            <input class="input" v-model="form.phone" placeholder="可选" />
          </label>
        </div>
        <footer class="modal-foot">
          <button class="btn" @click="editing = null">取消</button>
          <button class="btn btn-primary" @click="save" :disabled="saving">{{ saving ? "保存中..." : "保存" }}</button>
        </footer>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { listMembers, setMemberDisabled, updateMember, type Member } from "../api/admin";
import { useNotification } from "@/shell/notify";

const notify = useNotification();

const members = ref<Member[]>([]);
const search = ref("");
const editing = ref<Member | null>(null);
const saving = ref(false);
const form = reactive({ display_name: "", department: "", title: "", phone: "" });

onMounted(reload);

async function reload() {
  members.value = await listMembers(search.value);
}

function openEdit(m: Member) {
  editing.value = m;
  form.display_name = m.display_name;
  form.department = m.department;
  form.title = m.title;
  form.phone = m.phone;
}

async function save() {
  if (!editing.value) return;
  saving.value = true;
  try {
    await updateMember(editing.value.member_id, {
      display_name: form.display_name,
      department: form.department,
      title: form.title,
      phone: form.phone,
    });
    editing.value = null;
    await reload();
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "保存失败");
  } finally { saving.value = false; }
}

async function toggleStatus(m: Member) {
  await setMemberDisabled(m.member_id, m.status === "active");
  await reload();
}

function avatarColor(name: string) {
  const seed = name.split("").reduce((s, c) => s + c.charCodeAt(0), 0);
  const palette = [
    "linear-gradient(135deg,#1e5fd9,#4a85ee)",
    "linear-gradient(135deg,#7c4ddb,#5a2db5)",
    "linear-gradient(135deg,#0fa8a3,#0a7e7a)",
    "linear-gradient(135deg,#e8920e,#b86d05)",
    "linear-gradient(135deg,#1aa971,#0e7b51)",
  ];
  return palette[seed % palette.length];
}
</script>

<style scoped>
.page-head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; letter-spacing: -0.01em; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.head-actions { display: flex; gap: 8px; }
.search-input { width: 220px; height: 34px; font-size: 13px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }
.btn:hover { background: var(--bg); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th {
  text-align: left;
  font-size: 11.5px; font-weight: 600;
  color: var(--text-3); text-transform: uppercase; letter-spacing: .5px;
  padding: 11px 16px;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
}
.data-table td {
  padding: 12px 16px;
  font-size: 13px;
  border-bottom: 1px solid var(--border-soft);
  vertical-align: middle;
}
.data-table tbody tr:hover { background: var(--surface-2); }
.data-table tbody tr:last-child td { border-bottom: 0; }
.data-table tr.disabled { opacity: 0.55; }

.member-cell { display: flex; gap: 10px; align-items: center; }
.avatar {
  width: 32px; height: 32px;
  border-radius: 50%;
  color: white; font-weight: 600; font-size: 12px;
  display: grid; place-items: center;
}
.name { font-weight: 600; }
.email-tiny { font-size: 11px; color: var(--text-3); margin-top: 1px; }
.chip-dept {
  display: inline-block;
  padding: 2px 8px;
  background: var(--surface-2);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 11.5px;
}
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 12px; color: var(--text-2); }
.muted { color: var(--text-3); }
.time { color: var(--text-3); font-family: var(--ff-mono); font-size: 12px; }

.badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px; font-weight: 600;
}
.badge-success { background: var(--success-soft); color: var(--success); }
.badge-neutral { background: var(--bg-deep); color: var(--text-3); }

.actions { white-space: nowrap; }
.link-btn {
  background: transparent;
  border: 0;
  font-size: 12.5px;
  color: var(--primary);
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
}
.link-btn:hover { background: var(--primary-soft); }
.link-btn.warn { color: var(--warning); }
.link-btn.warn:hover { background: var(--warning-soft); }

.empty { text-align: center; color: var(--text-4); padding: 30px 0; }

/* Modal */
.modal-overlay {
  position: fixed; inset: 0;
  background: rgba(13,27,46,.42);
  z-index: 500;
  display: flex; align-items: center; justify-content: center;
}
.modal {
  background: var(--surface);
  border-radius: 14px;
  width: 460px;
  max-width: 90vw;
  box-shadow: 0 24px 64px rgba(13,27,46,.28);
  overflow: hidden;
}
.modal-head {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
}
.modal-head h3 { font-size: 15px; font-weight: 600; }
.close-btn {
  background: transparent;
  border: 0;
  font-size: 22px;
  color: var(--text-3);
  cursor: pointer;
  padding: 0 6px;
  border-radius: 4px;
}
.close-btn:hover { background: var(--bg); color: var(--text); }
.modal-body { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 6px; }
.label { font-size: 12px; font-weight: 500; color: var(--text-2); }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.modal-foot {
  padding: 12px 18px;
  border-top: 1px solid var(--border);
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
