<template>
  <section class="workspace">
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <span class="gradient-text">工作台</span>
        </h1>
        <div class="page-subtitle">
          <span>{{ auth.tenant?.name ?? '当前租户' }}</span>
          <span class="dot-sep">·</span>
          <span>{{ today }}</span>
        </div>
      </div>
    </div>

    <div class="kpi-grid">
      <div class="card card-pad kpi">
        <div class="kpi-head">
          <span class="kpi-icon kpi-icon-blue">◎</span>
          <span class="badge badge-info">本周</span>
        </div>
        <div class="kpi-value">{{ planCount }}</div>
        <div class="kpi-label">活跃计划</div>
      </div>
      <div class="card card-pad kpi">
        <div class="kpi-head">
          <span class="kpi-icon kpi-icon-orange">✎</span>
          <span class="badge badge-warning">本月</span>
        </div>
        <div class="kpi-value">{{ reportCount }}</div>
        <div class="kpi-label">已提报告</div>
      </div>
      <div class="card card-pad kpi">
        <div class="kpi-head">
          <span class="kpi-icon kpi-icon-green">✓</span>
          <span class="badge badge-success">在线</span>
        </div>
        <div class="kpi-value">live</div>
        <div class="kpi-label">系统状态</div>
      </div>
      <div class="card card-pad kpi">
        <div class="kpi-head">
          <span class="kpi-icon kpi-icon-purple">▲</span>
          <span class="badge badge-purple">v3.1</span>
        </div>
        <div class="kpi-value">{{ version }}</div>
        <div class="kpi-label">构建版本</div>
      </div>
    </div>

    <div class="row-2">
      <div class="card">
        <div class="card-head">
          <span class="card-title">计划层级 (来自字典服务)</span>
          <span class="badge">plan_level</span>
        </div>
        <div class="card-pad">
          <div class="chip-list">
            <span v-for="it in dictItems" :key="it.code" class="chip">
              <code>{{ it.code }}</code>
              <span>{{ it.name }}</span>
            </span>
          </div>
        </div>
      </div>

      <div class="card">
        <div class="card-head">
          <span class="card-title">快速入口</span>
        </div>
        <div class="card-pad shortcut-grid">
          <router-link to="/okr/plans" class="shortcut">
            <span class="shortcut-icon shortcut-blue">◎</span>
            <div>
              <div class="shortcut-title">创建计划</div>
              <div class="shortcut-sub">年 / 半年 / 月 / 周</div>
            </div>
          </router-link>
          <router-link to="/okr/reports" class="shortcut">
            <span class="shortcut-icon shortcut-orange">✎</span>
            <div>
              <div class="shortcut-title">提交日报</div>
              <div class="shortcut-sub">记录今日工作</div>
            </div>
          </router-link>
          <router-link to="/okr/reports" class="shortcut">
            <span class="shortcut-icon shortcut-purple">📋</span>
            <div>
              <div class="shortcut-title">提交周报</div>
              <div class="shortcut-sub">本周 Mon–Sun</div>
            </div>
          </router-link>
          <router-link to="/okr/rollup" class="shortcut">
            <span class="shortcut-icon shortcut-green">▲</span>
            <div>
              <div class="shortcut-title">周报汇总</div>
              <div class="shortcut-sub">按部门统计</div>
            </div>
          </router-link>
        </div>
      </div>
    </div>

    <div class="card" v-if="lastError">
      <div class="card-head">
        <span class="card-title">最近一次错误响应 (envelope 验证)</span>
        <span class="badge badge-danger">{{ lastError.error?.kind ?? 'error' }}</span>
      </div>
      <pre class="code-block">{{ JSON.stringify(lastError, null, 2) }}</pre>
    </div>
  </section>
</template>

<script setup lang="ts">
import { ref, onMounted } from "vue";
import { client } from "@/api/client";
import { useAuthStore } from "@/shell/auth/auth.store";
import { listPlans, listReports } from "@/modules/okr/api/okr";

