<script setup lang="ts">
import { ElButton, ElIcon } from 'element-plus'
import { Right, Share, Document as DocIcon } from '@element-plus/icons-vue'

withDefaults(
  defineProps<{
    title: string
    subtitle?: string
    badge?: { label: string; cls?: string }
    advanceText?: string
    allowDispute?: boolean
    submitOnly?: boolean
    saveDraft?: boolean
    submitting?: boolean
  }>(),
  { advanceText: '提交并推进', allowDispute: false, submitOnly: false, saveDraft: true, submitting: false }
)
const emit = defineEmits<{
  (e: 'saveDraft'): void
  (e: 'submitProgress'): void
  (e: 'advance'): void
  (e: 'advanceDispute'): void
}>()
</script>

<template>
  <div class="form-shell">
    <div class="form-shell-header">
      <div>
        <div class="row items-center gap-2">
          <h2 class="form-shell-title">{{ title }}</h2>
          <span v-if="badge" class="badge badge-square" :class="badge.cls || 'badge-info'">{{ badge.label }}</span>
        </div>
        <div v-if="subtitle" class="form-shell-sub">{{ subtitle }}</div>
      </div>
    </div>
    <div class="form-shell-body">
      <slot />
    </div>
    <div class="form-shell-footer">
      <el-button v-if="saveDraft" text :icon="DocIcon" @click="emit('saveDraft')">保存草稿</el-button>
      <div class="flex-1" />
      <el-button v-if="submitOnly" @click="emit('submitProgress')" :loading="submitting">提交进展</el-button>
      <el-button
        v-if="allowDispute"
        @click="emit('advanceDispute')"
        :loading="submitting"
        style="border-color: var(--purple); color: var(--purple);"
      >
        <el-icon><Share /></el-icon>&nbsp;标记存在争议 → 会商研究
      </el-button>
      <el-button type="primary" @click="emit('advance')" :loading="submitting">
        {{ advanceText }} <el-icon><Right /></el-icon>
      </el-button>
    </div>
  </div>
</template>
