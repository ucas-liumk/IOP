<script setup lang="ts">
import { computed, onMounted, onBeforeUnmount, ref, shallowRef, watch } from 'vue'
import { ElButton, ElIcon, ElTooltip } from 'element-plus'
import { ArrowLeft, Close, Paperclip, User, Flag, Calendar, Bell, ChatLineRound, Share, Tools, Document } from '@element-plus/icons-vue'
import AvatarBadge from '@/components/AvatarBadge.vue'
import FlowTimelineV from '@/components/FlowTimelineV.vue'
import StageStrip from '@/components/StageStrip.vue'
import HistoryTimeline from '@/components/HistoryTimeline.vue'
import MaterialList from '@/components/MaterialList.vue'
import CollabPanel from '@/components/CollabPanel.vue'
import MeasuresPanel from '@/components/MeasuresPanel.vue'
import FlowGraph from '@/components/FlowGraph.vue'
import FormSubmit    from '@/components/forms/FormSubmit.vue'
import FormReview    from '@/components/forms/FormReview.vue'
import FormPropose   from '@/components/forms/FormPropose.vue'
import FormMeeting   from '@/components/forms/FormMeeting.vue'
import FormArbitrate from '@/components/forms/FormArbitrate.vue'
import FormConsult   from '@/components/forms/FormConsult.vue'
import FormImplement from '@/components/forms/FormImplement.vue'
import FormEvaluate  from '@/components/forms/FormEvaluate.vue'
import { useProcessingStore } from '@/stores/processing'
import { filesApi, messagesApi } from '@/api'
import { priorityMeta, statusMeta, daysBetween } from '@/utils/helpers'
import type { StageCode } from '@/types'

const proc = useProcessingStore()
const activeStage = ref<StageCode>('submit')
const tab = ref<'main' | 'flow' | 'collab' | 'measures'>('main')

const FORM_MAP: Record<StageCode, any> = {
  submit: FormSubmit, review: FormReview, propose: FormPropose,
  meeting: FormMeeting, arbitrate: FormArbitrate, consult: FormConsult,
  implement: FormImplement, evaluate: FormEvaluate,
}

const detail = computed(() => proc.detail)
const problem = computed(() => detail.value?.problem)

watch(detail, () => {
  if (problem.value) activeStage.value = problem.value.currentStage as StageCode
}, { immediate: true })

function close() { proc.close() }
function onEsc(e: KeyboardEvent) { if (e.key === 'Escape') close() }

onMounted(() => {
  document.body.style.overflow = 'hidden'
  window.addEventListener('keydown', onEsc)
})
onBeforeUnmount(() => {
  document.body.style.overflow = ''
  window.removeEventListener('keydown', onEsc)
})

async function onDone() {
  await proc.reload()
}

async function refreshMessages() {
  if (!problem.value) return
  detail.value!.messages = await messagesApi.list(problem.value.id)
}
async function refreshFiles() {
  if (!problem.value) return
  detail.value!.attachments = await filesApi.list(problem.value.id)
}
</script>

