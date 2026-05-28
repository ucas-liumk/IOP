<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElButton, ElIcon } from 'element-plus'
import { Download, Refresh, List, CircleCheck, Warning, VideoPlay, Share, Document } from '@element-plus/icons-vue'
import VChart from 'vue-echarts'
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import { PieChart, LineChart, BarChart } from 'echarts/charts'
import { GridComponent, LegendComponent, TooltipComponent, TitleComponent } from 'echarts/components'
import KpiCard from '@/components/KpiCard.vue'
import BarRow from '@/components/BarRow.vue'
import StarRating from '@/components/StarRating.vue'
import { dashboardApi } from '@/api'
import { useProcessingStore } from '@/stores/processing'
import { useRouter } from 'vue-router'
import type { DashboardOverview } from '@/types'

use([CanvasRenderer, PieChart, LineChart, BarChart, GridComponent, LegendComponent, TooltipComponent, TitleComponent])

const router = useRouter()
const proc = useProcessingStore()
const data = ref<DashboardOverview | null>(null)
const period = ref('month')

onMounted(load)
async function load() { data.value = await dashboardApi.overview() }

const kpis = computed(() => {
  if (!data.value) return []
  const k = data.value.kpis
  return [
    { id: 'total',      label: '问题总数', value: k.total,         delta: 12.5,  icon: List,        color: 'submit' },
    { id: 'review',     label: '待审核',   value: k.pendingReview, delta: 5.2,   icon: Document,    color: 'review' },
    { id: 'assign',     label: '待分办',   value: k.pendingAssign, delta: -3.1,  icon: Share,       color: 'review',   deltaInverted: true },
    { id: 'processing', label: '正在办理', value: k.processing,    delta: 8.7,   icon: VideoPlay,   color: 'propose' },
    { id: 'done',       label: '已办结',   value: k.done,          delta: 15.3,  icon: CircleCheck, color: 'consult' },
    { id: 'overdue',    label: '超期问题', value: k.overdue,       delta: -21.4, icon: Warning,     color: 'arbitrate', deltaInverted: true, accent: 'var(--danger)' },
  ]
})

// ECharts options ------------------------------------------------------------
const CATEGORY_PALETTE = ['#1e5fd9', '#7c4ddb', '#0fa8a3', '#d63838', '#e8920e', '#b14fa0', '#8896ad']
const categoryOption = computed(() => {
  if (!data.value) return {}
  return {
    tooltip: { trigger: 'item' },
    legend: { show: false },
    series: [{
      name: '问题类型', type: 'pie',
      radius: ['56%', '78%'], avoidLabelOverlap: false,
      itemStyle: { borderRadius: 6, borderColor: '#fff', borderWidth: 2 },
      label: { show: false },
      data: data.value.categories.map((c, i) => ({ name: c.k, value: c.v, itemStyle: { color: CATEGORY_PALETTE[i % CATEGORY_PALETTE.length] } })),
    }],
  }
})

const trendOption = computed(() => {
  if (!data.value) return {}
  const months = data.value.trend.map((t) => t.month.slice(2))
  return {
    grid: { left: 36, right: 16, top: 28, bottom: 28 },
    tooltip: { trigger: 'axis' },
    legend: { right: 0, top: 0, icon: 'circle' },
    xAxis: { type: 'category', data: months, axisLine: { lineStyle: { color: '#d4dbe8' } }, axisTick: { show: false }, axisLabel: { color: '#7b8aa3', fontSize: 11 } },
    yAxis: { type: 'value', splitLine: { lineStyle: { color: '#eef1f7', type: 'dashed' } }, axisLabel: { color: '#7b8aa3', fontSize: 11 } },
    series: [
      {
        name: '提报', type: 'line', smooth: true, symbol: 'circle', symbolSize: 6,
        lineStyle: { width: 2.2, color: '#1e5fd9' }, itemStyle: { color: '#1e5fd9' },
        areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(30,95,217,0.18)' }, { offset: 1, color: 'rgba(30,95,217,0)' }] } },
        data: data.value.trend.map((t) => t.submit),
      },
      {
        name: '办结', type: 'line', smooth: true, symbol: 'circle', symbolSize: 6,
        lineStyle: { width: 2.2, color: '#1aa971' }, itemStyle: { color: '#1aa971' },
        areaStyle: { color: { type: 'linear', x: 0, y: 0, x2: 0, y2: 1, colorStops: [{ offset: 0, color: 'rgba(26,169,113,0.18)' }, { offset: 1, color: 'rgba(26,169,113,0)' }] } },
        data: data.value.trend.map((t) => t.done),
      },
    ],
  }
})

