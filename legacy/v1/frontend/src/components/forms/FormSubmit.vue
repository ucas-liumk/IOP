<script setup lang="ts">
import { ref } from 'vue'
import { ElForm, ElFormItem, ElInput, ElSelect, ElOption, ElDatePicker, ElRadioGroup, ElRadioButton } from 'element-plus'
import FormShell from './FormShell.vue'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const p = props.detail.problem
const draft = ref({
  title: p.title,
  category: p.category,
  priority: p.priority,
  dueDate: p.dueDate,
  description: p.description,
})

function onAdvance() {
  // 提报阶段为 "已发生", 这里不真正提交, 只展示
  emit('done')
}
</script>

<template>
  <FormShell
    title="问题提报"
    subtitle="详细描述问题、设置优先级与期望解决时间，提交后将进入审核分办环节。"
    :badge="{ label: 'Step 1 / 8', cls: 'badge-info' }"
    advanceText="提交问题"
    :save-draft="false"
    @advance="onAdvance"
  >
    <div class="form-grid-2">
      <el-form-item label="问题标题"><el-input v-model="draft.title" /></el-form-item>
      <el-form-item label="问题分类">
        <el-select v-model="draft.category">
          <el-option v-for="c in ['战略规划','运营效率','组织人事','合规风控','信息技术','品牌市场','行政后勤','其他']" :key="c" :label="c" :value="c" />
        </el-select>
      </el-form-item>
      <el-form-item label="优先级">
        <el-radio-group v-model="draft.priority">
          <el-radio-button value="critical">紧急</el-radio-button>
          <el-radio-button value="high">重要</el-radio-button>
          <el-radio-button value="normal">一般</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item label="期望解决时间">
        <el-date-picker v-model="draft.dueDate" type="date" value-format="YYYY-MM-DD" style="width: 100%;" />
      </el-form-item>
    </div>
    <el-form-item label="问题描述">
      <el-input v-model="draft.description" type="textarea" :rows="6" maxlength="2000" show-word-limit />
    </el-form-item>
  </FormShell>
</template>
