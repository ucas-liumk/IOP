<template>
  <section>
    <header class="page-head">
      <h1>仪表盘</h1>
      <p class="sub">{{ tenant?.tenant.name }} · {{ tenant?.tenant.slug }} · 创建于 {{ tenant?.tenant.created_at?.slice(0, 10) }}</p>
    </header>

    <div class="stat-grid">
      <article class="stat-card">
        <div class="stat-icon" style="background:var(--info-soft);color:var(--info);">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>
        </div>
        <div>
          <div class="stat-label">成员总数</div>
          <div class="stat-value">{{ tenant?.member_count ?? 0 }}</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon" style="background:var(--purple-soft);color:var(--purple);">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>
        </div>
        <div>
          <div class="stat-label">角色数</div>
          <div class="stat-value">{{ roleCount }}</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon" style="background:var(--warning-soft);color:var(--warning);">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
        </div>
        <div>
          <div class="stat-label">审计条目 (近期)</div>
          <div class="stat-value">{{ auditCount }}</div>
        </div>
      </article>
      <article class="stat-card">
        <div class="stat-icon" style="background:var(--success-soft);color:var(--success);">
          <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="20 6 9 17 4 12"/></svg>
        </div>
        <div>
          <div class="stat-label">租户状态</div>
          <div class="stat-value status">{{ statusLabel }}</div>
        </div>
      </article>
    </div>

    <div class="row-2">
      <article class="card">
        <header class="card-header">
          <span class="card-title">最近活动</span>
          <router-link to="/admin/audit" class="card-link">全部 →</router-link>
        </header>
        <ul class="activity">
          <li v-for="a in recentAudit" :key="a.id">
            <span class="actor-dot"></span>
            <div>
              <div class="act-title"><code>{{ a.action }}</code> by {{ a.actor === 'system' ? '系统' : a.actor.slice(0,8) }}</div>
              <div class="act-meta">{{ a.occurred_at.slice(5,16) }}</div>
            </div>
          </li>
          <li v-if="!recentAudit.length" class="empty">暂无活动</li>
        </ul>
      </article>

      <article class="card">
        <header class="card-header">
          <span class="card-title">快捷操作</span>
        </header>
        <div class="actions">
          <router-link to="/admin/members" class="action-btn">
            <span class="icon" style="background:var(--info-soft);color:var(--info);">+</span>
            <div>
              <div class="ttl">邀请成员</div>
              <div class="sub-ttl">添加新人到租户</div>
            </div>
          </router-link>
          <router-link to="/admin/roles" class="action-btn">
            <span class="icon" style="background:var(--purple-soft);color:var(--purple);">✚</span>
            <div>
              <div class="ttl">创建角色</div>
              <div class="sub-ttl">自定义权限组合</div>
            </div>
          </router-link>
          <router-link to="/admin/settings" class="action-btn">
            <span class="icon" style="background:var(--warning-soft);color:var(--warning);">⚙</span>
            <div>
              <div class="ttl">租户设置</div>
              <div class="sub-ttl">名称 / 配置</div>
            </div>
          </router-link>
          <router-link to="/me/settings" class="action-btn">
            <span class="icon" style="background:var(--success-soft);color:var(--success);">⊙</span>
            <div>
              <div class="ttl">个人设置</div>
              <div class="sub-ttl">改密码 / 会话</div>
            </div>
          </router-link>
        </div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { getTenant, listAudit, listRoles, type TenantInfo, type AuditEntry } from "../api/admin";

const tenant = ref<TenantInfo | null>(null);
const roleCount = ref(0);
const auditCount = ref(0);
const recentAudit = ref<AuditEntry[]>([]);

const statusLabel = computed(() => {
  const s = tenant.value?.tenant.status;
  return s === "active" ? "在线" : s === "suspended" ? "暂停" : s === "closed" ? "已关闭" : "—";
});

onMounted(async () => {
  try { tenant.value = await getTenant(); } catch {}
  try { roleCount.value = (await listRoles()).length; } catch {}
  try {
    const a = await listAudit();
    auditCount.value = a.length;
    recentAudit.value = a.slice(0, 6);
  } catch {}
});
</script>

<style scoped>
.page-head h1 { font-size: 22px; font-weight: 700; letter-spacing: -0.01em; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }

.stat-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 14px;
  margin: 20px 0;
}
.stat-card {
  display: flex;
  gap: 12px;
  align-items: center;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
}
.stat-icon {
  width: 38px; height: 38px;
  border-radius: 10px;
  display: grid; place-items: center;
  flex-shrink: 0;
}
.stat-label { font-size: 12px; color: var(--text-3); font-weight: 500; }
.stat-value { font-size: 22px; font-weight: 700; letter-spacing: -0.02em; margin-top: 2px; }
.stat-value.status { font-size: 16px; color: var(--success); }

.row-2 { display: grid; grid-template-columns: 1fr 1fr; gap: 14px; }

.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}
.card-header {
  display: flex; justify-content: space-between; align-items: center;
  padding: 14px 18px;
  border-bottom: 1px solid var(--border);
}
.card-title { font-size: 14px; font-weight: 600; }
.card-link { font-size: 12.5px; color: var(--primary); }

.activity { list-style: none; padding: 8px; }
.activity li {
  display: flex; gap: 10px; align-items: flex-start;
  padding: 8px 10px;
  border-radius: 6px;
}
.activity li:hover { background: var(--bg); }
.activity li.empty {
  text-align: center;
  color: var(--text-4);
  padding: 30px 0;
  display: block;
}
.actor-dot {
  width: 8px; height: 8px;
  background: var(--primary);
  border-radius: 50%;
  margin-top: 6px;
  flex-shrink: 0;
}
.act-title {
  font-size: 13px; color: var(--text);
}
.act-title code {
  background: var(--surface-2);
  padding: 1px 6px;
  border-radius: 3px;
  font-family: var(--ff-mono);
  font-size: 11.5px;
  color: var(--primary);
}
.act-meta {
  font-size: 11.5px; color: var(--text-3); margin-top: 2px;
}

.actions {
  padding: 12px;
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 8px;
}
.action-btn {
  display: flex; gap: 10px;
  align-items: center;
  padding: 12px;
  border-radius: 10px;
  color: inherit;
  text-decoration: none;
  border: 1px solid var(--border);
  transition: all 0.12s;
}
.action-btn:hover { border-color: var(--primary); box-shadow: var(--sh-1); text-decoration: none; }
.action-btn .icon {
  width: 34px; height: 34px;
  border-radius: 8px;
  display: grid; place-items: center;
  font-size: 16px;
  font-weight: 600;
}
.ttl { font-size: 13px; font-weight: 600; }
.sub-ttl { font-size: 11.5px; color: var(--text-3); margin-top: 2px; }
</style>
