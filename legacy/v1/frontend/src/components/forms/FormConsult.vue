<script setup lang="ts">
import { computed, ref } from 'vue'
import { ElFormItem, ElInput, ElSelect, ElOption, ElDatePicker, ElIcon, ElMessage } from 'element-plus'
import { ChatLineRound } from '@element-plus/icons-vue'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const submitting = ref(false)
const draft = ref({
  audience: props.detail.problem.participants.join(','),
  method: '公开评论',
  startDate: props.detail.consult?.startDate || '',
  endDate:   props.detail.consult?.endDate || '',
  brief:     props.detail.consult?.brief || '',
  revision:  props.detail.consult?.revision || '',
  note: '',
})
const c = computed(() => props.detail.consult || { totalCount: 0, supportCount: 0, neutralCount: 0, opposeCount: 0 })

async function submit(advance: boolean) {
  submitting.value = true
  try {
    await problemsApi.consult(props.detail.problem.id, {
      audience: draft.value.audience.split(',').map((s) => s.trim()).filter(Boolean),
      method: draft.value.method,
      startDate: draft.value.startDate,
      endDate: draft.value.endDate,
      brief: draft.value.brief,
      revision: draft.value.revision,
      note: draft.value.note || (advance ? '征求意见结束' : '征求意见保存'),
      advance,
    })
    ElMessage.success(advance ? '已进入督导落实' : '已保存')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="征求意见"
    subtitle="将拟提举措公开征求利益相关方意见，按反馈进行修订完善。"
    :badge="{ label: 'Step 6 / 8 · 共识路径', cls: 'badge-teal' }"
    advance-text="结束征求 → 督导落实"
    submit-only
    :submitting="submitting"
    @advance="submit(true)"
    @submit-progress="submit(false)"
  >
    <div class="banner banner-teal">
      <el-icon><ChatLineRound /></el-icon>
      <div><b>当前位于共识路径。</b> 因研提举措无显著争议，跳过会商与裁决，直接征求意见，按反馈完善后落实。</div>
    </div>

    <div class="form-grid-2">
      <el-form-item label="征求对象">
        <el-input v-model="draft.audience" placeholder="@部门 / @岗位 / 全员, 多个逗号分隔" />
      </el-form-item>
      <el-form-item label="征求方式">
        <el-select v-model="draft.method">
          <el-option v-for="m in ['问卷调查','定向访谈','公开评论','线下座谈']" :key="m" :label="m" :value="m" />
        </el-select>
      </el-form-item>
      <el-form-item label="开启日期">
        <el-date-picker v-model="draft.startDate" type="date" value-format="YYYY-MM-DD" style="width: 100%;" />
      </el-form-item>
      <el-form-item label="截止日期">
        <el-date-picker v-model="draft.endDate" type="date" value-format="YYYY-MM-DD" style="width: 100%;" />
      </el-form-item>
    </div>

    <el-form-item label="征求事项说明">
      <el-input v-model="draft.brief" type="textarea" :rows="4" placeholder="向被征求方说明本次征求的事项、重点关注点..." />
    </el-form-item>

    <el-form-item label="意见汇总">
      <div style="width: 100%;">
        <div class="consult-grid">
          <div class="consult-stat">
            <div class="consult-stat-v mono">{{ c.totalCount }}</div>
            <div class="consult-stat-l">已收集反馈</div>
          </div>
          <div class="consult-stat" style="background: var(--success-soft);">
            <div class="consult-stat-v mono" style="color: var(--success);">{{ c.supportCount }}</div>
            <div class="consult-stat-l">支持</div>
          </div>
          <div class="consult-stat" style="background: var(--neutral-soft);">
            <div class="consult-stat-v mono" style="color: var(--neutral);">{{ c.neutralCount }}</div>
            <div class="consult-stat-l">中立</div>
          </div>
          <div class="consult-stat" style="background: var(--danger-soft);">
            <div class="consult-stat-v mono" style="color: var(--danger);">{{ c.opposeCount }}</div>
            <div class="consult-stat-l">反对</div>
          </div>
        </div>
        <div class="consult-bar" v-if="c.totalCount > 0">
          <div :style="{ width: (c.supportCount / c.totalCount * 100) + '%', background: 'var(--success)' }" />
          <div :style="{ width: (c.neutralCount / c.totalCount * 100) + '%', background: 'var(--neutral)' }" />
          <div :style="{ width: (c.opposeCount / c.totalCount * 100) + '%', background: 'var(--danger)' }" />
        </div>
      </div>
    </el-form-item>

    <el-form-item label="修订说明">
      <el-input v-model="draft.revision" type="textarea" :rows="3" placeholder="根据反馈对原举措做了哪些调整..." />
    </el-form-item>
  </FormShell>
</template>

<style scoped>
.consult-grid { display: grid; grid-template-columns: repeat(4, 1fr); gap: 8px; margin-bottom: 10px; }
.consult-stat { padding: 12px; background: var(--surface-3); border-radius: 10px; text-align: center; }
.consult-stat-v { font-size: 22px; font-weight: 700; color: var(--text); }
.consult-stat-l { font-size: 11.5px; color: var(--text-3); margin-top: 2px; }
.consult-bar { display: flex; height: 10px; border-radius: 999px; overflow: hidden; background: var(--surface-3); }
</style>
