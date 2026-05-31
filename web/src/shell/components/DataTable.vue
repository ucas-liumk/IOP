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
              <slot :name="`head-${col.key}`" :col="col">{{ col.label }}</slot>
            </th>
          </tr>
        </thead>
        <tbody>
          <!-- Skeleton rows on first load (no data yet) keep column structure. -->
          <template v-if="loading && rows.length === 0">
            <tr v-for="i in skeletonRows" :key="`sk-${i}`" class="skeleton-row" aria-hidden="true">
              <td v-for="col in columns" :key="col.key" :class="alignClass(col)">
                <span class="sk-cell" :style="{ width: skeletonWidth(col) }" />
              </td>
            </tr>
          </template>
          <template v-else>
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
          </template>
        </tbody>
      </table>
    </div>

    <!-- Refresh overlay: subtle progress bar when re-loading with data present. -->
    <div v-if="loading && rows.length > 0" class="table-refresh" role="status" aria-label="加载中">
      <span class="refresh-bar" />
    </div>
  </article>
</template>

<script setup lang="ts">
import type { Column } from "./types";

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
    /** How many skeleton rows to show on first load. */
    skeletonRows?: number;
  }>(),
  { loading: false, sticky: true, skeletonRows: 8 },
);

function alignClass(col: Column): string | undefined {
  if (col.align === "right") return "ta-right";
  if (col.align === "center") return "ta-center";
  // numeric / tabular columns render right-aligned figures.
  if (col.align === "tabular") return "ta-right tabular-nums";
  return undefined;
}

// Skeleton bar width per column — narrower for fixed/narrow columns so it reads
// like real content rather than uniform stripes.
function skeletonWidth(col: Column): string {
  const w = typeof col.width === "number" ? col.width : 160;
  if (w <= 100) return "44%";
  if (w <= 200) return "68%";
  return "82%";
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
.data-table tbody tr:not(.skeleton-row):hover { background: var(--surface-2); }
.data-table tbody tr:last-child td { border-bottom: 0; }
.data-table .ta-right { text-align: right; }
.data-table .ta-center { text-align: center; }
.data-table .tabular-nums { font-variant-numeric: tabular-nums; }
.empty-cell {
  text-align: center;
  color: var(--text-4);
  padding: 24px 0 !important;
}

/* Skeleton rows (first load) */
.skeleton-row td { vertical-align: middle; }
.sk-cell {
  display: inline-block;
  height: 14px;
  border-radius: var(--r-sm);
  background: linear-gradient(
    90deg,
    var(--border-soft) 25%,
    var(--border) 37%,
    var(--border-soft) 63%
  );
  background-size: 400% 100%;
  animation: dt-shimmer 1.3s ease-in-out infinite;
}
.ta-right .sk-cell, .ta-center .sk-cell { display: inline-block; }
@keyframes dt-shimmer {
  0% { background-position: 100% 50%; }
  100% { background-position: 0 50%; }
}

/* Refresh bar (re-loading while data already shown) */
.is-loading .table-scroll { opacity: .6; pointer-events: none; }
.table-refresh {
  position: absolute;
  top: 0; left: 0; right: 0;
  height: 2px;
  overflow: hidden;
}
.refresh-bar {
  display: block;
  height: 100%;
  width: 40%;
  background: var(--primary);
  border-radius: var(--r-pill);
  animation: dt-progress 1.1s ease-in-out infinite;
}
@keyframes dt-progress {
  0% { transform: translateX(-100%); }
  100% { transform: translateX(300%); }
}
@media (prefers-reduced-motion: reduce) {
  .sk-cell { animation: none; }
  .refresh-bar { animation: none; width: 100%; opacity: .5; }
}
</style>
