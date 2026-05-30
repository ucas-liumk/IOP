<template>
  <section class="home">
    <!-- ===== Welcome ===== -->
    <div class="welcome">
      <div class="welcome-left">
        <h1>
          {{ greetingPrefix }}，{{ greetingName }}
          <span class="wave">👋</span>
        </h1>
        <div class="welcome-meta">
          {{ todayLabel }} ·
          5 月你完成了 <strong>{{ monthDone }}</strong> 项工作，
          <span class="delta">↑ {{ momPercent }}%</span> 较上月
        </div>
      </div>
      <div class="welcome-right">
        <div class="mini-chip">
          <span class="label">待我处理</span>
          <span class="num">{{ stats.todo }}</span>
        </div>
        <div class="mini-chip">
          <span class="label">我提报的</span>
          <span class="num">{{ stats.submitted }}</span>
        </div>
        <div class="mini-chip">
          <span class="label">抄送我</span>
          <span class="num">{{ stats.cc }}</span>
        </div>
        <div class="mini-chip overdue">
          <span class="label">已超期</span>
          <span class="num">{{ stats.overdue }}</span>
        </div>
      </div>
    </div>

    <!-- ===== Grid ===== -->
    <div class="grid-main">
      <!-- ===== LEFT COL ===== -->
      <div class="col-left">
        <!-- 每周议程 -->
        <article class="card">
          <header class="card-header">
            <div class="card-title">
              本周议程
              <span class="sub">· {{ weekRangeLabel }}</span>
            </div>
            <a class="card-link primary" href="#">查看月历 →</a>
          </header>
          <div class="agenda-toolbar">
            <button class="nav-arrow">‹</button>
            <span class="week-label">{{ weekLabel }}</span>
            <button class="nav-arrow">›</button>
            <button class="today-btn">回到今天</button>
            <div class="switch">
              <span class="on">周</span><span>月</span>
            </div>
          </div>
          <div class="agenda-grid">
            <div
              v-for="d in weekDays"
              :key="d.iso"
              class="agenda-day"
              :class="{ today: d.isToday, weekend: d.weekday >= 6 }"
            >
              <div class="day-head">
                <span class="day-weekday">{{ d.weekdayLabel }}</span>
                <span class="day-num" :class="{ 'today-num': d.isToday }">{{ d.dom }}</span>
                <span v-if="d.events.length" class="day-tag">{{ d.events.length }}</span>
              </div>
              <div v-for="ev in d.events" :key="ev.id" :class="['event', ev.color]">
                <div class="event-time">{{ ev.time }}</div>
                <div class="event-title">{{ ev.title }}</div>
              </div>
              <div v-if="!d.events.length" class="no-event">无安排</div>
            </div>
          </div>
        </article>

        <!-- 新通知 -->
        <article class="card">
          <header class="card-header">
            <div class="card-title">
              新通知
              <span class="badge-num">12</span>
            </div>
            <a class="card-link" href="#">全部通知 →</a>
          </header>
          <div class="notif-tabs">
            <button class="notif-tab on">全部<span class="n">12</span></button>
            <button class="notif-tab has-new">@我<span class="n">3</span></button>
            <button class="notif-tab">审批<span class="n">2</span></button>
            <button class="notif-tab">评论<span class="n">5</span></button>
            <button class="notif-tab">系统<span class="n">2</span></button>
          </div>
          <div class="notif-list">
            <div v-for="n in notifications" :key="n.id" class="notif-item" :class="{ unread: n.unread }">
              <div :class="['notif-avatar', n.kind]">
                {{ n.initial }}
                <span class="source-mark">
                  <svg width="8" height="8" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="6"/></svg>
                </span>
              </div>
              <div class="notif-body">
                <div class="notif-title">
                  <b>{{ n.actor }}</b> {{ n.verb }}
                  <span v-if="n.atMe" class="at">@你</span>
                  <span v-if="n.quote" class="quote">{{ n.quote }}</span>
                </div>
                <div class="notif-meta">
                  <span class="app-src">{{ n.app }}</span>
                  <span class="sep"></span>
                  <span>{{ n.where }}</span>
                  <span class="sep"></span>
                  <span>{{ n.time }}</span>
                </div>
                <div v-if="n.actions?.length" class="notif-actions">
                  <button v-for="(a, i) in n.actions" :key="a" :class="{ primary: i === 0 }">{{ a }}</button>
                </div>
              </div>
            </div>
          </div>
        </article>

        <!-- 我的待办 -->
        <article class="card">
          <header class="card-header">
            <div class="card-title">
              我的待办
              <span class="sub">· 来自你装载的应用</span>
            </div>
            <a class="card-link" href="#">全部待办 →</a>
          </header>
          <div class="todo-tabs">
            <button class="todo-tab t-review">待审核<span class="n">{{ todoCounts.review }}</span></button>
            <button class="todo-tab t-assign">待分办<span class="n">{{ todoCounts.assign }}</span></button>
            <button class="todo-tab t-propose">待研提<span class="n">{{ todoCounts.propose }}</span></button>
            <button class="todo-tab t-eval">待评价<span class="n">{{ todoCounts.eval }}</span></button>
          </div>
          <div class="todo-list">
            <div v-for="t in todos" :key="t.id" class="todo-item" :class="'pri-' + t.priority">
              <div class="pri-bar"></div>
              <div class="todo-content">
                <div class="todo-title-row">
                  <span class="title-text">{{ t.title }}</span>
                  <span v-if="t.priority === 'urgent'" class="pri-tag urgent">紧急</span>
                  <span v-else-if="t.priority === 'high'" class="pri-tag high">高</span>
                </div>
                <div class="todo-meta">
                  <span class="app-tag">
                    <svg width="10" height="10" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><circle cx="12" cy="12" r="9"/><circle cx="12" cy="12" r="4"/></svg>
                    {{ t.app }}
                  </span>
                  <span class="dot-sep"></span>
                  <span>{{ t.stage }}</span>
                  <span class="dot-sep"></span>
                  <span :class="['time', { over: t.over }]">{{ t.time }}</span>
                </div>
              </div>
              <div class="todo-actions">
                <button>处理</button>
              </div>
            </div>
          </div>
        </article>

        <!-- 最近 -->
        <article class="card">
          <header class="card-header">
            <div class="card-title">最近</div>
            <a class="card-link" href="#">查看更多 →</a>
          </header>
          <div class="docs-list">
            <div v-for="d in recent" :key="d.id" class="doc-item">
              <div class="doc-icon" :style="{ background: d.color }">
                <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
                  <path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/>
                </svg>
              </div>
              <div class="doc-info">
                <div class="doc-name">{{ d.name }}</div>
                <div class="doc-meta"><span>{{ d.app }}</span><span class="dot-sep"></span><span>{{ d.author }}</span></div>
              </div>
              <div class="doc-time">{{ d.time }}</div>
            </div>
          </div>
        </article>
      </div>

      <!-- ===== RIGHT COL ===== -->
      <div class="col-right">
        <!-- 平台公告 -->
        <article class="card">
          <header class="card-header">
            <div class="card-title">平台公告</div>
            <a class="card-link" href="#">全部 →</a>
          </header>
          <div class="ann-list">
            <div class="ann-item">
              <div class="ann-icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M3 11l18-5v12L3 14v-3z"/><path d="M11.6 16.8a3 3 0 1 1-5.8-1.6"/></svg>
              </div>
              <div class="ann-text">
                <div class="ann-title">平台 v1.0 正式上线<span class="ann-tag">NEW</span></div>
                <div class="ann-date">5月26日 · 产品团队</div>
              </div>
            </div>
            <div class="ann-item">
              <div class="ann-icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><ellipse cx="12" cy="5" rx="9" ry="3"/><path d="M21 12c0 1.66-4 3-9 3s-9-1.34-9-3"/><path d="M3 5v14c0 1.66 4 3 9 3s9-1.34 9-3V5"/></svg>
              </div>
              <div class="ann-text">
                <div class="ann-title">KingbaseV8 数据库支持已就绪</div>
                <div class="ann-date">5月20日 · 运维团队</div>
              </div>
            </div>
            <div class="ann-item">
              <div class="ann-icon">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M9 9h6v6H9z"/></svg>
              </div>
              <div class="ann-text">
                <div class="ann-title">应用市场新增 4 款推荐应用</div>
                <div class="ann-date">5月15日 · 应用中心</div>
              </div>
            </div>
          </div>
        </article>

        <!-- 快速链接 -->
        <article class="card">
          <header class="card-header">
            <div class="card-title">快速链接</div>
          </header>
          <div class="quick-links">
            <a class="q-link" href="#">
              <div class="q-ico" style="background: var(--primary-soft); color: var(--primary);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/></svg>
              </div>
              <div>
                <div class="q-name">常用文档</div>
                <div class="q-sub">12 篇</div>
              </div>
            </a>
            <a class="q-link" href="#">
              <div class="q-ico" style="background: var(--success-soft); color: var(--success);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/></svg>
              </div>
              <div>
                <div class="q-name">通讯录</div>
                <div class="q-sub">{{ memberCount }} 人</div>
              </div>
            </a>
            <a class="q-link" href="#">
              <div class="q-ico" style="background: var(--purple-soft); color: var(--purple);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><rect x="3" y="4" width="18" height="18" rx="2"/><line x1="16" y1="2" x2="16" y2="6"/><line x1="8" y1="2" x2="8" y2="6"/><line x1="3" y1="10" x2="21" y2="10"/></svg>
              </div>
              <div>
                <div class="q-name">会议室</div>
                <div class="q-sub">预订</div>
              </div>
            </a>
            <a class="q-link" href="#">
              <div class="q-ico" style="background: var(--teal-soft); color: var(--teal);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M9.09 9a3 3 0 0 1 5.83 1c0 2-3 3-3 3"/></svg>
              </div>
              <div>
                <div class="q-name">知识库</div>
                <div class="q-sub">检索</div>
              </div>
            </a>
          </div>
        </article>

        <!-- 推荐应用 -->
        <article class="card rec-mini-card">
          <header class="card-header">
            <div class="card-title">为你推荐 · 应用市场</div>
            <a class="card-link primary" href="#">浏览全部 →</a>
          </header>
          <div class="rec-list">
            <div class="rec-mini-item">
              <div class="icon-box" style="background: var(--cat-collab);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><line x1="8" y1="6" x2="21" y2="6"/><line x1="8" y1="12" x2="21" y2="12"/><line x1="8" y1="18" x2="21" y2="18"/></svg>
              </div>
              <div>
                <div class="rec-mini-name">周报助手 <span class="rating">4.9★</span></div>
                <div class="rec-mini-desc">AI 辅助生成部门周报</div>
              </div>
            </div>
            <div class="rec-mini-item">
              <div class="icon-box" style="background: var(--cat-biz);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10"/><path d="M9 12l2 2 4-4"/></svg>
              </div>
              <div>
                <div class="rec-mini-name">客户管理 CRM <span class="rating">4.6★</span></div>
                <div class="rec-mini-desc">销售线索到合同</div>
              </div>
            </div>
            <div class="rec-mini-item">
              <div class="icon-box" style="background: var(--cat-hr);">
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polygon points="14 2 18 6 7 17 3 17 3 13 14 2"/></svg>
              </div>
              <div>
                <div class="rec-mini-name">智能合同 <span class="rating">4.5★</span></div>
                <div class="rec-mini-desc">模板 + 电子签</div>
              </div>
            </div>
          </div>
        </article>
      </div>
    </div>
  </section>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from "vue";
