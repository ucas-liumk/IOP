<script setup lang="ts">
withDefaults(
  defineProps<{ label: string; value: number; max: number; color?: string; suffix?: string }>(),
  { color: 'var(--primary)', suffix: '' }
)
</script>

<template>
  <div class="bar-row">
    <div class="bar-label">{{ label }}</div>
    <div class="bar-track">
      <div class="bar-fill" :style="{ width: (value / max) * 100 + '%', background: color }" />
    </div>
    <div class="bar-value mono tabular">{{ value }}{{ suffix }}</div>
  </div>
</template>
