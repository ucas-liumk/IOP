<template>
  <div class="lc-center">
    <!-- Form grid -->
    <template v-if="!current">
      <header class="lc-c-head">
        <h2>表单中心</h2>
      </header>
      <div v-if="loading" class="lc-c-loading">加载中…</div>
      <div v-else-if="forms.length === 0" class="lc-c-empty">
        <div class="lc-c-emoji">📋</div>
        <div>还没有可用的表单</div>
      </div>
      <div v-else class="lc-grid">
        <button v-for="f in forms" :key="f.id" class="lc-card" @click="open(f)">
          <span class="lc-card-ico">{{ f.icon || "📋" }}</span>
          <span class="lc-card-name">{{ f.name }}</span>
          <span class="lc-card-meta">{{ f.fields.length }} 字段 · {{ f.entry_count }} 条数据</span>
        </button>
      </div>
    </template>

    <!-- Detail: fill + entries -->
    <template v-else>
      <header class="lc-d-head">
        <button class="lc-back" @click="back">← 返回</button>
        <h2>{{ current.icon }} {{ current.name }}</h2>
        <div class="lc-d-actions">
          <button class="btn btn-ghost" @click="exportCsv">导出 CSV</button>
        </div>
      </header>

      <div class="lc-d-cols">
        <!-- Fill form -->
        <section class="lc-fill">
          <h3>填写新记录</h3>
          <form @submit.prevent="submit">
            <div v-for="f in current.fields" :key="f.key" class="lc-fill-row">
              <label class="lc-fill-label">
                {{ f.label }} <span v-if="f.required" class="lc-req">*</span>
              </label>

              <textarea v-if="f.type === 'textarea'" v-model="formData[f.key]" class="lc-input area" :placeholder="f.label"></textarea>
              <select v-else-if="f.type === 'select'" v-model="formData[f.key]" class="lc-input">
                <option value="">请选择</option>
                <option v-for="o in f.options || []" :key="o" :value="o">{{ o }}</option>
              </select>
              <label v-else-if="f.type === 'checkbox'" class="lc-cb">
                <input type="checkbox" v-model="formData[f.key]" /> {{ f.label }}
              </label>
              <input v-else-if="f.type === 'number' || f.type === 'money'" type="number" v-model.number="formData[f.key]" class="lc-input" :placeholder="f.type === 'money' ? '金额（元）' : f.label" />
              <input v-else-if="f.type === 'date'" type="date" v-model="formData[f.key]" class="lc-input" />
              <input v-else type="text" v-model="formData[f.key]" class="lc-input" :placeholder="f.label" :inputmode="f.type === 'phone' ? 'tel' : 'text'" />
            </div>
            <div v-if="current.fields.length === 0" class="lc-fill-empty">此表单还没有字段</div>
            <button v-else class="btn btn-primary" type="submit" :disabled="submitting">{{ submitting ? "提交中…" : "提交" }}</button>
          </form>
        </section>

        <!-- Entries table -->
        <section class="lc-entries">
          <div class="lc-entries-head">
            <h3>数据（{{ total }}）</h3>
            <input v-model="search" class="lc-search" placeholder="搜索数据…" @keyup.enter="reloadEntries(1)" />
          </div>
          <div class="lc-table-wrap">
            <table class="lc-table">
              <thead>
                <tr>
                  <th v-for="f in current.fields" :key="f.key">{{ f.label }}</th>
                  <th>提交时间</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="e in entries" :key="e.id">
                  <td v-for="f in current.fields" :key="f.key">{{ cell(e.data[f.key], f.type) }}</td>
                  <td class="lc-ts">{{ fmtTime(e.created_at) }}</td>
                </tr>
                <tr v-if="entries.length === 0">
                  <td :colspan="current.fields.length + 1" class="lc-no-data">暂无数据</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="lc-pager" v-if="total > pageSize">
            <button class="btn btn-ghost" :disabled="page <= 1" @click="reloadEntries(page - 1)">上一页</button>
            <span>{{ page }} / {{ totalPages }}</span>
            <button class="btn btn-ghost" :disabled="page >= totalPages" @click="reloadEntries(page + 1)">下一页</button>
          </div>
        </section>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import * as api from "../api";
import type { FormDef, FormEntry } from "../api";
import { useNotification } from "@/shell/notify";

const notify = useNotification();

const forms = ref<FormDef[]>([]);
const loading = ref(false);
const current = ref<FormDef | null>(null);

const formData = reactive<Record<string, any>>({});
const submitting = ref(false);

const entries = ref<FormEntry[]>([]);
const total = ref(0);
const page = ref(1);
const pageSize = 20;
const search = ref("");
const totalPages = computed(() => Math.max(1, Math.ceil(total.value / pageSize)));

async function refresh() {
  loading.value = true;
  try {
    forms.value = await api.listForms(false);
  } finally {
    loading.value = false;
  }
}
onMounted(refresh);

async function open(f: FormDef) {
  try {
    current.value = await api.getForm(f.id);
    resetForm();
    await reloadEntries(1);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "打开表单失败");
  }
}

function resetForm() {
  for (const k of Object.keys(formData)) delete formData[k];
  for (const f of current.value?.fields || []) {
    formData[f.key] = f.type === "checkbox" ? false : "";
  }
}