import { useAuthStore } from "@/shell/auth/auth.store";
import { listPlans, listReports } from "@/modules/okr/api/okr";

const auth = useAuthStore();

const now = new Date();
const greetingName = computed(() => auth.user?.email?.split("@")[0] ?? "同学");
const greetingPrefix = computed(() => {
  const h = now.getHours();
  if (h < 6) return "凌晨好"; if (h < 12) return "上午好"; if (h < 14) return "中午好";
  if (h < 18) return "下午好"; return "晚上好";
});
const todayLabel = now.toLocaleDateString("zh-CN", { year: "numeric", month: "long", day: "numeric", weekday: "long" });

const monthDone = ref(0);
const momPercent = ref(18);
const memberCount = ref(38);

interface MiniStats { todo: number; submitted: number; cc: number; overdue: number; }
const stats = ref<MiniStats>({ todo: 12, submitted: 7, cc: 24, overdue: 2 });

// Build week (Mon→Sun) anchored on today
interface DayCell {
  iso: string; dom: number; weekday: number; weekdayLabel: string; isToday: boolean;
  events: { id: string; time: string; title: string; color: string }[];
}
const weekDays = ref<DayCell[]>([]);
const weekLabel = ref("");
const weekRangeLabel = ref("");

function buildWeek() {
  const todayIso = now.toISOString().slice(0, 10);
  const dow = now.getDay() === 0 ? 7 : now.getDay();
  const monday = new Date(now.getTime() - (dow - 1) * 86400000);
  const labels = ["MON","TUE","WED","THU","FRI","SAT","SUN"];

  const cells: DayCell[] = [];
  for (let i = 0; i < 7; i++) {
    const d = new Date(monday.getTime() + i * 86400000);
    const iso = d.toISOString().slice(0, 10);
    cells.push({
      iso, dom: d.getDate(), weekday: i + 1, weekdayLabel: labels[i],
      isToday: iso === todayIso,
      events: [],
    });
  }
  // Seed varied demo events; today gets two real-ish entries.
  const seed: Record<number, { time: string; title: string; color: string }[]> = {
    0: [{ time: "10:00", title: "周一例会", color: "" }],
    1: [{ time: "14:00", title: "客户响应优化评审", color: "warning" }],
    2: [
      { time: "09:30", title: "OKR Q3 对齐", color: "purple" },
      { time: "16:00", title: "海外合规事项落实", color: "danger" },
    ],
    3: [{ time: "11:00", title: "产品路线图分歧讨论", color: "warning" }],
    4: [{ time: "10:30", title: "财务月度复盘", color: "success" }],
    5: [],
    6: [],
  };
  cells.forEach((c, i) => {
    seed[i]?.forEach((e, j) => c.events.push({ id: `${i}-${j}`, ...e }));
  });
  weekDays.value = cells;

  const start = monday.toLocaleDateString("zh-CN", { month: "long", day: "numeric" });
  const end = new Date(monday.getTime() + 6 * 86400000)
    .toLocaleDateString("zh-CN", { month: "long", day: "numeric" });
  weekLabel.value = `${start} – ${end}`;
  weekRangeLabel.value = `第 ${getISOWeek(now)} 周`;
}

