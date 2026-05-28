# 问题协同研究解决平台 · 详细设计文档

| 项 | 内容 |
|---|---|
| 版本 | v1.0.0 |
| 起草日期 | 2026-05-25 |
| 对应代码版本 | git initial commit |

---

## 1. 系统架构

### 1.1 总体架构

```
┌────────────────────────────────────────────────────────────────────────────┐
│                              浏览器 (Chrome / Edge)                        │
│   ┌────────────────────────────────────────────────────────────────────┐   │
│   │  Vue 3 SPA   (Vue Router / Pinia / Element Plus / ECharts)         │   │
│   │   ├─ DashboardView                                                 │   │
│   │   ├─ ProblemsView                                                  │   │
│   │   └─ ProcessingModal (全屏弹层 + 8 个 Stage 表单)                  │   │
│   └────────────────────────┬───────────────────────────────────────────┘   │
│                            │ HTTP / JSON (/api/**, X-User-Id 头)            │
└────────────────────────────┼───────────────────────────────────────────────┘
                             │
                             ▼  (Nginx 反代 → Spring Boot)
┌────────────────────────────────────────────────────────────────────────────┐
│                       Spring Boot 3 Backend (JVM 17+)                      │
│   ┌────────────────────────────────────────────────────────────────────┐   │
│   │  WebMvc 层  Controller (Problem / Dashboard / File / Message ...)  │   │
│   │             ResponseWrapAdvice (统一封装 ApiResponse)                │   │
│   │             GlobalExceptionHandler (业务/参数/未知错误)              │   │
│   │             WebConfig 拦截器 (X-User-Id → UserContext)              │   │
│   ├────────────────────────────────────────────────────────────────────┤   │
│   │  Service 层  ProblemService (CRUD + 状态机推进, @Transactional)     │   │
│   │              DashboardService (聚合 SQL)                            │   │
│   │              StageEngine (状态机引擎, 纯函数)                       │   │
│   │              FileStorageService (MinIO 上传/下载/删除)              │   │
│   │              MessageService (留言 + @ 抽取)                         │   │
│   │              NotificationService (轮询接口聚合)                     │   │
│   ├────────────────────────────────────────────────────────────────────┤   │
│   │  持久层     MyBatis-Plus Mapper × 10                                │   │
│   │              Entity × 10                                            │   │
│   └─────────┬──────────────────────────────────────────────┬───────────┘   │
└─────────────┼──────────────────────────────────────────────┼───────────────┘
              │ JDBC                                         │ MinIO SDK
              ▼                                              ▼
        ┌───────────────────────────┐               ┌──────────────────┐
        │  PostgreSQL 16            │               │  MinIO           │
        │  ( = KingbaseV8 PG 模式)  │               │  bucket: gallant │
        │  - JSONB 字段             │               │  -files          │
        │  - 业务约束 CHECK         │               │                  │
        └───────────────────────────┘               └──────────────────┘
```

### 1.2 关键技术选型理由

| 选型 | 理由 |
|---|---|
| Vue 3 + TS | 团队前端栈，TS 类型安全 |
| Element Plus | Vue 3 生态最成熟、中文文档完善、企业项目首选 |
| ECharts | 大屏/仪表盘事实标准，中文社区强 |
| Spring Boot 3 + MyBatis-Plus | Java 主流企业栈；MP 提供 CRUD/分页/逻辑删除，减少样板代码 |
| PostgreSQL JDBC | KingbaseV8 PG 兼容模式可无缝衔接，开发期无须本地 KB |
| MinIO | 对象存储，S3 API 兼容；后续如换私有云只改 endpoint |
| 轮询通知 | 实现简单，可平滑升级到 SSE/WebSocket |

---

## 2. 状态机详细设计

### 2.1 阶段枚举

