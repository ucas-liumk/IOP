<template>
  <section>
    <header class="page-head">
      <div>
        <h1>审计日志</h1>
        <p class="sub">最近 20 条事件 · 异步收集，可能有 5-10 秒延迟</p>
      </div>
      <button class="btn" @click="reload">↻ 刷新</button>
    </header>

    <article class="card">
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 130px">时间</th>
            <th style="width: 180px">动作</th>
            <th style="width: 130px">执行者</th>
            <th style="width: 110px">资源</th>
            <th>详情</th>
            <th style="width: 80px">追踪 ID</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="a in entries" :key="a.id" @click="expand = expand === a.id ? '' : a.id">
            <td class="time">{{ a.occurred_at?.slice(5, 16) }}</td>
            <td><code class="action">{{ a.action }}</code></td>
            <td>{{ a.actor === 'system' ? '系统' : a.actor.slice(0, 8) + '…' }}</td>
            <td class="muted">{{ a.resource || '—' }}</td>
            <td class="detail">
              <span v-if="expand !== a.id" class="detail-preview">{{ detailPreview(a.detail) }}</span>
              <pre v-else class="detail-full">{{ JSON.stringify(parsedDetail(a.detail), null, 2) }}</pre>
            </td>
            <td class="trace"><code>{{ a.trace_id?.slice(0, 8) }}</code></td>
          </tr>
          <tr v-if="entries.length === 0"><td colspan="6" class="empty">暂无审计记录 · 跑几个动作（创建/登录）再来看</td></tr>
        </tbody>
      </table>
    </article>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { listAudit, type AuditEntry } from "../api/admin";

const entries = ref<AuditEntry[]>([]);
const expand = ref("");

onMounted(reload);
async function reload() { entries.value = await listAudit(); }

function parsedDetail(d: any) {
  if (typeof d === "string") { try { return JSON.parse(d); } catch { return d; } }
  if (d instanceof Uint8Array || Array.isArray(d)) {
    try { return JSON.parse(new TextDecoder().decode(new Uint8Array(d))); } catch { return d; }
  }
  return d;
}
function detailPreview(d: any) {
  const p = parsedDetail(d);
  if (!p) return "";
  if (typeof p === "string") return p.slice(0, 80);
  const s = JSON.stringify(p);
  return s.length > 80 ? s.slice(0, 80) + "…" : s;
}
</script>

<style scoped>
.page-head { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.btn { padding: 6px 12px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); }
.btn:hover { background: var(--bg); }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }

.data-table { width: 100%; border-collapse: collapse; }
.data-table th {
  text-align: left;
  font-size: 11.5px; font-weight: 600;
  color: var(--text-3); text-transform: uppercase; letter-spacing: .5px;
  padding: 10px 14px;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
}
.data-table td {
  padding: 11px 14px;
  font-size: 12.5px;
  border-bottom: 1px solid var(--border-soft);
  vertical-align: top;
  cursor: pointer;
}
.data-table tbody tr:hover { background: var(--surface-2); }
.data-table tbody tr:last-child td { border-bottom: 0; }
.time { font-family: var(--ff-mono); color: var(--text-2); }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; }
code.action { color: var(--primary); font-weight: 600; }
.muted { color: var(--text-3); }
.detail-preview { color: var(--text-2); font-family: var(--ff-mono); font-size: 11.5px; }
.detail-full {
  background: var(--bg-deep);
  padding: 10px;
  border-radius: 6px;
  font-family: var(--ff-mono);
  font-size: 11.5px;
  line-height: 1.55;
  color: var(--text);
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}
.trace code { color: var(--text-3); }
.empty { text-align: center; color: var(--text-4); padding: 36px 0; }
</style>
