-- +goose Up

CREATE TABLE bot_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chat_id VARCHAR(50) NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    intent VARCHAR(50) NOT NULL,
    state JSONB NOT NULL DEFAULT '{}',
    step INT NOT NULL DEFAULT 0,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bot_sessions_chat ON bot_sessions(chat_id, expires_at DESC);

CREATE TABLE bot_user_links (
    telegram_user_id BIGINT PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    telegram_username VARCHAR(255),
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_bot_user_links_user ON bot_user_links(user_id);

CREATE TABLE llm_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    provider VARCHAR(20) NOT NULL,
    api_key VARCHAR(500),
    model VARCHAR(100) NOT NULL,
    base_url VARCHAR(500),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down
DROP TABLE IF EXISTS llm_configs;
DROP TABLE IF EXISTS bot_user_links;
DROP TABLE IF EXISTS bot_sessions;
