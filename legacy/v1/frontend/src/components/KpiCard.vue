<script setup lang="ts">
import { computed, h, type Component } from 'vue'
import { ElIcon } from 'element-plus'
import { ArrowUp, ArrowDown } from '@element-plus/icons-vue'

const props = withDefaults(
  defineProps<{
    label: string
    value: number | string
    delta?: number
    deltaInverted?: boolean
    icon: Component
    color: string
    accent?: string
  }>(),
  { delta: 0, deltaInverted: false }
)

const positive = computed(() => (props.delta ?? 0) >= 0)
const isGood = computed(() => (props.deltaInverted ? !positive.value : positive.value))
</script>

<template>
  <div class="card kpi" style="padding: 18px 20px;">
    <div class="row items-center justify-between">
      <div class="text-soft text-sm font-semi">{{ label }}</div>
      <div class="kpi-icon" :class="`stage-bg-${color}`">
        <el-icon :size="16"><component :is="icon" /></el-icon>
      </div>
    </div>
    <div class="row items-baseline gap-2" style="margin-top: 10px;">
      <div class="kpi-value mono tabular" :style="{ color: accent || 'var(--text)' }">{{ value }}</div>
    </div>
    <div v-if="delta != null" class="row items-center gap-2" style="margin-top: 6px;">
      <span class="kpi-delta" :class="isGood ? 'up' : 'down'">
        <el-icon :size="11">
          <ArrowUp v-if="positive" />
          <ArrowDown v-else />
        </el-icon>
        {{ Math.abs(delta) }}%
      </span>
      <span class="text-xs text-muted">同比上周</span>
    </div>
  </div>
</template>
