<template>
  <section class="rollup">
    <header class="page-header">
      <div>
        <h1 class="page-title">周报汇总</h1>
        <div class="page-subtitle">按部门展示本周提交状态</div>
      </div>
      <div class="date-picker">
        <span class="label">查询周</span>
        <input class="input input-sm" v-model="week" type="date" @change="load" />
      </div>
    </header>

    <div class="stat-row">
      <div class="card card-pad stat">
        <div class="stat-label">总成员</div>
        <div class="stat-value">{{ rows.length }}</div>
      </div>
      <div class="card card-pad stat stat-ok">
        <div class="stat-label">已提交</div>
        <div class="stat-value">{{ submitted }}</div>
      </div>
      <div class="card card-pad stat stat-warn">
        <div class="stat-label">未提交</div>
        <div class="stat-value">{{ rows.length - submitted }}</div>
      </div>
      <div class="card card-pad stat stat-rate">
        <div class="stat-label">完成率</div>
        <div class="stat-value">
          {{ rows.length ? Math.round((submitted / rows.length) * 100) : 0 }}<span class="unit">%</span>
        </div>
      </div>
    </div>

    <div v-if="rows.length === 0" class="card empty-card">
      <div class="empty-icon">▲</div>
      <div class="empty-title">该周暂无数据</div>
      <div class="empty-sub">当前租户内还没有成员，或所选周没有报告记录</div>
    </div>

    <div v-else class="card table-card">
      <div class="card-head">
        <span class="card-title">提交明细</span>
        <span class="badge">week of {{ week }}</span>
      </div>
      <table class="data-table">
        <thead>
          <tr>
            <th style="width: 120px">部门</th>
            <th style="width: 140px">成员</th>
            <th style="width: 90px">提交状态</th>
            <th>摘要</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="r in rows" :key="r.member_id" :class="{ overdue: !r.submitted }">
            <td>
              <span class="dept-badge">{{ r.department || "未分配" }}</span>
            </td>
            <td>
              <div class="member">
                <div class="member-avatar">{{ r.owner_name?.[0] ?? '?' }}</div>
                <span>{{ r.owner_name }}</span>
              </div>
            </td>
            <td>
              <span v-if="r.submitted" class="badge badge-success badge-dot">已提交</span>
              <span v-else class="badge badge-warning badge-dot">未提交</span>
            </td>
            <td class="summary-cell">{{ r.summary || "—" }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { rollupWeekly, type RollupRow } from "../api/okr";

const week = ref(new Date().toISOString().slice(0, 10));
const rows = ref<RollupRow[]>([]);

const submitted = computed(() => rows.value.filter((r) => r.submitted).length);

onMounted(load);
async function load() { rows.value = await rollupWeekly(week.value); }
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: var(--sp-6); }
.page-title { font-size: 24px; font-weight: 700; letter-spacing: -0.01em; }
.page-subtitle { font-size: 13px; color: var(--text-3); margin-top: 4px; }

.date-picker { display: flex; gap: var(--sp-2); align-items: center; }
.date-picker .label { margin-bottom: 0; }
.input-sm { width: 160px; }

.stat-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--sp-3);
  margin-bottom: var(--sp-5);
}
.stat-label { font-size: 12px; color: var(--text-3); font-weight: 500; margin-bottom: var(--sp-2); }
.stat-value { font-size: 28px; font-weight: 700; letter-spacing: -0.02em; }
.stat .unit { font-size: 14px; color: var(--text-3); font-weight: 600; margin-left: 2px; }
.stat-ok .stat-value { color: var(--success); }
.stat-warn .stat-value { color: var(--warning); }
.stat-rate .stat-value { color: var(--primary); }

.empty-card { padding: var(--sp-9) 0; text-align: center; }
.empty-icon { font-size: 40px; color: var(--text-4); margin-bottom: var(--sp-3); }
.empty-title { font-size: 15px; font-weight: 600; color: var(--text-2); }
.empty-sub { font-size: 12.5px; color: var(--text-3); margin-top: 4px; }

.table-card { overflow: hidden; }
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th {
  text-align: left;
  font-size: 12px;
  font-weight: 600;
  color: var(--text-3);
  padding: var(--sp-3) var(--sp-4);
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
}
.data-table td {
  padding: var(--sp-3) var(--sp-4);
  font-size: 13.5px;
  border-bottom: 1px solid var(--border-soft);
  vertical-align: middle;
}
.data-table tbody tr:last-child td { border-bottom: 0; }
.data-table tbody tr:hover { background: var(--surface-2); }
.data-table tr.overdue { background: rgba(232, 146, 14, 0.04); }

.dept-badge {
  display: inline-block;
  padding: 2px 8px;
  background: var(--surface-3);
  border: 1px solid var(--border);
  border-radius: var(--r-pill);
  font-size: 11.5px;
  font-weight: 500;
  color: var(--text-2);
}

.member { display: flex; align-items: center; gap: var(--sp-2); }
.member-avatar {
  width: 24px;
  height: 24px;
  border-radius: 999px;
  background: linear-gradient(135deg, var(--primary) 0%, var(--purple) 100%);
  color: white;
  display: flex; align-items: center; justify-content: center;
  font-size: 11px;
  font-weight: 600;
}

.summary-cell {
  color: var(--text);
  font-size: 13px;
  line-height: 1.55;
  max-width: 480px;
}
</style>
