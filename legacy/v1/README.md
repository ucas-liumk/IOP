# 问题协同研究解决平台

企业内部问题协同研究/解决平台。8 节点状态机（提报 → 审核分办 → 研提举措 →〔有争议: 会商研究 → 争议裁决〕|〔无争议: 征求意见〕→ 督导落实 → 评价反馈），看板 + 问题清单 + 全屏办理弹层 三大界面。

## 技术栈

- **前端**：Vue 3 + TypeScript + Vite + Vue Router + Pinia + Element Plus + ECharts + Axios
- **后端**：Spring Boot 3 + MyBatis-Plus + PostgreSQL JDBC（生产换 KingbaseV8 驱动 / PG 兼容模式）
- **存储**：MinIO（后端中转上传，支持单文件 50MB）
- **测试**：JUnit 5 + Spring Boot Test / Vitest + Vue Test Utils

## 目录

```
gallant/
├── docker-compose.yml      本地 PostgreSQL 16 + MinIO
├── docs/
│   ├── prototype/          原始 React 设计稿（参考）
│   ├── schema.sql          建表 DDL
│   └── seed.sql            种子数据（8 个示范问题）
├── backend/                Spring Boot 后端
└── frontend/               Vue 3 前端
```

## 本地启动

```bash
# 1. 起依赖（数据库 + MinIO + bucket 初始化）
docker compose up -d

# 2. 后端
cd backend
mvn spring-boot:run
# http://localhost:8080/api

# 3. 前端
cd frontend
npm install
npm run dev
# http://localhost:5173
```

需要 **JDK 17+**、**Node 18+**、**Docker**。

## 测试

```bash
# 后端 (需要先 docker compose up -d db)
cd backend && mvn test
# 19 个测试: StageTest(9 单元) + ProblemServiceTest(8) + DashboardServiceTest(1) + ProblemControllerIT(5)

# 前端
cd frontend && npm test
# 13 个测试: helpers / stages / StageChip / AvatarBadge
```

**集成测试架构说明**：集成测试连接到 `docker compose` 启动的本地 PG（`localhost:5432`），每个测试方法前重置 schema + seed 保证隔离。**不使用 testcontainers** 是因为目前 testcontainers 1.20.4 内置的 docker-java 客户端 API 版本与 Docker 29 (colima 后端) 不兼容。如需切回 testcontainers，在 `pom.xml` 中取消注释并升级到 1.21+ 即可。

## 切换到 KingbaseV8（生产）

1. 在 `backend/pom.xml` 注释 `postgresql` 依赖
2. 取消注释 `kingbase8`（需先 `mvn install:install-file -Dfile=kingbase8-x.x.x.jar -DgroupId=cn.com.kingbase -DartifactId=kingbase8 -Dversion=8.6.0 -Dpackaging=jar`）
3. 修改 `backend/src/main/resources/application-dev.yml`：
   - `driver-class-name: com.kingbase8.Driver`
   - `url: jdbc:kingbase8://host:54321/gallant`
4. 修改 `MybatisPlusConfig` 中 `DbType.POSTGRE_SQL` → `DbType.KINGBASE_ES`
5. 确认 KingbaseV8 实例为 **PG 兼容模式**

## 设计决策记录

- **不区分角色，保留用户身份**：所有动作记录 `actor_user_id`，便于审计；无权限校验。当前请求用户通过 `X-User-Id` 请求头识别，未提供则使用 mock 用户 id=1（陈雨晴）。
- **状态机后端强校验**：`StageEngine` 校验阶段流转合法性，前端只展示当前阶段的表单。
- **轮询通知**：前端每 20s 拉取 `/notifications/unread`，后续可平滑升级到 SSE。
- **MinIO 后端中转**：所有文件上传/下载经后端，便于审计和后续接入扫描；大文件使用 multipart。
- **问题创建语义**：新建问题立即处于 `currentStage='review'`（等待审核），不停留在 `submit`。`submit` 历史条目记录创建动作。

## API 速览

| 模块 | 端点 |
|------|------|
| 看板 | `GET /dashboard/overview` |
| 问题列表 | `GET /problems?page=&size=&status=&stage=&priority=&tab=&query=` |
| 问题详情 | `GET /problems/{id}` |
| 创建问题 | `POST /problems` |
| 阶段动作 | `POST /problems/{id}/actions/{review|propose|meeting|arbitrate|consult|implement|evaluate}` |
| 留言 | `GET/POST /messages/problem/{id}` |
| 文件 | `POST /files/upload`, `GET /files/{id}/download`, `GET /files/problem/{id}` |
| 通知 | `GET /notifications/unread` |
| 阶段元数据 | `GET /stages` |
| 用户 | `GET /users`, `GET /users/me` |

## 端到端验证

完整流程（争议路径）：

```
[创建] pending/review
   → [审核分办 approve] processing/propose
   → [研提举措 +dispute] meeting/meeting
   → [会商完成 advance] arbitrate/arbitrate
   → [裁决发布] processing/implement
   → [督导申请办结] processing/evaluate
   → [评价提交] done/evaluate
```