function back() {
  current.value = null;
  entries.value = [];
  refresh();
}

async function submit() {
  if (!current.value) return;
  submitting.value = true;
  try {
    const payload: Record<string, any> = {};
    for (const f of current.value.fields) {
      const v = formData[f.key];
      if (v !== "" && v !== undefined && v !== null) payload[f.key] = v;
    }
    await api.submitEntry(current.value.id, payload);
    notify.success("提交成功");
    resetForm();
    await reloadEntries(1);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "提交失败");
  } finally {
    submitting.value = false;
  }
}

async function reloadEntries(p: number) {
  if (!current.value) return;
  const res = await api.listEntries(current.value.id, { page: p, page_size: pageSize, search: search.value });
  entries.value = res.data;
  total.value = res.total;
  page.value = res.page;
}

async function exportCsv() {
  if (!current.value) return;
  try {
    await api.exportEntries(current.value.id, `${current.value.code}_entries.csv`);
  } catch (e: any) {
    notify.error(e.response?.data?.error?.message ?? "导出失败");
  }
}

function cell(v: any, type: string) {
  if (v === undefined || v === null || v === "") return "";
  if (type === "checkbox") return v ? "是" : "否";
  if (Array.isArray(v)) return v.join(", ");
  return String(v);
}
function fmtTime(iso: string) {
  const d = new Date(iso);
  const p = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())} ${p(d.getHours())}:${p(d.getMinutes())}`;
}
</script>

<style scoped>
.lc-center { padding: 4px 0; }
.lc-c-head h2, .lc-d-head h2 { font-size: 20px; font-weight: 700; color: var(--text); margin: 0; }
.lc-c-loading, .lc-c-empty { color: var(--text-3); text-align: center; padding: 50px 0; font-size: 13px; }
.lc-c-emoji { font-size: 36px; margin-bottom: 8px; }

.lc-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 14px; margin-top: 16px; }
.lc-card {
  display: flex; flex-direction: column; gap: 6px; align-items: flex-start;
  background: var(--surface); border: 1px solid var(--border); border-radius: 12px;
  padding: 18px; cursor: pointer; text-align: left; transition: border-color .15s, box-shadow .15s;
}
.lc-card:hover { border-color: var(--primary); box-shadow: 0 4px 14px rgba(13,27,46,.08); }
.lc-card-ico { font-size: 26px; }
.lc-card-name { font-size: 15px; font-weight: 600; color: var(--text); }
.lc-card-meta { font-size: 12px; color: var(--text-3); }

.lc-d-head { display: flex; align-items: center; gap: 14px; margin-bottom: 18px; }
.lc-back { border: 0; background: transparent; color: var(--primary); font-size: 13px; cursor: pointer; }
.lc-d-actions { margin-left: auto; }

.lc-d-cols { display: grid; grid-template-columns: 340px 1fr; gap: 22px; align-items: start; }
.lc-fill { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; padding: 18px; }
.lc-fill h3, .lc-entries-head h3 { font-size: 14px; font-weight: 600; color: var(--text); margin: 0 0 12px; }
.lc-fill-row { margin-bottom: 12px; display: flex; flex-direction: column; gap: 5px; }
.lc-fill-label { font-size: 12.5px; color: var(--text-2); font-weight: 500; }
.lc-req { color: var(--danger); }
.lc-input {
  border: 1px solid var(--border); border-radius: 8px; padding: 8px 10px; font-size: 13px;
  background: var(--surface); color: var(--text); outline: none; width: 100%; font-family: inherit;
}
.lc-input:focus { border-color: var(--primary); }
.lc-input.area { min-height: 64px; resize: vertical; }
.lc-cb { font-size: 13px; color: var(--text-2); display: flex; align-items: center; gap: 6px; cursor: pointer; }
.lc-fill-empty { color: var(--text-4); font-size: 13px; padding: 10px 0; }

.lc-entries { min-width: 0; }
.lc-entries-head { display: flex; align-items: center; justify-content: space-between; gap: 10px; margin-bottom: 10px; }
.lc-search { border: 1px solid var(--border); border-radius: 8px; padding: 7px 11px; font-size: 13px; background: var(--surface); color: var(--text); outline: none; width: 200px; }
.lc-search:focus { border-color: var(--primary); }
.lc-table-wrap { overflow-x: auto; border: 1px solid var(--border); border-radius: 10px; background: var(--surface); }
.lc-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.lc-table th, .lc-table td { padding: 9px 12px; text-align: left; border-bottom: 1px solid var(--border); white-space: nowrap; }
.lc-table th { font-size: 12px; font-weight: 600; color: var(--text-3); background: var(--surface-2); position: sticky; top: 0; }
.lc-table td { color: var(--text); }
.lc-table tr:last-child td { border-bottom: 0; }
.lc-ts { color: var(--text-3); font-size: 12px; }
.lc-no-data { text-align: center; color: var(--text-4); padding: 24px 0 !important; }
.lc-pager { display: flex; align-items: center; justify-content: center; gap: 14px; margin-top: 12px; font-size: 13px; color: var(--text-3); }

@media (max-width: 880px) {
  .lc-d-cols { grid-template-columns: 1fr; }
}
</style>
