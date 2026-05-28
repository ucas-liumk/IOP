<script setup lang="ts">
import { ref } from 'vue'
import { ElFormItem, ElInput, ElSelect, ElOption, ElDatePicker, ElIcon, ElMessage } from 'element-plus'
import { Warning } from '@element-plus/icons-vue'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const submitting = ref(false)
const draft = ref({
  arbitrator: '',
  date: new Date().toISOString().slice(0, 10),
  overall: '',
  note: '',
})
const resolutions = ref(props.detail.disputes.map((d) => ({ disputeId: d.dispute.id, resolution: d.dispute.resolution || '' })))

async function advance() {
  submitting.value = true
  try {
    await problemsApi.arbitrate(props.detail.problem.id, {
      ...draft.value,
      note: draft.value.note || '裁决发布',
      resolutions: resolutions.value,
    })
    ElMessage.success('裁决已发布')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="争议裁决"
    subtitle="对会商后仍存在争议的事项，由主管方做出最终裁决。裁决结果对所有相关方具有约束力。"
    :badge="{ label: 'Step 5 / 8 · 终局决定', cls: 'badge-danger' }"
    advance-text="发布裁决 → 督导落实"
    :submitting="submitting"
    @advance="advance"
  >
    <div class="banner banner-danger">
      <el-icon><Warning /></el-icon>
      <div><b>裁决一经发布即生效</b>，相关方需按裁决方案执行。如需调整，需走变更流程。</div>
    </div>

    <div class="form-grid-2">
      <el-form-item label="裁决方">
        <el-select v-model="draft.arbitrator">
          <el-option v-for="d in ['CEO 办公室','COO 办公室','CTO 办公室','战略委员会','运营委员会']" :key="d" :label="d" :value="d" />
        </el-select>
      </el-form-item>
      <el-form-item label="裁决日期">
        <el-date-picker v-model="draft.date" type="date" value-format="YYYY-MM-DD" style="width: 100%;" />
      </el-form-item>
    </div>

    <el-form-item label="逐项裁决">
      <div class="col gap-3" style="width: 100%;">
        <div
          v-for="(d, i) in detail.disputes"
          :key="d.dispute.id"
          style="padding: 14px; background: var(--surface); border: 1px solid var(--danger-soft); border-left: 3px solid var(--danger); border-radius: 10px;"
        >
          <div class="row items-center gap-2" style="margin-bottom: 8px;">
            <span class="badge badge-danger">争议点 {{ i + 1 }}</span>
            <span class="font-semi">{{ d.dispute.point }}</span>
          </div>
          <div class="col" style="margin-bottom: 10px; gap: 4px;">
            <div v-for="p in d.positions" :key="p.id" class="row" style="gap: 12px; font-size: 12.5px;">
              <span style="color: var(--purple); font-weight: 600; padding: 1px 8px; background: var(--surface-3); border-radius: 4px;">{{ p.party }}</span>
              <span style="color: var(--text-2); line-height: 1.55;">{{ p.view }}</span>
            </div>
          </div>
          <el-input v-model="resolutions[i].resolution" type="textarea" :rows="3" placeholder="给出明确的裁决结论与执行方式..." />
        </div>
      </div>
    </el-form-item>

    <el-form-item label="综合裁决意见">
      <el-input v-model="draft.overall" type="textarea" :rows="4" placeholder="说明裁决的依据、原则、对各方的约束力..." />
    </el-form-item>
  </FormShell>
</template>
