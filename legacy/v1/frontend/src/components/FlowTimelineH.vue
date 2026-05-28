<script setup lang="ts">
import { computed } from 'vue'
import { ElIcon } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { useStagesStore, STAGE_META } from '@/stores/stages'
import type { Problem, StageHistory } from '@/types'

const props = defineProps<{ problem: Problem; history?: StageHistory[] }>()
const stagesStore = useStagesStore()

const path = computed(() => stagesStore.pathFor(props.problem.branch))
const currentIdx = computed(() => path.value.findIndex((s) => s.code === props.problem.currentStage))

function lastHistoryFor(code: string): StageHistory | undefined {
  if (!props.history) return undefined
  return [...props.history].reverse().find((h) => h.stage === code)
}
</script>

<template>
  <div class="flow-h">
    <template v-for="(s, i) in path" :key="s.code">
      <div
        v-if="i > 0"
        class="flow-h-line"
        :class="{
          done: problem.status === 'done' ? true : i <= currentIdx,
          overdue: problem.overdue && i === currentIdx,
        }"
      />
      <div
        class="flow-h-node"
        :class="{
          done: problem.status === 'done' ? true : i < currentIdx,
          current: i === currentIdx && problem.status !== 'done',
          overdue: i === currentIdx && problem.overdue,
          branch: !!s.branch,
        }"
      >
        <div class="flow-h-dot">
          <el-icon v-if="problem.status === 'done' || i < currentIdx" :size="11"><Check /></el-icon>
          <span v-else-if="i === currentIdx && !problem.overdue" class="flow-h-pulse" />
        </div>
        <div class="flow-h-label">
          <div class="flow-h-name">
            {{ STAGE_META[s.code].short }}
            <span v-if="s.branch === 'dispute'" class="flow-h-tag" style="background: var(--purple-soft); color: var(--purple);">争议</span>
          </div>
          <div v-if="lastHistoryFor(s.code)" class="flow-h-meta">
            {{ lastHistoryFor(s.code)!.occurredAt.slice(5, 10) }} · {{ lastHistoryFor(s.code)!.actorName }}
          </div>
          <div v-else-if="i === currentIdx" class="flow-h-meta">进行中</div>
          <div v-else class="flow-h-meta">—</div>
        </div>
      </div>
    </template>
  </div>
</template>