<template>
  <div class="proc-modal-mask" @click.self="close">
    <div class="proc-modal-inner">
      <div v-if="!detail || !problem" class="text-muted" style="padding: 80px; text-align: center;">加载中...</div>
      <template v-else>
        <!-- header -->
        <div class="proc-header">
          <el-button text :icon="ArrowLeft" size="small" @click="close">关闭</el-button>
          <div class="flex-1" style="min-width: 0;">
            <div class="row items-center gap-2" style="margin-bottom: 4px;">
              <span class="mono text-xs text-muted">{{ problem.id }}</span>
              <span class="badge badge-square" :class="priorityMeta(problem.priority).cls">{{ priorityMeta(problem.priority).label }}</span>
              <span class="badge" :class="statusMeta(problem.status).cls">{{ statusMeta(problem.status).label }}</span>
              <span v-if="problem.overdue" class="badge badge-danger">超期 {{ problem.overdueDays }} 天</span>
            </div>
            <h2 class="proc-title">{{ problem.title }}</h2>
          </div>
          <div class="proc-header-meta">
            <div class="row gap-3 text-xs text-muted">
              <span><el-icon><User /></el-icon> 提报：{{ detail.submitter?.name }}</span>
              <span><el-icon><Flag /></el-icon> 承办：{{ problem.handlerDept }}</span>
              <span>
                <el-icon><Calendar /></el-icon>
                期限：{{ problem.dueDate }}
                <b :style="{ color: problem.overdue ? 'var(--danger)' : daysBetween(problem.dueDate) <= 3 ? 'var(--warning)' : 'var(--text-2)' }">
                  ({{ problem.overdue ? `超期 ${problem.overdueDays} 天` : `剩 ${daysBetween(problem.dueDate)} 天` }})
                </b>
              </span>
            </div>
          </div>
          <div class="row gap-2">
            <el-button text :icon="Bell" />
            <el-button text :icon="Paperclip">附件</el-button>
            <el-button text :icon="Close" @click="close" />
          </div>
        </div>

        <!-- top stage strip -->
        <StageStrip :problem="problem" :active-stage="activeStage" @select="activeStage = $event" />

        <div class="proc-layout">
          <!-- left rail -->
          <div class="proc-rail">
            <div class="card">
              <div class="card-head"><div class="card-title">问题基本信息</div></div>
              <div class="card-pad col gap-3" style="font-size: 13px;">
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">问题等级</span><span class="badge badge-square" :class="priorityMeta(problem.priority).cls">{{ priorityMeta(problem.priority).label }}</span></div>
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">问题分类</span><span class="chip">{{ problem.category }}</span></div>
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">提报方</span><AvatarBadge :name="detail.submitter?.name || '?'" :size="20" /><span>{{ detail.submitter?.name }}</span><span class="text-xs text-muted">· {{ detail.submitter?.dept }}</span></div>
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">承办单位</span><span class="text-soft font-semi">{{ problem.handlerDept }}</span></div>
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">提报日期</span><span class="mono">{{ problem.submitDate }}</span></div>
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">办理期限</span><span class="mono">{{ problem.dueDate }}</span></div>
                <div class="row items-center gap-3"><span style="color: var(--text-3); min-width: 70px;">处理路径</span>
                  <span v-if="problem.branch === 'dispute'" class="badge badge-purple">争议 · 会商裁决</span>
                  <span v-else-if="problem.branch === 'consensus'" class="badge badge-teal">共识 · 征求意见</span>
                  <span v-else class="badge">待研提阶段判定</span>
                </div>
                <div class="row items-start gap-3">
                  <span style="color: var(--text-3); min-width: 70px;">参与方</span>
                  <div class="row items-center gap-1 flex-wrap">
                    <AvatarBadge v-for="d in problem.participants.slice(0, 4)" :key="d" :name="d" :size="20" />
                    <span v-if="problem.participants.length > 4" class="text-xs text-muted">+{{ problem.participants.length - 4 }}</span>
                  </div>
                </div>
                <div class="row flex-wrap gap-1" style="margin-top: 4px;">
                  <span v-for="t in problem.tags" :key="t" class="chip" style="font-size: 11px;">#{{ t }}</span>
                </div>
              </div>
            </div>

            <div class="card" style="margin-top: 14px;">
              <div class="card-head">
                <div class="card-title">办理流程</div>
                <el-button text size="small" @click="tab = 'flow'">关系图谱</el-button>
              </div>
              <div class="card-pad">
                <FlowTimelineV :problem="problem" :history="detail.history" @jump="activeStage = $event" />
              </div>
            </div>
          </div>

          <!-- center -->
          <div class="proc-main">
            <div class="proc-tabs">
              <div class="proc-tab" :class="{ active: tab === 'main' }" @click="tab = 'main'">
                <el-icon><Tools /></el-icon> 办理
                <span v-if="activeStage !== problem.currentStage" class="badge badge-warning" style="margin-left: 6px;">查看历史阶段</span>
              </div>
              <div class="proc-tab" :class="{ active: tab === 'flow' }" @click="tab = 'flow'">
                <el-icon><Share /></el-icon> 流程图谱
              </div>
              <div class="proc-tab" :class="{ active: tab === 'collab' }" @click="tab = 'collab'">
                <el-icon><ChatLineRound /></el-icon> 协同留言
                <span class="tab-count">{{ detail.messages.length }}</span>
              </div>
              <div class="proc-tab" :class="{ active: tab === 'measures' }" @click="tab = 'measures'">
                <el-icon><Document /></el-icon> 举措清单
                <span class="tab-count">{{ detail.measures.length }}</span>
              </div>
            </div>

            <component
              v-if="tab === 'main'"
              :is="FORM_MAP[activeStage]"
              :detail="detail"
              :key="activeStage + '-' + problem.currentStage"
              @done="onDone"
            />

            <FlowGraph v-else-if="tab === 'flow'" :problem="problem" :history="detail.history" @jump="(s) => { activeStage = s; tab = 'main' }" />
            <CollabPanel v-else-if="tab === 'collab'" :problem-id="problem.id" :messages="detail.messages" :participants="problem.participants" @refresh="refreshMessages" />
            <MeasuresPanel v-else-if="tab === 'measures'" :measures="detail.measures" />
          </div>

          <!-- right rail -->
          <div class="proc-rail-right">
            <div class="card">
              <div class="card-head">
                <div class="card-title">历史进展</div>
                <span class="chip mono">{{ detail.history.length }}</span>
              </div>
              <div class="card-pad" style="max-height: 480px; overflow: auto;">
                <HistoryTimeline :history="detail.history" />
              </div>
            </div>

            <div class="card" style="margin-top: 14px;">
              <div class="card-head"><div class="card-title">相关材料</div></div>
              <div class="card-pad">
                <MaterialList :problem-id="problem.id" :attachments="detail.attachments" :active-stage="activeStage" @refresh="refreshFiles" />
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
  </div>
</template>

<style scoped>
.proc-tabs {
  display: flex; gap: 4px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--r-md);
  padding: 4px; margin-bottom: 12px;
}
.proc-tab {
  padding: 8px 14px; font-size: 13px; font-weight: 500;
  color: var(--text-2); cursor: pointer; border-radius: 7px;
  display: flex; align-items: center; gap: 6px;
}
.proc-tab:hover { background: var(--surface-3); }
.proc-tab.active { background: var(--primary-soft); color: var(--primary); font-weight: 600; }
.tab-count {
  background: var(--surface-3); padding: 1px 7px; border-radius: 999px;
  font-size: 11px; color: var(--text-3); font-weight: 600; margin-left: 4px;
}
.proc-tab.active .tab-count { background: var(--primary-soft); color: var(--primary); }
</style>