```java
public enum Stage {
    SUBMIT   ("submit",    "问题提报", null,             1),
    REVIEW   ("review",    "审核分办", null,             2),
    PROPOSE  ("propose",   "研提举措", null,             3),
    MEETING  ("meeting",   "会商研究", Branch.DISPUTE,   4),
    ARBITRATE("arbitrate", "争议裁决", Branch.DISPUTE,   5),
    CONSULT  ("consult",   "征求意见", Branch.CONSENSUS, 6),
    IMPLEMENT("implement", "督导落实", null,             7),
    EVALUATE ("evaluate",  "评价反馈", null,             8);
}
```

`branch` 为空表示**公共节点**（任何路径都经过），非空表示**仅当 problem.branch 等于该值时存在**。

### 2.2 流转函数

```
next(current, branch) →
  PROPOSE  + DISPUTE   → MEETING
  PROPOSE  + CONSENSUS → CONSULT
  ARBITRATE + *        → IMPLEMENT
  CONSULT   + *        → IMPLEMENT
  EVALUATE  + *        → ∅   (终态)
  其他: 按 seq 找下一个满足 branch 约束的节点
```

### 2.3 校验规则
`StageEngine.validateStageAction(problem, requestedStage)`：
1. 若 `problem.status == 'done'` 且 requested ≠ EVALUATE → 拒绝
2. 若 `requestedStage.seq > current.seq` → 拒绝（"不能在未到达的阶段执行"）
3. 否则放行（允许在已完成阶段重新提交或编辑）

### 2.4 PROPOSE 分支决策时序

```
用户在 propose 表单上切换 hasDispute 开关
   │
   └─► UI 实时刷新分支预览（紫色 vs 青色）
       │
       └─► 点击底部按钮:
             ├─ "提交并进入征求意见"  → POST /actions/propose hasDispute=false
             └─ "标记存在争议 → 会商研究" → POST /actions/propose hasDispute=true

后端:
  ProblemService.propose() {
    全量替换 measures
    若 hasDispute: 全量替换 disputes
    branch = hasDispute ? DISPUTE : CONSENSUS
    next = StageEngine.advance(problem, branch)       // MEETING or CONSULT
    更新 problem: branch, currentStage, status, latest, progress
    写 stage_history (branchChoice 字段记录本次选择)
  }
```

### 2.5 状态映射

`Stage.deriveProblemStatus(boolean evaluated)`：

| 当前阶段 | 已评价 | status |
|---|---|---|
| SUBMIT | false | pending |
| MEETING | false | meeting |
| ARBITRATE | false | arbitrate |
| CONSULT | false | consulting |
| EVALUATE | true | done |
| 其他 | false | processing |

超期 (`overdue=true`) 是独立字段，前端 UI 优先展示；后端不覆盖 status。

---

## 3. 数据库设计

### 3.1 ER 图（核心实体）

```
                ┌──────────┐
                │ app_user │
                └────┬─────┘
                     │ submitter_id / actor_user_id / uploader_id / evaluator_id
       ┌─────────────┼──────────────────────────────────┐
       │             │                                  │
       ▼             ▼                                  ▼
┌──────────────┐  ┌──────────────┐                ┌────────────┐
│   problem    │──┤stage_history │ (1:N, 顺序)    │ evaluation │
│ (业务编号 PK)│  └──────────────┘                └────────────┘
└───┬────┬─────┘
    │    │
    │    └─────────────┬──────────────┬──────────────┬─────────────┐
    │                  │              │              │             │
    ▼                  ▼              ▼              ▼             ▼
┌─────────┐      ┌───────────┐  ┌──────────┐  ┌──────────┐  ┌──────────────┐
│ measure │      │  dispute  │  │ message  │  │attachment│  │ consult_stat │
└─────────┘      └─────┬─────┘  └──────────┘  └──────────┘  └──────────────┘
                       │
                       ▼
              ┌──────────────────┐
              │ dispute_position │
              └──────────────────┘
```

### 3.2 表清单

