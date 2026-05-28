<template>
  <section class="reports">
    <header class="page-header">
      <div>
        <h1 class="page-title">日报 / 周报</h1>
        <div class="page-subtitle">记录每日工作，每周汇总</div>
      </div>
      <div class="header-actions">
        <button class="btn" :class="{ 'btn-primary': showDaily }" @click="toggle('daily')">
          {{ showDaily ? '关闭' : '+ 写日报' }}
        </button>
        <button class="btn" :class="{ 'btn-primary': showWeekly }" @click="toggle('weekly')">
          {{ showWeekly ? '关闭' : '+ 写周报' }}
        </button>
      </div>
    </header>

    <div class="form-row">
      <div v-if="showDaily" class="card form-card">
        <div class="card-head">
          <span class="card-title">日报</span>
          <span class="badge badge-warning">today</span>
        </div>
        <form @submit.prevent="postDaily" class="card-pad form-body">
          <label class="field">
            <span class="label">日期</span>
            <input class="input" v-model="daily.day" type="date" required />
          </label>
          <label class="field">
            <span class="label">今日工作总结</span>
            <textarea class="textarea" v-model="daily.summary" placeholder="今天完成了什么 / 遇到什么问题 / 明天计划做什么" required></textarea>
          </label>
          <button class="btn btn-primary btn-block" :disabled="loading">
            {{ loading ? "提交中…" : "提交日报" }}
          </button>
        </form>
      </div>

      <div v-if="showWeekly" class="card form-card">
        <div class="card-head">
          <span class="card-title">周报</span>
          <span class="badge badge-info">this week</span>
        </div>
        <form @submit.prevent="postWeekly" class="card-pad form-body">
          <label class="field">
            <span class="label">本周内任一日期</span>
            <input class="input" v-model="weekly.week_contains" type="date" required />
          </label>
          <label class="field">
            <span class="label">本周总结</span>
            <textarea class="textarea" v-model="weekly.summary" placeholder="本周完成 / 下周计划 / 风险阻塞" required></textarea>
          </label>
          <button class="btn btn-primary btn-block" :disabled="loading">
            {{ loading ? "提交中…" : "提交周报" }}
          </button>
        </form>
      </div>
    </div>

    <div class="card filter-bar">
      <div class="filter-group">
        <span class="label">筛选类型</span>
        <div class="seg">
          <button :class="['seg-btn', { active: filterType === '' }]" @click="filterType = ''; reload()">全部</button>
          <button :class="['seg-btn', { active: filterType === 'daily' }]" @click="filterType = 'daily'; reload()">日报</button>
          <button :class="['seg-btn', { active: filterType === 'weekly' }]" @click="filterType = 'weekly'; reload()">周报</button>
        </div>
      </div>
      <div class="filter-meta">共 <strong>{{ reports.length }}</strong> 条</div>
    </div>

    <div v-if="reports.length === 0" class="empty">
      <div class="empty-icon">✎</div>
      <div class="empty-title">暂无报告</div>
      <div class="empty-sub">点击上方按钮提交今日日报或本周周报</div>
    </div>

    <div v-else class="report-list">
      <article v-for="r in reports" :key="r.id" class="card report-card">
        <div class="report-head">
          <span :class="['stage-chip', 'type-' + r.type]">
            <span class="dot"></span>{{ r.type === 'daily' ? '日报' : '周报' }}
          </span>
          <span class="report-period">
            {{ r.period.start.slice(0,10) }}
            <span v-if="r.type === 'weekly'"> → {{ r.period.end.slice(0,10) }}</span>
          </span>
          <span class="report-submit">提交于 {{ new Date(r.submitted_at).toLocaleString('zh-CN') }}</span>
        </div>
        <div class="report-body">{{ r.summary }}</div>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from "vue";
import { listReports, submitDaily, submitWeekly, type Report } from "../api/okr";

