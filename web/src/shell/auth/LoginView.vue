<template>
  <div class="login-page">
    <div class="login-aside">
      <div class="aside-mark">I</div>
      <h1 class="aside-title">一体化办公平台</h1>
      <p class="aside-tagline">高效协同 · 创造价值</p>
      <div class="aside-foot">v3.1 · {{ year }}</div>
    </div>

    <div class="login-form-wrap">
      <form class="card login-form" @submit.prevent="submit">
        <div class="form-head">
          <h2>欢迎回来</h2>
          <p class="sub">使用用户名登录</p>
        </div>

        <label class="field">
          <span class="label">用户名</span>
          <input class="input" v-model="username" type="text" required autofocus autocomplete="username" placeholder="请输入用户名" />
        </label>
        <label class="field">
          <span class="label">密码</span>
          <PasswordField
            v-model="password"
            :minlength="10"
            required
            autocomplete="current-password"
            placeholder="至少 10 位，含字母与数字"
          />
        </label>

        <div v-if="error" class="form-error">
          <span class="badge badge-danger badge-dot">登录失败</span>
          <span>{{ error }}</span>
        </div>

        <button class="btn btn-primary btn-block" :disabled="loading" type="submit">
          {{ loading ? "登录中…" : "登录" }}
        </button>

        <div class="form-foot">
          <span class="muted">还没有账号？</span>
          <router-link to="/register" class="link">立即注册</router-link>
        </div>
      </form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from "vue";
import { useRouter } from "vue-router";
import { useAuthStore } from "./auth.store";
import { homeForRole } from "./guard";
import PasswordField from "./PasswordField.vue";

const auth = useAuthStore();
const router = useRouter();
const username = ref("");
const password = ref("");
const loading = ref(false);
const error = ref("");
const year = new Date().getFullYear();

async function submit() {
  loading.value = true;
  error.value = "";
  try {
    await auth.login(username.value, password.value);
    if (auth.tenants.length > 0 && !auth.tenant) {
      await auth.switchTenant(auth.tenants[0].id);
    }
    // Route by role: must-change → security page; platform admin → platform console;
    // otherwise the tenant workspace.
    router.push(homeForRole());
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
  width: 56px;
  height: 56px;
  border-radius: var(--r-md);
  background: linear-gradient(135deg, var(--primary) 0%, #4a7ce8 100%);
  color: #fff;
  font-size: 28px;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
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

@media (max-width: 900px) {
  .login-page { grid-template-columns: 1fr; }
  .login-aside { padding: var(--sp-7); min-height: 280px; }
}
</style>
