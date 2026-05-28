<script setup lang="ts">
import AvatarBadge from '@/components/AvatarBadge.vue'
import { STAGE_META } from '@/stores/stages'
import type { StageHistory } from '@/types'

defineProps<{ history: StageHistory[] }>()
</script>

<template>
  <div class="history">
    <div
      v-for="(h, i) in [...history].reverse()"
      :key="h.id"
      class="history-row"
    >
      <div class="history-rail">
        <div class="history-dot" :class="`stage-bg-${STAGE_META[h.stage].color}`">
          {{ STAGE_META[h.stage].icon }}
        </div>
        <div v-if="i < history.length - 1" class="history-line" />
      </div>
      <div class="history-body">
        <div class="row items-center gap-2 flex-wrap" style="margin-bottom: 3px;">
          <span class="stage-chip" :class="`stage-bg-${STAGE_META[h.stage].color}`"><span class="dot" />{{ STAGE_META[h.stage].name }}</span>
          <span class="text-xs text-muted mono">{{ h.occurredAt }}</span>
        </div>
        <div class="text-sm">{{ h.note }}</div>
        <div class="row items-center gap-2 text-xs text-muted" style="margin-top: 4px;">
          <AvatarBadge :name="h.actorName" :size="16" />
          <span>{{ h.actorName }}</span>
          <span>·</span>
          <span>{{ h.actorDept }}</span>
        </div>
        <div v-if="h.files && h.files.length" class="col gap-1" style="margin-top: 6px;">
          <div
            v-for="f in h.files"
            :key="f"
            class="row items-center gap-2 text-xs"
            style="padding: 4px 8px; background: var(--surface-3); border-radius: 6px;"
          >
            📄 <span class="text-soft">{{ f }}</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
