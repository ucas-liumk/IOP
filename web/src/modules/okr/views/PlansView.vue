<template>
  <section class="plans">
    <header class="head">
      <h1>工作安排 / OKR</h1>
      <button @click="showCreate = !showCreate">{{ showCreate ? "取消" : "新建计划" }}</button>
    </header>

    <form v-if="showCreate" class="card create" @submit.prevent="create">
      <select v-model="form.level" required>
        <option value="year">年度</option>
        <option value="half_year">半年</option>
        <option value="month">月度</option>
        <option value="week">周度</option>
      </select>
      <input v-model="form.title" placeholder="目标标题" required />
      <input v-model="form.from" type="date" required />
      <input v-model="form.to" type="date" required />
      <button type="submit" :disabled="loading">提交</button>
    </form>

    <div class="filter">
      <label>层级:
        <select v-model="filterLevel" @change="reload">
          <option value="">全部</option>
          <option value="year">年度</option>
          <option value="half_year">半年</option>
          <option value="month">月度</option>
          <option value="week">周度</option>
        </select>
      </label>
    </div>

    <p v-if="plans.length === 0" class="muted">暂无计划. 上面新建一个开始.</p>

    <ul class="list">
      <li v-for="p in plans" :key="p.id" class="card">
        <div class="row">
          <span :class="['badge', p.level]">{{ levelLabel(p.level) }}</span>
          <strong>{{ p.title }}</strong>
          <span class="muted">{{ formatDate(p.period.start) }} → {{ formatDate(p.period.end) }}</span>
          <span :class="['status', p.status]">{{ p.status }}</span>
        </div>
        <div class="items" v-if="p.items?.length">
          <div v-for="it in p.items" :key="it.id" class="item">
            <span class="weight">{{ it.weight }}%</span>
            <span class="title">{{ it.title }}</span>
            <span class="progress">{{ it.progress_pct }}%</span>
            <span :class="['itemstatus', it.status]">{{ it.status }}</span>
          </div>
        </div>
        <details>
          <summary>添加条目</summary>
          <form @submit.prevent="addItem(p)" class="inline">
            <input v-model="newItem[p.id].title" placeholder="条目标题" required />
            <input v-model.number="newItem[p.id].weight" type="number" min="1" max="100" placeholder="权重" required />
            <button type="submit">添加</button>
          </form>
        </details>
      </li>
    </ul>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref, watch } from "vue";
import { addItem as apiAddItem, createPlan, listPlans, type Plan } from "../api/okr";

const plans = ref<Plan[]>([]);
const filterLevel = ref("");
const showCreate = ref(false);
const loading = ref(false);
const newItem = reactive<Record<string, { title: string; weight: number }>>({});

const form = reactive({
  level: "week",
  title: "",
  from: today(),
  to: addDays(today(), 7),
});

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
  } catch (e: any) {
    alert(e.response?.data?.error?.message ?? "创建失败");
  } finally { loading.value = false; }
}

async function addItem(p: Plan) {
  const draft = newItem[p.id];
  if (!draft?.title) return;
  try {
    await apiAddItem(p.id, draft.title, draft.weight);
    draft.title = ""; draft.weight = 0;
    await reload();
  } catch (e: any) { alert(e.response?.data?.error?.message ?? "添加失败"); }
}

function today() { return new Date().toISOString().slice(0, 10); }
function addDays(d: string, n: number) {
  const dt = new Date(d); dt.setDate(dt.getDate() + n);
  return dt.toISOString().slice(0, 10);
}
function formatDate(s: string) { return s.slice(0, 10); }
function levelLabel(l: string) { return { year: "年度", half_year: "半年", month: "月度", week: "周度" }[l] ?? l; }
</script>

<style scoped>
.plans { max-width: 1000px; margin: 0 auto; }
.head { display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-4); }
.head h1 { font-size: 20px; }
.head button { padding: var(--space-2) var(--space-3); background: var(--color-primary); color: white; border: 0; border-radius: var(--radius); cursor: pointer; }
.card { padding: var(--space-3); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius); margin-bottom: var(--space-3); }
.create { display: grid; grid-template-columns: 120px 1fr 140px 140px 100px; gap: var(--space-2); align-items: center; }
.create input, .create select { padding: var(--space-2); border: 1px solid var(--color-border); border-radius: 4px; }
.create button { background: var(--color-primary); color: white; padding: var(--space-2); border: 0; border-radius: var(--radius); cursor: pointer; }
.filter { margin-bottom: var(--space-3); }
.row { display: flex; gap: var(--space-3); align-items: center; }
.badge { padding: 2px 8px; border-radius: 4px; font-size: 12px; background: var(--color-bg); }
.badge.year { background: #e3f2ff; color: var(--color-primary); }
.badge.half_year { background: #d6f5e8; color: var(--color-success); }
.status { margin-left: auto; padding: 2px 8px; border-radius: 4px; font-size: 12px; background: var(--color-bg); }
.status.active { background: rgba(0, 168, 112, 0.12); color: var(--color-success); }
.status.closed { background: rgba(213, 73, 65, 0.12); color: var(--color-danger); }
.items { margin-top: var(--space-3); }
.item { display: grid; grid-template-columns: 60px 1fr 60px 80px; gap: var(--space-2); padding: var(--space-2); border-top: 1px solid var(--color-border); font-size: 13px; }
.weight, .progress { color: var(--color-text-muted); }
.itemstatus { font-size: 11px; }
details { margin-top: var(--space-2); }
summary { font-size: 13px; color: var(--color-text-muted); cursor: pointer; }
.inline { display: flex; gap: var(--space-2); margin-top: var(--space-2); }
.inline input { flex: 1; padding: var(--space-1) var(--space-2); border: 1px solid var(--color-border); border-radius: 4px; }
.inline button { padding: var(--space-1) var(--space-3); background: var(--color-bg); border: 1px solid var(--color-border); border-radius: 4px; cursor: pointer; }
.muted { color: var(--color-text-muted); }
</style>