const reports = ref<Report[]>([]);
const filterType = ref("");
const showDaily = ref(false);
const showWeekly = ref(false);
const loading = ref(false);
const daily = reactive({ day: today(), summary: "" });
const weekly = reactive({ week_contains: today(), summary: "" });

onMounted(reload);
function toggle(which: 'daily' | 'weekly') {
  if (which === 'daily') { showDaily.value = !showDaily.value; if (showDaily.value) showWeekly.value = false; }
  else { showWeekly.value = !showWeekly.value; if (showWeekly.value) showDaily.value = false; }
}
async function reload() { reports.value = await listReports(filterType.value || undefined); }
async function postDaily() {
  loading.value = true;
  try {
    await submitDaily({ day: daily.day, summary: daily.summary, entries: [] });
    daily.summary = ""; showDaily.value = false;
    await reload();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "提交失败"); }
  finally { loading.value = false; }
}
async function postWeekly() {
  loading.value = true;
  try {
    await submitWeekly({ week_contains: weekly.week_contains, summary: weekly.summary, entries: [] });
    weekly.summary = ""; showWeekly.value = false;
    await reload();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "提交失败"); }
  finally { loading.value = false; }
}
function today() { return new Date().toISOString().slice(0, 10); }
</script>

<style scoped>
.page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: var(--sp-6); }
.page-title { font-size: 24px; font-weight: 700; letter-spacing: -0.01em; }
.page-subtitle { font-size: 13px; color: var(--text-3); margin-top: 4px; }
.header-actions { display: flex; gap: var(--sp-2); }

.form-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(380px, 1fr));
  gap: var(--sp-4);
  margin-bottom: var(--sp-4);
}
.form-card { box-shadow: var(--sh-2); }
.form-body { display: flex; flex-direction: column; gap: var(--sp-4); }
.field { display: flex; flex-direction: column; }
.btn-block { width: 100%; justify-content: center; padding: 10px; font-weight: 600; }

.filter-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--sp-3) var(--sp-5);
  margin-bottom: var(--sp-4);
}
.filter-group { display: flex; align-items: center; gap: var(--sp-3); }
.filter-meta { font-size: 13px; color: var(--text-3); }
.filter-meta strong { color: var(--text); font-weight: 600; }

.seg { display: inline-flex; background: var(--surface-3); border-radius: var(--r-pill); padding: 3px; gap: 2px; }
.seg-btn {
  padding: 5px 12px; font-size: 12.5px; font-weight: 500;
  border: 0; background: transparent; border-radius: var(--r-pill);
  color: var(--text-2); cursor: pointer; transition: all 0.15s;
}
.seg-btn:hover { color: var(--text); }
.seg-btn.active { background: var(--surface); color: var(--primary); font-weight: 600; box-shadow: var(--sh-1); }

.empty { text-align: center; padding: var(--sp-9) 0; color: var(--text-3); }
.empty-icon { font-size: 40px; color: var(--text-4); margin-bottom: var(--sp-3); }
.empty-title { font-size: 15px; font-weight: 600; color: var(--text-2); }
.empty-sub { font-size: 12.5px; margin-top: 4px; }

.report-list { display: flex; flex-direction: column; gap: var(--sp-3); }
.report-card { padding: var(--sp-4) var(--sp-5); transition: all 0.15s; }
.report-card:hover { box-shadow: var(--sh-2); border-color: var(--border-strong); }
.report-head { display: flex; align-items: center; gap: var(--sp-3); margin-bottom: var(--sp-2); }
.report-period { font-size: 12.5px; color: var(--text-2); font-family: var(--ff-mono); font-weight: 500; }
.report-submit { font-size: 11.5px; color: var(--text-3); margin-left: auto; }
.report-body { font-size: 13.5px; color: var(--text); white-space: pre-wrap; line-height: 1.65; }

.type-daily { background: var(--warning-soft); color: var(--warning); }
.type-weekly { background: var(--info-soft); color: var(--info); }
</style>