function getISOWeek(d: Date) {
  const date = new Date(Date.UTC(d.getFullYear(), d.getMonth(), d.getDate()));
  date.setUTCDate(date.getUTCDate() + 4 - (date.getUTCDay() || 7));
  const yearStart = new Date(Date.UTC(date.getUTCFullYear(), 0, 1));
  return Math.ceil(((+date - +yearStart) / 86400000 + 1) / 7);
}

// Notifications — demonstrative, mix of types
interface Notif {
  id: string; kind: string; initial: string; actor: string; verb: string;
  atMe?: boolean; quote?: string; app: string; where: string; time: string;
  actions?: string[]; unread?: boolean;
}
const notifications = ref<Notif[]>([
  { id: "1", kind: "mention", initial: "张", actor: "张明华", verb: "在评论中提到了",
    atMe: true, quote: "@你 这个方案的预算分配需要再核对一下，是否考虑加上 Q3 的浮动？",
    app: "OKR", where: "完成 Q3 战略目标", time: "12 分钟前",
    actions: ["回复", "查看"], unread: true },
  { id: "2", kind: "approval", initial: "李", actor: "李雨晨", verb: "提交了一份审批",
    quote: "差旅报销 ¥3,820 · 北京 → 上海 客户拜访",
    app: "审批", where: "费用报销", time: "1 小时前",
    actions: ["通过", "驳回"], unread: true },
  { id: "3", kind: "comment", initial: "王", actor: "王芳",
    verb: "在你的周报上发表了评论",
    quote: "本周的成果很扎实，下周的几个风险项需要重点跟进。",
    app: "OKR", where: "演示用户 · 周报", time: "3 小时前",
    actions: ["查看周报"] },
  { id: "4", kind: "system", initial: "系", actor: "系统",
    verb: "OKR 应用 v1.0.3 发布，新增「计划复制」能力",
    app: "系统", where: "应用更新", time: "昨天" },
  { id: "5", kind: "urgent", initial: "急", actor: "海外合规专班",
    verb: "请求你审批",
    quote: "海外合规事项落实方案，截止 5/30 · 已超期 3 天",
    app: "OKR", where: "审核分办", time: "2 天前", actions: ["立即处理"], unread: true },
]);

