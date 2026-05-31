<template>
  <section class="admin-page">
    <PageHeader title="系统监控" sub="服务 / 数据库 / Redis / 对象存储健康 + 平台计数">
      <template #actions>
        <label class="auto">
          <input type="checkbox" v-model="autoRefresh" /> 自动刷新 (10s)
        </label>
        <button class="btn btn-ghost" @click="reload">刷新</button>
      </template>
    </PageHeader>

    <!-- Health cards -->
    <div class="cards">
      <article v-for="h in healthCards" :key="h.name" class="card health" :class="h.ok ? 'ok' : 'down'">
        <div class="h-top">
          <span class="dot" :class="h.ok ? 'ok' : 'down'"></span>
          <span class="h-name">{{ h.label }}</span>
        </div>
        <div class="h-status">{{ h.ok ? '正常' : '异常' }}</div>
        <div class="h-meta">
          <span v-if="h.latency != null">{{ h.latency }} ms</span>
          <span v-if="h.error" class="h-err">{{ h.error }}</span>
        </div>
      </article>
      <article v-if="healthCards.length === 0" class="card health muted-card">
        <div class="h-status">无健康探针数据</div>
        <div class="h-meta">后端未配置 health 探针，仅展示数据库连接池与计数。</div>
      </article>
    </div>

    <!-- Counters -->
    <article class="card panel">
      <h3 class="panel-title">平台计数</h3>
      <div class="kpis">
        <div v-for="(v, k) in (snapshot?.counters ?? {})" :key="k" class="kpi">
          <div class="kpi-val">{{ v }}</div>
          <div class="kpi-lbl">{{ counterLabel(String(k)) }}</div>
        </div>
        <div v-if="!snapshot || Object.keys(snapshot.counters).length === 0" class="muted">无计数数据</div>
      </div>
    </article>

    <!-- DB pool -->
    <article class="card panel">
      <h3 class="panel-title">数据库连接池</h3>
      <div class="kpis">
        <div class="kpi"><div class="kpi-val">{{ pool.acquired_conns ?? '—' }}</div><div class="kpi-lbl">使用中</div></div>
        <div class="kpi"><div class="kpi-val">{{ pool.idle_conns ?? '—' }}</div><div class="kpi-lbl">空闲</div></div>
        <div class="kpi"><div class="kpi-val">{{ pool.total_conns ?? '—' }}</div><div class="kpi-lbl">总连接</div></div>
        <div class="kpi"><div class="kpi-val">{{ pool.max_conns ?? '—' }}</div><div class="kpi-lbl">上限</div></div>
        <div class="kpi"><div class="kpi-val">{{ pool.acquire_count ?? '—' }}</div><div class="kpi-lbl">累计获取</div></div>
        <div class="kpi"><div class="kpi-val">{{ pool.empty_acquire_count ?? '—' }}</div><div class="kpi-lbl">空池等待</div></div>
      </div>
    </article>

    <!-- Redis info (best effort) -->
    <article v-if="snapshot?.redis && Object.keys(snapshot.redis).length" class="card panel">
      <h3 class="panel-title">Redis</h3>
      <dl class="kv">
        <template v-for="(v, k) in snapshot.redis" :key="k">
          <dt>{{ k }}</dt><dd>{{ v }}</dd>
        </template>
      </dl>
    </article>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from "vue";
import { PageHeader } from "@/shell/components";
import { getMonitor, type MonitorSnapshot, type DBPoolStats } from "../api/system";

const snapshot = ref<MonitorSnapshot | null>(null);
const autoRefresh = ref(false);
let timer: number | undefined;

const pool = computed<Partial<DBPoolStats>>(() => snapshot.value?.db_pool ?? {});

interface HealthCard { name: string; label: string; ok: boolean; latency?: number; error?: string }
const healthCards = computed<HealthCard[]>(() => {
  const h = snapshot.value?.health ?? {};
  return Object.entries(h).map(([name, c]: [string, any]) => {
    const ok = c?.ok === true || c?.status === "ok" || c?.status === "up" || c?.status === "healthy";
    return {
      name,
      label: healthLabel(name),
      ok,
      latency: typeof c?.latency_ms === "number" ? c.latency_ms : undefined,
      error: c?.error || (!ok && typeof c?.status === "string" ? c.status : undefined),
    };
  });
});

onMounted(() => {
  reload();
});
onUnmounted(stopTimer);

async function reload() {
  snapshot.value = await getMonitor();
}

function startTimer() {
  stopTimer();
  timer = window.setInterval(reload, 10000);
}
function stopTimer() {
  if (timer != null) { window.clearInterval(timer); timer = undefined; }
}

watch(autoRefresh, (on) => { on ? startTimer() : stopTimer(); });

function healthLabel(name: string): string {
  const map: Record<string, string> = {
    db: "数据库 (PostgreSQL)", postgres: "数据库 (PostgreSQL)", pg: "数据库 (PostgreSQL)",
    redis: "Redis", cache: "Redis",
    minio: "对象存储 (MinIO)", s3: "对象存储 (MinIO)", storage: "对象存储",
  };
  return map[name.toLowerCase()] ?? name;
}

function counterLabel(k: string): string {
  const map: Record<string, string> = {
    active_organizations: "活跃组织",
    platform_users: "平台用户",
    active_sessions: "活跃会话",
  };
  return map[k] ?? k;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.auto { display: inline-flex; align-items: center; gap: 6px; font-size: 12.5px; color: var(--text-2); cursor: pointer; }

.cards { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 14px; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.health { padding: 16px 18px; display: flex; flex-direction: column; gap: 6px; }
.health.ok { border-left: 3px solid var(--success); }
.health.down { border-left: 3px solid var(--danger); }
.muted-card { border-left: 3px solid var(--border-strong); }
.h-top { display: flex; align-items: center; gap: 8px; }
.dot { width: 9px; height: 9px; border-radius: 50%; display: inline-block; }
.dot.ok { background: var(--success); }
.dot.down { background: var(--danger); }
.h-name { font-size: 13px; font-weight: 600; color: var(--text); }
.h-status { font-size: 20px; font-weight: 700; color: var(--text); }
.h-meta { font-size: 11.5px; color: var(--text-3); display: flex; flex-direction: column; gap: 2px; }
.h-err { color: var(--danger); word-break: break-all; }

.panel { padding: 18px 20px; }
.panel-title { font-size: 13px; font-weight: 600; color: var(--text-2); margin: 0 0 14px; }
.kpis { display: grid; grid-template-columns: repeat(auto-fill, minmax(110px, 1fr)); gap: 16px; }
.kpi { display: flex; flex-direction: column; gap: 3px; }
.kpi-val { font-size: 22px; font-weight: 700; color: var(--text); font-variant-numeric: tabular-nums; }
.kpi-lbl { font-size: 11.5px; color: var(--text-3); }
.muted { color: var(--text-4); font-size: 12.5px; }

.kv { display: grid; grid-template-columns: 200px 1fr; gap: 6px 14px; margin: 0; font-size: 12.5px; }
.kv dt { color: var(--text-3); font-family: var(--ff-mono); }
.kv dd { margin: 0; color: var(--text); font-family: var(--ff-mono); word-break: break-all; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
</style>
