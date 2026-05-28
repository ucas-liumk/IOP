<script setup lang="ts">
import type { Measure } from '@/types'

defineProps<{ measures: Measure[] }>()

const STATUS_LABEL: Record<string, { label: string; cls: string }> = {
  proposed:    { label: '已提出', cls: 'badge' },
  drafting:    { label: '起草中', cls: 'badge-warning' },
  approved:    { label: '已通过', cls: 'badge-teal' },
  in_progress: { label: '执行中', cls: 'badge-info' },
  completed:   { label: '已完成', cls: 'badge-success' },
}
</script>

<template>
  <div class="form-shell">
    <div class="form-shell-header">
      <div>
        <h2 class="form-shell-title">举措清单</h2>
        <div class="form-shell-sub">由研提阶段提出，经会商/裁决或征求意见后落实的全部举措。</div>
      </div>
    </div>
    <div class="form-shell-body">
      <div v-if="measures.length === 0" style="padding: 32px; text-align: center; color: var(--text-3);">尚未提出任何举措</div>
      <div v-else class="col gap-3">
        <div v-for="m in measures" :key="m.id" class="measure-card">
          <div class="row items-center justify-between" style="margin-bottom: 6px;">
            <div class="row items-center gap-2">
              <span class="measure-num mono">{{ m.code }}</span>
              <span class="font-semi">{{ m.title }}</span>
            </div>
            <div class="row gap-2">
              <span v-if="m.hasDispute" class="badge badge-purple">来自争议</span>
              <span class="chip">{{ m.owner }}</span>
              <span class="badge" :class="STATUS_LABEL[m.status].cls">{{ STATUS_LABEL[m.status].label }}</span>
            </div>
          </div>
          <div v-if="m.progress" class="row items-center gap-3" style="margin-top: 6px;">
            <span class="text-xs text-muted">进度</span>
            <div style="flex: 1; height: 6px; background: var(--surface-3); border-radius: 999px; overflow: hidden;">
              <div :style="{ width: m.progress + '%', height: '100%', background: 'var(--primary)' }" />
            </div>
            <span class="mono text-xs">{{ m.progress }}%</span>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
