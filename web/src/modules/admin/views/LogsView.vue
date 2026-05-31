<template>
  <section class="admin-page">
    <PageHeader title="日志" sub="操作日志 + 登录日志 · 异步收集，可能有数秒延迟">
      <template #actions>
        <button class="btn btn-ghost" @click="reload">刷新</button>
      </template>
    </PageHeader>

    <!-- Tabs -->
    <div class="tabs">
      <button class="tab" :class="{ active: tab === 'oper' }" @click="switchTab('oper')">操作日志</button>
      <button class="tab" :class="{ active: tab === 'login' }" @click="switchTab('login')">登录日志</button>
    </div>

    <!-- Filters -->
    <div class="filters">
      <input class="input" v-model="filter.actor" placeholder="执行者 (member id 前缀/全 id)" @keyup.enter="reload" />
      <input v-if="tab === 'oper'" class="input" v-model="filter.action" placeholder="动作 (如 iam.member_created)" @keyup.enter="reload" />
      <input class="input date" type="date" v-model="filter.from" title="起始日期" />
      <input class="input date" type="date" v-model="filter.to" title="结束日期" />
      <button class="btn btn-primary" @click="reload">查询</button>
      <button class="btn btn-ghost" @click="resetFilters">重置</button>
    </div>

    <article class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 150px">时间</th>
            <th style="width: 200px">动作</th>
            <th style="width: 140px">执行者</th>
            <th style="width: 110px">资源</th>
            <th>详情</th>
            <th style="width: 90px">追踪 ID</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in entries" :key="a.id" @click="openDetail(a)" class="clickable">
            <td class="time">{{ a.occurred_at?.slice(0, 19).replace('T', ' ') }}</td>
            <td><code class="action">{{ a.action }}</code></td>
            <td>{{ actorLabel(a.actor) }}</td>
            <td class="muted">{{ a.resource || '—' }}</td>
            <td class="detail"><span class="detail-preview">{{ detailPreview(a.detail) }}</span></td>
            <td class="trace"><code>{{ a.trace_id?.slice(0, 8) }}</code></td>
          </tr>
          <tr v-if="entries.length === 0">
            <td colspan="6" class="empty-cell">{{ loading ? "加载中…" : "暂无日志记录" }}</td>
          </tr>
        </tbody>
      </table>
    </article>

    <!-- Pager (array endpoints have no total; infer next from full page) -->
    <div class="pager">
      <button class="btn btn-ghost btn-sm" :disabled="page <= 1" @click="prevPage">上一页</button>
      <span class="page-no">第 {{ page }} 页</span>
      <button class="btn btn-ghost btn-sm" :disabled="!hasNext" @click="nextPage">下一页</button>
    </div>

    <!-- Detail drawer -->
    <div v-if="detail" class="drawer-overlay" @click.self="detail = null">
      <aside class="drawer">
        <header class="drawer-head">
          <h3>日志详情</h3>
          <button class="x" @click="detail = null">✕</button>
        </header>
        <dl class="kv">
          <dt>时间</dt><dd>{{ detail.occurred_at?.slice(0, 19).replace('T', ' ') }}</dd>
          <dt>动作</dt><dd><code class="action">{{ detail.action }}</code></dd>
          <dt>执行者</dt><dd>{{ detail.actor || '—' }}</dd>
          <dt>资源</dt><dd>{{ detail.resource || '—' }}</dd>
          <dt>资源 ID</dt><dd><code>{{ detail.resource_id || '—' }}</code></dd>
          <dt>追踪 ID</dt><dd><code>{{ detail.trace_id || '—' }}</code></dd>
        </dl>
        <div class="detail-block">
          <span class="block-label">详情 JSON</span>
          <pre class="detail-full">{{ JSON.stringify(parsedDetail(detail.detail), null, 2) }}</pre>
        </div>
      </aside>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { PageHeader } from "@/shell/components";
import { listOperLogs, listLoginLogs, type AuditEntry, type LogFilter } from "../api/admin";

const PAGE_SIZE = 20;

const tab = ref<"oper" | "login">("oper");
const entries = ref<AuditEntry[]>([]);
const detail = ref<AuditEntry | null>(null);
const page = ref(1);
const hasNext = ref(false);
const loading = ref(false);

const filter = reactive({ actor: "", action: "", from: "", to: "" });

onMounted(reload);

function buildFilter(): LogFilter {
  return {
    actor: filter.actor.trim() || undefined,
    action: filter.action.trim() || undefined,
    from: filter.from || undefined,
    to: filter.to || undefined,
    page: page.value,
    pageSize: PAGE_SIZE,
  };
}

async function reload() {
  loading.value = true;
  try {
    const f = buildFilter();
    entries.value = tab.value === "oper" ? await listOperLogs(f) : await listLoginLogs(f);
    hasNext.value = entries.value.length >= PAGE_SIZE;
  } finally { loading.value = false; }
}

