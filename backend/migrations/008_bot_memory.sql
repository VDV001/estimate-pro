-- +goose Up

-- Conversation memory: stores recent exchanges per user for context-aware responses.
CREATE TABLE bot_memory (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    chat_id VARCHAR(50) NOT NULL,
    role VARCHAR(10) NOT NULL,  -- 'user' or 'esti'
    content TEXT NOT NULL,
    intent VARCHAR(50),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_bot_memory_user_recent ON bot_memory(user_id, created_at DESC);

-- User preferences learned over time (communication style, etc.)
CREATE TABLE bot_user_prefs (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    style VARCHAR(20) NOT NULL DEFAULT 'casual',  -- casual, formal, brief
    language VARCHAR(5) NOT NULL DEFAULT 'ru',     -- preferred response language
    notes TEXT NOT NULL DEFAULT '',                 -- LLM-generated notes about the user
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- +goose Down

DROP TABLE IF EXISTS bot_user_prefs;
DROP TABLE IF EXISTS bot_memory;
