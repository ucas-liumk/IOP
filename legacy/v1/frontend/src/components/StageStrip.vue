<script setup lang="ts">
import { computed } from 'vue'
import { ElIcon } from 'element-plus'
import { Check } from '@element-plus/icons-vue'
import { useStagesStore } from '@/stores/stages'
import type { Problem, StageCode } from '@/types'

const props = defineProps<{ problem: Problem; activeStage: StageCode }>()
const emit = defineEmits<{ (e: 'select', stage: StageCode): void }>()

const stagesStore = useStagesStore()
const path = computed(() => stagesStore.pathFor(props.problem.branch))
const currentIdx = computed(() => path.value.findIndex((s) => s.code === props.problem.currentStage))
</script>

<template>
  <div class="stage-strip">
    <template v-for="(s, i) in path" :key="s.code">
      <div v-if="i > 0" class="stage-strip-line" :class="{ done: problem.status === 'done' || i <= currentIdx }" />
      <button
        class="stage-strip-node"
        :class="{
          done: problem.status === 'done' ? true : i < currentIdx,
          current: i === currentIdx && problem.status !== 'done',
          active: s.code === activeStage,
          branch: !!s.branch,
        }"
        @click="emit('select', s.code as StageCode)"
      >
        <div class="stage-strip-dot">
          <el-icon v-if="problem.status === 'done' || i < currentIdx" :size="12"><Check /></el-icon>
          <span v-else-if="i === currentIdx" class="flow-h-pulse" />
          <span v-else class="text-xs font-bold">{{ i + 1 }}</span>
        </div>
        <div>
          <div class="stage-strip-name">{{ s.label }}</div>
          <div class="stage-strip-sub">
            {{ problem.status === 'done' || i < currentIdx ? '已完成' : i === currentIdx ? '进行中' : '未开始' }}
            <span v-if="s.branch === 'dispute'" class="badge badge-purple" style="margin-left: 4px; font-size: 9px;">争议</span>
            <span v-else-if="s.branch === 'consensus'" class="badge badge-teal" style="margin-left: 4px; font-size: 9px;">共识</span>
          </div>
        </div>
      </button>
    </template>
  </div>
</template>
