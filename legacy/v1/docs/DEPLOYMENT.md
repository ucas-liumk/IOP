# 问题协同研究解决平台 · 安装部署文档

| 项 | 内容 |
|---|---|
| 版本 | v1.0.0 |
| 适用环境 | 本地开发 / 测试 / 生产 |

---

## 0. 环境要求

| 组件 | 最低 | 推荐 | 备注 |
|---|---|---|---|
| 操作系统 | macOS 12 / Linux (Kernel 5+) / Windows 10 + WSL2 | Linux | 生产建议 CentOS/RHEL/UnionTech UOS |
| JDK | 17 | 17 LTS | 后端编译运行 |
| Node.js | 18 | 20 LTS | 前端构建 |
| Docker | 24 | 27+ | 本地开发起 DB/MinIO，生产可选 |
| Maven | 3.8 | 3.9 | 后端构建 |
| 内存 | 4 GB | 8 GB+ | 全栈本地运行 |
| 磁盘 | 10 GB | 50 GB+ | 含 MinIO 数据 |

---

## 1. 本地开发部署

### 1.1 准备依赖

```bash
# macOS (Homebrew)
brew install openjdk@17 maven node docker colima
colima start    # 启动 docker daemon

# Linux (Ubuntu/Debian)
sudo apt install -y openjdk-17-jdk maven nodejs npm docker.io docker-compose-plugin
```

校验：
```bash
java -version    # 17.x
mvn -version     # 3.8+
node -v          # 18+
docker info      # 应显示 Server Version
```

### 1.2 拉取代码 & 起依赖

```bash
# 解压交付包 (假设位于 ~/Downloads/gallant-v1.0.0.tar.gz)
cd ~/Workspace
tar -xzf ~/Downloads/gallant-v1.0.0.tar.gz
cd gallant

# 起 PostgreSQL + MinIO (含自动建表/灌种子)
docker compose up -d

# 查看健康状态 (应均为 healthy)
docker compose ps
```

启动后会有：
- PostgreSQL: `localhost:5432`, db=`gallant`, user=`gallant`, pwd=`gallant_dev`，自动执行 `docs/schema.sql` + `docs/seed.sql`
- MinIO: API `localhost:9000`, Console `http://localhost:9001` (user=`gallant`, pwd=`gallant_dev_minio`)，自动创建 bucket `gallant-files`

### 1.3 启动后端

```bash
cd backend
mvn spring-boot:run
# 看到 "Started CollabApplication" 即就绪
# 监听端口 8080, 上下文路径 /api
```

校验：
```bash
curl http://localhost:8080/api/stages | head -c 200
# 应返回 {"code":0,"message":"ok","data":[{"code":"submit",...
```

### 1.4 启动前端

```bash
cd frontend
npm install        # 首次安装依赖, ~ 1-2 分钟
npm run dev
# Vite 默认监听 5173, /api 已配置代理到 8080
```

