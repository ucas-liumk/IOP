<template>
  <section class="admin-page">
    <PageHeader title="字典管理" sub="字典类型（左）+ 字典数据（右）· 平台默认值只读，租户级覆盖可编辑">
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

      <!-- Dict data table -->
      <article class="card data-pane">
        <div class="pane-head data-head">
          <span><code>{{ active }}</code></span>
          <span class="meta">{{ data?.items.length ?? 0 }} 项 · {{ overrideCount }} 已覆盖</span>
        </div>
        <EmptyState v-if="!active" title="选择一个字典类型" sub="从左侧选择字典类型查看与编辑其字典数据。" icon="◷" />
        <table v-else-if="data" class="data-table">
          <thead>
            <tr>
              <th style="width: 140px">编码</th>
              <th style="width: 150px">默认名称</th>
              <th>租户覆盖名称</th>
              <th style="width: 90px">排序</th>
              <th style="width: 70px">启用</th>
              <th style="width: 140px"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in data.items" :key="it.code">
              <td><code>{{ it.code }}</code></td>
              <td class="muted">{{ it.name }}</td>
              <td>
                <input class="input input-sm" v-model="editing[it.code].name" :placeholder="it.name" />
              </td>
              <td>
                <input type="number" min="0" class="input input-sm sort-input" v-model.number="editing[it.code].sort_order" />
              </td>
              <td><input type="checkbox" v-model="editing[it.code].active" /></td>
              <td class="actions">
                <button class="link-btn" v-perm="'dict:write'" @click="save(it.code)">保存</button>
                <button class="link-btn warn" v-perm="'dict:write'" v-if="data.overrides[it.code]" @click="clear(it.code)">恢复默认</button>
              </td>
            </tr>
            <tr v-if="data.items.length === 0">
              <td colspan="6" class="empty-cell">该字典类型暂无数据项</td>
            </tr>
          </tbody>
        </table>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref } from "vue";
import { PageHeader, EmptyState } from "@/shell/components";
import { listDictTypes, getDictType, setDictOverride, clearDictOverride, type DictTypeItems } from "../api/admin";
import { useNotification } from "@/shell/notify";

const notify = useNotification();

const types = ref<string[]>([]);
const active = ref("");
const data = ref<DictTypeItems | null>(null);
const editing = reactive<Record<string, { name: string; sort_order: number; active: boolean }>>({});

const overrideCount = computed(() => Object.keys(data.value?.overrides ?? {}).length);

onMounted(load);

async function load() {
  types.value = await listDictTypes();
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
  data.value = await getDictType(active.value);
  for (const k of Object.keys(editing)) delete editing[k];
  for (const it of data.value.items) {
    const o = data.value.overrides[it.code];
    editing[it.code] = {
      name: o?.name ?? "",
      sort_order: o?.sort_order ?? it.sort_order,
      active: o?.active ?? it.active,
    };
  }
}

async function save(code: string) {
  const e = editing[code];
  if (!e.name) { notify.warning("覆盖名称必填（留空请点恢复默认）"); return; }
  try {
    await setDictOverride(active.value, code, { name: e.name, sort_order: e.sort_order, active: e.active });
    notify.success("已保存覆盖");
    await loadType();
  } catch (err: any) { notify.error(err.response?.data?.error?.message ?? "保存失败"); }
}

async function clear(code: string) {
  try {
    await clearDictOverride(active.value, code);
    notify.success("已恢复默认");
    await loadType();
  } catch (err: any) { notify.error(err.response?.data?.error?.message ?? "操作失败"); }
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
.data-table td { padding: 9px 14px; font-size: 13px; border-bottom: 1px solid var(--border-soft); vertical-align: middle; }
.data-table tbody tr:last-child td { border-bottom: 0; }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; color: var(--primary); }
td.muted { color: var(--text-3); }
.input { padding: 5px 8px; border: 1px solid var(--border-strong); border-radius: 5px; font-size: 12.5px; background: var(--surface); }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input-sm { width: 100%; }
.sort-input { width: 70px; }
.actions { white-space: nowrap; }
.link-btn { background: transparent; border: 0; font-size: 12px; color: var(--primary); cursor: pointer; padding: 3px 6px; border-radius: 3px; }
.link-btn:hover { background: var(--primary-soft); }
.link-btn.warn { color: var(--warning); }
.link-btn.warn:hover { background: var(--warning-soft); }
.empty-cell { text-align: center; color: var(--text-4); padding: 36px 0; }

.btn { padding: 7px 14px; font-size: 13px; border: 1px solid var(--border); border-radius: 7px; background: var(--surface); cursor: pointer; }
.btn:hover { background: var(--bg); }
.btn-ghost { background: var(--surface); }
</style>
