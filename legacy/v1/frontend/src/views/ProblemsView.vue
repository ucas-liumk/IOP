<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  ElButton, ElButtonGroup, ElInput, ElOption, ElSelect, ElTabs, ElTabPane,
  ElTable, ElTableColumn,
} from 'element-plus'
import { Download, Plus, Refresh, Search, Right, Grid, List } from '@element-plus/icons-vue'
import AvatarBadge from '@/components/AvatarBadge.vue'
import FlowTimelineH from '@/components/FlowTimelineH.vue'
import StageChip from '@/components/StageChip.vue'
import { problemsApi } from '@/api'
import { useProcessingStore } from '@/stores/processing'
import { useStagesStore } from '@/stores/stages'
import { priorityMeta, statusMeta, daysBetween } from '@/utils/helpers'
import type { PageResult, Problem } from '@/types'

const route = useRoute()
const proc = useProcessingStore()
const stagesStore = useStagesStore()

const view = ref<'card' | 'table'>('card')
const tab = ref('all')
const query = ref('')
const status = ref('all')
const priority = ref('all')
const stage = ref<string>((route.query.stage as string) || 'all')

const result = ref<PageResult<Problem>>({ items: [], total: 0, page: 1, size: 50 })

async function load() {
  result.value = await problemsApi.list({
    page: 1, size: 50, tab: tab.value, query: query.value,
    status: status.value, priority: priority.value, stage: stage.value,
  })
}

onMounted(load)
watch([tab, status, priority, stage], load)
let qTimer: number | undefined
watch(query, () => {
  if (qTimer) clearTimeout(qTimer)
  qTimer = window.setTimeout(load, 250)
})

const counts = ref({ all: 0, mine: 0, assigned: 0, overdue: 0, done: 0 })
async function loadCounts() {
  const all = await problemsApi.list({ size: 1 })
  counts.value.all = all.total
  counts.value.mine     = (await problemsApi.list({ size: 1, tab: 'mine' })).total
  counts.value.assigned = (await problemsApi.list({ size: 1, tab: 'assigned' })).total
  counts.value.overdue  = (await problemsApi.list({ size: 1, tab: 'overdue' })).total
  counts.value.done     = (await problemsApi.list({ size: 1, tab: 'done' })).total
}
onMounted(loadCounts)

function open(p: Problem) { proc.open(p.id) }
function reset() {
  query.value = ''; status.value = 'all'; priority.value = 'all'; stage.value = 'all'
}

function remainStr(p: Problem) {
  if (p.status === 'done') return '已完成'
  if (p.overdue) return `超期 ${p.overdueDays} 天`
  return `剩 ${daysBetween(p.dueDate)} 天`
}
function remainColor(p: Problem) {
  if (p.status === 'done') return 'var(--success)'
  if (p.overdue) return 'var(--danger)'
  if (daysBetween(p.dueDate) <= 3) return 'var(--warning)'
  return 'var(--text-2)'
}
</script>