// 阶段分布带颜色
const STAGE_COLOR: Record<string, string> = {
  propose:   'var(--stage-propose)',
  meeting:   'var(--stage-meeting)',
  arbitrate: 'var(--stage-arbitrate)',
  consult:   'var(--stage-consult)',
  implement: 'var(--stage-implement)',
  review:    'var(--stage-review)',
  submit:    'var(--stage-submit)',
}
const STAGE_LABEL: Record<string, string> = {
  submit: '问题提报', review: '审核分办', propose: '研提举措',
  meeting: '会商研究', arbitrate: '争议裁决', consult: '征求意见',
  implement: '督导落实', evaluate: '评价反馈',
}
const procBandTotal = computed(() => data.value?.processingBreakdown.reduce((s, p) => s + p.v, 0) || 0)

function goProblems(stage?: string) {
  router.push({ path: '/problems', query: stage ? { stage } : {} })
}
</script>

<template>
  <div v-if="data" class="page">
    <!-- header -->
    <div class="page-header">
      <div>
        <h1 class="page-title">
          <span style="background: linear-gradient(90deg, var(--primary) 0%, var(--accent) 100%); -webkit-background-clip: text; -webkit-text-fill-color: transparent;">
            全局态势总览
          </span>
        </h1>
        <div class="page-subtitle">问题协同研究解决平台 · 数据更新于 {{ new Date().toISOString().slice(0, 10) }} 09:00</div>
      </div>
      <div class="page-actions">
        <el-button-group>
          <el-button v-for="p in ['today', 'week', 'month', 'quarter', 'year']" :key="p" size="small" :type="period === p ? 'primary' : ''" @click="period = p">
            {{ { today: '今日', week: '本周', month: '本月', quarter: '本季度', year: '本年度' }[p] }}
          </el-button>
        </el-button-group>
        <el-button size="small" :icon="Download">导出报表</el-button>
        <el-button size="small" :icon="Refresh" @click="load">刷新</el-button>
      </div>
    </div>

    <!-- KPI grid -->
    <div class="dash-grid">
      <KpiCard v-for="k in kpis" :key="k.id" v-bind="k" />
    </div>

    <!-- Processing breakdown band -->
    <div class="card" style="margin-top: 14px; padding: 18px 22px;">
      <div class="row items-center justify-between" style="margin-bottom: 14px;">
        <div>
          <div class="font-semi" style="font-size: 14px;">办理中 · 阶段分布</div>
          <div class="text-xs text-muted" style="margin-top: 2px;">当前共 <b class="mono">{{ procBandTotal }}</b> 个问题正在办理，点击进入对应阶段</div>
        </div>
        <div class="row gap-3 items-center">
          <span class="chip"><span style="width:6px;height:6px;border-radius:999px;background:var(--purple);" />争议路径</span>
          <span class="chip"><span style="width:6px;height:6px;border-radius:999px;background:var(--teal);" />共识路径</span>
        </div>
      </div>
      <div class="proc-band">
        <div
          v-for="p in data.processingBreakdown"
          :key="p.k"
          class="proc-band-seg"
          :style="{ width: (p.v / procBandTotal * 100) + '%', background: STAGE_COLOR[p.k as string] || 'var(--neutral)' }"
          @click="goProblems(p.k as string)"
        >
          <div class="proc-band-content">
            <div class="proc-band-label">{{ STAGE_LABEL[p.k as string] || p.k }}</div>
            <div class="proc-band-value mono">{{ p.v }}</div>
          </div>
        </div>
      </div>
      <div class="proc-band-legend">
        <div v-for="p in data.processingBreakdown" :key="p.k" class="proc-band-li">
          <span style="width:6px;height:6px;border-radius:999px;display:inline-block" :style="{ background: STAGE_COLOR[p.k as string] }" />
          <span class="text-soft">{{ STAGE_LABEL[p.k as string] }}</span>
          <span class="mono font-semi">{{ p.v }}</span>
        </div>
      </div>
    </div>

    <!-- Category + Dispute + Overdue -->
    <div class="dash-row cols-3">
      <div class="card card-pad">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title">问题类型分布</div>
        </div>
        <div class="row items-center gap-5">
          <VChart :option="categoryOption" autoresize style="width: 180px; height: 180px;" />
          <div class="col gap-2 flex-1">
            <div
              v-for="(c, i) in data.categories"
              :key="c.k as string"
              class="row items-center gap-2 justify-between"
              style="padding: 3px 0;"
            >
              <div class="row items-center gap-2">
                <span style="width:6px;height:6px;border-radius:999px;display:inline-block" :style="{ background: CATEGORY_PALETTE[i % CATEGORY_PALETTE.length] }" />
                <span class="text-sm text-soft">{{ c.k }}</span>
              </div>
              <span class="mono text-sm font-semi" style="min-width: 28px; text-align: right;">{{ c.v }}</span>
            </div>
          </div>
        </div>
      </div>

      <div class="card card-pad">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title" style="color: var(--purple);">争议研究分支</div>
          <span class="badge badge-purple">双路径</span>
        </div>
        <div class="text-xs text-muted" style="margin-bottom: 12px;">研提举措后路径分流统计</div>
        <div class="dispute-bar" v-if="data.disputeStats.totalPropose > 0">
          <div
            class="dispute-seg dispute-consensus"
            :style="{ width: (data.disputeStats.consultPath / data.disputeStats.totalPropose * 100) + '%' }"
          >
            <div class="dispute-seg-label">
              <span class="mono font-bold" style="font-size: 18px;">{{ data.disputeStats.consultPath }}</span>
              <span class="text-xs">无争议 · 征求意见</span>
            </div>
          </div>
          <div
            class="dispute-seg dispute-dispute"
            :style="{ width: (data.disputeStats.withDispute / data.disputeStats.totalPropose * 100) + '%' }"
          >
            <div class="dispute-seg-label">
              <span class="mono font-bold" style="font-size: 18px;">{{ data.disputeStats.withDispute }}</span>
              <span class="text-xs">有争议 · 会商裁决</span>
            </div>
          </div>
        </div>
        <div class="dispute-grid" style="margin-top: 14px;">
          <div class="dispute-stat">
            <div class="dispute-stat-v mono">{{ data.disputeStats.disputeRate }}%</div>
            <div class="dispute-stat-l">争议触发率</div>
          </div>
          <div class="dispute-stat">
            <div class="dispute-stat-v mono">{{ data.disputeStats.avgMeetings }}</div>
            <div class="dispute-stat-l">平均会商次数</div>
          </div>
          <div class="dispute-stat">
            <div class="dispute-stat-v mono">{{ data.disputeStats.arbitrateDone }}</div>
            <div class="dispute-stat-l">已裁决</div>
          </div>
        </div>
      </div>

      <div class="card card-pad" style="background: linear-gradient(135deg, #fef0f0 0%, #ffffff 60%); border-color: var(--danger-soft);">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title" style="color: var(--danger);">超期预警 TOP 5 单位</div>
          <el-button text size="small" @click="goProblems()">查看全部</el-button>
        </div>
        <div class="col gap-3">
          <div v-for="(u, i) in data.overdueByDept" :key="u.k as string" class="overdue-row">
            <div class="overdue-rank" :class="i < 3 ? `r${i + 1}` : ''">{{ i + 1 }}</div>
            <div class="flex-1">
              <div class="font-semi text-sm">{{ u.k }}</div>
              <div class="overdue-bar">
                <div class="overdue-bar-fill" :style="{ width: (Number(u.v) / Number(data.overdueByDept[0].v) * 100) + '%' }" />
              </div>
            </div>
            <div class="mono font-bold" style="color: var(--danger); font-size: 16px;">
              {{ u.v }}<span class="text-xs text-muted" style="font-weight: normal;">个</span>
            </div>
          </div>
          <div v-if="!data.overdueByDept.length" class="text-muted text-sm">暂无超期</div>
        </div>
      </div>
    </div>

    <!-- Top units -->
    <div class="dash-row cols-2">
      <div class="card card-pad">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title">单位提报数 TOP 10</div>
        </div>
        <div class="col gap-1">
          <BarRow
            v-for="u in data.topSubmitterDepts"
            :key="u.k as string"
            :label="String(u.k)"
            :value="Number(u.v)"
            :max="Number(data.topSubmitterDepts[0].v)"
            color="var(--primary)"
            suffix="个"
          />
        </div>
      </div>
      <div class="card card-pad">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title">单位办理数 TOP 10</div>
        </div>
        <div class="col gap-1">
          <BarRow
            v-for="u in data.topHandlerDepts"
            :key="u.k as string"
            :label="String(u.k)"
            :value="Number(u.v)"
            :max="Number(data.topHandlerDepts[0].v)"
            color="var(--success)"
            suffix="个"
          />
        </div>
      </div>
    </div>

    <!-- Trend + Satisfaction -->
    <div class="dash-row cols-trend">
      <div class="card card-pad">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title">近 12 月提报与办结趋势</div>
        </div>
        <VChart :option="trendOption" autoresize style="height: 240px;" />
      </div>

      <div class="card card-pad">
        <div class="row items-center justify-between" style="margin-bottom: 12px;">
          <div class="card-title">单位满意度排名</div>
          <span class="chip">满分 5.0</span>
        </div>
        <table class="dash-table">
          <thead>
            <tr>
              <th style="width: 40px;">排名</th>
              <th>单位</th>
              <th style="width: 60px; text-align: right;">评分</th>
              <th style="width: 100px; text-align: right;">星级</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(u, i) in data.satisfaction.slice(0, 8)" :key="u.name">
              <td>
                <span class="rank-pill" :class="[i < 3 ? 'top' : '', i === 0 ? 'gold' : i === 1 ? 'silver' : i === 2 ? 'bronze' : '']">{{ i + 1 }}</span>
              </td>
              <td class="font-semi">{{ u.name }}</td>
              <td class="mono font-semi" style="text-align: right;">{{ u.score }}</td>
              <td style="text-align: right;"><StarRating :score="Number(u.score)" :size="12" /></td>
            </tr>
            <tr v-if="!data.satisfaction.length">
              <td colspan="4" class="text-muted text-sm" style="text-align: center; padding: 20px;">暂无评价数据</td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
  <div v-else class="page text-muted" style="padding: 60px; text-align: center;">加载中...</div>
