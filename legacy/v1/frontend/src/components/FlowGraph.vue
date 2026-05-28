<script setup lang="ts">
import { computed } from 'vue'
import { STAGE_META } from '@/stores/stages'
import type { Problem, StageHistory, StageCode } from '@/types'

const props = defineProps<{ problem: Problem; history: StageHistory[] }>()
const emit = defineEmits<{ (e: 'jump', stage: StageCode): void }>()

const STAGES: StageCode[] = ['submit', 'review', 'propose', 'meeting', 'arbitrate', 'consult', 'implement', 'evaluate']
const POSITIONS: Record<StageCode, { x: number; y: number }> = {
  submit:    { x: 90, y: 180 },
  review:    { x: 200, y: 180 },
  propose:   { x: 310, y: 180 },
  meeting:   { x: 420, y: 110 },
  arbitrate: { x: 530, y: 110 },
  consult:   { x: 475, y: 250 },
  implement: { x: 640, y: 180 },
  evaluate:  { x: 750, y: 180 },
}
const EDGES: [StageCode, StageCode][] = [
  ['submit', 'review'], ['review', 'propose'],
  ['propose', 'meeting'], ['meeting', 'arbitrate'], ['arbitrate', 'implement'],
  ['propose', 'consult'], ['consult', 'implement'],
  ['implement', 'evaluate'],
]
const EDGE_BRANCH: Record<string, 'dispute' | 'consensus'> = {
  'propose-meeting': 'dispute', 'meeting-arbitrate': 'dispute', 'arbitrate-implement': 'dispute',
  'propose-consult': 'consensus', 'consult-implement': 'consensus',
}

const taken = computed(() => new Set(props.history.map((h) => h.stage)))

function edgeActive(a: StageCode, b: StageCode): boolean {
  const branch = EDGE_BRANCH[`${a}-${b}`]
  if (a === 'propose' && b === 'meeting')  return props.problem.branch === 'dispute'
  if (a === 'propose' && b === 'consult')  return props.problem.branch === 'consensus'
  if (branch && branch !== props.problem.branch) return false
  return taken.value.has(a) || taken.value.has(b)
}

function edgeColor(a: StageCode, b: StageCode): string {
  const branch = EDGE_BRANCH[`${a}-${b}`]
  if (!edgeActive(a, b)) return 'var(--border-strong)'
  if (branch === 'dispute') return 'var(--purple)'
  if (branch === 'consensus') return 'var(--teal)'
  return 'var(--success)'
}

function pathD(a: StageCode, b: StageCode): string {
  const p1 = POSITIONS[a], p2 = POSITIONS[b]
  const mx = (p1.x + p2.x) / 2
  return `M ${p1.x} ${p1.y} C ${mx} ${p1.y}, ${mx} ${p2.y}, ${p2.x} ${p2.y}`
}

function nodeFill(s: StageCode): string {
  if (s === props.problem.currentStage && props.problem.status !== 'done') return 'var(--primary)'
  if (taken.value.has(s)) return 'var(--success)'
  return 'white'
}
function nodeStroke(s: StageCode): string {
  if (s === props.problem.currentStage && props.problem.status !== 'done') return 'var(--primary)'
  if (taken.value.has(s)) return 'var(--success)'
  const meta = STAGE_META[s]
  if (meta.color === 'meeting' || meta.color === 'arbitrate') return 'var(--purple)'
  if (meta.color === 'consult') return 'var(--teal)'
  return 'var(--border-strong)'
}
function nodeOpacity(s: StageCode): number {
  const branchOnly = (s === 'meeting' || s === 'arbitrate') ? 'dispute' : s === 'consult' ? 'consensus' : null
  if (branchOnly && props.problem.branch && branchOnly !== props.problem.branch) return 0.3
  return 1
}
</script>

<template>
  <div class="form-shell">
    <div class="form-shell-header">
      <div>
        <h2 class="form-shell-title">流程图谱</h2>
        <div class="form-shell-sub">显示完整 8 节点关系网，已走过的路径高亮显示；点击节点跳转到对应阶段。</div>
      </div>
      <div class="row gap-3 text-xs">
        <span class="chip"><span style="width:6px;height:6px;border-radius:999px;background:var(--success)" />已完成</span>
        <span class="chip"><span style="width:6px;height:6px;border-radius:999px;background:var(--primary)" />当前</span>
        <span class="chip"><span style="width:6px;height:6px;border-radius:999px;background:var(--purple)" />争议路径</span>
        <span class="chip"><span style="width:6px;height:6px;border-radius:999px;background:var(--teal)" />共识路径</span>
      </div>
    </div>
    <div class="form-shell-body" style="padding: 0; background: var(--surface-2);">
      <svg viewBox="0 0 840 360" style="width: 100%; display: block; padding: 16px;">
        <g v-for="(e, i) in EDGES" :key="i">
          <path
            :d="pathD(e[0], e[1])"
            :stroke="edgeColor(e[0], e[1])"
            :stroke-width="edgeActive(e[0], e[1]) ? 2.5 : 1.5"
            :stroke-dasharray="EDGE_BRANCH[`${e[0]}-${e[1]}`] && !edgeActive(e[0], e[1]) ? '6 4' : ''"
            fill="none"
          />
        </g>
        <g v-for="(s, i) in STAGES" :key="s" style="cursor: pointer;" :style="{ opacity: nodeOpacity(s) }" @click="emit('jump', s)">
          <circle :cx="POSITIONS[s].x" :cy="POSITIONS[s].y" r="26"
                  :fill="nodeFill(s)" :stroke="nodeStroke(s)" stroke-width="2.5" />
          <text :x="POSITIONS[s].x" :y="POSITIONS[s].y + 5"
                text-anchor="middle" font-size="13" font-weight="700"
                :fill="(s === problem.currentStage && problem.status !== 'done') || taken.has(s) ? 'white' : 'var(--text)'">
            {{ i + 1 }}
          </text>
          <text :x="POSITIONS[s].x" :y="POSITIONS[s].y + 50" text-anchor="middle" font-size="12" font-weight="600" fill="var(--text)">
            {{ STAGE_META[s].name }}
          </text>
        </g>
      </svg>
    </div>
    <div class="form-shell-footer">
      <div class="text-xs text-muted">
        当前路径：
        <b v-if="problem.branch === 'dispute'" style="color: var(--purple);">争议研究路径（会商 → 裁决）</b>
        <b v-else-if="problem.branch === 'consensus'" style="color: var(--teal);">共识路径（征求意见）</b>
        <span v-else>研提举措后将根据争议情况确定</span>
      </div>
    </div>
  </div>
</template>
