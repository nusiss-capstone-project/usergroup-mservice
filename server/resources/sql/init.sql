-- usergroup-mservice schema (PostgreSQL)

CREATE TABLE IF NOT EXISTS user_group (
    id          BIGSERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    rule_config JSONB        NOT NULL,
    expression  TEXT         NOT NULL,
    status      VARCHAR(32)  NOT NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_group_status ON user_group (status);

CREATE TABLE IF NOT EXISTS user_full_info (
    id         BIGSERIAL PRIMARY KEY,
    user_id    BIGINT       NOT NULL,
    profile    JSONB        NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS uk_user_full_info_user_id
    ON user_full_info (user_id);

CREATE INDEX IF NOT EXISTS idx_user_full_info_gin
    ON user_full_info
    USING GIN (profile jsonb_path_ops);
