<script setup lang="ts">
import { ref } from 'vue'
import {
  ElFormItem, ElInput, ElSelect, ElOption, ElDatePicker, ElRadioGroup, ElRadioButton, ElMessage,
} from 'element-plus'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const decision = ref<'approve' | 'modify' | 'reject'>('approve')
const submitting = ref(false)
const draft = ref({
  reviewNote: '',
  handlerName: '',
  handlerDept: '',
  priority: props.detail.problem.priority,
  dueDate: props.detail.problem.dueDate,
  assignNote: '',
})

async function advance() {
  submitting.value = true
  try {
    await problemsApi.review(props.detail.problem.id, { decision: decision.value, ...draft.value })
    ElMessage.success('审核已提交')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="审核分办"
    subtitle="核实问题真实性与优先级，决定是否受理、分派至承办单位。"
    :badge="{ label: 'Step 2 / 8' }"
    :advance-text="decision === 'approve' ? '通过并分办 → 研提举措' : decision === 'reject' ? '退回提报方' : '提交'"
    :submitting="submitting"
    @advance="advance"
  >
    <el-form-item label="审核决定">
      <el-radio-group v-model="decision">
        <el-radio-button value="approve">通过并受理</el-radio-button>
        <el-radio-button value="modify">退回补充材料</el-radio-button>
        <el-radio-button value="reject">不予受理</el-radio-button>
      </el-radio-group>
    </el-form-item>

    <el-form-item label="审核意见">
      <el-input v-model="draft.reviewNote" type="textarea" :rows="4" placeholder="说明审核结论..." />
    </el-form-item>

    <template v-if="decision === 'approve'">
      <div class="form-grid-2">
        <el-form-item label="承办单位">
          <el-select v-model="draft.handlerDept" filterable>
            <el-option v-for="d in ['运营优化办','战略部','人力资源部','法务合规部','研发中心','产品中心','行政中心','财务部','客户成功部']" :key="d" :label="d" :value="d" />
          </el-select>
        </el-form-item>
        <el-form-item label="承办人 / 牵头">
          <el-input v-model="draft.handlerName" placeholder="承办人或牵头单位" />
        </el-form-item>
        <el-form-item label="确认优先级">
          <el-radio-group v-model="draft.priority">
            <el-radio-button value="critical">紧急</el-radio-button>
            <el-radio-button value="high">重要</el-radio-button>
            <el-radio-button value="normal">一般</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="办理期限">
          <el-date-picker v-model="draft.dueDate" type="date" value-format="YYYY-MM-DD" style="width: 100%;" />
        </el-form-item>
      </div>
      <el-form-item label="分办说明">
        <el-input v-model="draft.assignNote" type="textarea" :rows="3" placeholder="向承办单位说明处置要点..." />
      </el-form-item>
    </template>
  </FormShell>
</template>
