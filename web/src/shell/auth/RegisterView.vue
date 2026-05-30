<template>
  <div class="login-page">
    <div class="login-aside">
      <div class="aside-mark">I</div>
      <h1 class="aside-title">一体化办公平台</h1>
      <p class="aside-tagline">高效协同 · 创造价值</p>
      <div class="aside-foot">v3.1 · {{ year }}</div>
    </div>

    <div class="login-form-wrap">
      <!-- Submitted state -->
      <div v-if="submitted" class="card login-form done-card">
        <div class="done-icon">
          <svg width="44" height="44" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.2"
               stroke-linecap="round" stroke-linejoin="round">
            <circle cx="12" cy="12" r="10" />
            <path d="m9 12 2 2 4-4" />
          </svg>
        </div>
        <h2 class="done-title">申请已提交</h2>
        <p class="done-sub">您的账号申请正在等待管理员审批，<br/>审批通过后您将可以登录。</p>
        <div class="done-meta">
          <div class="meta-row"><span class="lbl">用户名</span><span class="val">{{ formCache.username }}</span></div>
          <div class="meta-row"><span class="lbl">真实姓名</span><span class="val">{{ formCache.real_name }}</span></div>
          <div class="meta-row"><span class="lbl">所在单位</span><span class="val">{{ formCache.organization }}</span></div>
          <div v-if="formCache.phone" class="meta-row"><span class="lbl">手机号</span><span class="val">{{ formCache.phone }}</span></div>
          <div class="meta-row"><span class="lbl">申请编号</span><code class="val">{{ submittedId.slice(0, 8) }}…</code></div>
        </div>
        <router-link to="/login" class="btn btn-primary btn-block">返回登录</router-link>
      </div>

      <!-- Form state -->
      <form v-else class="card login-form" @submit.prevent="submit">
        <div class="form-head">
          <h2>申请账号</h2>
          <p class="sub">提交后由管理员审批，通过即可登录</p>
        </div>

        <label class="field">
          <span class="label">用户名</span>
          <input class="input" v-model="username" type="text" required autofocus autocomplete="username"
                 placeholder="3-32 位字母/数字/-/_，以字母开头" />
        </label>
        <label class="field">
          <span class="label">真实姓名</span>
          <input class="input" v-model="realName" type="text" required maxlength="32"
                 placeholder="请填写真实姓名" />
        </label>
        <label class="field">
          <span class="label">所在单位</span>
          <select v-model="organizationId" class="input" required :disabled="orgsLoading || orgs.length === 0">
            <option value="" disabled>
              {{ orgsLoading ? '加载中…' : (orgs.length === 0 ? '暂无可选单位，请联系管理员添加' : '请选择') }}
            </option>
            <option v-for="o in orgs" :key="o.id" :value="o.id">{{ o.name }}</option>
          </select>
        </label>
        <label class="field">
          <span class="label">手机号 <span class="optional">（可选）</span></span>
          <input class="input" v-model="phone" type="tel" maxlength="11" autocomplete="tel"
                 inputmode="numeric" pattern="^1[3-9]\d{9}$"
                 placeholder="11 位手机号" />
          <span class="field-hint">短信验证码功能即将上线，当前可留空</span>
        </label>
        <label class="field">
          <span class="label">密码</span>
          <PasswordField
            v-model="password"
            :minlength="10"
            required
            autocomplete="new-password"
            placeholder="至少 10 位，含字母与数字"
          />
        </label>
        <label class="field">
          <span class="label">确认密码</span>
          <PasswordField
            v-model="password2"
            required
            autocomplete="new-password"
            placeholder="再次输入密码"
          />
        </label>

        <div v-if="error" class="form-error">
          <span class="badge badge-danger badge-dot">提交失败</span>
          <span>{{ error }}</span>
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" type="submit">
          {{ loading ? "提交中…" : "提交申请" }}
        </button>

        <div class="form-foot">
          <span class="muted">已有账号？</span>
          <router-link to="/login" class="link">返回登录</router-link>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from "vue";
import { client } from "@/api/client";
import PasswordField from "./PasswordField.vue";

interface Org { id: string; name: string; slug: string }
const orgs = ref<Org[]>([]);
const orgsLoading = ref(true);

const username = ref("");
const realName = ref("");
const organizationId = ref("");
const phone = ref("");
const password = ref("");
const password2 = ref("");
const loading = ref(false);
const error = ref("");
const submitted = ref(false);
const submittedId = ref("");
const formCache = reactive({ username: "", real_name: "", organization: "", phone: "" });
const year = new Date().getFullYear();

onMounted(async () => {
  try {
    const r = await client.get("/public/organizations");
    orgs.value = r.data?.data?.organizations ?? [];
  } catch {} finally { orgsLoading.value = false; }
});

