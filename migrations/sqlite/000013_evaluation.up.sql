-- Persist evaluation runs and their metric results (Lite).
-- Mirrors migrations/versioned/000087. Row ids are generated in Go, so there
-- are no server-side defaults on the id columns.

CREATE TABLE IF NOT EXISTS evaluation_runs (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    dataset_id VARCHAR(255) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    err_msg TEXT NOT NULL DEFAULT '',
    total INTEGER NOT NULL DEFAULT 0,
    finished INTEGER NOT NULL DEFAULT 0,
    params TEXT,
    start_time DATETIME,
    end_time DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_evaluation_runs_tenant
    ON evaluation_runs (tenant_id);

CREATE TABLE IF NOT EXISTS evaluation_metrics (
    id VARCHAR(36) PRIMARY KEY,
    run_id VARCHAR(255) NOT NULL,
    tenant_id INTEGER NOT NULL,
    retrieval_metrics TEXT,
    generation_metrics TEXT,
    cost REAL,
    latency_ms INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_evaluation_metrics_run
    ON evaluation_metrics (run_id);
CREATE INDEX IF NOT EXISTS idx_evaluation_metrics_tenant
    ON evaluation_metrics (tenant_id);
