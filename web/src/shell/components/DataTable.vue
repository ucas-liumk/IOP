<template>
  <article class="card" :class="{ 'is-loading': loading }">
    <div class="table-scroll" :class="{ sticky }">
      <table class="data-table">
        <thead>
          <tr>
            <th
              v-for="col in columns"
              :key="col.key"
              :style="{ width: typeof col.width === 'number' ? col.width + 'px' : col.width }"
              :class="alignClass(col)"
            >
              {{ col.label }}
            </th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(row, i) in rows" :key="rowKey ? row[rowKey] : i">
            <td v-for="col in columns" :key="col.key" :class="alignClass(col)">
              <slot :name="`cell-${col.key}`" :row="row" :col="col">{{ row[col.key] }}</slot>
            </td>
          </tr>
          <tr v-if="rows.length === 0 && !loading">
            <td :colspan="columns.length" class="empty-cell">
              <slot name="empty">{{ emptyText ?? "暂无数据" }}</slot>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Loading overlay: shimmer skeleton over the table area. -->
    <div v-if="loading" class="table-loading" role="status" aria-label="加载中">
      <SkeletonLoader :lines="6" :height="16" :last-short="false" />
    </div>
  </article>
</template>

<script setup lang="ts">
import type { Column } from "./types";
import SkeletonLoader from "./SkeletonLoader.vue";

const props = withDefaults(
  defineProps<{
    columns: Column[];
    rows: any[];
    rowKey?: string;
    emptyText?: string;
    /** Show a skeleton/loading overlay over the table body. */
    loading?: boolean;
    /** Make the header stick to the top while the body scrolls. */
    sticky?: boolean;
  }>(),
  { loading: false, sticky: true },
);

function alignClass(col: Column): string | undefined {
  if (col.align === "right") return "ta-right";
  if (col.align === "center") return "ta-center";
  return undefined;
}

// silence unused-prop lint for props consumed only in template
void props;
</script>

<style scoped>
.card {
  position: relative;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  overflow: hidden;
}
.table-scroll {
  width: 100%;
  overflow: auto;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th {
  text-align: left;
  font-size: 11.5px;
  font-weight: 600;
  color: var(--text-3);
  text-transform: uppercase;
  letter-spacing: .5px;
  padding: 11px 16px;
  background: var(--surface-2);
  border-bottom: 1px solid var(--border);
}
/* Sticky header: thead cells stick to the top of the scroll container. */
.table-scroll.sticky thead th {
  position: sticky;
  top: 0;
  z-index: 2;
}
.data-table td {
  padding: 12px 16px;
  font-size: 13px;
  border-bottom: 1px solid var(--border-soft);
  vertical-align: middle;
}
.data-table tbody tr:hover { background: var(--surface-2); }
.data-table tbody tr:last-child td { border-bottom: 0; }
.data-table .ta-right { text-align: right; }
.data-table .ta-center { text-align: center; }
.empty-cell {
  text-align: center;
  color: var(--text-4);
  padding: 24px 0 !important;
}
.is-loading .table-scroll { opacity: .35; pointer-events: none; }
.table-loading {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  justify-content: center;
  padding: 24px 16px;
  background: var(--surface);
}
</style>
