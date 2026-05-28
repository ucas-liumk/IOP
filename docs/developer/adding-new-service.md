# 添加一个新服务包

> 适用于通用/支持子域 (CRUD 为主, 业务规则少). 不要给这种模块用 DDD 四层.

以加一个 "Knowledge Base (kb)" 服务包为例.

## 步骤 1: 目录结构

```bash
mkdir -p server/internal/services/kb
```

服务包内一般 3-5 个文件:

```
kb/
├── types.go        # 类型定义 + Repository interface
├── service.go      # 业务逻辑 (Service struct + 方法)
├── pg_repo.go      # PostgreSQL 实现
└── http.go         # Gin handler + RegisterRoutes
```

## 步骤 2: 数据库表

在 `server/migrations/tenant_template/000004_kb.up.sql` (递增编号):

```sql
CREATE TABLE IF NOT EXISTS kb_article (
    id          UUID PRIMARY KEY,
    author      UUID NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL,
    tags        TEXT[] DEFAULT ARRAY[]::TEXT[],
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS kb_article_author_idx ON kb_article(author);
```

平台级表 (跨租户) 放 `migrations/public/`.

## 步骤 3: types.go

```go
package kb

import (
	"context"
	"time"

	"github.com/leo/iop/server/internal/shared/kernel"
)

type Article struct {
	ID        kernel.ID `json:"id"`
	Author    kernel.ID `json:"author"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	Tags      []string  `json:"tags"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Repository interface {
	Save(ctx context.Context, a *Article) error
	Get(ctx context.Context, id kernel.ID) (*Article, error)
	List(ctx context.Context, p kernel.Pagination) ([]*Article, error)
}
```

## 步骤 4: pg_repo.go

走 `tenantdb.TenantDB.Transaction` (自动 SET LOCAL search_path).

## 步骤 5: service.go + http.go

参考 `services/dictionary/` 现成模板.

## 步骤 6: 在 app/app.go wire 进去

```go
import "github.com/leo/iop/server/internal/services/kb"

// in Build():
a.KB = kb.NewService(kb.NewPGRepo(pool, tenantDB), clk)

// in Engine():
kb.RegisterRoutes(authT, a.KB)
```

## 步骤 7: 加 OpenAPI

在 `server/api/openapi.yaml` 加 paths + schema.

## 步骤 8: 发事件 (如果有的话)

```go
bus.Publish(ctx, "kb.article_created", ArticleCreated{...})
```

记得在 `docs/architecture/event-catalog.md` 登记 topic.

## 步骤 9: 测试

```go
// services/kb/service_test.go
func TestSaveAndGet(t *testing.T) {
	// 用真实 PG (从 docker-compose), 不要 mock repo
}
```

集成测试放 `test/integration/`.

## 不要做的事

- ❌ 不要给 CRUD-only 模块写 domain/application/infrastructure/interface 四层
- ❌ 不要在 service.go 里直接拼 SQL (用 repo 接口)
- ❌ 不要让 service A import service B 内部类型 (用接口注入)
- ❌ 不要跨 schema JOIN

## 何时升级为有界上下文 (contexts/)?

只有当**所有**以下都为真时:
- 业务规则复杂 (多种状态机, 多个不变量, 跨实体一致性)
- 多人协作开发 (需要纪律边界)
- 团队对 DDD 有基本理解

否则保持服务包.  保持简单是 v3.1 的核心信条.
