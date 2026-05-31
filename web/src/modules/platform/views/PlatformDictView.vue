<template>
  <section class="admin-page">
    <PageHeader title="字典管理" sub="平台级字典类型（左）+ 字典数据（右）· 平台台为共享字典只读视图，租户级覆盖在组织控制台编辑">
      <template #actions>
        <button class="btn btn-ghost" @click="load">刷新</button>
      </template>
    </PageHeader>

    <div class="split">
      <!-- Dict type list -->
      <article class="card list-pane">
        <div class="pane-head">字典类型 · {{ types.length }}</div>
        <ul class="type-list">
          <li
            v-for="t in types"
            :key="t"
            class="type-item"
            :class="{ active: t === active }"
            @click="select(t)"
          >
            <code class="ti-code">{{ t }}</code>
          </li>
          <li v-if="types.length === 0" class="muted">尚无字典类型</li>
        </ul>
      </article>

      <!-- Dict data table (read-only at platform scope) -->
      <article class="card data-pane">
        <div class="pane-head data-head">
          <span><code>{{ active }}</code></span>
          <span class="meta">{{ items.length }} 项 · 只读</span>
        </div>
        <EmptyState v-if="!active" title="选择一个字典类型" sub="从左侧选择字典类型查看其字典数据。" icon="◷" />
        <table v-else class="data-table">
          <thead>
            <tr>
              <th style="width: 180px">编码</th>
              <th>名称</th>
              <th style="width: 90px">排序</th>
              <th style="width: 80px">启用</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in items" :key="it.code">
              <td><code>{{ it.code }}</code></td>
              <td>{{ it.name }}</td>
              <td class="muted">{{ it.sort_order }}</td>
              <td>
                <span class="badge" :class="it.active ? 'on' : 'off'">{{ it.active ? '启用' : '停用' }}</span>
              </td>
            </tr>
            <tr v-if="items.length === 0">
              <td colspan="4" class="empty-cell">{{ loading ? '加载中…' : '该字典类型暂无数据项' }}</td>
            </tr>
          </tbody>
        </table>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { onMounted, ref } from "vue";
import { PageHeader, EmptyState } from "@/shell/components";
import { PLATFORM_DICT_TYPES, lookupDict, type DictItem } from "../api/system";

const types = ref<string[]>([]);
const active = ref("");
const items = ref<DictItem[]>([]);
const loading = ref(false);

onMounted(load);

async function load() {
  types.value = [...PLATFORM_DICT_TYPES];
  if (!active.value || !types.value.includes(active.value)) {
    active.value = types.value[0] ?? "";
  }
  if (active.value) await loadType();
}

function select(t: string) {
  active.value = t;
  loadType();
}

async function loadType() {
  loading.value = true;
  try {
    items.value = await lookupDict(active.value);
  } finally {
    loading.value = false;
  }
}
</script>

<style scoped>
.admin-page { display: flex; flex-direction: column; gap: var(--sp-5); }
.split { display: grid; grid-template-columns: 240px 1fr; gap: 16px; align-items: start; }
.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; }
.list-pane { padding: 12px; }
.pane-head { font-size: 11.5px; font-weight: 600; color: var(--text-3); text-transform: uppercase; letter-spacing: .5px; padding: 4px 8px 8px; }
.data-pane { overflow: hidden; }
.data-head { display: flex; justify-content: space-between; align-items: center; padding: 14px 16px; border-bottom: 1px solid var(--border); }
.data-head .meta { font-size: 12px; color: var(--text-3); text-transform: none; letter-spacing: 0; }

.type-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }
.type-item { padding: 9px 10px; border-radius: 7px; cursor: pointer; }
.type-item:hover { background: var(--surface-2); }
.type-item.active { background: var(--primary-soft); }
.type-item.active .ti-code { color: var(--primary); }
.ti-code { font-family: var(--ff-mono); font-size: 12px; color: var(--text-2); background: transparent; padding: 0; }
.muted { color: var(--text-4); font-size: 12.5px; padding: 8px; }

.data-table { width: 100%; border-collapse: collapse; }
.data-table th {
  text-align: left; font-size: 11.5px; font-weight: 600; color: var(--text-3);
  text-transform: uppercase; letter-spacing: .5px; padding: 10px 14px;
  background: var(--surface-2); border-bottom: 1px solid var(--border);
}
.data-table td { padding: 10px 14px; font-size: 13px; border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
.data-table tbody tr:last-child td { border-bottom: 0; }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; color: var(--primary); }
td.muted { color: var(--text-3); }
.badge { font-size: 11px; font-weight: 600; padding: 2px 9px; border-radius: 999px; }
.badge.on { background: var(--success-soft); color: var(--success); }
.badge.off { background: var(--surface-2); color: var(--text-3); }
.empty-cell { text-align: center; color: var(--text-4); padding: 36px 0; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-ghost { background: var(--surface); }
</style>