| 表 | 主键 | 说明 |
|---|---|---|
| `app_user` | id (BIGSERIAL) | 用户表，记录身份用 |
| `problem` | id (VARCHAR(32), 业务编号) | 问题主表 |
| `stage_history` | id (BIGSERIAL) | 阶段历史 / 审计流水（追加，不更新） |
| `measure` | id (BIGSERIAL) | 举措清单（按 problem_id, code 唯一） |
| `dispute` | id (BIGSERIAL) | 争议点 |
| `dispute_position` | id (BIGSERIAL) | 各方观点（属于 dispute） |
| `message` | id (BIGSERIAL) | 协同留言 |
| `attachment` | id (BIGSERIAL) | 文件元数据（实体在 MinIO） |
| `consult_stat` | problem_id (FK & PK) | 征求意见的反馈汇总 |
| `evaluation` | id (BIGSERIAL) | 评价（一个 problem 可有多条，按 party） |

完整 DDL 见 [schema.sql](schema.sql)。

### 3.3 关键设计决策

- **业务编号 PK**：`problem.id = WTyyyyMMdd-NNN`，作为对外编号也是主键，便于跨系统引用
- **JSONB 字段**：`tags`、`participants`、`files`、`mentions`、`read_by` 用 JSONB 存数组，避免子表
- **stage_history append-only**：永不更新，永不删除（除问题被 CASCADE 删除）
- **measure code 唯一**：同一 problem 内 `code='M1'` 唯一，避免重复
- **CHECK 约束**：priority/status/branch/stage 都有 CHECK，防止脏数据
- **JSONB 查询**：参与方过滤用 `participants @> '[\"dept\"]'::jsonb`，需建 GIN 索引（生产再加）

### 3.4 索引策略

| 表 | 索引 | 用途 |
|---|---|---|
| problem | idx_problem_status, idx_problem_stage, idx_problem_submitter, idx_problem_handler_dept, idx_problem_due_date, idx_problem_category | 看板聚合 + 列表过滤 |
| stage_history | idx_history_problem (problem_id, occurred_at DESC) | 详情页历史时间线 |
| attachment | idx_attachment_problem (problem_id, stage) | 按阶段分组材料 |

---

## 4. API 设计

### 4.1 通用约定

- **基础路径**：`/api`
- **响应格式**：`{ code: int, message: string, data: any }`，`code=0` 成功
- **错误码**：
  - `0`：成功
  - `40000-40999`：业务错误（用户输入/状态非法）
  - `40001`：参数校验失败
  - `50000`：服务异常
- **用户标识**：请求头 `X-User-Id: <id>`，未提供则默认 mock 用户 (id=1)
- **时间格式**：`yyyy-MM-dd HH:mm:ss`（Asia/Shanghai）

### 4.2 接口清单（关键）

#### Problem
```
GET    /problems?page=1&size=20&status=&stage=&priority=&tab=&query=
GET    /problems/{id}                       → ProblemDetail (含 history/measures/disputes/messages/attachments/consult/evaluations)
POST   /problems                            → 创建; 默认 stage=review, status=pending
POST   /problems/{id}/actions/review        body: ReviewActionRequest
POST   /problems/{id}/actions/propose       body: ProposeActionRequest (关键: hasDispute / measures / disputes)
POST   /problems/{id}/actions/meeting       body: MeetingActionRequest (advance=true 推进 arbitrate)
POST   /problems/{id}/actions/arbitrate     body: ArbitrateActionRequest
POST   /problems/{id}/actions/consult       body: ConsultActionRequest (advance=true 推进 implement)
POST   /problems/{id}/actions/implement     body: ImplementActionRequest
POST   /problems/{id}/actions/evaluate      body: EvaluateActionRequest (提交即 done)
```

#### Dashboard
```
GET    /dashboard/overview
       → { kpis, processingBreakdown, categories, topSubmitterDepts,
            topHandlerDepts, overdueByDept, trend, satisfaction, disputeStats }
```