function switchTab(t: "oper" | "login") {
  if (tab.value === t) return;
  tab.value = t;
  page.value = 1;
  reload();
}

function resetFilters() {
  filter.actor = ""; filter.action = ""; filter.from = ""; filter.to = "";
  page.value = 1;
  reload();
}

function nextPage() { if (hasNext.value) { page.value++; reload(); } }
function prevPage() { if (page.value > 1) { page.value--; reload(); } }

function openDetail(a: AuditEntry) {
  // Login logs are flat events; the detail drawer is most useful for oper logs,
  // but we open it for both — the JSON block adapts to whatever payload exists.
  detail.value = a;
}

function actorLabel(actor: string): string {
  if (!actor) return "—";
  if (actor === "system") return "系统";
  return actor.length > 12 ? actor.slice(0, 8) + "…" : actor;
}

function parsedDetail(d: any) {
  if (!d) return d;
  if (typeof d === "string") { try { return JSON.parse(d); } catch { return d; } }
  if (d instanceof Uint8Array || Array.isArray(d)) {
    try { return JSON.parse(new TextDecoder().decode(new Uint8Array(d))); } catch { return d; }
  }
  return d;
}
function detailPreview(d: any) {
  const p = parsedDetail(d);
  if (!p) return "—";
  if (typeof p === "string") return p.slice(0, 90);
  const s = JSON.stringify(p);
  return s.length > 90 ? s.slice(0, 90) + "…" : s;
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-4); }

.tabs { display: flex; gap: 4px; border-bottom: 1px solid var(--border); }
.tab {
  background: transparent; border: 0; border-bottom: 2px solid transparent;
  padding: 9px 16px; font-size: 13.5px; font-weight: 500; color: var(--text-3);
  cursor: pointer; margin-bottom: -1px;
}
.tab:hover { color: var(--text-2); }
.tab.active { color: var(--primary); border-bottom-color: var(--primary); font-weight: 600; }

.filters { display: flex; gap: 8px; flex-wrap: wrap; align-items: center; }
.filters .input { flex: 0 1 220px; }
.filters .input.date { flex: 0 0 auto; width: 150px; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th {
  text-align: left; font-size: 11.5px; font-weight: 600; color: var(--text-3);
  text-transform: uppercase; letter-spacing: .5px; padding: 10px 14px;
  background: var(--surface-2); border-bottom: 1px solid var(--border);
}
.data-table td { padding: 11px 14px; font-size: 12.5px; border-bottom: 1px solid var(--border-soft); vertical-align: top; }
.data-table tbody tr.clickable { cursor: pointer; }
.data-table tbody tr:hover { background: var(--surface-2); }
.data-table tbody tr:last-child td { border-bottom: 0; }
.time { font-family: var(--ff-mono); color: var(--text-2); white-space: nowrap; }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; }
code.action { color: var(--primary); font-weight: 600; }
.muted { color: var(--text-3); }
.detail-preview { color: var(--text-2); font-family: var(--ff-mono); font-size: 11.5px; }
.trace code { color: var(--text-3); }
.empty-cell { text-align: center; color: var(--text-4); padding: 36px 0; }

.pager { display: flex; gap: 10px; align-items: center; justify-content: flex-end; }
.page-no { font-size: 12.5px; color: var(--text-3); }
.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover:not(:disabled) { background: var(--bg); }
.btn:disabled { opacity: .45; cursor: not-allowed; }
.btn-primary { background: var(--primary); color: white; border-color: var(--primary); }
.btn-primary:hover { background: var(--primary-hover); }
.btn-sm { padding: 5px 11px; font-size: 12px; }
.input { padding: 7px 11px; border: 1px solid var(--border-strong); border-radius: 6px; font-size: 13px; background: var(--surface); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }

.drawer-overlay { position: fixed; inset: 0; background: rgba(13,27,46,.4); z-index: 110; display: flex; justify-content: flex-end; backdrop-filter: blur(2px); }
.drawer { width: min(520px, 94vw); height: 100%; background: var(--surface); box-shadow: var(--sh-4); padding: 20px 22px; overflow: auto; display: flex; flex-direction: column; gap: 16px; }
.drawer-head { display: flex; justify-content: space-between; align-items: center; }
.drawer-head h3 { font-size: 16px; font-weight: 600; margin: 0; }
.x { background: transparent; border: 0; font-size: 16px; color: var(--text-3); cursor: pointer; }
.kv { display: grid; grid-template-columns: 90px 1fr; gap: 8px 14px; margin: 0; font-size: 13px; }
.kv dt { color: var(--text-3); }
.kv dd { margin: 0; color: var(--text); word-break: break-all; }
.detail-block { display: flex; flex-direction: column; gap: 6px; }
.block-label { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; }
.detail-full {
  background: var(--bg-deep); padding: 12px; border-radius: 8px; margin: 0;
  font-family: var(--ff-mono); font-size: 11.5px; line-height: 1.6; color: var(--text);
  white-space: pre-wrap; word-break: break-all;
}
</style>
