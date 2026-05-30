<template>
  <article class="stat-card">
    <div class="stat-icon" :style="iconStyle">
      <slot name="icon">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path :d="icon"/>
        </svg>
      </slot>
    </div>
    <div class="stat-content">
      <div class="stat-label">{{ label }}</div>
      <div class="stat-value" :class="{ small: smallValue }">{{ value }}<span v-if="unit" class="unit">{{ unit }}</span></div>
      <div v-if="delta" :class="['stat-delta', deltaClass]">
        <span>{{ deltaArrow }}</span> {{ delta }}
      </div>
    </div>
  </article>
</template>

<script setup lang="ts">
import { computed } from "vue";

const props = defineProps<{
  label: string;
  value: string | number;
  unit?: string;
  icon?: string;          // SVG path d
  color?: string;         // text color of icon
  bg?: string;            // background of icon container
  delta?: string;         // "+12%" or "-3%"
  smallValue?: boolean;   // for non-numeric values like "live"
}>();

const iconStyle = computed(() => ({
  background: props.bg ?? "var(--info-soft)",
  color: props.color ?? "var(--info)",
}));
const deltaClass = computed(() => props.delta?.startsWith("-") ? "down" : "up");
const deltaArrow = computed(() => props.delta?.startsWith("-") ? "↓" : "↑");
</script>

<style scoped>
.stat-card {
  display: flex;
  gap: 12px;
  align-items: center;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 12px;
  padding: 16px;
}
.stat-icon {
  width: 38px;
  height: 38px;
  border-radius: 10px;
  display: grid;
  place-items: center;
  flex-shrink: 0;
}
.stat-content { min-width: 0; flex: 1; }
.stat-label {
  font-size: 12px;
  color: var(--text-3);
  font-weight: 500;
}
.stat-value {
  font-size: 22px;
  font-weight: 700;
  letter-spacing: -0.02em;
  margin-top: 2px;
  font-feature-settings: "tnum";
}
.stat-value.small { font-size: 16px; color: var(--success); }
.stat-value .unit { font-size: 14px; color: var(--text-3); margin-left: 2px; font-weight: 600; }
.stat-delta {
  font-size: 11.5px;
  font-weight: 600;
  margin-top: 2px;
}
.stat-delta.up { color: var(--success); }
.stat-delta.down { color: var(--danger); }
</style>
