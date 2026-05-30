<template>
  <section class="me-page">
    <PageHeader title="安全设置" sub="修改密码以保证账号安全" />

    <div v-if="forced" class="forced-banner">
      <strong>请先修改初始密码</strong>
      <span>为了账号安全，使用默认/初始密码登录后必须设置新密码才能继续使用平台。</span>
    </div>

    <article class="card">
      <h3 class="section-title">修改密码</h3>
      <p class="section-sub">建议每 3 个月更换一次密码。密码至少 10 位，包含字母与数字。</p>

      <form @submit.prevent="submit" class="form">
        <label class="field">
          <span class="label">当前密码</span>
          <PasswordField v-model="oldPw" required autocomplete="current-password" placeholder="请输入当前密码" />
        </label>
        <label class="field">
          <span class="label">新密码</span>
          <PasswordField v-model="newPw" required :minlength="10" autocomplete="new-password" placeholder="至少 10 位，含字母与数字" />
        </label>
        <label class="field">
          <span class="label">确认新密码</span>
          <PasswordField v-model="newPw2" required autocomplete="new-password" placeholder="再次输入新密码" />
        </label>

        <div v-if="error" class="form-error">{{ error }}</div>
        <div v-if="success" class="form-ok">密码已更新</div>

        <button type="submit" class="btn btn-primary" :disabled="busy">
          {{ busy ? '更新中…' : '更新密码' }}
        </button>
      </form>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from "vue";
import { useRoute, useRouter } from "vue-router";
import { PageHeader } from "@/shell/components";
import { changePassword } from "@/modules/admin/api/admin";
import { useAuthStore } from "@/shell/auth/auth.store";
import { homeForRole } from "@/shell/auth/guard";
import PasswordField from "@/shell/auth/PasswordField.vue";

const route = useRoute();
const router = useRouter();
const auth = useAuthStore();
const forced = computed(() => route.query.forced === "1" || !!auth.user?.password_must_change);

const oldPw = ref("");
const newPw = ref("");
const newPw2 = ref("");
const busy = ref(false);
const error = ref("");
const success = ref(false);

async function submit() {
  error.value = "";
  success.value = false;
  if (newPw.value !== newPw2.value) {
    error.value = "两次新密码不一致";
    return;
  }
  busy.value = true;
  try {
    await changePassword(oldPw.value, newPw.value);
    success.value = true;
    oldPw.value = ""; newPw.value = ""; newPw2.value = "";
    // Clear the forced flag locally and let the user into the app.
    if (auth.user) auth.user.password_must_change = false;
    if (forced.value) setTimeout(() => router.push(homeForRole()), 800);
  } catch (e: any) {
    error.value = e.response?.data?.error?.message ?? "更新失败";
  } finally { busy.value = false; }
}
</script>

<style scoped>
.me-page { display: flex; flex-direction: column; gap: var(--sp-5); max-width: 560px; }
.forced-banner {
  display: flex; flex-direction: column; gap: 3px;
  background: var(--warning-soft); color: var(--warning);
  border: 1px solid var(--warning); border-radius: 10px;
  padding: 12px 16px; font-size: 13px;
}
.forced-banner strong { font-size: 13.5px; }
.forced-banner span { color: var(--text-2); font-size: 12.5px; }
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--sh-1);
  padding: 24px;
}
.section-title { font-size: 15px; font-weight: 600; margin: 0; }
.section-sub { font-size: 12.5px; color: var(--text-3); margin: 6px 0 18px; }
.form { display: flex; flex-direction: column; gap: 14px; }
.field { display: flex; flex-direction: column; gap: 5px; }
.field .label { font-size: 12px; color: var(--text-2); }
.form-error {
  font-size: 12.5px;
  background: var(--danger-soft);
  color: var(--danger);
  padding: 8px 12px;
  border-radius: 7px;
}
.form-ok {
  font-size: 12.5px;
  background: var(--success-soft);
  color: var(--success);
  padding: 8px 12px;
  border-radius: 7px;
}
</style>
