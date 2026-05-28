<template>
  <section class="reports">
    <header class="head">
      <h1>报告 / 日报 + 周报</h1>
      <div>
        <button @click="showDaily = !showDaily">{{ showDaily ? "取消" : "写日报" }}</button>
        <button @click="showWeekly = !showWeekly">{{ showWeekly ? "取消" : "写周报" }}</button>
      </div>
    </header>

    <form v-if="showDaily" class="card form" @submit.prevent="postDaily">
      <h3>日报</h3>
      <input v-model="daily.day" type="date" required />
      <textarea v-model="daily.summary" placeholder="今日重点 / 总结" required></textarea>
      <button :disabled="loading">提交日报</button>
    </form>

    <form v-if="showWeekly" class="card form" @submit.prevent="postWeekly">
      <h3>周报</h3>
      <input v-model="weekly.week_contains" type="date" required />
      <textarea v-model="weekly.summary" placeholder="本周完成 / 下周计划 / 风险" required></textarea>
      <button :disabled="loading">提交周报</button>
    </form>

    <div class="filter">
      <label>类型:
        <select v-model="filterType" @change="reload">
          <option value="">全部</option>
          <option value="daily">日报</option>
          <option value="weekly">周报</option>
        </select>
      </label>
    </div>

    <ul class="list">
      <li v-for="r in reports" :key="r.id" class="card">
        <div class="row">
          <span :class="['badge', r.type]">{{ r.type === 'daily' ? '日报' : '周报' }}</span>
          <span class="muted">{{ r.period.start.slice(0,10) }} → {{ r.period.end.slice(0,10) }}</span>
          <span class="muted">提交于 {{ new Date(r.submitted_at).toLocaleString() }}</span>
        </div>
        <div class="summary">{{ r.summary }}</div>
      </li>
    </ul>
    <p v-if="reports.length === 0" class="muted">暂无报告.</p>
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

const daily = reactive({ day: today(), summary: "", entries: [] as any[] });
const weekly = reactive({ week_contains: today(), summary: "", entries: [] as any[] });

onMounted(reload);

async function reload() {
  reports.value = await listReports(filterType.value || undefined);
}

async function postDaily() {
  loading.value = true;
  try {
    await submitDaily({ day: daily.day, summary: daily.summary, entries: [] });
    daily.summary = "";
    showDaily.value = false;
    await reload();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "提交失败"); }
  finally { loading.value = false; }
}

async function postWeekly() {
  loading.value = true;
  try {
    await submitWeekly({ week_contains: weekly.week_contains, summary: weekly.summary, entries: [] });
    weekly.summary = "";
    showWeekly.value = false;
    await reload();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "提交失败"); }
  finally { loading.value = false; }
}

function today() { return new Date().toISOString().slice(0, 10); }
</script>

<style scoped>
.reports { max-width: 1000px; margin: 0 auto; }
.head { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-4); }
.head h1 { font-size: 20px; }
.head button { padding: var(--space-2) var(--space-3); margin-left: var(--space-2); background: var(--color-primary); color: white; border: 0; border-radius: var(--radius); cursor: pointer; }
.card { padding: var(--space-3); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius); margin-bottom: var(--space-3); }
.form { display: flex; flex-direction: column; gap: var(--space-2); }
.form h3 { margin-bottom: var(--space-2); }
.form input, .form textarea { padding: var(--space-2); border: 1px solid var(--color-border); border-radius: 4px; font-family: inherit; font-size: 14px; }
.form textarea { min-height: 80px; }
.form button { padding: var(--space-2); background: var(--color-primary); color: white; border: 0; border-radius: var(--radius); cursor: pointer; }
.filter { margin-bottom: var(--space-3); }
.row { display: flex; gap: var(--space-3); align-items: center; flex-wrap: wrap; }
.badge { padding: 2px 8px; border-radius: 4px; font-size: 12px; background: var(--color-bg); }
.badge.daily { background: #fff4e3; color: #c67200; }
.badge.weekly { background: #e3f2ff; color: var(--color-primary); }
.summary { margin-top: var(--space-2); white-space: pre-wrap; }
.muted { color: var(--color-text-muted); font-size: 13px; }
</style>