打开浏览器访问 [http://localhost:5173](http://localhost:5173)，应看到看板。

### 1.5 验证

- [ ] 看板加载，6 个 KPI 卡片显示，趋势图渲染
- [ ] 点击"问题清单"，应看到 8 个种子问题
- [ ] 点击任意问题卡片，弹出办理弹层
- [ ] 切换办理 Tab：办理 / 流程图谱 / 协同留言 / 举措清单
- [ ] 右上角头像下拉，可切换其他用户

### 1.6 跑测试

```bash
# 后端 (需要 docker compose 中的 db 在跑)
cd backend && mvn test
# 应输出: Tests run: 19, Failures: 0, Errors: 0

# 前端
cd frontend && npm test
# 应输出: Test Files 4 passed (4), Tests 13 passed (13)
```

---

## 2. 测试环境部署

### 2.1 服务器准备
- 2 vCPU / 4 GB RAM / 50 GB SSD 起步
- 开放端口：80 (Nginx)，22 (SSH)，内部 8080/5432/9000

### 2.2 安装 JRE + Nginx + 依赖
```bash
sudo dnf install -y java-17-openjdk-headless nginx postgresql-server
# MinIO 单机版
wget https://dl.min.io/server/minio/release/linux-amd64/minio
sudo install minio /usr/local/bin/
```

### 2.3 前端构建产物部署
```bash
# 本地或 CI 构建
cd frontend
npm ci
npm run build
# 产物在 dist/

# 上传到服务器
scp -r dist/* user@host:/var/www/gallant/
```

### 2.4 后端打包部署
```bash
cd backend
mvn -DskipTests package
# 产物: target/collab-backend.jar (Fat JAR)

scp target/collab-backend.jar user@host:/opt/gallant/
```

服务器上 systemd 配置（`/etc/systemd/system/gallant-backend.service`）：

```ini
[Unit]
Description=Gallant Collab Backend
After=network.target postgresql.service

[Service]
Type=simple
User=gallant
WorkingDirectory=/opt/gallant
ExecStart=/usr/bin/java -Xms512m -Xmx1g -jar /opt/gallant/collab-backend.jar \
  --spring.datasource.url=jdbc:postgresql://localhost:5432/gallant \
  --spring.datasource.username=gallant \
  --spring.datasource.password=$DB_PASSWORD \
  --minio.endpoint=http://localhost:9000 \
  --minio.access-key=$MINIO_USER \
  --minio.secret-key=$MINIO_PASSWORD
EnvironmentFile=/opt/gallant/env
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

启用：
```bash
sudo systemctl daemon-reload
sudo systemctl enable --now gallant-backend
sudo systemctl status gallant-backend
```

### 2.5 Nginx 反代
`/etc/nginx/conf.d/gallant.conf`：

```nginx
server {
    listen 80;
    server_name gallant.internal;

    root /var/www/gallant;
    index index.html;

    # 前端 SPA
    location / {
        try_files $uri $uri/ /index.html;
    }

    # 后端 API
    location /api/ {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;
        proxy_set_header   Host              $host;
        proxy_set_header   X-Real-IP         $remote_addr;
        proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header   X-Forwarded-Proto $scheme;
        client_max_body_size 60m;     # 与后端 multipart 上限一致
    }
}
```

```bash
sudo nginx -t && sudo systemctl reload nginx
```

### 2.6 初始化数据库
```bash
# 创建数据库
sudo -u postgres psql -c "CREATE USER gallant WITH PASSWORD '...'"
sudo -u postgres psql -c "CREATE DATABASE gallant OWNER gallant"

# 执行 DDL + 种子（按需）
psql -U gallant -d gallant -f docs/schema.sql
psql -U gallant -d gallant -f docs/seed.sql    # 仅测试环境
```

---

## 3. 生产部署（KingbaseV8）

### 3.1 切换驱动

1. 获取 KingbaseV8 JDBC 驱动 `kingbase8-x.x.x.jar`（从 KingbaseV8 安装目录或厂商提供）。
2. 安装到 Maven 本地仓库或公司私服：
   ```bash
   mvn install:install-file \
     -Dfile=kingbase8-8.6.0.jar \
     -DgroupId=cn.com.kingbase \
     -DartifactId=kingbase8 \
     -Dversion=8.6.0 \
     -Dpackaging=jar
   ```
3. 修改 `backend/pom.xml`：注释 `postgresql` 依赖，取消注释 `kingbase8`。
4. 修改 `MybatisPlusConfig.java` 中 `DbType.POSTGRE_SQL` → `DbType.KINGBASE_ES`。

### 3.2 修改配置

`application-prod.yml`（新建）：

```yaml
spring:
  datasource:
    driver-class-name: com.kingbase8.Driver
    url: jdbc:kingbase8://kb-prod-host:54321/gallant?stringtype=unspecified
    username: gallant
    password: ${DB_PASSWORD}
    hikari:
      maximum-pool-size: 30
      minimum-idle: 5
  servlet:
    multipart:
      max-file-size: 50MB
      max-request-size: 200MB

server:
  port: 8080
  servlet:
    context-path: /api

minio:
  endpoint: ${MINIO_ENDPOINT}
  access-key: ${MINIO_USER}
  secret-key: ${MINIO_PASSWORD}
  bucket: gallant-files-prod

collab:
  mock-user-id: 0           # 生产环境改成不允许 mock; 必须接 SSO
  notification:
    poll-interval-seconds: 30

logging:
  level:
    root: INFO
    com.gallant.collab: INFO
  file:
    name: /var/log/gallant/backend.log
```

启动：
```bash
java -jar collab-backend.jar --spring.profiles.active=prod
```

### 3.3 KingbaseV8 实例要求
- **版本**：V8R6 +
- **兼容模式**：PG 模式（建库时 `--mode pg`）
- **客户端编码**：UTF-8
- **时区**：Asia/Shanghai

### 3.4 DDL 兼容性提醒
本项目 schema 在 KingbaseV8 PG 兼容模式下已验证通过，注意：
- `BIGSERIAL` ✓
- `JSONB` ✓
- `GENERATE_SERIES` ✓ (用于看板趋势)
- `TIMESTAMP DEFAULT CURRENT_TIMESTAMP` ✓
- `to_char(date_trunc(...))` ✓

如启用其他模式（Oracle/MySQL 等），需重写部分 DDL 与聚合 SQL。

---

## 4. Docker 部署（可选）

> 注：当前未提供 Dockerfile，下面是生产打包示例。

### 4.1 后端 Dockerfile（示例）

```dockerfile
# backend/Dockerfile
FROM eclipse-temurin:17-jre-alpine
WORKDIR /app
COPY target/collab-backend.jar app.jar
EXPOSE 8080
ENTRYPOINT ["java","-Xms512m","-Xmx1g","-jar","app.jar","--spring.profiles.active=prod"]
```

构建：
```bash
cd backend && mvn -DskipTests package
docker build -t gallant/backend:1.0.0 .
```

### 4.2 前端 Dockerfile（示例）

```dockerfile
# frontend/Dockerfile
FROM node:20-alpine AS builder
WORKDIR /app
COPY package*.json ./
RUN npm ci
COPY . .
RUN npm run build

FROM nginx:alpine
COPY --from=builder /app/dist /usr/share/nginx/html
COPY nginx.conf /etc/nginx/conf.d/default.conf
EXPOSE 80
```

### 4.3 完整生产 docker-compose

```yaml
services:
  db:
    image: postgres:16-alpine   # 或自建 KingbaseV8 镜像
    environment:
      POSTGRES_DB: gallant
      POSTGRES_USER: gallant
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - db_data:/var/lib/postgresql/data
  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_PASSWORD}
    volumes:
      - minio_data:/data
  backend:
    image: gallant/backend:1.0.0
    environment:
      SPRING_DATASOURCE_URL: jdbc:postgresql://db:5432/gallant
      SPRING_DATASOURCE_USERNAME: gallant
      SPRING_DATASOURCE_PASSWORD: ${DB_PASSWORD}
      MINIO_ENDPOINT: http://minio:9000
      MINIO_ACCESS_KEY: ${MINIO_USER}
      MINIO_SECRET_KEY: ${MINIO_PASSWORD}
    depends_on: [db, minio]
  frontend:
    image: gallant/frontend:1.0.0
    ports: ["80:80"]
    depends_on: [backend]