async function submit() {
  error.value = "";
  if (password.value !== password2.value) {
    error.value = "两次密码不一致";
    return;
  }
  const trimmedPhone = phone.value.trim();
  if (trimmedPhone && !/^1[3-9]\d{9}$/.test(trimmedPhone)) {
    error.value = "手机号格式错误";
    return;
  }
  if (!organizationId.value) {
    error.value = "请选择所在单位";
    return;
  }
  loading.value = true;
  try {
    const res = await client.post("/auth/register", {
      username: username.value.trim(),
      real_name: realName.value.trim(),
      organization_id: organizationId.value,
      phone: trimmedPhone || undefined,
      password: password.value,
    });
    const data = res.data?.data ?? res.data;
    submittedId.value = data.application_id ?? "";
    const orgName = orgs.value.find((o) => o.id === organizationId.value)?.name ?? "";
    Object.assign(formCache, {
      username: username.value.trim(),
      real_name: realName.value.trim(),
      organization: orgName,
      phone: trimmedPhone,
    });
    submitted.value = true;
  } catch (e: any) {
    error.value = e.response?.data?.error?.message ?? "提交失败";
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  display: grid;
  grid-template-columns: 480px 1fr;
  background: var(--bg);
}

.login-aside {
  background-image: url("/login-bg.png");
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  color: #0d1b2e;
  padding: var(--sp-10) var(--sp-8) var(--sp-8);
  display: flex;
  flex-direction: column;
  gap: var(--sp-5);
  position: relative;
  overflow: hidden;
}
.login-aside::before {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(180deg, rgba(255,255,255,.35) 0%, rgba(255,255,255,0) 40%);
  pointer-events: none;
}
.login-aside > * { position: relative; z-index: 1; }

.aside-mark {
  width: 56px; height: 56px;
  border-radius: var(--r-md);
  background: linear-gradient(135deg, var(--primary) 0%, #4a7ce8 100%);
  color: #fff;
  font-size: 28px;
  font-weight: 800;
  display: flex; align-items: center; justify-content: center;
  letter-spacing: -0.04em;
  box-shadow: 0 6px 18px rgba(30,95,217,.32);
}
.aside-title {
  font-size: 30px;
  font-weight: 700;
  letter-spacing: 0.02em;
  line-height: 1.2;
  color: #0d1b2e;
  margin: 0;
}
.aside-tagline {
  font-size: 15px;
  font-weight: 500;
  letter-spacing: 0.04em;
  color: rgba(13, 27, 46, 0.62);
  margin-top: -4px;
}
.aside-foot {
  margin-top: auto;
  font-size: 12px;
  color: rgba(13, 27, 46, 0.45);
  font-family: var(--ff-mono);
}

.login-form-wrap {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--sp-8);
}
.login-form {
  width: 100%;
  max-width: 420px;
  padding: var(--sp-8);
  box-shadow: var(--sh-3);
  display: flex;
  flex-direction: column;
  gap: var(--sp-5);
}
.form-head h2 { font-size: 24px; font-weight: 700; letter-spacing: -0.01em; }
.form-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.field { display: flex; flex-direction: column; }
.optional { color: var(--text-4); font-weight: 400; }
.field-hint {
  margin-top: 4px;
  font-size: 11.5px;
  color: var(--text-3);
}
.btn-block { width: 100%; justify-content: center; padding: 11px; font-size: 14px; font-weight: 600; }
.form-error {
  display: flex;
  gap: var(--sp-2);
  align-items: center;
  padding: var(--sp-3) var(--sp-4);
  background: var(--danger-soft);
  border-radius: var(--r-sm);
  font-size: 13px;
  color: var(--danger);
}
.form-foot {
  font-size: 13px;
  color: var(--text-3);
  text-align: center;
}
.muted { color: var(--text-3); }
.link {
  color: var(--primary);
  text-decoration: none;
  font-weight: 500;
}
.link:hover { text-decoration: underline; }

/* Submitted state */
.done-card {
  text-align: center;
  gap: var(--sp-4);
  align-items: center;
}
.done-icon {
  width: 64px; height: 64px;
  border-radius: 50%;
  background: var(--success-soft);
  color: var(--success);
  display: grid; place-items: center;
  margin: 4px auto 0;
}
.done-title {
  font-size: 20px;
  font-weight: 700;
  letter-spacing: -.01em;
  color: var(--text);
}
.done-sub {
  font-size: 13px;
  color: var(--text-2);
  line-height: 1.6;
}
.done-meta {
  width: 100%;
  background: var(--surface-2);
  border-radius: var(--r-md);
  padding: var(--sp-4);
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.meta-row {
  display: flex;
  justify-content: space-between;
  font-size: 12.5px;
}
.meta-row .lbl { color: var(--text-3); }
.meta-row .val {
  color: var(--text);
  font-weight: 500;
  font-family: inherit;
}
.meta-row code.val {
  font-family: var(--ff-mono);
  font-size: 11.5px;
}

@media (max-width: 900px) {
  .login-page { grid-template-columns: 1fr; }
  .login-aside { padding: var(--sp-7); min-height: 220px; }
}
</style>
