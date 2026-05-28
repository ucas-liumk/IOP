<script setup lang="ts">
import { STAGE_META } from '@/stores/stages'
import type { StageCode } from '@/types'

const props = defineProps<{ stage: StageCode; size?: 'sm' | 'lg' }>()
const meta = STAGE_META[props.stage]
</script>

<template>
  <span v-if="meta" class="stage-chip" :class="[`stage-bg-${meta.color}`, size === 'lg' ? 'lg' : '']">
    <span class="dot" />
    {{ meta.name }}
  </span>
</template>
