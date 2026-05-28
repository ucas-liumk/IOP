<script setup lang="ts">
withDefaults(defineProps<{ score?: number; size?: number; max?: number }>(), {
  score: 0,
  size: 14,
  max: 5,
})
</script>

<template>
  <span class="stars" style="display: inline-flex; gap: 1px;">
    <svg v-for="i in max" :key="i" :width="size" :height="size" viewBox="0 0 24 24">
      <defs>
        <linearGradient :id="`g-half-${i}-${size}`">
          <stop offset="50%" stop-color="#f5a524" />
          <stop offset="50%" stop-color="#e6eaf2" />
        </linearGradient>
      </defs>
      <polygon
        points="12 2 15.09 8.26 22 9.27 17 14.14 18.18 21.02 12 17.77 5.82 21.02 7 14.14 2 9.27 8.91 8.26"
        :fill="i <= Math.floor(score) ? '#f5a524' : i - 1 < score ? `url(#g-half-${i}-${size})` : '#e6eaf2'"
      />
    </svg>
  </span>
</template>