<template>
  <div class="page">
    <div class="page-header">
      <div>
        <h1 class="page-title">问题清单</h1>
        <div class="page-subtitle">共 <b class="mono">{{ result.total }}</b> 个问题</div>
      </div>
      <div class="page-actions">
        <el-button-group>
          <el-button size="small" :type="view === 'card' ? 'primary' : ''" @click="view = 'card'" :icon="Grid">卡片</el-button>
          <el-button size="small" :type="view === 'table' ? 'primary' : ''" @click="view = 'table'" :icon="List">表格</el-button>
        </el-button-group>
        <el-button size="small" :icon="Download">导出</el-button>
        <el-button type="primary" size="small" :icon="Plus">提报新问题</el-button>
      </div>
    </div>

    <div class="card" style="overflow: visible;">
      <el-tabs v-model="tab" class="problems-tabs">
        <el-tab-pane label="全部" name="all" />
        <el-tab-pane name="mine"><template #label>我提报的<span class="tab-count">{{ counts.mine }}</span></template></el-tab-pane>
        <el-tab-pane name="assigned"><template #label>分给我的<span class="tab-count">{{ counts.assigned }}</span></template></el-tab-pane>
        <el-tab-pane name="overdue"><template #label>超期预警<span class="tab-count danger">{{ counts.overdue }}</span></template></el-tab-pane>
        <el-tab-pane name="done"><template #label>已办结<span class="tab-count">{{ counts.done }}</span></template></el-tab-pane>
      </el-tabs>

      <div class="filter-bar">
        <el-input v-model="query" :prefix-icon="Search" placeholder="搜索问题编号、标题或描述…" clearable style="width: 320px;" />
        <el-select v-model="status" placeholder="全部状态" style="width: 140px;">
          <el-option label="全部状态" value="all" />
          <el-option label="待办" value="pending" />
          <el-option label="办理中" value="processing" />
          <el-option label="会商中" value="meeting" />
          <el-option label="裁决中" value="arbitrate" />
          <el-option label="征求意见中" value="consulting" />
          <el-option label="已办结" value="done" />
          <el-option label="超期" value="overdue" />
        </el-select>
        <el-select v-model="priority" placeholder="全部优先级" style="width: 140px;">
          <el-option label="全部优先级" value="all" />
          <el-option label="紧急" value="critical" />
          <el-option label="重要" value="high" />
          <el-option label="一般" value="normal" />
        </el-select>
        <el-select v-model="stage" placeholder="全部阶段" style="width: 140px;">
          <el-option label="全部阶段" value="all" />
          <el-option v-for="s in stagesStore.stages" :key="s.code" :label="s.label" :value="s.code" />
        </el-select>
        <el-button size="small" :icon="Refresh" @click="reset">重置</el-button>
        <div class="flex-1" />
        <span class="text-xs text-muted">符合 <b class="mono">{{ result.items.length }}</b> / {{ result.total }}</span>
      </div>

      <div style="padding: 14px;">
        <div v-if="result.items.length === 0" style="padding: 40px; text-align: center; color: var(--text-3);">
          没有符合条件的问题
        </div>

        <template v-else-if="view === 'card'">
          <div
            v-for="p in result.items"
            :key="p.id"
            class="problem-card"
            :class="{ overdue: p.overdue }"
            @click="open(p)"
          >
            <div class="row items-start justify-between gap-4">
              <div class="flex-1" style="min-width: 0;">
                <div class="row items-center gap-2" style="margin-bottom: 6px;">
                  <span class="badge badge-square" :class="priorityMeta(p.priority).cls">{{ priorityMeta(p.priority).label }}</span>
                  <span class="text-xs text-muted mono">{{ p.id }}</span>
                  <span class="text-xs text-muted">·</span>
                  <span class="text-xs text-muted">{{ p.category }}</span>
                  <span v-for="t in p.tags.slice(0, 2)" :key="t" class="chip" style="font-size: 11px; padding: 2px 7px;">#{{ t }}</span>
                </div>
                <div class="row items-center gap-2" style="margin-bottom: 4px;">
                  <h3 style="font-size: 17px; font-weight: 700; margin: 0; letter-spacing: -0.01em;">{{ p.title }}</h3>
                  <StageChip :stage="p.currentStage" />
                </div>
                <div class="text-sm text-muted" style="margin-bottom: 12px; line-height: 1.6; max-width: 920px;">{{ p.description }}</div>
              </div>
              <div class="col items-end gap-2" style="flex-shrink: 0;">
                <span class="badge badge-lg badge-dot" :class="statusMeta(p.status).cls">{{ statusMeta(p.status).label }}</span>
                <div class="col items-end" style="font-size: 11px; color: var(--text-3);">
                  <div>承办：<span class="font-semi text-soft">{{ p.handlerDept }}</span></div>
                  <div>期限：<span class="mono">{{ p.dueDate }}</span> · <span :style="{ color: remainColor(p), fontWeight: 600 }">{{ remainStr(p) }}</span></div>
                </div>
              </div>
            </div>

            <FlowTimelineH :problem="p" />

            <div class="row items-center justify-between gap-3" style="margin-top: 12px;">
              <div class="text-xs text-soft flex-1" style="min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;">
                最新进展：{{ p.latest }}
              </div>
              <div class="row gap-2 items-center">
                <div v-if="p.participants.length > 0" class="row items-center gap-1">
                  <AvatarBadge v-for="d in p.participants.slice(0, 3)" :key="d" :name="d" :size="22" />
                  <div v-if="p.participants.length > 3" class="avatar" style="width: 22px; height: 22px; font-size: 10px; background: var(--surface-3); color: var(--text-3);">+{{ p.participants.length - 3 }}</div>
                </div>
                <el-button size="small" @click.stop="open(p)">查看详情</el-button>
                <el-button v-if="p.status !== 'done'" type="primary" size="small" @click.stop="open(p)">
                  {{ p.currentStage === 'evaluate' ? '去评价' : '去办理' }}
                  <el-icon><Right /></el-icon>
                </el-button>
              </div>
            </div>
          </div>
        </template>

        <el-table v-else :data="result.items" @row-click="(row: Problem) => open(row)" row-class-name="cursor-row">
          <el-table-column prop="id" label="编号" width="160" />
          <el-table-column label="标题">
            <template #default="{ row }">
              <div class="font-semi">{{ row.title }}</div>
              <div class="text-xs text-muted" style="white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 480px;">{{ row.latest }}</div>
            </template>
          </el-table-column>
          <el-table-column prop="category" label="类型" width="110" />
          <el-table-column prop="handlerDept" label="承办单位" width="120" />
          <el-table-column label="当前阶段" width="110">
            <template #default="{ row }"><StageChip :stage="row.currentStage" /></template>
          </el-table-column>
          <el-table-column label="优先级" width="80">
            <template #default="{ row }">
              <span class="badge badge-square" :class="priorityMeta(row.priority).cls">{{ priorityMeta(row.priority).label }}</span>
            </template>
          </el-table-column>
          <el-table-column label="状态" width="100">
            <template #default="{ row }">
              <span class="badge" :class="statusMeta(row.status).cls">{{ statusMeta(row.status).label }}</span>
            </template>
          </el-table-column>
          <el-table-column label="办理期限" width="140">
            <template #default="{ row }">
              <div class="mono text-sm">{{ row.dueDate }}</div>
              <div class="text-xs" :style="{ color: row.overdue ? 'var(--danger)' : 'var(--text-3)' }">{{ remainStr(row) }}</div>
            </template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button size="small" text @click.stop="open(row)">查看</el-button>
              <el-button v-if="row.status !== 'done'" size="small" type="primary" @click.stop="open(row)">办理</el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
    </div>
  </div>
</template>

<style scoped>
.filter-bar { display: flex; align-items: center; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--border-soft); }
.problems-tabs { padding: 0 16px; }
.problems-tabs :deep(.el-tabs__header) { margin: 0; }
.tab-count { background: var(--surface-3); padding: 1px 7px; border-radius: 999px; font-size: 11px; color: var(--text-3); font-weight: 600; margin-left: 6px; }
.tab-count.danger { background: var(--danger-soft); color: var(--danger); }
:deep(.cursor-row) { cursor: pointer; }
</style>