const todoCounts = { review: 3, assign: 2, propose: 4, eval: 3 };
interface Todo { id: string; title: string; priority: "urgent"|"high"|"normal"; app: string; stage: string; time: string; over?: boolean }
const todos = ref<Todo[]>([
  { id: "t1", title: "海外合规事项落实", priority: "urgent", app: "OKR", stage: "待审核", time: "超期 3 天", over: true },
  { id: "t2", title: "H2 产品路线图分歧", priority: "urgent", app: "OKR", stage: "待研提举措", time: "今天" },
  { id: "t3", title: "客户响应延迟优化", priority: "high", app: "OKR", stage: "待分办", time: "明天" },
  { id: "t4", title: "Q2 营销预算超支", priority: "high", app: "OKR", stage: "待裁决", time: "本周内" },
  { id: "t5", title: "远程办公政策修订", priority: "normal", app: "OKR", stage: "待征求意见", time: "5 月 30 日" },
  { id: "t6", title: "组织架构调整方案评审", priority: "normal", app: "OKR", stage: "待评价", time: "下周" },
]);

interface RecentDoc { id: string; name: string; app: string; author: string; time: string; color: string }
const recent = ref<RecentDoc[]>([]);

onMounted(async () => {
  buildWeek();
  // Pull real data where possible
  try {
    const plans = await listPlans("week");
    monthDone.value = plans.length * 5; // demo derivation
    stats.value.submitted = Math.max(plans.length, 1);
  } catch {}
  try {
    const reports = await listReports();
    stats.value.todo = Math.max(reports.length, 6);
    recent.value = reports.slice(0, 4).map((r, i) => ({
      id: r.id,
      name: (r.type === "daily" ? "日报：" : "周报：") + r.summary.slice(0, 30) + "…",
      app: "OKR",
      author: greetingName.value,
      time: new Date(r.submitted_at).toLocaleDateString("zh-CN"),
      color: ["#1e5fd9", "#7c4ddb", "#0fa8a3", "#e8920e"][i % 4],
    }));
  } catch {}
  if (recent.value.length === 0) {
    recent.value = [
      { id: "r1", name: "2026 H2 战略规划草稿", app: "文档", author: "陈雨晴", time: "昨天", color: "#1e5fd9" },
      { id: "r2", name: "客户响应延迟优化-讨论纪要", app: "OKR", author: "李明", time: "2 天前", color: "#7c4ddb" },
      { id: "r3", name: "海外合规事项-合规清单.xlsx", app: "文档", author: "王芳", time: "3 天前", color: "#1aa971" },
    ];
  }
});
</script>

