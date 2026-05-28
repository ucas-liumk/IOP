import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import { stagesApi } from '@/api'
import type { StageDef, StageCode, BranchCode } from '@/types'

export const STAGE_META: Record<StageCode, { name: string; short: string; icon: string; color: string; desc: string }> = {
  submit:    { name: '问题提报', short: '提报',    icon: '📥', color: 'submit',    desc: '提报问题、上传证据材料、设定期望解决时间' },
  review:    { name: '审核分办', short: '审核分办', icon: '✅', color: 'review',    desc: '核实问题真实性、判定优先级、分派至承办单位' },
  propose:   { name: '研提举措', short: '研提举措', icon: '💡', color: 'propose',   desc: '承办方提出解决举措，标注是否存在争议' },
  meeting:   { name: '会商研究', short: '会商',    icon: '👥', color: 'meeting',   desc: '多方组织会商，记录各方意见和分歧点' },
  arbitrate: { name: '争议裁决', short: '裁决',    icon: '⚖️', color: 'arbitrate', desc: '主管方做出最终裁决，形成约束性方案' },
  consult:   { name: '征求意见', short: '征求意见', icon: '📣', color: 'consult',   desc: '面向利益相关方公开征求意见并汇总修订' },
  implement: { name: '督导落实', short: '督导',    icon: '🚩', color: 'implement', desc: '跟踪举措落实进度，督促按期完成' },
  evaluate:  { name: '评价反馈', short: '评价',    icon: '⭐', color: 'evaluate',  desc: '提报方与相关方对处理结果打分评价' },
}

export const useStagesStore = defineStore('stages', () => {
  const stages = ref<StageDef[]>([])

  async function load() {
    if (stages.value.length === 0) stages.value = await stagesApi.all()
    return stages.value
  }

  const byCode = computed(() => Object.fromEntries(stages.value.map((s) => [s.code, s])))

  function pathFor(branch: BranchCode): StageDef[] {
    return stages.value.filter((s) => !s.branch || s.branch === branch)
  }

  return { stages, load, byCode, pathFor }
})
