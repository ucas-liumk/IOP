<script setup lang="ts">
import { ref } from 'vue'
import { ElFormItem, ElInput, ElIcon, ElMessage } from 'element-plus'
import { Share } from '@element-plus/icons-vue'
import FormShell from './FormShell.vue'
import { problemsApi } from '@/api'
import type { ProblemDetail } from '@/types'

const props = defineProps<{ detail: ProblemDetail }>()
const emit = defineEmits<{ (e: 'done'): void }>()

const submitting = ref(false)
const round = ref({ attendees: '', summary: '', consensus: '', pending: '' })

async function submit(advance: boolean) {
  submitting.value = true
  try {
    await problemsApi.meeting(props.detail.problem.id, {
      ...round.value,
      note: round.value.summary || '本轮会商',
      advance,
    })
    ElMessage.success(advance ? '会商完成，已进入争议裁决' : '本轮纪要已保存')
    emit('done')
  } finally { submitting.value = false }
}
</script>

<template>
  <FormShell
    title="会商研究"
    subtitle="组织多方会商，对争议点进行讨论，记录形成共识与未达成共识的部分。"
    :badge="{ label: 'Step 4 / 8 · 争议路径', cls: 'badge-purple' }"
    advance-text="完成会商 → 争议裁决"
    submit-only
    :submitting="submitting"
    @advance="submit(true)"
    @submit-progress="submit(false)"
  >
    <div class="banner banner-purple">
      <el-icon><Share /></el-icon>
      <div>
        <b>当前位于争议路径。</b> 共记录 <b>{{ detail.disputes.length }}</b> 个争议点，会商目标是尽量形成共识，未达成共识的部分将进入争议裁决。
      </div>
    </div>

    <el-form-item v-if="detail.disputes.length > 0" label="待会商的争议点">
      <div class="col gap-2" style="width: 100%;">
        <div
          v-for="(d, i) in detail.disputes"
          :key="d.dispute.id"
          style="padding: 12px; background: var(--surface-3); border-radius: 10px; border-left: 3px solid var(--purple);"
        >
          <div class="row items-center gap-2">
            <span class="badge badge-purple">{{ i + 1 }}</span>
            <span class="font-semi">{{ d.dispute.point }}</span>
          </div>
          <div class="col" style="margin-top: 8px; gap: 4px;">
            <div v-for="p in d.positions" :key="p.id" class="row" style="gap: 12px; font-size: 12.5px;">
              <span style="color: var(--purple); font-weight: 600; padding: 1px 8px; background: var(--surface); border-radius: 4px;">{{ p.party }}</span>
              <span style="color: var(--text-2); line-height: 1.55;">{{ p.view }}</span>
            </div>
          </div>
        </div>
      </div>
    </el-form-item>

    <el-form-item label="参会方">
      <el-input v-model="round.attendees" placeholder="@部门 / @人员" />
    </el-form-item>
    <el-form-item label="会商纪要">
      <el-input v-model="round.summary" type="textarea" :rows="4" placeholder="各方陈述、讨论焦点、形成共识与遗留分歧..." />
    </el-form-item>
    <div class="form-grid-2">
      <el-form-item label="已达成共识">
        <el-input v-model="round.consensus" type="textarea" :rows="3" placeholder="逐条列出本轮新达成的共识..." />
      </el-form-item>
      <el-form-item label="未达成共识 (转裁决)">
        <el-input v-model="round.pending" type="textarea" :rows="3" placeholder="逐条列出仍存在分歧的事项..." />
      </el-form-item>
    </div>
  </FormShell>
</template>