</template>

<style scoped>
.dash-grid { display: grid; grid-template-columns: repeat(6, 1fr); gap: 14px; }
.dash-row { display: grid; gap: 14px; margin-top: 14px; }
.dash-row.cols-2 { grid-template-columns: 1fr 1fr; }
.dash-row.cols-3 { grid-template-columns: 1.2fr 1.1fr 1fr; }
.dash-row.cols-trend { grid-template-columns: 2fr 1fr; }

.proc-band { display: flex; width: 100%; height: 64px; border-radius: 12px; overflow: hidden; gap: 3px; }
.proc-band-seg { display: flex; align-items: center; justify-content: center; cursor: pointer; transition: all .2s ease; min-width: 60px; }
.proc-band-seg:hover { filter: brightness(1.08); }
.proc-band-content { text-align: center; color: white; display: flex; flex-direction: column; gap: 2px; padding: 0 8px; }
.proc-band-label { font-size: 11.5px; font-weight: 500; opacity: 0.95; }
.proc-band-value { font-size: 18px; font-weight: 700; line-height: 1.1; }
.proc-band-legend { display: flex; gap: 24px; flex-wrap: wrap; margin-top: 12px; font-size: 12.5px; }
.proc-band-li { display: flex; align-items: center; gap: 6px; }