#### File
```
POST   /files/upload         multipart: problemId, stage, file
GET    /files/problem/{id}   → Attachment[]
GET    /files/{id}/download  → 流式响应, Content-Disposition 带原名
DELETE /files/{id}
```

#### Message / Notification / User / Stages
```
GET    /messages/problem/{id}     POST /messages/problem/{id} body:{content}
GET    /notifications/unread      POST /notifications/messages/{id}/read
GET    /users                     GET  /users/me
GET    /stages                    → 8 阶段元数据 (code/label/branch/seq)
```

### 4.3 关键 DTO

```java
public class ProposeActionRequest {
  Boolean hasDispute;          // ★ 分支决策
  List<MeasureInput> measures;
  List<DisputeInput> disputes; // 仅 hasDispute=true 时有效
  String  note;
}

public class ProblemDetail {
  Problem               problem;
  AppUser               submitter;
  List<StageHistory>    history;
  List<Measure>         measures;
  List<DisputeWithPositions> disputes;
  List<Message>         messages;
  List<Attachment>      attachments;
  ConsultStat           consult;
  List<Evaluation>      evaluations;
}
```

---

## 5. 前端架构

### 5.1 路由

| 路径 | 视图 | 说明 |
|---|---|---|
| `/` | redirect `/dashboard` | |
| `/dashboard` | DashboardView | 全局看板 |
| `/problems` | ProblemsView | 问题清单（双视图） |

办理弹层不是独立路由，是一个**全局组件 + Pinia store** —— 任何页面调 `useProcessingStore().open(problemId)` 即弹出。

### 5.2 状态管理 (Pinia stores)

| Store | 职责 |
|---|---|
| `useUserStore` | 当前用户 + 全部用户列表 + 切换用户 (demo) |
| `useStagesStore` | 8 阶段元数据 (从后端拉一次缓存) + `pathFor(branch)` 计算路径 |
| `useNotificationStore` | 启停轮询 + 未读消息聚合 |
| `useProcessingStore` | 办理弹层的开关 + 当前 problemId + ProblemDetail 缓存 |

### 5.3 组件树

```
App.vue
└─ AppLayout.vue
   ├─ 顶部导航 (nav-tabs / 用户切换 / 通知徽章 / 提报问题按钮)
   ├─ <router-view/>
   │   ├─ DashboardView.vue
   │   │   ├─ KpiCard × 6
   │   │   ├─ ProcessingBreakdownBand (内置)
   │   │   ├─ CategoryDonut (ECharts)
   │   │   ├─ DisputeCard
   │   │   ├─ OverdueCard
   │   │   ├─ TopUnitsCard × 2 (BarRow)
   │   │   ├─ TrendCard (ECharts)
   │   │   └─ SatisfactionTable + StarRating
   │   └─ ProblemsView.vue
   │       ├─ ElTabs
   │       ├─ Filter bar (search + 3 selects)
   │       ├─ ProblemCard (含 FlowTimelineH) ×n  ── 卡片视图
   │       └─ ElTable                            ── 表格视图
   └─ ProcessingModal.vue  (条件渲染)
      ├─ 顶栏 (id / priority / status / overdue / 关闭)
      ├─ StageStrip (8 节点条)
      ├─ 三栏布局
      │  ├─ 左: 问题基本信息 + FlowTimelineV
      │  ├─ 中: ProcessingTabs
      │  │   ├─ 办理: 动态渲染 Form{Submit|Review|Propose|Meeting|Arbitrate|Consult|Implement|Evaluate}
      │  │   ├─ 流程图谱: FlowGraph (SVG)
      │  │   ├─ 协同留言: CollabPanel
      │  │   └─ 举措清单: MeasuresPanel
      │  └─ 右: HistoryTimeline + MaterialList
      └─ ESC / 遮罩点击关闭
```

