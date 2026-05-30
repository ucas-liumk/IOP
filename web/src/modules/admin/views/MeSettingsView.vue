<template>
  <section>
    <header class="page-head">
      <h1>个人设置</h1>
      <p class="sub">{{ auth.user?.email }} · 上次登录 {{ lastLogin || '—' }}</p>
    </header>

    <div class="layout">
      <article class="card">
        <header class="card-head"><span class="card-title">修改密码</span></header>
        <form class="form-body" @submit.prevent="changePw">
          <label class="field">
            <span class="label">当前密码</span>
            <input class="input" v-model="pw.old" type="password" required />
          </label>
          <label class="field">
            <span class="label">新密码</span>
            <input class="input" v-model="pw.new" type="password" required minlength="10" />
            <span class="hint">至少 10 位，包含字母和数字</span>
          </label>
          <label class="field">
            <span class="label">确认新密码</span>
            <input class="input" v-model="pw.confirm" type="password" required minlength="10" />
            <span v-if="pw.confirm && pw.confirm !== pw.new" class="hint error">两次输入不一致</span>
          </label>
          <div class="form-actions">
            <button class="btn btn-primary" :disabled="!canSubmitPw || pwSaving">{{ pwSaving ? '提交中…' : '保存密码' }}</button>
            <span v-if="pwSaved" class="ok-tag">✓ 密码已修改</span>
          </div>
        </form>
      </article>

      <article class="card">
        <header class="card-head">
          <span class="card-title">登录会话</span>
          <span class="meta">显示最近 20 次登录</span>
        </header>
        <ul class="session-list">
          <li v-for="s in sessions" :key="s.id" :class="{ current: s.current, revoked: s.revoked }">
            <div class="s-icon">
              <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="2" y="4" width="20" height="16" rx="2"/><line x1="2" y1="10" x2="22" y2="10"/></svg>
            </div>
            <div class="s-info">
              <div class="s-title">
                {{ briefUA(s.user_agent) }}
                <span v-if="s.current" class="tag current-tag">当前</span>
                <span v-if="s.revoked" class="tag revoked-tag">已注销</span>
              </div>
              <div class="s-meta">{{ s.ip_address || '—' }} · 登录于 {{ s.issued_at?.slice(5,16) }} · 过期于 {{ s.expires_at?.slice(5,16) }}</div>
            </div>
            <button v-if="!s.revoked && !s.current" class="link-btn warn" @click="revoke(s.id)">注销</button>
          </li>
          <li v-if="sessions.length === 0" class="empty">暂无会话记录</li>
        </ul>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import { changePassword, listSessions, revokeSession, type Session } from "../api/admin";

const auth = useAuthStore();
const pw = reactive({ old: "", new: "", confirm: "" });
const pwSaving = ref(false);
const pwSaved = ref(false);
const sessions = ref<Session[]>([]);
const lastLogin = ref("");

const canSubmitPw = computed(() => pw.old && pw.new && pw.new === pw.confirm && pw.new.length >= 10);

onMounted(reload);
async function reload() {
  sessions.value = await listSessions();
  const current = sessions.value.find((s) => s.current);
  lastLogin.value = current?.issued_at ?? sessions.value[0]?.issued_at ?? "";
}

async function changePw() {
  pwSaving.value = true;
  pwSaved.value = false;
  try {
    await changePassword(pw.old, pw.new);
    pw.old = ""; pw.new = ""; pw.confirm = "";
    pwSaved.value = true;
    setTimeout(() => (pwSaved.value = false), 2400);
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "改密码失败"); }
  finally { pwSaving.value = false; }
}

async function revoke(id: string) {
  if (!confirm("确定注销该会话？")) return;
  await revokeSession(id);
  await reload();
}

function briefUA(ua: string) {
  if (!ua) return "未知设备";
  if (ua.includes("Macintosh") || ua.includes("Mac OS")) return "Mac · " + (ua.match(/Chrome|Safari|Firefox/)?.[0] ?? "浏览器");
  if (ua.includes("Windows")) return "Windows · " + (ua.match(/Chrome|Edge|Firefox/)?.[0] ?? "浏览器");
  if (ua.includes("iPhone")) return "iPhone Safari";
  if (ua.includes("Android")) return "Android";
  if (ua.includes("curl")) return "curl (CLI)";
  return ua.slice(0, 40);
}
</script>

<style scoped>
.page-head { margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }

.layout { display: grid; grid-template-columns: 1fr 1.4fr; gap: 14px; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.card-head { display: flex; justify-content: space-between; align-items: center; padding: 14px 18px; border-bottom: 1px solid var(--border); }
.card-title { font-size: 14px; font-weight: 600; }
.meta { font-size: 11.5px; color: var(--text-3); }

.form-body { padding: 18px; display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 5px; }
.label { font-size: 12.5px; font-weight: 500; color: var(--text-2); }
.input { padding: 8px 11px; border: 1px solid var(--border-strong); border-radius: 7px; font-size: 13px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.hint { font-size: 11.5px; color: var(--text-3); }
.hint.error { color: var(--danger); }
.form-actions { display: flex; align-items: center; gap: 10px; margin-top: 6px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-primary:disabled { opacity: 0.5; cursor: not-allowed; }
.ok-tag { color: var(--success); font-size: 12.5px; font-weight: 600; }

.session-list { list-style: none; padding: 8px; max-height: 460px; overflow-y: auto; }
.session-list li {
  display: flex; gap: 12px; align-items: center;
  padding: 10px 12px;
  border-radius: 8px;
  border-bottom: 1px solid var(--border-soft);
}
.session-list li:last-child { border-bottom: 0; }
.session-list li.revoked { opacity: 0.55; }
.session-list li.empty {
  text-align: center; color: var(--text-4); padding: 40px 0;
  border-bottom: 0;
  display: block;
}
.s-icon {
  width: 32px; height: 32px;
  background: var(--primary-soft);
  color: var(--primary);
  border-radius: 7px;
  display: grid; place-items: center;
  flex-shrink: 0;
}
.s-info { flex: 1; min-width: 0; }
.s-title { font-size: 13px; font-weight: 600; display: flex; gap: 6px; align-items: center; }
.s-meta { font-size: 11.5px; color: var(--text-3); margin-top: 2px; }
.tag { font-size: 10px; font-weight: 700; padding: 1px 5px; border-radius: 3px; letter-spacing: .3px; }
.current-tag { background: var(--success-soft); color: var(--success); }
.revoked-tag { background: var(--bg-deep); color: var(--text-3); }
.link-btn { background: transparent; border: 0; font-size: 12.5px; cursor: pointer; padding: 4px 8px; border-radius: 4px; }
.link-btn.warn { color: var(--warning); }
.link-btn.warn:hover { background: var(--warning-soft); }
</style>
