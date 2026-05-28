<template>
  <section class="rollup">
    <header class="head">
      <h1>周报汇总</h1>
      <input v-model="week" type="date" @change="load" />
    </header>
    <p v-if="rows.length === 0" class="muted">该周暂无数据 (或当前租户无成员).</p>
    <table v-else>
      <thead>
        <tr>
          <th>部门</th>
          <th>成员</th>
          <th>已提交</th>
          <th>摘要</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="r in rows" :key="r.member_id" :class="{ overdue: !r.submitted }">
          <td>{{ r.department || "(未填)" }}</td>
          <td>{{ r.owner_name }}</td>
          <td>
            <span v-if="r.submitted" class="ok">✓</span>
            <span v-else class="warn">未提交</span>
          </td>
          <td>{{ r.summary || "" }}</td>
        </tr>
      </tbody>
    </table>
    <div class="stats" v-if="rows.length > 0">
      <span>共 <strong>{{ rows.length }}</strong> 名成员</span>
      <span>已提交: <strong class="ok">{{ submitted }}</strong></span>
      <span>未提交: <strong class="warn">{{ rows.length - submitted }}</strong></span>
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
.rollup { max-width: 1100px; margin: 0 auto; }
.head { display: flex; justify-content: space-between; margin-bottom: var(--space-4); align-items: center; }
.head h1 { font-size: 20px; }
.head input { padding: var(--space-2); border: 1px solid var(--color-border); border-radius: 4px; }
table { width: 100%; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius); border-collapse: collapse; overflow: hidden; }
th, td { padding: var(--space-2) var(--space-3); text-align: left; border-bottom: 1px solid var(--color-border); font-size: 14px; }
th { background: var(--color-bg); color: var(--color-text-muted); font-weight: 600; font-size: 13px; }
tr.overdue { background: #fff8e6; }
.ok { color: var(--color-success); font-weight: 600; }
.warn { color: #c67200; font-weight: 600; }
.muted { color: var(--color-text-muted); }
.stats { margin-top: var(--space-3); display: flex; gap: var(--space-6); font-size: 14px; color: var(--color-text-muted); }
.stats strong { color: var(--color-text); }
.stats strong.ok { color: var(--color-success); }
.stats strong.warn { color: #c67200; }
</style>
