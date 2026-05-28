<script setup lang="ts">
import { ref } from 'vue'
import {
  ElFormItem, ElInput, ElIcon, ElSwitch, ElButton, ElMessage,
} from 'element-plus'
import { Plus, Delete, Share, Right } from '@element-plus/icons-vue'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const hasDispute = ref<boolean>(props.detail.disputes.length > 0)
const submitting = ref(false)
const note = ref('')

interface MeasureDraft { code: string; title: string; owner: string; hasDispute: boolean }
interface DisputeDraft { point: string; positions: { party: string; view: string }[] }

const measures = ref<MeasureDraft[]>(
  props.detail.measures.length
    ? props.detail.measures.map((m) => ({ code: m.code, title: m.title, owner: m.owner, hasDispute: m.hasDispute }))
    : [{ code: 'M1', title: '', owner: '', hasDispute: false }]
)
const disputes = ref<DisputeDraft[]>(
  props.detail.disputes.length
    ? props.detail.disputes.map((d) => ({
        point: d.dispute.point,
        positions: d.positions.map((p) => ({ party: p.party, view: p.view })),
      }))
    : [{ point: '', positions: [{ party: '', view: '' }] }]
)

function addMeasure()       { measures.value.push({ code: 'M' + (measures.value.length + 1), title: '', owner: '', hasDispute: false }) }
function removeMeasure(i: number) { measures.value.splice(i, 1) }
function addDispute()       { disputes.value.push({ point: '', positions: [{ party: '', view: '' }] }) }
function addPosition(i: number)   { disputes.value[i].positions.push({ party: '', view: '' }) }

async function submit(branchOverride?: 'dispute' | 'consensus') {
  const branch = branchOverride ?? (hasDispute.value ? 'dispute' : 'consensus')
  submitting.value = true
  try {
    await problemsApi.propose(props.detail.problem.id, {
      hasDispute: branch === 'dispute',
      measures: measures.value,
      disputes: branch === 'dispute' ? disputes.value : [],
      note: note.value || `提交研提，${branch === 'dispute' ? '存在' : '无显著'}争议`,
    })
    ElMessage.success(branch === 'dispute' ? '已进入会商研究' : '已进入征求意见')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="研提举措"
    subtitle="承办单位组织研究并提出具体举措，标注每条举措是否存在争议。若存在争议，将进入会商研究与争议裁决；否则进入征求意见。"
    :badge="{ label: 'Step 3 / 8 · 关键分支', cls: 'badge-warning' }"
    advance-text="提交并进入征求意见"
    :allow-dispute="!hasDispute"
    :submitting="submitting"
    @advance="submit('consensus')"
    @advance-dispute="submit('dispute')"
  >
    <!-- 分支切换 -->
    <div class="dispute-toggle">
      <div class="row items-center gap-3 flex-1">
        <div class="dispute-toggle-icon" :class="{ on: hasDispute }">
          <el-icon><Share /></el-icon>
        </div>
        <div class="flex-1">
          <div class="font-semi">是否存在争议？</div>
          <div class="text-xs text-muted">勾选后，问题将进入"会商研究 → 争议裁决"路径；否则进入"征求意见"。</div>
        </div>
      </div>
      <el-switch v-model="hasDispute" />
    </div>

    <!-- 分支预览 -->
    <div class="branch-preview" :class="{ dispute: hasDispute }">
      <div class="branch-flow">
        <div class="branch-node">研提举措</div>
        <el-icon><Right /></el-icon>
        <template v-if="hasDispute">
          <div class="branch-node purple">会商研究</div>
          <el-icon><Right /></el-icon>
          <div class="branch-node danger">争议裁决</div>
        </template>
        <template v-else>
          <div class="branch-node teal">征求意见</div>
        </template>
        <el-icon><Right /></el-icon>
        <div class="branch-node">督导落实</div>
        <el-icon><Right /></el-icon>
        <div class="branch-node">评价反馈</div>
      </div>
    </div>

    <!-- 举措清单 -->
    <el-form-item label="拟提举措">
      <div class="col gap-3" style="width: 100%;">
        <div v-for="(m, i) in measures" :key="i" class="measure-card">
          <div class="row items-center justify-between" style="margin-bottom: 8px;">
            <div class="row items-center gap-2">
              <span class="measure-num mono">{{ 'M' + (i + 1) }}</span>
              <span class="text-sm font-semi">举措 {{ i + 1 }}</span>
              <span v-if="m.hasDispute" class="badge badge-purple">争议</span>
            </div>
            <el-button text :icon="Delete" @click="removeMeasure(i)" />
          </div>
          <el-input v-model="m.title" type="textarea" :rows="2" placeholder="举措内容（具体可执行的动作）" />
          <div class="row gap-3" style="margin-top: 8px;">
            <el-input v-model="m.owner" placeholder="责任方 / 部门" />
            <label class="row items-center gap-2 text-sm text-soft" style="padding-left: 8px;">
              <input type="checkbox" v-model="m.hasDispute" />本条存在争议
            </label>
          </div>
        </div>
        <el-button :icon="Plus" @click="addMeasure">添加举措</el-button>
      </div>
    </el-form-item>

    <!-- 争议点 -->
    <el-form-item v-if="hasDispute" label="争议点详情">
      <div class="col gap-3" style="width: 100%;">
        <div
          v-for="(d, i) in disputes"
          :key="i"
          class="measure-card"
          style="border-color: var(--purple-soft); background: var(--purple-soft);"
        >
          <div class="row items-center gap-2" style="margin-bottom: 8px;">
            <span class="badge badge-purple">争议点 {{ i + 1 }}</span>
            <el-input v-model="d.point" placeholder="争议点描述（例如：基础设施 ROI 量化口径）" style="background: white;" />
          </div>
          <div class="col gap-2">
            <div v-for="(pos, j) in d.positions" :key="j" class="row gap-2">
              <el-input v-model="pos.party" placeholder="方" style="flex: 0 0 140px; background: white;" />
              <el-input v-model="pos.view" placeholder="观点" style="background: white;" />
            </div>
            <el-button size="small" text :icon="Plus" @click="addPosition(i)">添加一方观点</el-button>
          </div>
        </div>
        <el-button :icon="Plus" @click="addDispute" style="border-color: var(--purple); color: var(--purple);">添加争议点</el-button>
      </div>
    </el-form-item>

    <el-form-item label="提交说明">
      <el-input v-model="note" type="textarea" :rows="2" placeholder="本次研提的说明" />
    </el-form-item>
  </FormShell>
</template>
