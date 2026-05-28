# IOP — PG 备份与恢复 runbook

## 目标

- RPO: < 24h (单实例 + 每日备份)
- RTO: < 4h (单租户恢复); < 2h (全库恢复)

## 备份位置

- 本地: `/var/backups/iop/{daily,weekly}/iop-<UTC>.dump` (pg_dump custom format, gzip 6)
- 异机: 若 `IOP_BACKUP_S3` 设置 → 同步到 S3/MinIO

## 全库恢复 (灾难场景)

```bash
# 1. 准备空 PG 实例 (注意: 不能恢复到生产实例！先恢复到 staging)
docker compose -f deployments/docker-compose.yml exec -T db psql -U iop -d postgres -c "DROP DATABASE IF EXISTS iop"
docker compose exec -T db psql -U iop -d postgres -c "CREATE DATABASE iop"

# 2. 恢复
pg_restore -h <host> -p <port> -U iop -d iop --clean --if-exists --no-owner /path/to/iop-<ts>.dump

# 3. 验证
docker compose exec -T db psql -U iop -d iop -c "SELECT scope, count(*) FROM public.migration_history GROUP BY scope"

# 4. 命门测试
cd server && IOP_INTEGRATION=1 go test ./test/integration/...
```

## 单租户恢复 (从全备拆出某个 tenant_<slug>)

```bash
# 1. 从全备里提取该 tenant 的 schema
pg_restore -l /path/to/iop-<ts>.dump | grep -E "SCHEMA tenant_<slug>|TABLE tenant_<slug>\." > restore-list.txt

# 2. 在 staging 上还原以验证
pg_restore -h staging -U iop -d iop_staging -L restore-list.txt /path/to/iop-<ts>.dump

# 3. 验证数据
docker compose exec -T db psql -U iop -d iop_staging -c "SET search_path TO tenant_<slug>, public; SELECT count(*) FROM member"

# 4. 选择性回写生产 (只在确认数据正确后, 由人工执行)
#    例: 用 \copy + 业务对比工具迁移数据
```

## 完整性校验

每次备份自动跑 `pg_restore -l` 校验. CI 周期任务 (M5+) 在 staging 跑一次端到端恢复演练.

## 关键人员

- 备份责任人: ops
- 恢复决策人: cto + lead engineer
- 联系方式: 见 docs/operations/oncall.md

## v1.5+ 升级路径

- `pg_basebackup` + WAL 归档 → RPO < 5 分钟
- 主从复制 + 自动 failover → RTO < 30 分钟
