<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElFormItem, ElInput, ElSelect, ElOption, ElRate, ElRadioGroup, ElRadioButton, ElMessage } from 'element-plus'
import StarRating from '@/components/StarRating.vue'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const submitting = ref(false)
const scores = ref({ quality: 4.5, speed: 4.5, collab: 4.5, satisfaction: 4.5 })
const draft = ref({
  party: '提报方',
  comment: '',
  archiveBestPractice: 'no' as 'yes' | 'no',
})

const overall = computed(() =>
  ((scores.value.quality + scores.value.speed + scores.value.collab + scores.value.satisfaction) / 4).toFixed(1)
)

const dims = [
  { k: 'quality',      label: '处理质量',   desc: '举措是否切实解决了问题，方案的完整性与可执行性' },
  { k: 'speed',        label: '处理效率',   desc: '是否在合理期限内完成，响应是否及时' },
  { k: 'collab',       label: '协同表现',   desc: '跨部门沟通是否顺畅，是否充分征求各方意见' },
  { k: 'satisfaction', label: '总体满意度', desc: '对本次问题处理的总体感受' },
] as const

async function advance() {
  submitting.value = true
  try {
    await problemsApi.evaluate(props.detail.problem.id, {
      party: draft.value.party,
      quality: scores.value.quality,
      speed: scores.value.speed,
      collab: scores.value.collab,
      satisfaction: scores.value.satisfaction,
      comment: draft.value.comment,
      archiveBestPractice: draft.value.archiveBestPractice === 'yes',
    })
    ElMessage.success('评价已提交, 问题闭环')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="评价反馈"
    subtitle="对本次问题协同解决的处理结果进行打分与评价，作为承办单位的考核依据。"
    :badge="{ label: 'Step 8 / 8 · 闭环', cls: 'badge-success' }"
    advance-text="提交评价 · 完成办结"
    :save-draft="false"
    :submitting="submitting"
    @advance="advance"
  >
    <div class="evaluate-overall">
      <div>
        <div class="text-xs text-muted">综合评分</div>
        <div class="row items-baseline gap-3" style="margin-top: 4px;">
          <div style="font-size: 44px; font-weight: 700;" class="mono">{{ overall }}</div>
          <div class="text-muted">/ 5.0</div>
        </div>
        <StarRating :score="Number(overall)" :size="20" />
      </div>
      <div class="text-sm text-muted" style="max-width: 360px;">
        您的评价将记入承办单位 <b class="text-soft">{{ detail.problem.handlerDept }}</b> 的协同办理满意度排行。
      </div>
    </div>

    <el-form-item label="评价身份">
      <el-select v-model="draft.party" style="width: 200px;">
        <el-option label="提报方" value="提报方" />
        <el-option label="相关方" value="相关方" />
        <el-option label="第三方" value="第三方" />
      </el-select>
    </el-form-item>

    <el-form-item label="维度评分">
      <div class="col gap-3" style="width: 100%;">
        <div v-for="d in dims" :key="d.k" class="row items-center gap-4">
          <div style="min-width: 110px;">
            <div class="font-semi text-sm">{{ d.label }}</div>
            <div class="text-xs text-muted">{{ d.desc }}</div>
          </div>
          <el-rate v-model="scores[d.k]" allow-half show-score :max="5" />
        </div>
      </div>
    </el-form-item>

    <el-form-item label="文字评价">
      <el-input v-model="draft.comment" type="textarea" :rows="5" placeholder="例如：流程推进高效，会商安排合理..." />
    </el-form-item>

    <el-form-item label="是否推荐归档为最佳实践">
      <el-radio-group v-model="draft.archiveBestPractice">
        <el-radio-button value="yes">推荐归档</el-radio-button>
        <el-radio-button value="no">不推荐</el-radio-button>
      </el-radio-group>
    </el-form-item>
  </FormShell>
</template>

<style scoped>
.evaluate-overall {
  display: flex; align-items: center; justify-content: space-between; gap: 24px;
  padding: 22px 24px;
  background: linear-gradient(135deg, #fff8e6 0%, var(--surface) 80%);
  border: 1px solid #fde8b0; border-radius: 14px; margin-bottom: 20px;
}
</style>