<style scoped>
.home {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}

/* ===== Welcome ===== */
.welcome {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 18px 24px;
  background: linear-gradient(110deg, #eef3fe 0%, #f1ecfb 50%, #e3f6f5 100%);
  border-radius: 14px;
  position: relative;
  overflow: hidden;
  border: 1px solid rgba(255,255,255,.6);
}
.welcome::before {
  content: "";
  position: absolute; top: -50%; right: -8%;
  width: 280px; height: 280px;
  background: radial-gradient(circle, rgba(30,95,217,.10), transparent 60%);
  border-radius: 50%; pointer-events: none;
}
.welcome-left { position: relative; z-index: 1; }
.welcome h1 {
  margin: 0;
  font-size: 20px; font-weight: 700;
  color: var(--text); letter-spacing: -.3px;
}
.welcome h1 .wave {
  display: inline-block;
  animation: wave 2.4s ease-in-out infinite;
  transform-origin: 70% 70%;
}
@keyframes wave {
  0%,60%,100% { transform: rotate(0); }
  10% { transform: rotate(14deg); } 20% { transform: rotate(-8deg); }
  30% { transform: rotate(14deg); } 40% { transform: rotate(-4deg); }
  50% { transform: rotate(10deg); }
}
.welcome-meta { margin-top: 4px; font-size: 13px; color: var(--text-2); }
.welcome-meta .delta { color: var(--success); font-weight: 600; }
.welcome-right {
  display: flex; gap: 8px; flex-wrap: wrap;
  position: relative; z-index: 1;
}
.mini-chip {
  display: flex; align-items: center; gap: 8px;
  background: rgba(255,255,255,.7);
  backdrop-filter: blur(8px);
  border: 1px solid rgba(255,255,255,.9);
  padding: 8px 14px;
  border-radius: 10px;
  font-size: 13px;
  color: var(--text-2);
  cursor: pointer;
  white-space: nowrap;
  transition: background .15s, transform .15s;
}
.mini-chip:hover { background: rgba(255,255,255,.95); transform: translateY(-1px); }
.mini-chip .label { color: var(--text-3); }
.mini-chip .num {
  font-size: 16px; font-weight: 700; color: var(--text);
  font-feature-settings: "tnum";
}
.mini-chip.overdue .num { color: var(--danger); }

/* ===== Grid ===== */
.grid-main {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 340px;
  gap: 18px;
  align-items: start;
}
.col-left { display: flex; flex-direction: column; gap: 18px; min-width: 0; }
.col-right { display: flex; flex-direction: column; gap: 16px; position: sticky; top: 76px; }

.card {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--sh-1);
  overflow: hidden;
}
.card-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 16px 22px;
  border-bottom: 1px solid var(--border);
}
.card-title {
  font-size: 15px; font-weight: 600; color: var(--text);
  display: flex; align-items: center; gap: 10px;
}
.card-title .badge-num {
  background: var(--accent); color: #fff;
  font-size: 11px; font-weight: 700;
  padding: 1px 7px; border-radius: 999px;
}
.card-title .sub { font-weight: 400; color: var(--text-3); font-size: 13px; }
.card-link {
  font-size: 13px; color: var(--text-3);
  display: inline-flex; align-items: center; gap: 4px;
  transition: color .15s;
}
.card-link:hover { color: var(--primary); }
.card-link.primary { color: var(--primary); font-weight: 500; }

/* ===== Agenda ===== */
.agenda-toolbar {
  display: flex; align-items: center; gap: 14px;
  padding: 10px 22px 14px;
  font-size: 13px; color: var(--text-2);
}
.nav-arrow {
  width: 26px; height: 26px; border-radius: 6px;
  display: grid; place-items: center;
  color: var(--text-3); cursor: pointer;
  background: transparent; border: 0;
  transition: background .12s, color .12s;
}
.nav-arrow:hover { background: var(--bg); color: var(--text); }
.week-label { font-weight: 600; color: var(--text); }
.today-btn {
  font-size: 12px; padding: 4px 10px;
  border: 1px solid var(--border); border-radius: 6px;
  background: transparent; color: var(--text-2); cursor: pointer;
}
.today-btn:hover { background: var(--bg); }
.switch {
  margin-left: auto;
  display: flex; background: var(--bg);
  border-radius: 7px; padding: 2px; font-size: 12px;
}
.switch span { padding: 4px 10px; border-radius: 5px; cursor: pointer; color: var(--text-3); }
.switch .on { background: var(--surface); color: var(--text); font-weight: 600; box-shadow: var(--sh-1); }