### 5.4 关键交互流程

#### 5.4.1 切换办理 Tab + 提交动作

```
ProcessingModal.open(id)
  ↓ Pinia: useProcessingStore.open(id)
  ↓ http GET /problems/{id} → ProblemDetail 存入 store
  ↓ 渲染顶栏 + StageStrip + 左中右

中部表单 e.g. FormPropose
  ↓ 用户填表 + 切换 hasDispute
  ↓ 点击 "提交并进入征求意见" / "标记存在争议 → 会商研究"
  ↓ http POST /problems/{id}/actions/propose
  ↓ emit('done')
ProcessingModal.onDone()
  ↓ proc.reload() → 重新拉 detail → 刷新视图
  ↓ activeStage 跟随新的 currentStage
```

#### 5.4.2 通知轮询

```
App.vue mounted
  ↓ useNotificationStore.startPolling()
  ↓ setInterval(20s) → GET /notifications/unread
  ↓ digest.value 更新
  ↓ 顶栏 ElBadge 自动响应
```

### 5.5 设计令牌 (CSS Variables)

定义在 `src/styles/tokens.css`，与原型完全一致：
- 品牌色：`--primary: #1e5fd9` (蓝)，`--accent: #d63838` (红)
- 8 阶段色：`--stage-submit/review/propose/meeting/arbitrate/consult/implement/evaluate`
- 间距体系：`--sp-1` (4px) → `--sp-10` (72px)
- 圆角：`--r-sm/md/lg/xl/pill`
- 阴影：`--sh-1` → `--sh-4`

Element Plus 主题色通过 `--el-color-primary` 等变量覆盖（见 `global.css`）。

---

## 6. 关键流程时序图

### 6.1 提报 → 评价完整闭环（争议路径）

```
用户 (陈雨晴, CTO)         前端                后端 ProblemService        DB
   │                          │                       │                    │
   ├─[点击 "提报问题"]──────►│                       │                    │
   │                          ├──POST /problems─────►│                    │
   │                          │                       ├─create()──────────►│
   │                          │                       │                    │ INSERT problem (stage=review, status=pending)
   │                          │                       │                    │ INSERT stage_history (stage=submit)
   │                          │◄──{id, ...}──────────┤                    │
   │◄─[办理弹层展开]──────────┤                       │                    │
   │                          │                       │                    │
   ├─[审核分办 approve]──────►│                       │                    │
   │                          ├──POST .../review─────►│                    │
   │                          │                       ├─review()──────────►│ UPDATE problem (stage=propose, status=processing, handler=...)
   │                          │                       │                    │ INSERT stage_history (stage=review)
   │                          │◄──Problem────────────┤                    │
   │                          ├──reload detail────────►│                    │
   │                          │                       │                    │
   ├─[研提举措 hasDispute=true]►                       │                    │
   │                          ├──POST .../propose────►│                    │
   │                          │                       ├─propose()─────────►│ DELETE measures, INSERT n × measures
   │                          │                       │                    │ DELETE disputes, INSERT n × disputes + positions
   │                          │                       │                    │ UPDATE problem (branch=dispute, stage=meeting, status=meeting)
   │                          │                       │                    │ INSERT stage_history (stage=propose, branch_choice=dispute)
   │                          │                       │                    │
   │ (会商 N 轮 → meeting advance) │                                              │
   │ (裁决 → arbitrate)           │                                              │
   │ (督导 → implement advance)   │                                              │
   │                                                                             │
   ├─[评价提交]──────────────►│                       │                    │
   │                          ├──POST .../evaluate───►│                    │
   │                          │                       ├─evaluate()────────►│ INSERT evaluation (overall=avg)
   │                          │                       │                    │ UPDATE problem (status=done, progress=100)
   │                          │                       │                    │ INSERT stage_history (stage=evaluate)
   │                          │◄──Problem(status=done)┤                    │
   │◄─[闭环, 看板 done++]─────┤                       │                    │
```

