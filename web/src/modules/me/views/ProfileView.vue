<template>
  <section class="me-page">
    <PageHeader title="个人资料" sub="查看你的账号信息，由管理员维护" />

    <article class="card">
      <div class="profile-head">
        <div class="avatar">{{ initials }}</div>
        <div>
          <div class="display-name">{{ auth.user?.username || '—' }}</div>
          <div class="display-sub">
            <span v-if="admin.has_platform_access" class="role-tag platform">平台角色</span>
            <span v-else-if="admin.is_tenant_admin" class="role-tag tenant">租户管理员</span>
            <span v-else class="role-tag">成员</span>
            <span class="separator">·</span>
            <span>{{ auth.tenant?.name ?? '—' }}</span>
          </div>
        </div>
      </div>

      <div class="info-list">
        <div class="info-row">
          <span class="info-label">用户名</span>
          <span class="info-value"><code>{{ auth.user?.username || '—' }}</code></span>
        </div>
        <div class="info-row">
          <span class="info-label">手机号</span>
          <span class="info-value">
            <template v-if="auth.user?.phone">{{ auth.user.phone }}</template>
            <span v-else class="muted">未绑定</span>
          </span>
        </div>
        <div class="info-row">
          <span class="info-label">邮箱</span>
          <span class="info-value">
            <template v-if="auth.user?.email">{{ auth.user.email }}</template>
            <span v-else class="muted">未绑定</span>
          </span>
        </div>
        <div class="info-row">
          <span class="info-label">当前租户</span>
          <span class="info-value">{{ auth.tenant?.name || '—' }} <code v-if="auth.tenant">{{ auth.tenant.slug }}</code></span>
        </div>
        <div class="info-row">
          <span class="info-label">用户 ID</span>
          <span class="info-value"><code>{{ auth.user?.id }}</code></span>
        </div>
      </div>

      <div class="profile-foot">
        <span class="muted">姓名 / 邮箱 / 手机号修改请联系所在单位管理员</span>
      </div>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { PageHeader } from "@/shell/components";
import { useAuthStore } from "@/shell/auth/auth.store";
import { getMyAdminFlags, type MeAdmin } from "@/modules/admin/api/admin";

const auth = useAuthStore();
const admin = ref<MeAdmin>({ is_tenant_admin: false, is_platform_admin: false, has_platform_access: false });

onMounted(async () => {
  admin.value = await getMyAdminFlags();
});

const initials = computed(() => {
  const s = auth.user?.username || auth.user?.email || auth.user?.phone || "?";
  return s.slice(0, 2).toUpperCase();
});
</script>

<style scoped>
.me-page { display: flex; flex-direction: column; gap: var(--sp-5); max-width: 720px; }
.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--sh-1);
  padding: 24px;
}
.profile-head {
  display: flex; align-items: center; gap: 16px;
  padding-bottom: 18px;
  border-bottom: 1px solid var(--border);
}
.avatar {
  width: 60px; height: 60px;
  border-radius: 50%;
  background: linear-gradient(135deg, var(--primary), #5b8bf5);
  color: #fff;
  display: grid; place-items: center;
  font-size: 22px;
  font-weight: 700;
}
.display-name { font-size: 18px; font-weight: 700; color: var(--text); }
.display-sub {
  font-size: 13px; color: var(--text-3);
  margin-top: 4px;
  display: flex; align-items: center; gap: 8px;
}
.separator { color: var(--text-4); }

.role-tag {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 600;
  background: var(--bg-deep);
  color: var(--text-3);
}
.role-tag.tenant { background: var(--info-soft); color: var(--info); }
.role-tag.platform { background: var(--purple-soft); color: var(--purple); }

.info-list { display: flex; flex-direction: column; padding: 12px 0; }
.info-row {
  display: flex; align-items: center;
  padding: 10px 0;
  border-bottom: 1px dashed var(--border);
}
.info-row:last-child { border-bottom: 0; }
.info-label {
  width: 120px;
  font-size: 12.5px;
  color: var(--text-3);
}
.info-value {
  flex: 1; min-width: 0;
  font-size: 13.5px;
  color: var(--text);
  display: flex; align-items: center; gap: 8px;
}
.info-value code {
  background: var(--surface-2);
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--ff-mono);
  font-size: 11.5px;
  color: var(--text-2);
}

.profile-foot {
  margin-top: 14px;
  padding-top: 14px;
  border-top: 1px solid var(--border);
}
.muted { color: var(--text-3); font-size: 12.5px; }
</style>
