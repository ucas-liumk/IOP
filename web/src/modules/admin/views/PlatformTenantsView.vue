<template>
  <section>
    <header class="page-head">
      <div>
        <h1>租户管理 <span class="badge-platform">平台级</span></h1>
        <p class="sub">列出本平台上所有租户 · 仅平台管理员可见</p>
      </div>
      <button class="btn btn-primary">+ 开通租户</button>
    </header>

    <article class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 220px">租户</th>
            <th style="width: 120px">Slug</th>
            <th style="width: 180px">Schema</th>
            <th style="width: 90px">状态</th>
            <th style="width: 120px">创建时间</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="t in tenants" :key="t.id">
            <td>
              <div class="tenant-cell">
                <div class="t-logo" :style="{ background: colorFor(t.name) }">{{ t.name[0] }}</div>
                <div>
                  <div class="t-name">{{ t.name }}</div>
                  <div class="t-id"><code>{{ t.id.slice(0, 8) }}…</code></div>
                </div>
              </div>
            </td>
            <td><code>{{ t.slug }}</code></td>
            <td><code class="mono">{{ t.schema_name }}</code></td>
            <td>
              <span class="badge" :class="'status-' + t.status">
                <span class="dot"></span>{{ statusLabel(t.status) }}
              </span>
            </td>
            <td class="time">{{ t.created_at?.slice(0, 10) }}</td>
            <td class="actions">
              <button v-if="t.status === 'active'" class="link-btn warn" @click="suspend(t)">暂停</button>
              <button v-else-if="t.status === 'suspended'" class="link-btn" @click="resume(t)">恢复</button>
            </td>
          </tr>
          <tr v-if="tenants.length === 0"><td colspan="6" class="empty">尚无租户</td></tr>
        </tbody>
      </table>
    </article>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { listAllTenants, suspendTenant, resumeTenant, type PlatformTenant } from "../api/admin";

const tenants = ref<PlatformTenant[]>([]);

onMounted(reload);
async function reload() { tenants.value = await listAllTenants(); }

async function suspend(t: PlatformTenant) {
  if (!confirm(`确定暂停租户 "${t.name}"？`)) return;
  await suspendTenant(t.id);
  await reload();
}
async function resume(t: PlatformTenant) {
  await resumeTenant(t.id);
  await reload();
}

function statusLabel(s: string) { return { active: "运行中", suspended: "已暂停", closed: "已关闭" }[s] ?? s; }
function colorFor(name: string) {
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
.page-head h1 { font-size: 22px; font-weight: 700; display: flex; align-items: center; gap: 10px; }
.badge-platform { font-size: 11px; font-weight: 700; padding: 2px 8px; background: var(--purple-soft); color: var(--purple); border-radius: 999px; letter-spacing: .3px; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }
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
}
.data-table tbody tr:hover { background: var(--surface-2); }

.tenant-cell { display: flex; gap: 10px; align-items: center; }
.t-logo {
  width: 32px; height: 32px;
  border-radius: 7px;
  color: white; font-weight: 700;
  display: grid; place-items: center;
}
.t-name { font-weight: 600; }
.t-id { font-size: 11px; color: var(--text-3); margin-top: 2px; }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; }
code.mono { color: var(--text-2); }
.time { color: var(--text-3); font-family: var(--ff-mono); font-size: 12px; }

.badge {
  display: inline-flex; align-items: center; gap: 5px;
  padding: 2px 8px;
  border-radius: 999px;
  font-size: 11.5px; font-weight: 600;
}
.badge .dot { width: 5px; height: 5px; background: currentColor; border-radius: 999px; }
.badge.status-active { background: var(--success-soft); color: var(--success); }
.badge.status-suspended { background: var(--warning-soft); color: var(--warning); }
.badge.status-closed { background: var(--danger-soft); color: var(--danger); }

.actions { white-space: nowrap; }
.link-btn { background: transparent; border: 0; font-size: 12.5px; color: var(--primary); cursor: pointer; padding: 4px 8px; border-radius: 4px; }
.link-btn:hover { background: var(--primary-soft); }
.link-btn.warn { color: var(--warning); }
.link-btn.warn:hover { background: var(--warning-soft); }

.empty { text-align: center; color: var(--text-4); padding: 36px 0; }
</style>