### 6.2 文件上传链路

```
用户拖拽文件
  ↓ ElUpload :http-request="customUpload"
  ↓ FormData(problemId, stage, file)
  ↓ POST /api/files/upload (multipart)
  ↓ FileController.upload()
  ↓ FileStorageService.upload()
    ├─ 构造 objectKey = problemId/stage/{uuid}-{safeName}
    ├─ client.putObject(bucket, objectKey, stream)
    └─ INSERT attachment (元数据)
  ↓ Attachment 返回前端
  ↓ MaterialList 刷新
```

---

## 7. 部署架构（生产）

```
        互联网 (内网)
            │
            ▼
   ┌─────────────────┐
   │  Nginx (443)    │  HTTPS 终结, 静态文件, /api 反代
   └────┬────────┬───┘
        │        │
  /     │        │  /api
        │        │
        ▼        ▼
  ┌─────────┐ ┌─────────────────┐
  │ dist/   │ │ Spring Boot     │  端口 8080 (上下文 /api)
  │ Vue SPA │ │ JVM 17+         │
  └─────────┘ └────┬────────┬───┘
                   │        │
                   ▼        ▼
            ┌─────────┐ ┌─────────────┐
            │ KingbaseV8│ │ MinIO       │
            │ PG 兼容   │ │ Bucket      │
            └─────────┘ └─────────────┘
```

### 7.1 高可用建议
- Spring Boot 多实例 + Nginx upstream 负载
- MinIO 4 节点纠删码模式
- KingbaseV8 主备 + 定期备份到对象存储
- 接通 SSO 后用 Redis Cluster 存 session
- Nginx + ModSecurity 做 WAF

---

## 8. 测试设计

### 8.1 测试金字塔

| 层级 | 工具 | 数量 | 覆盖 |
|---|---|---|---|
| 后端单元 | JUnit 5 | 9 (StageTest) | 状态机推进逻辑 |
| 后端集成 | Spring Boot Test + 本地 PG | 8 (Service) + 5 (Controller) + 1 (Dashboard) | 业务流程 + REST 接口 |
| 前端单元 | Vitest | 13 | helpers, store, 关键组件 |
| 前端 E2E | (未引入, 后续 Playwright) | 0 | — |
| 手工烟雾 | curl 脚本 | 1 | 完整争议闭环（创建→评价） |

### 8.2 集成测试隔离策略

每个测试方法 `@BeforeEach` 执行：
1. 删除所有表（CASCADE）
2. 执行 `schema.sql`
3. 执行 `seed.sql`
4. 设置 mock 用户

保证测试间完全独立。代价：每个测试 ~500ms（19 个测试共 ~10s）。

### 8.3 CI 建议

```
# .github/workflows/ci.yml (示意)
on: [push, pull_request]
jobs:
  backend:
    services: { postgres: { image: postgres:16 } }
    steps:
      - run: cd backend && mvn -B test
  frontend:
    steps:
      - run: cd frontend && npm ci && npm test && npm run build
```

---

## 9. 已知问题 & 后续优化

| 问题 | 现状 | 计划 |
|---|---|---|
| 看板首屏冷启动慢 | < 1s 可接受 | 加 Redis 缓存聚合结果 (5min TTL) |
| 测试不用 testcontainers | 用本地 PG, 需 docker compose up | 升级 testcontainers 至 1.21+ 验证 |
| 通知是轮询 | 20s 可接受 | 升级到 SSE |
| 文件下载经后端中转 | 流式传输, 但大文件压力大 | 引入 presigned URL 直传/直下 (大文件) |
| 无 SSO / RBAC | mock 用户 | 接公司 SSO，按部门做行级权限 |
| 看板 dispute avgMeetings 硬编码 2.4 | 简化 | 改成 stage_history 实时聚合 |
