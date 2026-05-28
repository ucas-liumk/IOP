<template>
  <section class="login">
    <h1>登录 IOP</h1>
    <form class="card" @submit.prevent="submit">
      <label>
        <span>邮箱</span>
        <input v-model="email" type="email" required autofocus />
      </label>
      <label>
        <span>密码</span>
        <input v-model="password" type="password" required minlength="10" />
      </label>
      <div v-if="error" class="err">{{ error }}</div>
      <button :disabled="loading" type="submit">
        {{ loading ? "登录中..." : "登录" }}
      </button>
    </form>
    <p class="muted">
      首次使用请先通过 <code>tenantctl</code> 创建租户和账号。
    </p>
  </section>
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
.login { max-width: 380px; margin: 80px auto; padding: var(--space-6); }
.card { display: flex; flex-direction: column; gap: var(--space-4); padding: var(--space-6); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius); }
label { display: flex; flex-direction: column; gap: var(--space-1); }
input { padding: var(--space-2) var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius); font-size: 14px; }
input:focus { outline: 2px solid var(--color-primary); }
button { padding: var(--space-3); background: var(--color-primary); color: white; border: 0; border-radius: var(--radius); font-weight: 600; cursor: pointer; }
button:disabled { opacity: 0.6; cursor: not-allowed; }
.err { color: var(--color-danger); font-size: 13px; }
.muted { margin-top: var(--space-4); color: var(--color-text-muted); font-size: 13px; }
code { background: var(--color-bg); padding: 2px 6px; border-radius: 3px; }
</style>
