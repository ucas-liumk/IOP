-- ================================================================
-- 问题协同研究解决平台 - 数据库 DDL
-- 兼容: PostgreSQL 12+ / KingbaseV8 PG 兼容模式
-- 约定: 主键统一 BIGINT (problem 用业务编号 VARCHAR)
--      时间统一 TIMESTAMP
--      所有动态枚举用 VARCHAR + CHECK，便于后续扩展
-- ================================================================

DROP TABLE IF EXISTS evaluation CASCADE;
DROP TABLE IF EXISTS consult_stat CASCADE;
DROP TABLE IF EXISTS attachment CASCADE;
DROP TABLE IF EXISTS message CASCADE;
DROP TABLE IF EXISTS dispute_position CASCADE;
DROP TABLE IF EXISTS dispute CASCADE;
DROP TABLE IF EXISTS measure CASCADE;
DROP TABLE IF EXISTS stage_history CASCADE;
DROP TABLE IF EXISTS problem CASCADE;
DROP TABLE IF EXISTS app_user CASCADE;

-- ============================ 用户 ============================
CREATE TABLE app_user (
  id           BIGSERIAL PRIMARY KEY,
  name         VARCHAR(64)  NOT NULL,
  dept         VARCHAR(128) NOT NULL,
  avatar_color VARCHAR(16),
  created_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
COMMENT ON TABLE  app_user      IS '用户表（保留身份用于审计，不做权限）';
COMMENT ON COLUMN app_user.dept IS '所在部门';

CREATE INDEX idx_user_name ON app_user(name);

-- ============================ 问题 ============================
CREATE TABLE problem (
  id              VARCHAR(32)  PRIMARY KEY,           -- 业务编号 WT20260510-001
  title           VARCHAR(256) NOT NULL,
  description     TEXT,
  category        VARCHAR(32)  NOT NULL,              -- 战略规划/运营效率/...
  priority        VARCHAR(16)  NOT NULL DEFAULT 'normal',
  status          VARCHAR(16)  NOT NULL DEFAULT 'pending',
  branch          VARCHAR(16),                        -- dispute/consensus/NULL
  current_stage   VARCHAR(16)  NOT NULL DEFAULT 'submit',
  submitter_id    BIGINT       NOT NULL REFERENCES app_user(id),
  submitter_dept  VARCHAR(128) NOT NULL,
  handler_name    VARCHAR(128),                       -- 承办单位 / 承办人
  handler_dept    VARCHAR(128),
  submit_date     DATE         NOT NULL,
  due_date        DATE,
  progress        INTEGER      NOT NULL DEFAULT 0,    -- 0~100
  overdue         BOOLEAN      NOT NULL DEFAULT FALSE,
  overdue_days    INTEGER      NOT NULL DEFAULT 0,
  latest          TEXT,                               -- 最新进展摘要
  tags            JSONB        NOT NULL DEFAULT '[]'::jsonb,
  participants    JSONB        NOT NULL DEFAULT '[]'::jsonb,
  created_at      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  updated_at      TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_priority CHECK (priority IN ('critical','high','normal','low')),
  CONSTRAINT chk_status   CHECK (status IN ('pending','processing','meeting','arbitrate','consulting','done','overdue')),
  CONSTRAINT chk_branch   CHECK (branch IS NULL OR branch IN ('dispute','consensus')),
  CONSTRAINT chk_stage    CHECK (current_stage IN ('submit','review','propose','meeting','arbitrate','consult','implement','evaluate'))
);
COMMENT ON TABLE problem IS '问题主表';

CREATE INDEX idx_problem_status        ON problem(status);
CREATE INDEX idx_problem_stage         ON problem(current_stage);
CREATE INDEX idx_problem_submitter     ON problem(submitter_id);
CREATE INDEX idx_problem_handler_dept  ON problem(handler_dept);
CREATE INDEX idx_problem_due_date      ON problem(due_date);
CREATE INDEX idx_problem_category      ON problem(category);

-- ============================ 阶段历史 ============================
CREATE TABLE stage_history (
  id            BIGSERIAL PRIMARY KEY,
  problem_id    VARCHAR(32)  NOT NULL REFERENCES problem(id) ON DELETE CASCADE,
  stage         VARCHAR(16)  NOT NULL,
  occurred_at   TIMESTAMP    NOT NULL,
  actor_user_id BIGINT       REFERENCES app_user(id),
  actor_name    VARCHAR(64)  NOT NULL,                -- 冗余, 兼容机构/系统类操作
  actor_dept    VARCHAR(128) NOT NULL,
  note          TEXT,
  files         JSONB        NOT NULL DEFAULT '[]'::jsonb,  -- ["filename.pdf", ...]
  branch_choice VARCHAR(16),                                -- 当 stage=propose 时记录分支
  created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_history_stage CHECK (stage IN ('submit','review','propose','meeting','arbitrate','consult','implement','evaluate'))
);
COMMENT ON TABLE stage_history IS '阶段历史 / 审计流水';

CREATE INDEX idx_history_problem    ON stage_history(problem_id, occurred_at DESC);
CREATE INDEX idx_history_actor      ON stage_history(actor_user_id);

-- ============================ 举措 ============================
CREATE TABLE measure (
  id           BIGSERIAL PRIMARY KEY,
  problem_id   VARCHAR(32)  NOT NULL REFERENCES problem(id) ON DELETE CASCADE,
  code         VARCHAR(8)   NOT NULL,              -- M1, M2 ...
  title        TEXT         NOT NULL,
  owner        VARCHAR(128),
  status       VARCHAR(16)  NOT NULL DEFAULT 'proposed',
  has_dispute  BOOLEAN      NOT NULL DEFAULT FALSE,
  progress     INTEGER      NOT NULL DEFAULT 0,
  display_order INTEGER     NOT NULL DEFAULT 0,
  created_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  updated_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_measure_status CHECK (status IN ('proposed','drafting','approved','in_progress','completed')),
  CONSTRAINT uk_measure UNIQUE (problem_id, code)
);
COMMENT ON TABLE measure IS '举措清单';

CREATE INDEX idx_measure_problem ON measure(problem_id);

-- ============================ 争议点 ============================
CREATE TABLE dispute (
  id           BIGSERIAL PRIMARY KEY,
  problem_id   VARCHAR(32)  NOT NULL REFERENCES problem(id) ON DELETE CASCADE,
  point        TEXT         NOT NULL,
  resolution   TEXT,                                  -- 裁决结论
  display_order INTEGER     NOT NULL DEFAULT 0,
  created_at   TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_dispute_problem ON dispute(problem_id);

CREATE TABLE dispute_position (
  id         BIGSERIAL PRIMARY KEY,
  dispute_id BIGINT       NOT NULL REFERENCES dispute(id) ON DELETE CASCADE,
  party      VARCHAR(128) NOT NULL,
  view_text  TEXT         NOT NULL,
  created_at TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_position_dispute ON dispute_position(dispute_id);

-- ============================ 留言 ============================
CREATE TABLE message (
  id            BIGSERIAL PRIMARY KEY,
  problem_id    VARCHAR(32) NOT NULL REFERENCES problem(id) ON DELETE CASCADE,
  actor_user_id BIGINT      REFERENCES app_user(id),
  actor_name    VARCHAR(64) NOT NULL,
  content       TEXT        NOT NULL,
  mentions      JSONB       NOT NULL DEFAULT '[]'::jsonb,  -- ["@张三","@运营中心"]
  occurred_at   TIMESTAMP   NOT NULL DEFAULT CURRENT_TIMESTAMP,
  read_by       JSONB       NOT NULL DEFAULT '[]'::jsonb   -- [user_id, ...]
);
CREATE INDEX idx_message_problem ON message(problem_id, occurred_at DESC);

-- ============================ 附件 ============================
CREATE TABLE attachment (
  id              BIGSERIAL PRIMARY KEY,
  problem_id      VARCHAR(32)  NOT NULL REFERENCES problem(id) ON DELETE CASCADE,
  stage           VARCHAR(16)  NOT NULL,
  file_name       VARCHAR(256) NOT NULL,
  file_size       BIGINT       NOT NULL,
  content_type    VARCHAR(128),
  object_key      VARCHAR(512) NOT NULL UNIQUE,        -- MinIO 对象 key
  uploader_id     BIGINT       REFERENCES app_user(id),
  uploader_name   VARCHAR(64)  NOT NULL,
  uploaded_at     TIMESTAMP    DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_attachment_problem ON attachment(problem_id, stage);

-- ============================ 征求意见反馈汇总 ============================
CREATE TABLE consult_stat (
  problem_id    VARCHAR(32) PRIMARY KEY REFERENCES problem(id) ON DELETE CASCADE,
  total_count   INTEGER NOT NULL DEFAULT 0,
  support_count INTEGER NOT NULL DEFAULT 0,
  neutral_count INTEGER NOT NULL DEFAULT 0,
  oppose_count  INTEGER NOT NULL DEFAULT 0,
  start_date    DATE,
  end_date      DATE,
  brief         TEXT,
  revision      TEXT,
  updated_at    TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================ 评价 ============================
CREATE TABLE evaluation (
  id            BIGSERIAL PRIMARY KEY,
  problem_id    VARCHAR(32)  NOT NULL REFERENCES problem(id) ON DELETE CASCADE,
  evaluator_id  BIGINT       REFERENCES app_user(id),
  evaluator_name VARCHAR(64) NOT NULL,
  party         VARCHAR(32)  NOT NULL,                -- 提报方/相关方/...
  quality       NUMERIC(3,1) NOT NULL,
  speed         NUMERIC(3,1) NOT NULL,
  collab        NUMERIC(3,1) NOT NULL,
  satisfaction  NUMERIC(3,1) NOT NULL,
  overall       NUMERIC(3,1) NOT NULL,
  comment_text  TEXT,
  archive_best_practice BOOLEAN NOT NULL DEFAULT FALSE,
  created_at    TIMESTAMP    DEFAULT CURRENT_TIMESTAMP,
  CONSTRAINT chk_score CHECK (
    quality BETWEEN 0 AND 5 AND speed BETWEEN 0 AND 5 AND
    collab BETWEEN 0 AND 5  AND satisfaction BETWEEN 0 AND 5
  )
);
CREATE INDEX idx_eval_problem ON evaluation(problem_id);
