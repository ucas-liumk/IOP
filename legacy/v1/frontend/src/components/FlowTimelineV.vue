<script setup lang="ts">
import { computed } from 'vue'
import { ElIcon } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { useStagesStore } from '@/stores/stages'
import type { Problem, StageHistory, StageCode } from '@/types'

const props = defineProps<{ problem: Problem; history: StageHistory[] }>()
const emit = defineEmits<{ (e: 'jump', stage: StageCode): void }>()

const stagesStore = useStagesStore()
const path = computed(() => stagesStore.pathFor(props.problem.branch))
const currentIdx = computed(() => path.value.findIndex((s) => s.code === props.problem.currentStage))

function lastHistoryFor(code: string): StageHistory | undefined {
  return [...props.history].reverse().find((h) => h.stage === code)
}
</script>

<template>
  <div class="flow-v">
    <div
      v-for="(s, i) in path"
      :key="s.code"
      class="flow-v-row clickable"
      :class="{
        done: problem.status === 'done' ? true : i < currentIdx,
        current: i === currentIdx && problem.status !== 'done',
        branch: !!s.branch,
      }"
      @click="emit('jump', s.code as StageCode)"
    >
      <div class="flow-v-rail">
        <div class="flow-v-dot">
          <el-icon v-if="problem.status === 'done' || i < currentIdx" :size="10"><Check /></el-icon>
          <span v-else-if="i === currentIdx" class="flow-v-pulse" />
        </div>
        <div v-if="i < path.length - 1" class="flow-v-line" :class="{ done: problem.status === 'done' || i < currentIdx }" />
      </div>
      <div class="flow-v-body">
        <div class="flow-v-title">
          <span>{{ s.label }}</span>
          <span
            v-if="s.branch"
            class="flow-v-tag"
            :style="{
              background: s.branch === 'dispute' ? 'var(--purple-soft)' : 'var(--teal-soft)',
              color: s.branch === 'dispute' ? 'var(--purple)' : 'var(--teal)',
            }"
          >
            {{ s.branch === 'dispute' ? '争议路径' : '共识路径' }}
          </span>
        </div>
        <div class="flow-v-meta">
          <template v-if="(problem.status === 'done' || i < currentIdx) && lastHistoryFor(s.code)">
            <span class="badge badge-success badge-square">已完成</span>
            <span class="text-xs text-muted">{{ lastHistoryFor(s.code)!.occurredAt.slice(5, 16) }} · {{ lastHistoryFor(s.code)!.actorName }}</span>
          </template>
          <template v-else-if="i === currentIdx">
            <span class="badge badge-info badge-square">进行中</span>
            <span class="text-xs text-muted">您的任务</span>
          </template>
          <span v-else class="text-xs text-muted">未开始</span>
        </div>
      </div>
    </div>
  </div>
</template>
