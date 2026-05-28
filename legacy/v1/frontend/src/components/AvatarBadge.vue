<script setup lang="ts">
import { computed } from 'vue'
import { colorOf } from '@/utils/helpers'

const props = withDefaults(defineProps<{ name?: string; size?: number; color?: string }>(), {
  name: '?',
  size: 28,
})
const bg = computed(() => props.color || colorOf(props.name))
const ch = computed(() => (props.name || '?').slice(0, 1))
</script>

<template>
  <div class="avatar" :style="{ width: size + 'px', height: size + 'px', fontSize: size * 0.42 + 'px', background: bg }">
    {{ ch }}
  </div>
</template>
