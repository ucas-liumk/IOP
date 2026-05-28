<template>
  <div class="login-page">
    <div class="login-aside">
      <div class="aside-mark">I</div>
      <h1 class="aside-title">
        <span class="gradient-text">IOP</span>
      </h1>
      <p class="aside-tagline">企业内部协同办公平台</p>
      <ul class="aside-features">
        <li><span class="dot"></span>多租户 PG schema 隔离</li>
        <li><span class="dot"></span>OKR 四级计划 + 日 / 周报</li>
        <li><span class="dot"></span>跨部门工作汇总</li>
        <li><span class="dot"></span>限流 / 幂等 / 慢查询防护</li>
      </ul>
      <div class="aside-foot">v3.1 · {{ year }}</div>
    </div>

    <div class="login-form-wrap">
      <form class="card login-form" @submit.prevent="submit">
        <div class="form-head">
          <h2>欢迎回来</h2>
          <p class="sub">使用工作邮箱登录</p>
        </div>

        <label class="field">
          <span class="label">邮箱</span>
          <input class="input" v-model="email" type="email" required autofocus placeholder="you@company.com" />
        </label>
        <label class="field">
          <span class="label">密码</span>
          <input class="input" v-model="password" type="password" required minlength="10" placeholder="至少 10 位，含字母与数字" />
        </label>

        <div v-if="error" class="form-error">
          <span class="badge badge-danger badge-dot">登录失败</span>
          <span>{{ error }}</span>
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" type="submit">
          {{ loading ? "登录中…" : "登录" }}
        </button>

        <div class="form-foot">
          <span class="muted">首次使用？</span>
          联系管理员通过 <code>tenantctl</code> 创建账号
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "./auth.store";

const auth = useAuthStore();
const router = useRouter();
const email = ref("");
const password = ref("");
const loading = ref(false);
const error = ref("");
const year = new Date().getFullYear();

async function submit() {
  loading.value = true;
  error.value = "";
  try {
    await auth.login(email.value, password.value);
    if (auth.tenants.length > 0 && !auth.tenant) {
      await auth.switchTenant(auth.tenants[0].id);
    }
    router.push("/");
  } catch (e: any) {
    error.value = e.response?.data?.error?.message ?? "登录失败";
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
  background: linear-gradient(160deg, #0d1b2e 0%, #1a3066 55%, #1e5fd9 100%);
  color: white;
  padding: var(--sp-10) var(--sp-8) var(--sp-8);
  display: flex;
  flex-direction: column;
  gap: var(--sp-6);
  position: relative;
  overflow: hidden;
}
.login-aside::before {
  content: "";
  position: absolute;
  inset: 0;
  background:
    radial-gradient(circle at 20% 110%, rgba(214, 56, 56, 0.18), transparent 50%),
    radial-gradient(circle at 90% 20%, rgba(124, 77, 219, 0.22), transparent 55%);
  pointer-events: none;
}
.login-aside > * { position: relative; z-index: 1; }

.aside-mark {
  width: 56px;
  height: 56px;
  border-radius: var(--r-md);
  background: linear-gradient(135deg, #ffffff 0%, #dbe6fd 100%);
  color: var(--primary);
  font-size: 28px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  letter-spacing: -0.04em;
  box-shadow: var(--sh-3);
}
.aside-title { font-size: 48px; font-weight: 800; letter-spacing: -0.03em; line-height: 1; }
.gradient-text {
  background: linear-gradient(90deg, #ffffff 0%, #dbe6fd 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
.aside-tagline {
  font-size: 17px;
  font-weight: 500;
  color: rgba(255, 255, 255, 0.85);
  margin-top: -8px;
}
.aside-features {
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: var(--sp-3);
  margin-top: var(--sp-6);
}
.aside-features li {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  font-size: 14px;
  color: rgba(255, 255, 255, 0.8);
}
.aside-features .dot {
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.6);
}
.aside-foot {
  margin-top: auto;
  font-size: 12px;
  color: rgba(255, 255, 255, 0.5);
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
.form-head h2 {
  font-size: 24px;
  font-weight: 700;
  letter-spacing: -0.01em;
}
.form-head .sub {
  font-size: 13px;
  color: var(--text-3);
  margin-top: 4px;
}

.field {
  display: flex;
  flex-direction: column;
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
  font-size: 12px;
  color: var(--text-3);
  text-align: center;
}
.form-foot code {
  background: var(--surface-3);
  padding: 1px 6px;
  border-radius: 4px;
  font-family: var(--ff-mono);
  font-size: 11px;
}
.muted { color: var(--text-3); }

@media (max-width: 900px) {
  .login-page { grid-template-columns: 1fr; }
  .login-aside { padding: var(--sp-7); }
  .aside-features { display: none; }
}
</style>