.dispute-bar { display: flex; border-radius: 10px; overflow: hidden; height: 88px; background: var(--surface-3); }
.dispute-seg { display: flex; align-items: center; justify-content: center; padding: 8px 12px; transition: width .8s ease; min-width: 80px; }
.dispute-consensus { background: linear-gradient(135deg, var(--teal) 0%, #0fa8a3aa 100%); color: white; }
.dispute-dispute   { background: linear-gradient(135deg, var(--purple) 0%, #7c4ddbaa 100%); color: white; }
.dispute-seg-label { display: flex; flex-direction: column; align-items: center; gap: 2px; }
.dispute-grid { display: grid; grid-template-columns: repeat(3, 1fr); gap: 8px; }
.dispute-stat { background: var(--surface-3); padding: 10px 12px; border-radius: 10px; text-align: center; }
.dispute-stat-v { font-size: 18px; font-weight: 700; color: var(--text); }
.dispute-stat-l { font-size: 11px; color: var(--text-3); margin-top: 2px; }

.overdue-row { display: flex; align-items: center; gap: 12px; }
.overdue-rank { width: 22px; height: 22px; border-radius: 6px; background: var(--surface-3); display: flex; align-items: center; justify-content: center; font-size: 12px; font-weight: 700; color: var(--text-3); flex-shrink: 0; }
.overdue-rank.r1 { background: linear-gradient(135deg, #f5a524, #e87f0e); color: white; }
.overdue-rank.r2 { background: linear-gradient(135deg, #a3b1c8, #7b8aa3); color: white; }
.overdue-rank.r3 { background: linear-gradient(135deg, #d4a574, #b08555); color: white; }
.overdue-bar { margin-top: 4px; height: 4px; background: var(--danger-soft); border-radius: 999px; overflow: hidden; }
.overdue-bar-fill { height: 100%; background: var(--danger); border-radius: 999px; transition: width .8s ease; }

.dash-table { width: 100%; border-collapse: separate; border-spacing: 0; }
.dash-table th, .dash-table td { padding: 9px 8px; border-bottom: 1px solid var(--border-soft); font-size: 13px; text-align: left; }
.dash-table th { font-weight: 500; color: var(--text-3); font-size: 11.5px; text-transform: uppercase; letter-spacing: 0.04em; }
.dash-table tbody tr:hover { background: var(--surface-3); }
.dash-table tbody tr:last-child td { border-bottom: none; }
.rank-pill { display: inline-flex; align-items: center; justify-content: center; width: 22px; height: 22px; border-radius: 6px; background: var(--surface-3); font-size: 11px; font-weight: 700; color: var(--text-3); }
.rank-pill.top.gold   { background: linear-gradient(135deg, #f5a524, #e87f0e); color: white; }
.rank-pill.top.silver { background: linear-gradient(135deg, #a3b1c8, #7b8aa3); color: white; }
.rank-pill.top.bronze { background: linear-gradient(135deg, #d4a574, #b08555); color: white; }
</style>
