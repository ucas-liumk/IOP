<template>
  <section>
    <header class="page-head">
      <div>
        <h1>字典管理</h1>
        <p class="sub">平台默认值 (只读) + 租户级 override (可编辑)</p>
      </div>
    </header>

    <div class="layout">
      <aside class="types">
        <button
          v-for="t in types"
          :key="t"
          class="type-btn"
          :class="{ active: t === active }"
          @click="active = t; load()"
        >{{ t }}</button>
      </aside>

      <article class="card">
        <header class="card-head">
          <span class="card-title">{{ active }}</span>
          <span class="meta">{{ data?.items.length ?? 0 }} 项 · {{ overrideCount }} 已覆盖</span>
        </header>
        <table class="data-table" v-if="data">
          <thead>
            <tr>
              <th>编码</th>
              <th>默认名称</th>
              <th>租户覆盖名称</th>
              <th>排序</th>
              <th>启用</th>
              <th></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="it in data.items" :key="it.code">
              <td><code>{{ it.code }}</code></td>
              <td class="muted">{{ it.name }}</td>
              <td>
                <input
                  class="input input-sm"
                  v-model="editing[it.code].name"
                  :placeholder="it.name"
                />
              </td>
              <td>
                <input
                  type="number"
                  min="0"
                  class="input input-sm sort-input"
                  v-model.number="editing[it.code].sort_order"
                />
              </td>
              <td>
                <input type="checkbox" v-model="editing[it.code].active" />
              </td>
              <td class="actions">
                <button class="link-btn" @click="save(it.code)">保存</button>
                <button class="link-btn warn" v-if="data.overrides[it.code]" @click="clear(it.code)">恢复默认</button>
              </td>
            </tr>
          </tbody>
        </table>
      </article>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from "vue";
import { listDictTypes, getDictType, setDictOverride, clearDictOverride, type DictTypeItems } from "../api/admin";
import { useNotification } from "@/shell/notify";

const notify = useNotification();

const types = ref<string[]>([]);
const active = ref("plan_level");
const data = ref<DictTypeItems | null>(null);
const editing = reactive<Record<string, { name: string; sort_order: number; active: boolean }>>({});

const overrideCount = computed(() => Object.keys(data.value?.overrides ?? {}).length);

onMounted(async () => {
  types.value = await listDictTypes();
  if (!types.value.includes(active.value)) active.value = types.value[0];
  await load();
});

async function load() {
  data.value = await getDictType(active.value);
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
  if (!e.name) { notify.warning("覆盖名称必填（或留空再点恢复默认）"); return; }
  try {
    await setDictOverride(active.value, code, { name: e.name, sort_order: e.sort_order, active: e.active });
    await load();
  } catch (err: any) { notify.error(err.response?.data?.error?.message ?? "保存失败"); }
}

async function clear(code: string) {
  await clearDictOverride(active.value, code);
  await load();
}
</script>

<style scoped>
.page-head { margin-bottom: 20px; }
.page-head h1 { font-size: 22px; font-weight: 700; }
.page-head .sub { font-size: 13px; color: var(--text-3); margin-top: 4px; }

.layout { display: grid; grid-template-columns: 200px 1fr; gap: 14px; }

.types { display: flex; flex-direction: column; gap: 4px; }
.type-btn {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 11px 14px;
  text-align: left;
  font-size: 13px;
  font-weight: 500;
  color: var(--text-2);
  cursor: pointer;
  transition: all 0.12s;
  font-family: var(--ff-mono);
}
.type-btn:hover { border-color: var(--border-strong); }
.type-btn.active { background: var(--primary-soft); border-color: var(--primary-softer); color: var(--primary); font-weight: 600; }

.card { background: var(--surface); border: 1px solid var(--border); border-radius: 12px; overflow: hidden; }
.card-head { display: flex; justify-content: space-between; align-items: center; padding: 12px 18px; border-bottom: 1px solid var(--border); }
.card-title { font-size: 14px; font-weight: 600; font-family: var(--ff-mono); }
.meta { font-size: 12px; color: var(--text-3); }

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
  padding: 9px 14px;
  font-size: 13px;
  border-bottom: 1px solid var(--border-soft);
}
.data-table tbody tr:last-child td { border-bottom: 0; }
code { background: var(--surface-2); padding: 2px 6px; border-radius: 3px; font-family: var(--ff-mono); font-size: 11.5px; color: var(--primary); }
.muted { color: var(--text-3); }
.input { padding: 5px 8px; border: 1px solid var(--border-strong); border-radius: 5px; font-size: 12.5px; }
.input:focus { outline: 2px solid var(--primary-soft); border-color: var(--primary); }
.input-sm { width: 100%; }
.sort-input { width: 70px; }
.actions { white-space: nowrap; }
.link-btn { background: transparent; border: 0; font-size: 12px; color: var(--primary); cursor: pointer; padding: 3px 6px; border-radius: 3px; }
.link-btn:hover { background: var(--primary-soft); }
.link-btn.warn { color: var(--warning); }
.link-btn.warn:hover { background: var(--warning-soft); }
</style>
