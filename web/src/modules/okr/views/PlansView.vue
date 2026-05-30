<template>
  <section class="plans">
    <header class="page-header">
      <div>
        <h1 class="page-title">工作安排 · OKR</h1>
        <div class="page-subtitle">按年 / 半年 / 月 / 周分层管理目标</div>
      </div>
      <button class="btn btn-primary" @click="showCreate = !showCreate">
        {{ showCreate ? "取消" : "+ 新建计划" }}
      </button>
    </header>

    <div v-if="showCreate" class="card create-card">
      <div class="card-head">
        <span class="card-title">新建计划</span>
        <span class="badge">draft</span>
      </div>
      <form @submit.prevent="create" class="form-grid">
        <label class="field">
          <span class="label">层级</span>
          <select class="select" v-model="form.level" required>
            <option value="year">年度</option>
            <option value="half_year">半年</option>
            <option value="month">月度</option>
            <option value="week">周度</option>
          </select>
        </label>
        <label class="field field-wide">
          <span class="label">目标</span>
          <input class="input" v-model="form.title" placeholder="例如：完成 v1.1 上线" required />
        </label>
        <label class="field">
          <span class="label">起始日期</span>
          <input class="input" v-model="form.from" type="date" required />
        </label>
        <label class="field">
          <span class="label">结束日期</span>
          <input class="input" v-model="form.to" type="date" required />
        </label>
        <div class="form-actions">
          <button class="btn" type="button" @click="showCreate = false">取消</button>
          <button class="btn btn-primary" type="submit" :disabled="loading">
            {{ loading ? "提交中…" : "创建" }}
          </button>
        </div>
      </form>
    </div>

    <div class="card filter-bar">
      <div class="filter-group">
        <span class="label">筛选层级</span>
        <div class="seg">
          <button v-for="opt in levelOpts" :key="opt.v"
                  :class="['seg-btn', { active: filterLevel === opt.v }]"
                  @click="filterLevel = opt.v; reload()">
            {{ opt.label }}
          </button>
        </div>
      </div>
      <div class="filter-meta">共 <strong>{{ plans.length }}</strong> 条</div>
    </div>

    <div v-if="plans.length === 0" class="empty">
      <div class="empty-icon">◎</div>
      <div class="empty-title">暂无计划</div>
      <div class="empty-sub">点击上方「新建计划」开始</div>
    </div>

    <div v-else class="plan-list">
      <article v-for="p in plans" :key="p.id" class="card plan-card">
        <div class="plan-head">
          <span :class="['stage-chip', 'level-' + p.level]">
            <span class="dot"></span>{{ levelLabel(p.level) }}
          </span>
          <h3 class="plan-title">{{ p.title }}</h3>
          <span class="plan-period">
            {{ formatDate(p.period.start) }} → {{ formatDate(p.period.end) }}
          </span>
          <span :class="['badge', 'status-badge', 'status-' + p.status]">{{ statusLabel(p.status) }}</span>
        </div>

        <div class="plan-items" v-if="p.items?.length">
          <div v-for="it in p.items" :key="it.id" class="plan-item">
            <div class="item-weight">{{ it.weight }}%</div>
            <div class="item-body">
              <div class="item-title">{{ it.title }}</div>
              <div class="bar-track">
                <div class="bar-fill" :style="{ width: it.progress_pct + '%', background: progressColor(it.progress_pct) }"></div>
              </div>
            </div>
            <div class="item-progress">{{ it.progress_pct }}%</div>
            <span :class="['badge', 'badge-square', 'item-status', 'item-' + it.status]">{{ it.status }}</span>
          </div>
        </div>
        <div v-else class="plan-empty">尚未添加条目</div>

        <details class="add-item">
          <summary>+ 添加条目</summary>
          <form class="add-item-form" @submit.prevent="addItem(p)">
            <input class="input" v-model="newItem[p.id].title" placeholder="条目标题" required />
            <input class="input" v-model.number="newItem[p.id].weight" type="number" min="1" max="100" placeholder="权重 %" required />
            <button class="btn btn-primary" type="submit">添加</button>
          </form>
        </details>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from "vue";
import { addItem as apiAddItem, createPlan, listPlans, type Plan } from "../api/okr";
import { useNotification } from "@/shell/notify";

const notify = useNotification();

const plans = ref<Plan[]>([]);
const filterLevel = ref("");
const showCreate = ref(false);
const loading = ref(false);
const newItem = reactive<Record<string, { title: string; weight: number }>>({});

const levelOpts = [
  { v: "", label: "全部" },
  { v: "year", label: "年度" },
  { v: "half_year", label: "半年" },
  { v: "month", label: "月度" },
  { v: "week", label: "周度" },
];

const form = reactive({ level: "week", title: "", from: today(), to: addDays(today(), 7) });

onMounted(reload);
watch(plans, (list) => list.forEach((p) => (newItem[p.id] ??= { title: "", weight: 0 })));

async function reload() {
  plans.value = await listPlans(filterLevel.value || undefined);
}

async function create() {
  loading.value = true;
  try {
    await createPlan({ level: form.level, from: form.from, to: form.to, title: form.title });
    form.title = "";
    showCreate.value = false;
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "创建失败"); }
  finally { loading.value = false; }
}

