<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElFormItem, ElInput, ElSlider, ElSelect, ElOption, ElMessage } from 'element-plus'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

interface MProg { id: number; title: string; owner: string; progress: number; status: string; comment: string; hasDispute: boolean; code: string }
const tracking = ref<MProg[]>(
  props.detail.measures.map((m) => ({
    id: m.id, code: m.code, title: m.title, owner: m.owner,
    progress: m.progress ?? (m.status === 'completed' ? 100 : m.status === 'in_progress' ? 60 : 0),
    status: m.status, comment: '', hasDispute: m.hasDispute,
  }))
)

const overall = computed(() =>
  tracking.value.length ? Math.round(tracking.value.reduce((s, t) => s + t.progress, 0) / tracking.value.length) : 0
)

const submitting = ref(false)
const note = ref('')

async function submit(advance: boolean) {
  submitting.value = true
  try {
    await problemsApi.implement(props.detail.problem.id, {
      measureProgress: tracking.value.map((t) => ({
        measureId: t.id, progress: t.progress, status: t.status, comment: t.comment,
      })),
      note: note.value || (advance ? '督导落实完成' : '督导落实进度更新'),
      advance,
    })
    ElMessage.success(advance ? '已进入评价反馈' : '进度已保存')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="督导落实"
    subtitle="跟踪每一条举措的落实进度，提交阶段性进展，最终申请办结。"
    :badge="{ label: 'Step 7 / 8' }"
    advance-text="申请办结 → 评价反馈"
    submit-only
    :submitting="submitting"
    @advance="submit(true)"
    @submit-progress="submit(false)"
  >
    <div class="implement-overview">
      <div class="flex-1">
        <div class="text-xs text-muted" style="margin-bottom: 6px;">总体落实进度</div>
        <div class="row items-center gap-3">
          <div style="font-size: 30px; font-weight: 700; color: var(--primary);" class="mono">{{ overall }}%</div>
          <div style="flex: 1; height: 10px; background: rgba(255,255,255,0.6); border-radius: 999px; overflow: hidden; min-width: 200px;">
            <div style="height: 100%; background: var(--primary); border-radius: 999px; transition: width .6s ease;" :style="{ width: overall + '%' }" />
          </div>
        </div>
      </div>
      <div class="row gap-4">
        <div style="text-align: center;">
          <div class="text-xs text-muted">举措数</div>
          <div class="mono font-bold text-lg">{{ tracking.length }}</div>
        </div>
        <div style="text-align: center;">
          <div class="text-xs text-muted">已完成</div>
          <div class="mono font-bold text-lg" style="color: var(--success);">{{ tracking.filter((t) => t.progress >= 100).length }}</div>
        </div>
        <div style="text-align: center;">
          <div class="text-xs text-muted">进行中</div>
          <div class="mono font-bold text-lg" style="color: var(--warning);">{{ tracking.filter((t) => t.progress > 0 && t.progress < 100).length }}</div>
        </div>
      </div>
    </div>

    <el-form-item label="举措落实跟踪">
      <div class="col gap-3" style="width: 100%;">
        <div v-for="(m, i) in tracking" :key="m.id" class="measure-card">
          <div class="row items-center justify-between" style="margin-bottom: 8px;">
            <div class="row items-center gap-2">
              <span class="measure-num mono">{{ m.code || ('M' + (i + 1)) }}</span>
              <span class="font-semi">{{ m.title }}</span>
              <span v-if="m.hasDispute" class="badge badge-purple">争议来源</span>
            </div>
            <span class="chip">{{ m.owner }}</span>
          </div>
          <div class="row items-center gap-3" style="margin-bottom: 8px;">
            <span class="text-xs text-muted" style="min-width: 60px;">进度 {{ m.progress }}%</span>
            <el-slider v-model="m.progress" style="flex: 1;" />
            <el-select v-model="m.status" style="width: 130px;">
              <el-option label="未开始"   value="proposed" />
              <el-option label="起草中"   value="drafting" />
              <el-option label="已通过"   value="approved" />
              <el-option label="进行中"   value="in_progress" />
              <el-option label="已完成"   value="completed" />
            </el-select>
          </div>
          <el-input v-model="m.comment" type="textarea" :rows="2" placeholder="本阶段落实情况、遇到的问题、下一步动作..." />
        </div>
      </div>
    </el-form-item>

    <el-form-item label="本次说明">
      <el-input v-model="note" type="textarea" :rows="2" placeholder="本次督导更新的总体说明" />
    </el-form-item>
  </FormShell>
</template>

<style scoped>
.implement-overview {
  display: flex; align-items: center; gap: 24px;
  padding: 16px 18px;
  background: linear-gradient(90deg, var(--primary-soft) 0%, var(--surface) 60%);
  border: 1px solid var(--border); border-radius: 12px; margin-bottom: 16px;
}
</style>
