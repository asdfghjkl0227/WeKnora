-- Migration 000088: per-call model usage records.
--
-- Before this change, token usage was only written to logs. This table gives
-- every model call a durable row so the model management page can show call
-- counts, cache hit rate and cost per model, aggregated over time.
--
-- model_id holds the model name reported by the chat/embedding layer (the same
-- string already written to the "[LLM Usage]" log line), which is the human
-- readable identity shown in the UI.

CREATE TABLE IF NOT EXISTS model_usage_records (
    id VARCHAR(36) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    model_id VARCHAR(255) NOT NULL,
    purpose VARCHAR(64) NOT NULL DEFAULT '',
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    cached_tokens INTEGER NOT NULL DEFAULT 0,
    cache_read_tokens INTEGER NOT NULL DEFAULT 0,
    cost DOUBLE PRECISION NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_model_usage_records_tenant_model_time
    ON model_usage_records (tenant_id, model_id, created_at);