async function addItem(p: Plan) {
  const draft = newItem[p.id];
  if (!draft?.title) return;
  try {
    await apiAddItem(p.id, draft.title, draft.weight);
    draft.title = ""; draft.weight = 0;
    await reload();
  } catch (e: any) { notify.error(e.response?.data?.error?.message ?? "添加失败"); }
}

function today() { return new Date().toISOString().slice(0, 10); }
function addDays(d: string, n: number) {
  const dt = new Date(d); dt.setDate(dt.getDate() + n);
  return dt.toISOString().slice(0, 10);
}
function formatDate(s: string) { return s.slice(0, 10); }
function levelLabel(l: string) { return { year: "年度", half_year: "半年", month: "月度", week: "周度" }[l] ?? l; }
function statusLabel(s: string) { return { draft: "草稿", active: "进行中", closed: "已关闭" }[s] ?? s; }
function progressColor(p: number) {
  if (p >= 100) return "var(--success)";
  if (p >= 50) return "var(--info)";
  if (p > 0) return "var(--warning)";
  return "var(--text-4)";
}
</script>

<style scoped>
.page-header {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  margin-bottom: var(--sp-6);
}
.page-title { font-size: 24px; font-weight: 700; letter-spacing: -0.01em; }
.page-subtitle { font-size: 13px; color: var(--text-3); margin-top: 4px; }

.create-card { margin-bottom: var(--sp-4); box-shadow: var(--sh-2); }
.form-grid {
  padding: var(--sp-5);
  display: grid;
  grid-template-columns: 1fr 2fr 1fr 1fr;
  gap: var(--sp-4);
  align-items: end;
}
.field { display: flex; flex-direction: column; }
.field-wide { grid-column: span 1; }
.form-actions {
  grid-column: 1 / -1;
  display: flex;
  gap: var(--sp-2);
  justify-content: flex-end;
}

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

.seg {
  display: inline-flex;
  background: var(--surface-3);
  border-radius: var(--r-pill);
  padding: 3px;
  gap: 2px;
}
.seg-btn {
  padding: 5px 12px;
  font-size: 12.5px;
  font-weight: 500;
  border: 0;
  background: transparent;
  border-radius: var(--r-pill);
  color: var(--text-2);
  cursor: pointer;
  transition: all 0.15s;
}
.seg-btn:hover { color: var(--text); }
.seg-btn.active { background: var(--surface); color: var(--primary); font-weight: 600; box-shadow: var(--sh-1); }

.empty {
  text-align: center;
  padding: var(--sp-9) 0;
  color: var(--text-3);
}
.empty-icon { font-size: 40px; color: var(--text-4); margin-bottom: var(--sp-3); }
.empty-title { font-size: 15px; font-weight: 600; color: var(--text-2); }
.empty-sub { font-size: 12.5px; margin-top: 4px; }

.plan-list { display: flex; flex-direction: column; gap: var(--sp-3); }
.plan-card {
  padding: var(--sp-5);
  transition: all 0.15s;
}
.plan-card:hover { box-shadow: var(--sh-2); border-color: var(--border-strong); }

.plan-head {
  display: flex;
  align-items: center;
  gap: var(--sp-3);
  margin-bottom: var(--sp-3);
}
.plan-title { font-size: 15px; font-weight: 600; flex: 1; }
.plan-period { font-size: 12px; color: var(--text-3); font-family: var(--ff-mono); }

.level-year { background: rgba(30,95,217,.12); color: var(--info); }
.level-half_year { background: rgba(124,77,219,.12); color: var(--purple); }
.level-month { background: rgba(232,146,14,.12); color: var(--warning); }
.level-week { background: rgba(42,136,86,.12); color: var(--stage-consult); }

.status-badge { font-size: 11px; }
.status-draft { background: var(--neutral-soft); color: var(--neutral); }
.status-active { background: var(--info-soft); color: var(--info); }
.status-closed { background: var(--success-soft); color: var(--success); }

.plan-items {
  border-top: 1px solid var(--border-soft);
  padding-top: var(--sp-3);
  display: flex;
  flex-direction: column;
  gap: var(--sp-2);
}
.plan-item {
  display: grid;
  grid-template-columns: 50px 1fr 50px 60px;
  gap: var(--sp-3);
  align-items: center;
  padding: var(--sp-2) 0;
}
.item-weight {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-2);
  text-align: right;
}
.item-title { font-size: 13px; margin-bottom: 4px; }
.item-progress { font-size: 12px; color: var(--text-2); text-align: right; font-family: var(--ff-mono); }
.item-status { font-size: 10.5px; padding: 2px 6px; }
.item-todo { background: var(--neutral-soft); color: var(--neutral); }
.item-doing { background: var(--info-soft); color: var(--info); }
.item-done { background: var(--success-soft); color: var(--success); }
.item-blocked { background: var(--danger-soft); color: var(--danger); }

.plan-empty {
  padding: var(--sp-3) 0;
  font-size: 12.5px;
  color: var(--text-3);
  border-top: 1px solid var(--border-soft);
}

.add-item {
  margin-top: var(--sp-3);
  border-top: 1px solid var(--border-soft);
  padding-top: var(--sp-3);
}
.add-item summary {
  font-size: 12.5px;
  color: var(--primary);
  cursor: pointer;
  font-weight: 500;
  list-style: none;
}
.add-item summary::-webkit-details-marker { display: none; }
.add-item-form {
  display: grid;
  grid-template-columns: 2fr 100px auto;
  gap: var(--sp-2);
  margin-top: var(--sp-3);
}
</style>