.agenda-grid {
  display: grid;
  grid-template-columns: repeat(7, 1fr);
  border-top: 1px solid var(--border);
}
.agenda-day {
  padding: 12px 12px 14px;
  border-right: 1px solid var(--border);
  min-height: 168px;
  display: flex; flex-direction: column; gap: 6px;
  background: var(--surface);
  transition: background .12s;
}
.agenda-day:last-child { border-right: none; }
.agenda-day:hover { background: var(--surface-2); }
.agenda-day.today { background: var(--primary-soft); }
.agenda-day.today:hover { background: #e3ecff; }
.agenda-day.weekend { background: var(--surface-2); }
.day-head { display: flex; align-items: baseline; gap: 6px; margin-bottom: 4px; }
.day-weekday {
  font-size: 11px; font-weight: 500; color: var(--text-3);
  text-transform: uppercase; letter-spacing: .6px;
}
.day-num {
  font-size: 18px; font-weight: 700; color: var(--text);
  font-feature-settings: "tnum"; letter-spacing: -.5px;
}
.day-num.today-num {
  color: #fff; background: var(--primary);
  width: 26px; height: 26px; border-radius: 50%;
  display: inline-grid; place-items: center;
  font-size: 13px;
}
.day-tag { margin-left: auto; font-size: 10px; color: var(--text-3); }
.agenda-day.today .day-tag { color: var(--primary); font-weight: 600; }

.event {
  border-left: 3px solid var(--primary);
  padding: 4px 8px 4px 6px;
  background: var(--surface-2);
  border-radius: 4px;
  cursor: pointer;
  transition: background .12s;
}
.event:hover { background: var(--bg-deep); }
.agenda-day.today .event { background: rgba(255,255,255,.7); }
.event-time {
  font-size: 10.5px; color: var(--text-3);
  font-feature-settings: "tnum"; font-weight: 500;
}
.event-title {
  font-size: 12px; color: var(--text); font-weight: 500;
  line-height: 1.35; margin-top: 1px;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.event.purple { border-left-color: var(--purple); }
.event.warning { border-left-color: var(--warning); }
.event.danger { border-left-color: var(--danger); }
.event.success { border-left-color: var(--success); }
.no-event { font-size: 11px; color: var(--text-4); font-style: italic; margin-top: 4px; }

/* ===== Notifications ===== */
.notif-tabs {
  display: flex; gap: 4px;
  padding: 10px 22px 12px;
  border-bottom: 1px solid var(--border);
}
.notif-tab {
  padding: 5px 11px; border-radius: 999px;
  font-size: 12.5px; font-weight: 500; color: var(--text-3);
  cursor: pointer; background: transparent; border: 0;
  display: inline-flex; align-items: center; gap: 5px;
  transition: background .12s, color .12s;
}
.notif-tab:hover { background: var(--bg); color: var(--text-2); }
.notif-tab.on { background: var(--primary-soft); color: var(--primary); }
.notif-tab .n {
  font-weight: 700; font-size: 10.5px;
  padding: 0 5px; border-radius: 999px;
  min-width: 14px; text-align: center;
  background: rgba(0,0,0,.06);
}
.notif-tab.on .n { background: rgba(30,95,217,.18); }
.notif-tab.has-new::after {
  content: ""; width: 6px; height: 6px;
  background: var(--accent); border-radius: 50%;
}

.notif-list { padding: 6px 10px 8px; max-height: 480px; overflow-y: auto; }
.notif-item {
  display: flex; gap: 12px;
  padding: 12px;
  border-radius: 8px;
  position: relative;
  transition: background .12s;
  cursor: pointer;
}
.notif-item:hover { background: var(--bg); }
.notif-item.unread::before {
  content: "";
  position: absolute; left: 4px; top: 22px;
  width: 6px; height: 6px; border-radius: 50%; background: var(--accent);
}
.notif-avatar {
  flex-shrink: 0;
  width: 36px; height: 36px; border-radius: 50%;
  display: grid; place-items: center;
  color: #fff; font-weight: 600; font-size: 13px;
  position: relative;
}
.notif-avatar.system { background: linear-gradient(135deg,#6b7891,#41526b); }
.notif-avatar.mention { background: linear-gradient(135deg,#7c4ddb,#5a2db5); }
.notif-avatar.approval { background: linear-gradient(135deg,#1aa971,#0e7b51); }
.notif-avatar.urgent { background: linear-gradient(135deg,#e23a3a,#a82323); }
.notif-avatar.comment { background: linear-gradient(135deg,#0fa8a3,#0a7e7a); }
.notif-avatar .source-mark {
  position: absolute; bottom: -2px; right: -2px;
  width: 14px; height: 14px; border-radius: 4px;
  background: var(--cat-collab);
  border: 2px solid var(--surface);
  display: grid; place-items: center; color: #fff;
}

.notif-body { flex: 1; min-width: 0; }
.notif-title { font-size: 13px; color: var(--text); line-height: 1.5; }
.notif-title b { font-weight: 600; }
.notif-title .at { color: var(--primary); font-weight: 600; }
.notif-title .quote {
  display: block;
  margin-top: 6px;
  padding: 7px 10px;
  background: var(--surface-2);
  border-left: 2px solid var(--border-strong);
  border-radius: 4px;
  font-size: 12px;
  color: var(--text-2);
  font-weight: 400;
}
.notif-meta {
  margin-top: 6px;
  display: flex; align-items: center; gap: 6px;
  font-size: 11.5px; color: var(--text-3);
}
.notif-meta .app-src {
  display: inline-flex; align-items: center; gap: 4px;
  padding: 1px 6px;
  background: var(--primary-soft); color: var(--primary);
  border-radius: 4px; font-weight: 500;
}
.notif-meta .sep { width: 2px; height: 2px; background: var(--text-4); border-radius: 50%; }
.notif-actions { margin-top: 8px; display: flex; gap: 6px; }
.notif-actions button {
  font-size: 11.5px; padding: 4px 10px; border-radius: 6px;
  border: 1px solid var(--border); background: var(--surface);
  color: var(--text-2); cursor: pointer;
  transition: background .12s, border-color .12s, color .12s;
}
.notif-actions button:hover { background: var(--bg); border-color: var(--border-strong); }
.notif-actions button.primary {
  background: var(--primary); border-color: var(--primary); color: #fff;
}
.notif-actions button.primary:hover { background: var(--primary-hover); }

/* ===== Todos ===== */
.todo-tabs {
  display: flex; gap: 6px; padding: 12px 22px;
  flex-wrap: wrap; border-bottom: 1px solid var(--border);
}
.todo-tab {
  display: inline-flex; align-items: center; gap: 6px;
  padding: 5px 10px; border-radius: 999px;
  font-size: 12px; font-weight: 500;
  cursor: pointer; border: 0;
  transition: background .15s;
}
.todo-tab .n {
  font-weight: 700; font-size: 11px;
  padding: 0 5px; border-radius: 999px; min-width: 14px;
  text-align: center; background: rgba(0,0,0,.06);
}
.todo-tab.t-review { background: var(--primary-soft); color: var(--primary); }
.todo-tab.t-review .n { background: rgba(30,95,217,.18); }
.todo-tab.t-assign { background: var(--teal-soft); color: var(--teal); }
.todo-tab.t-assign .n { background: rgba(15,168,163,.2); }
.todo-tab.t-propose { background: var(--purple-soft); color: var(--purple); }
.todo-tab.t-propose .n { background: rgba(124,77,219,.2); }
.todo-tab.t-eval { background: var(--warning-soft); color: var(--warning); }
.todo-tab.t-eval .n { background: rgba(232,146,14,.2); }

.todo-list { padding: 4px 12px 8px; }
.todo-item {
  display: flex; gap: 10px;
  padding: 11px 10px;
  border-radius: 8px;
  cursor: pointer; position: relative;
  transition: background .15s;
}
.todo-item:hover { background: var(--bg); }
.todo-item .pri-bar {
  width: 3px; border-radius: 2px;
  flex-shrink: 0;
  background: var(--text-4);
}
.todo-item.pri-urgent .pri-bar { background: var(--danger); }
.todo-item.pri-high .pri-bar { background: var(--warning); }
.todo-content { flex: 1; min-width: 0; }
.todo-title-row {
  display: flex; align-items: center; gap: 8px;
  font-size: 13px; font-weight: 500; color: var(--text);
}
.title-text {
  overflow: hidden; white-space: nowrap; text-overflow: ellipsis;
  flex: 1; min-width: 0;
}
.pri-tag {
  font-size: 10px; padding: 1px 5px; border-radius: 3px;
  font-weight: 600; letter-spacing: .3px;
}
.pri-tag.urgent { background: var(--danger-soft); color: var(--danger); }
.pri-tag.high { background: var(--warning-soft); color: var(--warning); }
.todo-meta {
  margin-top: 3px;
  font-size: 11.5px; color: var(--text-3);
  display: flex; align-items: center; gap: 6px; flex-wrap: wrap;
}
.app-tag {
  display: inline-flex; align-items: center; gap: 4px;
  color: var(--primary); font-weight: 500;
}
.dot-sep { width: 2px; height: 2px; background: var(--text-4); border-radius: 50%; }
.time.over { color: var(--danger); font-weight: 500; }
.todo-actions {
  display: flex; flex-direction: column; gap: 4px;
  align-self: center;
  opacity: 0; transition: opacity .15s;
}
.todo-item:hover .todo-actions { opacity: 1; }
.todo-actions button {
  font-size: 11px; padding: 4px 10px;
  border-radius: 5px;
  background: var(--surface);
  border: 1px solid var(--border);
  color: var(--text-2); cursor: pointer;
}
.todo-actions button:hover { background: var(--primary); color: #fff; border-color: var(--primary); }

/* ===== Recent docs ===== */
.docs-list { padding: 4px 10px 10px; }
.doc-item {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 12px; border-radius: 8px;
  cursor: pointer; transition: background .12s;
}
.doc-item:hover { background: var(--bg); }
.doc-icon {
  width: 32px; height: 32px; border-radius: 7px;
  display: grid; place-items: center;
  flex-shrink: 0; color: #fff;
}
.doc-info { flex: 1; min-width: 0; }
.doc-name {
  font-size: 13px; font-weight: 500; color: var(--text);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.doc-meta {
  margin-top: 2px;
  font-size: 11.5px; color: var(--text-3);
  display: flex; align-items: center; gap: 6px;
}
.doc-time { font-size: 11.5px; color: var(--text-3); white-space: nowrap; }

/* ===== Right Col ===== */
.ann-list { padding: 6px 8px 12px; }
.ann-item {
  display: flex; gap: 10px; padding: 10px 12px;
  border-radius: 8px; cursor: pointer;
  transition: background .15s;
}
.ann-item:hover { background: var(--bg); }
.ann-item .ann-icon {
  flex-shrink: 0;
  width: 28px; height: 28px; border-radius: 7px;
  background: var(--primary-soft); color: var(--primary);
  display: grid; place-items: center;
}
.ann-item:nth-child(2) .ann-icon { background: var(--success-soft); color: var(--success); }
.ann-item:nth-child(3) .ann-icon { background: var(--purple-soft); color: var(--purple); }
.ann-text { flex: 1; min-width: 0; }
.ann-title {
  font-size: 13px; color: var(--text); font-weight: 500;
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
}
.ann-date { font-size: 11.5px; color: var(--text-3); margin-top: 2px; }
.ann-tag {
  display: inline-block;
  font-size: 10px; padding: 1px 5px;
  background: var(--primary-soft); color: var(--primary);
  border-radius: 3px;
  margin-left: 6px; font-weight: 600;
  vertical-align: 1px;
}

.quick-links {
  padding: 10px;
  display: grid; grid-template-columns: repeat(2, 1fr); gap: 6px;
}
.q-link {
  display: flex; align-items: center; gap: 10px;
  padding: 10px 12px;
  border-radius: 8px;
  cursor: pointer; transition: background .15s;
  color: inherit; text-decoration: none;
}
.q-link:hover { background: var(--bg); text-decoration: none; }
.q-link .q-ico {
  width: 30px; height: 30px; border-radius: 7px;
  display: grid; place-items: center; flex-shrink: 0;
}
.q-name { font-size: 13px; font-weight: 500; color: var(--text); }
.q-sub { font-size: 11px; color: var(--text-3); margin-top: 1px; }

.rec-mini-card {
  border: 1px dashed var(--primary-softer);
  background: linear-gradient(135deg, var(--primary-soft) 0%, var(--surface) 60%);
}
.rec-mini-card .card-header { border-bottom-color: var(--primary-softer); }
.rec-list { padding: 8px 8px 14px; }
.rec-mini-item {
  display: flex; align-items: center; gap: 10px;
  padding: 10px;
  border-radius: 8px;
  cursor: pointer;
  background: var(--surface);
  border: 1px solid var(--border);
  margin: 6px 4px;
  transition: transform .15s, box-shadow .15s, border-color .15s;
}
.rec-mini-item:hover {
  transform: translateY(-1px);
  box-shadow: var(--sh-2);
  border-color: var(--border-strong);
}
.rec-mini-item .icon-box {
  width: 32px; height: 32px; border-radius: 8px;
  display: grid; place-items: center;
  color: #fff; flex-shrink: 0;
}
.rec-mini-name {
  font-size: 13px; font-weight: 600; color: var(--text);
  display: flex; align-items: center; gap: 6px;
}
.rec-mini-name .rating { font-size: 11px; color: var(--warning); font-weight: 700; }
.rec-mini-desc {
  font-size: 11.5px; color: var(--text-3);
  white-space: nowrap; overflow: hidden; text-overflow: ellipsis;
  margin-top: 1px;
}

/* Responsive */
@media (max-width: 1200px) {
  .grid-main { grid-template-columns: 1fr; }
  .col-right { position: static; }
}
</style>
