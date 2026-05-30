<template>
  <section class="ph">
    <PageHeader title="平台概览" sub="全平台治理视图 · 仅平台管理员可见" />

    <div class="stat-grid">
      <article class="stat-card" @click="$router.push('/platform/organizations')">
        <div class="stat-ico" style="background:var(--info-soft);color:var(--info)">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="6" width="18" height="15" rx="1"/><path d="M3 10h18"/></svg>
        </div>
        <div class="stat-body"><div class="stat-label">组织机构</div><div class="stat-value">{{ stats.organizations }}</div></div>
      </article>
      <article class="stat-card" @click="$router.push('/platform/users')">
        <div class="stat-ico" style="background:var(--purple-soft);color:var(--purple)">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>
        </div>
        <div class="stat-body"><div class="stat-label">全局用户</div><div class="stat-value">{{ stats.users }}</div></div>
      </article>
      <article class="stat-card alert" @click="$router.push('/platform/registrations')">
        <div class="stat-ico" style="background:var(--warning-soft);color:var(--warning)">
          <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><line x1="20" y1="8" x2="20" y2="14"/><line x1="23" y1="11" x2="17" y2="11"/></svg>
        </div>
        <div class="stat-body"><div class="stat-label">待审批注册</div><div class="stat-value">{{ stats.pending_registrations }}</div></div>
      </article>
    </div>

    <article class="card">
      <h3 class="card-title">快捷入口</h3>
      <div class="quick-grid">
        <RouterLink class="quick" to="/platform/organizations">
          <span class="q-title">组织管理</span>
          <span class="q-sub">开通 / 暂停 / 恢复组织</span>
        </RouterLink>
        <RouterLink class="quick" to="/platform/users">
          <span class="q-title">全局用户</span>
          <span class="q-sub">跨组织用户 / 重置密码</span>
        </RouterLink>
        <RouterLink class="quick" to="/platform/registrations">
          <span class="q-title">注册申请</span>
          <span class="q-sub">审批全平台入驻申请</span>
        </RouterLink>
        <RouterLink class="quick" to="/platform/rbac">
          <span class="q-title">平台角色</span>
          <span class="q-sub">菜单权限 + 成员</span>
        </RouterLink>
        <RouterLink class="quick" to="/platform/menus">
          <span class="q-title">菜单目录</span>
          <span class="q-sub">全平台菜单 / 权限巡检</span>
        </RouterLink>
      </div>
    </article>

    <article class="card">
      <h3 class="card-title">平台管理员能做什么</h3>
      <ul class="role-list">
        <li><strong>组织机构</strong>：开通 / 暂停 / 关闭所有组织（租户），数据物理隔离</li>
        <li><strong>全局用户</strong>：跨组织创建 / 停用 / 重置密码，并指派为某组织的成员或管理员</li>
        <li><strong>注册申请</strong>：审批全平台的入驻申请（任意目标组织）</li>
        <li><strong>平台角色 / 菜单目录</strong>：配置平台侧角色权限，巡检两套控制台的菜单 / 权限目录</li>
        <li class="muted">组织<strong>内部</strong>的成员 / 角色 / 部门 / 设置由各组织的<strong>组织管理员</strong>自治；平台管理员不直接介入</li>
      </ul>
    </article>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive } from "vue";
import { PageHeader } from "@/shell/components";
import { getPlatformStats } from "@/modules/admin/api/admin";

const stats = reactive({ organizations: 0, users: 0, pending_registrations: 0 });
onMounted(async () => {
  try { Object.assign(stats, await getPlatformStats()); } catch {}
});
</script>

<style scoped>
.ph { display: flex; flex-direction: column; gap: var(--sp-5); max-width: 980px; }
.stat-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(220px, 1fr)); gap: 14px; }
.stat-card {
  display: flex; align-items: center; gap: 14px;
  background: var(--surface); border: 1px solid var(--border); border-radius: 14px;
  padding: 18px 20px; cursor: pointer; transition: all .15s; box-shadow: var(--sh-1);
}
.stat-card:hover { transform: translateY(-2px); box-shadow: var(--sh-2); border-color: var(--border-strong); }
.stat-ico { width: 44px; height: 44px; border-radius: 12px; display: grid; place-items: center; }
.stat-label { font-size: 12.5px; color: var(--text-3); }
.stat-value { font-size: 26px; font-weight: 700; color: var(--text); line-height: 1.1; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 14px; padding: 20px 22px; box-shadow: var(--sh-1); }
.card-title { font-size: 15px; font-weight: 600; margin: 0 0 12px; }
.quick-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 10px; }
.quick {
  display: flex; flex-direction: column; gap: 3px;
  padding: 12px 14px; border: 1px solid var(--border); border-radius: 10px;
  background: var(--surface); text-decoration: none; transition: all .15s;
}
.quick:hover { border-color: var(--primary); background: var(--primary-soft); transform: translateY(-1px); }
.q-title { font-size: 13.5px; font-weight: 600; color: var(--text); }
.quick:hover .q-title { color: var(--primary); }
.q-sub { font-size: 11.5px; color: var(--text-3); }
.role-list { margin: 0; padding-left: 18px; display: flex; flex-direction: column; gap: 7px; font-size: 13px; color: var(--text-2); line-height: 1.6; }
.role-list .muted { color: var(--text-3); }
</style>
