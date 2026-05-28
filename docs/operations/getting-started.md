# IOP 生产部署 · Getting Started

## 一次性准备

1. **域名 + TLS 证书**

   申请域名后用 certbot / acme.sh 获取 Let's Encrypt 证书:

   ```bash
   docker run -it --rm \
     -v /etc/letsencrypt:/etc/letsencrypt \
     certbot/certbot certonly --standalone -d iop.your-company.com
   ```

   把 `fullchain.pem` / `privkey.pem` 复制到 `deployments/nginx/certs/`.

2. **创建 .env**

   `deployments/.env`:

   ```env
   IOP_DB_USER=iop
   IOP_DB_PASSWORD=<strong-random-32+>
   IOP_DB_NAME=iop
   IOP_JWT_SECRET=<random-64-bytes-base64>
   IOP_MINIO_ACCESS_KEY=<access-key>
   IOP_MINIO_SECRET_KEY=<secret-key>
   IMAGE_TAG=v1.0.0
   ```

3. **首次启动**

   ```bash
   cd deployments
   docker compose -f docker-compose.prod.yml up -d db redis minio
   # 等数据库就绪
   docker compose -f docker-compose.prod.yml exec db pg_isready -U iop -d iop
   # 跑迁移
   docker compose -f docker-compose.prod.yml run --rm server /usr/local/bin/migrate up
   # 启动业务进程
   docker compose -f docker-compose.prod.yml up -d server web nginx backup
   ```

4. **验证**

   ```bash
   curl https://iop.your-company.com/api/version | jq .
   ```

## 创建首个租户 + 管理员

```bash
docker compose -f docker-compose.prod.yml exec server tenantctl tenant create --slug=acme --name="ACME, Inc."
docker compose -f docker-compose.prod.yml exec server tenantctl user create --email=admin@acme.com --password=Init123abc!
docker compose -f docker-compose.prod.yml exec server tenantctl member join \
  --tenant=<tenant-id> --user=<user-id> --name="管理员" --email=admin@acme.com
docker compose -f docker-compose.prod.yml exec server tenantctl role grant \
  --tenant=<tenant-id> --member=<member-id> --code=tenant_admin
```

要求 admin 立即登录并改密码.

## 升级流程

```bash
# 1. 拉取新镜像 tag
export IMAGE_TAG=v1.1.0

# 2. 跑新增的 platform 迁移 (无停机)
docker compose -f docker-compose.prod.yml run --rm server migrate up

# 3. 滚动重启 server (web 通常无状态可直接 up)
docker compose -f docker-compose.prod.yml up -d --no-deps server

# 4. 跑新增的 tenant_template 迁移到所有现有租户
docker compose -f docker-compose.prod.yml exec server tenantctl tenant migrate-all  # M5+
```

## 监控指标

- Prometheus: 抓取 `https://iop.example.com/metrics` (需在 nginx allow list 中加 prometheus 节点)
- 关键指标:
  - `iop_http_request_duration_seconds` (route / status)
  - `iop_pg_slow_query_total{severity=warn|error}` — **>1s 必须告警**
  - Go runtime: `go_memstats_*`, `go_goroutines`
- 日志: 容器 stdout JSON, 配 Loki / ELK 即可

## 健康检查

```bash
curl https://iop.example.com/livez    # 进程存活
curl https://iop.example.com/readyz   # 主路径依赖 (PG)
curl https://iop.example.com/healthz  # 详细依赖矩阵
```

readyz 返 503 时 → nginx 流量摘除 (默认 LB 配置), DB 恢复后自动加回.

## 常见故障

| 现象 | 排查 | 解决 |
|---|---|---|
| /readyz 503 | `docker compose ps db` 查 PG 健康 | PG 重启 / 扩容 |
| 上传 413 | nginx `client_max_body_size` 太小 | 改 conf 后 `docker compose exec nginx nginx -s reload` |
| 全部租户 429 | 单租户脚本失控 | 通过 audit log 定位 → `tenantctl tenant suspend --id=<id>` 临时停用 |
| Redis 挂 | server 仍可用但限流降级 | 重启 Redis (volume 不丢) |

## 备份与恢复

见 `deployments/backup/restore_runbook.md`.
