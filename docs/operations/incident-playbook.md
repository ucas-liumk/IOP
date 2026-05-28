# 事故响应手册

## 0. 严重程度划分

| Sev | 影响范围 | 响应 SLA | 通知 |
|---|---|---|---|
| Sev1 | 全部租户登录失败 / 数据丢失 | 即时 | 全员 + CTO |
| Sev2 | 单租户不可用 / 性能严重劣化 | < 15 分钟 | 值班 + 主管 |
| Sev3 | 单功能故障 / 性能轻度劣化 | < 1 小时 | 值班 |
| Sev4 | 文案 / 视觉小 bug | < 1 天 | 工单 |

## 1. 全部租户登录失败

1. `curl https://iop.example.com/api/version` 看 server 进程
2. `curl https://iop.example.com/readyz` 看依赖矩阵
3. 若 PG 挂 → 见 §pg-down
4. 若 Redis 挂 → 登录仍能用 (session 黑名单失效) , 用户可继续
5. 若 server 进程挂 → `docker compose ps server`, 看日志 `docker logs deployments-server-1 --tail 200`
6. 检查 nginx error log: `docker logs deployments-nginx-1 --tail 200`

## 2. PG 不可用 (pg-down)

1. `docker compose exec db pg_isready` — 若 fail, 查看磁盘空间 `df -h`
2. 若磁盘满: 见 §disk-full
3. 重启容器: `docker compose restart db`
4. 若数据损坏 → 见 §restore

## 3. 单租户不可用

可能原因:
- 租户被 suspend → 查 audit log
- 该租户 schema 损坏 → `psql -c "\dn" | grep tenant_<slug>`
- 该租户 rate limit 长期触发 → 查 Redis: `redis-cli KEYS "rl:t:<tenant>:*"`

恢复路径:
- 数据损坏: 见 `deployments/backup/restore_runbook.md` § 单租户恢复

## 4. 磁盘满 (disk-full)

1. `du -sh /var/lib/docker/volumes/*` 找占用大的卷
2. 通常是 pg-data 或 minio-data
3. **不要**直接删除卷; 先备份再压缩或迁移到大盘
4. PG vacuum: `docker compose exec db vacuumdb -U iop -d iop --analyze`
5. MinIO 清理: 删除超过保留期的 audit attachment (M5+ 自动化)

## 5. 数据恢复 (restore)

见 `deployments/backup/restore_runbook.md`. 关键:
- **不要直接恢复到生产实例** — 先恢复到 staging 验证
- 单租户恢复优先, 避免全库 rollback

## 6. 安全事件

如发现:
- 异常登录 (audit log `iam.login_failed` 暴增)
- 异常 schema 创建 (tenancy.tenant_created 频次)
- 慢查询 (>10s) 突然飙升 — 可能是注入尝试

立即:
1. 暂停所有租户新建 (临时禁用 /tenants POST 在 nginx 层)
2. 抓取 access log + audit log 做后续追查
3. 强制所有 session 失效: `docker compose exec server tenantctl session revoke-all` (M6+)
4. 通知安全 / 法务

## 7. 性能劣化

诊断顺序:
1. /metrics `iop_http_request_duration_seconds` P99
2. /metrics `iop_pg_slow_query_total` — 查日志找具体 SQL
3. `EXPLAIN ANALYZE` 找缺失索引
4. PG `pg_stat_activity` 看长事务

## 8. 联系人

- 值班: 见 `docs/operations/oncall.md`
- DBA 兜底: dba@your-company.com
- 安全: sec@your-company.com
