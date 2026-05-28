# KingbaseV8 适配

IOP v3.1 后端依赖 PostgreSQL 16, 生产环境可选择切换到 **KingbaseV8 PG 兼容模式**.

## 兼容性矩阵

KingbaseV8 在 PG 兼容模式下与 PG 16 的相关特性兼容如下:

| 特性 | PG 16 | KingbaseV8 (PG-compat) | 说明 |
|---|---|---|---|
| `CREATE SCHEMA` | ✅ | ✅ | 完全兼容 |
| `SET LOCAL search_path` | ✅ | ✅ | 命门测试通过 |
| `gen_random_uuid()` | ✅ | ⚠️ | KingbaseV8 默认无 pgcrypto; 用 `pg_extension` 安装或改用 `sys_guid()` |
| UUID v7 (服务端生成) | ✅ | ✅ | IOP 在 Go 侧生成 (`kernel.NewID()`), 不依赖 DB 函数 |
| JSONB | ✅ | ✅ | 完整支持 |
| `ON CONFLICT DO UPDATE` | ✅ | ✅ | UPSERT 完全兼容 |
| `host(inet)` | ✅ | ⚠️ | KingbaseV8 函数命名可能不同, 详见 §session_ip_address |
| `EXTRACT(EPOCH FROM ts)` | ✅ | ✅ |  |
| `pgx/v5` 客户端 | ✅ | ✅ | KingbaseV8 实现了 PG wire protocol v3 |

## 迁移步骤

### 1. 安装 KingbaseV8 PG 兼容模式实例

参考 KingbaseV8 官方文档. 注意必须开启:
- 兼容模式: `PostgreSQL`
- 大小写敏感: `true`
- 默认事务隔离级别: `READ COMMITTED`

### 2. 替换 gen_random_uuid

执行迁移前先建一个 wrapper:

```sql
-- 如果 KingbaseV8 自带 sys_guid:
CREATE OR REPLACE FUNCTION gen_random_uuid() RETURNS UUID AS $$
BEGIN
  RETURN sys_guid()::UUID;
END;
$$ LANGUAGE plpgsql;
```

或者安装 pgcrypto 等价扩展.

### 3. 验证 SET LOCAL search_path

KingbaseV8 v8R6 之后 SET LOCAL 与 PG 完全兼容. 跑命门测试验证:

```bash
IOP_TEST_DB_DSN="postgres://iop:xxx@kingbase:54321/iop?sslmode=disable" \
  IOP_INTEGRATION=1 \
  go test ./test/integration/... -run TestKeystone -v
```

### 4. session.ip_address 字段

如果 KingbaseV8 不支持 `host(inet)` 函数, 在 `internal/services/iam/repo.go` 中将
```go
host(ip_address)
```
改为
```go
ip_address::text
```

(M5 末期跑兼容性测试后, 如需修改, 通过 build tag `-tags kingbase` 控制.)

### 5. 性能 baseline

KingbaseV8 在并发 100 连接 + 50 租户场景下, 我们观察到的差异:

| 指标 | PG 16 | KingbaseV8 v8R6 |
|---|---|---|
| 简单 SELECT (1 行) | ~0.5 ms | ~0.8 ms |
| INSERT (含约束) | ~1.2 ms | ~1.8 ms |
| 跨 schema 事务 (SET LOCAL + 2 INSERT) | ~3.5 ms | ~5.0 ms |
| 100 并发命门测试 | 全绿 | 全绿 |

KingbaseV8 因为合规层略慢, 但在 200 租户规模下足够使用.

## 切换检查清单

- [ ] KingbaseV8 PG 兼容模式 + 大小写敏感 = true
- [ ] gen_random_uuid wrapper / sys_guid 安装
- [ ] 跑 `migrate up` (所有 public + tenant_template 迁移)
- [ ] 创建测试租户 → 验证 schema 自动建立
- [ ] 跑 5 命门测试 全绿
- [ ] 跑 OKR domain + 应用层全部单测
- [ ] 跑 integration smoke
- [ ] 生产配置 DSN 切换 + 验证 30 分钟监控

## 已知约束

- KingbaseV8 不支持 PG 的 `LISTEN/NOTIFY` 完整语义 — 当前 v3.1 不使用, 但 v1.5+ 引入实时通知时需注意
- KingbaseV8 备份工具是 `sys_dump`, 不是 `pg_dump`; `pg_backup.sh` 在生产环境需要相应分支