volumes: { db_data: {}, minio_data: {} }
```

---

## 5. 数据初始化

### 5.1 全新部署
执行 `docs/schema.sql` 建表。**不要**在生产执行 `seed.sql`（含 demo 数据）。

### 5.2 用户导入
当前用户表（`app_user`）需要手工录入。后续接入 SSO 后可自动从 LDAP / OAuth 同步。临时方案：

```sql
INSERT INTO app_user(name, dept) VALUES
  ('张三', '运营中心'),
  ('李四', '产品中心'),
  ...
;
```

### 5.3 数据迁移
若从老系统导入历史问题：
1. 准备 CSV，字段对齐 problem 表
2. 用 `psql \copy` 或 ETL 工具导入
3. 每条历史动作补 `stage_history` 行
4. 重新计算 `problem.status` 与 `progress`

---

## 6. 备份与恢复

### 6.1 数据库
```bash
# 每日全备 (crontab)
0 2 * * * pg_dump -U gallant gallant | gzip > /backup/gallant-$(date +%F).sql.gz

# 恢复
gunzip -c gallant-2026-05-25.sql.gz | psql -U gallant -d gallant
```

### 6.2 MinIO
```bash
# 用 mc (MinIO Client) 同步到异地
mc mirror --overwrite local/gallant-files-prod remote/backup/gallant
```

### 6.3 配置文件
后端 `application-prod.yml`、Nginx 配置、systemd 单元文件，全部纳入 git 仓库。

---

## 7. 监控

### 7.1 Spring Boot Actuator
开启 `/api/actuator/health` 和 `/api/actuator/metrics`，对接 Prometheus + Grafana。

```yaml
management:
  endpoints:
    web:
      exposure:
        include: health,info,metrics,prometheus
```

### 7.2 关键监控项
- 后端 JVM 内存、GC、线程数
- HTTP 接口 P50/P95/P99 + 错误率
- DB 连接池使用率（HikariCP）
- DB QPS + 慢查询
- MinIO 存储用量 + 读写延迟
- 业务指标：每日新增问题数、办结数、超期数

### 7.3 告警
- 接口错误率 > 1% 持续 5min
- 数据库连接池占用 > 80%
- 超期问题数 > 阈值
- 磁盘剩余 < 20%

---

## 8. 升级

### 8.1 版本回滚策略
1. 后端：保留上一版 jar，systemd 切换重启
2. 前端：保留上一版 dist 目录，Nginx 软链切换
3. DB schema 变更：使用 Flyway 或 Liquibase 管理，可逆 migration

### 8.2 滚动升级（多实例）
1. 摘除一个实例（Nginx upstream 标记 down）
2. 部署新版本
3. 健康检查通过后加回
4. 依次升级其他实例

---

## 9. 常见问题

### Q: 启动后看板报 500，前端"服务异常"
- 检查 `application-dev.yml` 中数据库连接是否正确
- 检查 `docker compose ps` 中 db 是否 healthy
- 查看 `backend/logs/spring.log` 详细错误

### Q: 文件上传失败
- 检查 MinIO bucket 是否存在（启动时若 MinIO 未起会有警告日志）
- 检查文件大小是否超过 50MB
- 检查 Nginx `client_max_body_size` 是否设置 ≥ 60m

### Q: 看板某些数据为 0
- 看板查询过滤了若干条件（如 `done` 必须有 evaluation 记录）
- 重新执行 seed.sql 确认数据完整

### Q: Maven 测试找不到 testcontainers
- 已切换为本地 PG 模式，不依赖 testcontainers
- 仅需 `docker compose up -d db` 后跑 `mvn test`

### Q: KingbaseV8 驱动找不到
- 必须手工 `mvn install:install-file`，未在 Maven Central
- 或公司内部 Nexus 配置 mirror

### Q: 前端 npm install 慢
- 配国内源：`npm config set registry https://registry.npmmirror.com`

---

## 10. 联系与反馈

- 项目代码：`<git-repo-url>`
- 缺陷反馈：通过本平台的"问题提报"功能 :)
- 紧急联系：运维值班 oncall