interface DictItem { type_code: string; code: string; name: string; sort_order: number; active: boolean }

const auth = useAuthStore();
const version = ref<string>("…");
const dictItems = ref<DictItem[]>([]);
const planCount = ref(0);
const reportCount = ref(0);
const lastError = ref<any>(null);
const today = new Date().toISOString().slice(0, 10);

onMounted(async () => {
  try {
    const res = await fetch("/version");
    const j = await res.json();
    version.value = j?.version ?? "?";
  } catch { version.value = "—"; }

  try {
    const res = await client.get("/dict/plan_level");
    dictItems.value = res.data?.data?.items ?? [];
  } catch {}

  try { planCount.value = (await listPlans("week")).length; } catch {}
  try { reportCount.value = (await listReports()).length; } catch {}

  try { await client.get("/dict/__doesnotexist__"); }
  catch (e: any) { lastError.value = e.response?.data; }
});
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; margin-bottom: var(--sp-7); }
.page-title { font-size: 32px; font-weight: 800; letter-spacing: -0.02em; line-height: 1.1; }
.gradient-text {
  background: linear-gradient(90deg, var(--primary) 0%, var(--accent) 100%);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
}
.page-subtitle {
  font-size: 13px;
  color: var(--text-3);
  margin-top: 6px;
  display: flex;
  align-items: center;
  gap: var(--sp-2);
}
.dot-sep { color: var(--text-4); }

.kpi-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--sp-4);
  margin-bottom: var(--sp-6);
}
.kpi-head {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--sp-3);
}
.kpi-icon {
  font-size: 16px;
  background: var(--primary-soft);
  color: var(--primary);
}
.kpi-icon-blue { background: var(--info-soft); color: var(--info); }
.kpi-icon-orange { background: var(--warning-soft); color: var(--warning); }
.kpi-icon-green { background: var(--success-soft); color: var(--success); }
.kpi-icon-purple { background: var(--purple-soft); color: var(--purple); }
.kpi-label {
  font-size: 12.5px;
  color: var(--text-3);
  margin-top: 4px;
  font-weight: 500;
}

.row-2 {
  display: grid;
  grid-template-columns: 1fr 1.2fr;
  gap: var(--sp-4);
  margin-bottom: var(--sp-6);
}

.chip-list { display: flex; gap: var(--sp-2); flex-wrap: wrap; }
.chip code {
  font-family: var(--ff-mono);
  font-size: 11px;
  color: var(--primary);
  font-weight: 600;
}

.shortcut-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--sp-3); }
.shortcut {
  display: flex;
  gap: var(--sp-3);
  align-items: center;
  padding: var(--sp-3);
  border: 1px solid var(--border);
  border-radius: var(--r-md);
  text-decoration: none;
  color: var(--text);
  transition: all 0.15s;
}
.shortcut:hover {
  border-color: var(--primary);
  box-shadow: var(--sh-2);
  transform: translateY(-1px);
  text-decoration: none;
}
.shortcut-icon {
  width: 36px; height: 36px;
  border-radius: var(--r-sm);
  display: flex; align-items: center; justify-content: center;
  font-size: 16px;
  flex-shrink: 0;
}
.shortcut-blue { background: var(--info-soft); color: var(--info); }
.shortcut-orange { background: var(--warning-soft); color: var(--warning); }
.shortcut-purple { background: var(--purple-soft); color: var(--purple); }
.shortcut-green { background: var(--success-soft); color: var(--success); }
.shortcut-title { font-weight: 600; font-size: 13px; }
.shortcut-sub { font-size: 11.5px; color: var(--text-3); margin-top: 2px; }

.code-block {
  margin: var(--sp-4);
  background: var(--surface-3);
  padding: var(--sp-3) var(--sp-4);
  border-radius: var(--r-sm);
  font-family: var(--ff-mono);
  font-size: 12px;
  line-height: 1.6;
  overflow-x: auto;
  color: var(--text-2);
}
</style>
