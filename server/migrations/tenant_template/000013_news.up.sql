-- 时政资讯 (gov news CMS, 政务门户 / 人民网频道-style) tables — per tenant.
-- Lives INSIDE each tenant_<slug> schema, so no schema prefix and no tenant_id
-- column — schema isolation handles that. Applied to every existing tenant
-- schema on boot via SyncAllSchemas.
-- Aggregates: news_category (栏目) and news_article (文章).

CREATE TABLE IF NOT EXISTS news_category (
    id         UUID PRIMARY KEY,
    name       TEXT NOT NULL,
    order_num  INT  NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS news_category_order_idx ON news_category(order_num, name);

CREATE TABLE IF NOT EXISTS news_article (
    id            UUID PRIMARY KEY,
    category_id   UUID REFERENCES news_category(id) ON DELETE SET NULL,
    title         TEXT NOT NULL,
    summary       TEXT NOT NULL DEFAULT '',
    content       TEXT NOT NULL DEFAULT '',
    cover_url     TEXT NOT NULL DEFAULT '',
    author        TEXT NOT NULL DEFAULT '',
    status        TEXT NOT NULL DEFAULT 'draft',  -- 'draft' | 'published'
    published_at  TIMESTAMPTZ,
    views         INT  NOT NULL DEFAULT 0,
    created_by    UUID,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS news_article_status_idx    ON news_article(status, published_at DESC);
CREATE INDEX IF NOT EXISTS news_article_category_idx  ON news_article(category_id);
CREATE INDEX IF NOT EXISTS news_article_created_idx   ON news_article(created_at DESC);
