<script setup lang="ts">
import { computed } from 'vue'
import { ElButton, ElIcon, ElUpload, ElMessage, type UploadRequestOptions } from 'element-plus'
import { Download, Delete, Upload } from '@element-plus/icons-vue'
import { filesApi } from '@/api'
import { STAGE_META } from '@/stores/stages'
import type { Attachment, StageCode } from '@/types'

const props = defineProps<{ problemId: string; attachments: Attachment[]; activeStage: StageCode }>()
const emit = defineEmits<{ (e: 'refresh'): void }>()

const groups = computed(() => {
  const map: Record<string, Attachment[]> = {}
  for (const a of props.attachments) {
    if (!map[a.stage]) map[a.stage] = []
    map[a.stage].push(a)
  }
  return Object.entries(map)
})

async function customUpload(opts: UploadRequestOptions) {
  try {
    await filesApi.upload(props.problemId, props.activeStage, opts.file as File, (p) => opts.onProgress?.({ percent: p } as any))
    ElMessage.success('上传成功')
    emit('refresh')
  } catch (e) { /* http interceptor already toasts */ }
}

async function remove(a: Attachment) {
  await filesApi.delete(a.id)
  ElMessage.success('已删除')
  emit('refresh')
}
</script>

<template>
  <div class="col gap-3">
    <el-upload :show-file-list="false" :http-request="customUpload" multiple>
      <el-button size="small" :icon="Upload">上传到「{{ STAGE_META[activeStage].name }}」</el-button>
    </el-upload>
    <div v-for="[stage, items] in groups" :key="stage">
      <div class="row items-center justify-between" style="margin-bottom: 6px;">
        <span class="stage-chip" :class="`stage-bg-${STAGE_META[stage as StageCode]?.color || 'submit'}`" style="padding: 1px 7px; font-size: 10.5px;">
          <span class="dot" />{{ STAGE_META[stage as StageCode]?.name || stage }}
        </span>
        <span class="text-xs text-muted">{{ items.length }}</span>
      </div>
      <div class="col" style="gap: 2px;">
        <div v-for="a in items" :key="a.id" class="file-row" style="display: flex; align-items: center; gap: 10px; padding: 8px 10px; border-radius: 8px; background: var(--surface-3);">
          <div style="width: 32px; height: 32px; border-radius: 8px; display: flex; align-items: center; justify-content: center; background: var(--surface);">📄</div>
          <div style="flex: 1; min-width: 0;">
            <div style="font-size: 13px; font-weight: 500; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{{ a.fileName }}</div>
            <div style="font-size: 11px; color: var(--text-3);">{{ Math.round(a.fileSize / 1024) }}KB · {{ a.uploadedAt?.slice(0, 16) }} · {{ a.uploaderName }}</div>
          </div>
          <a :href="filesApi.downloadUrl(a.id)" target="_blank">
            <el-button text :icon="Download" />
          </a>
          <el-button text :icon="Delete" @click="remove(a)" />
        </div>
      </div>
    </div>
    <div v-if="attachments.length === 0" class="text-muted text-sm" style="padding: 12px; text-align: center;">尚无文件</div>
  </div>
</template>
