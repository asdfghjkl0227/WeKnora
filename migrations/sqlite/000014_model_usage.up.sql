-- Per-call model usage records (Lite). Mirrors migrations/versioned/000088.

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
    cost REAL NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_model_usage_records_tenant_model_time
    ON model_usage_records (tenant_id, model_id, created_at);
