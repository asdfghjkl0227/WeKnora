-- Migration 000087: persist evaluation runs and their metric results.
--
-- Before this change, evaluation results lived in process memory
-- (evaluationMemoryStorage in internal/application/service/evaluation.go)
-- and were lost on restart. These two tables give every run a durable record:
--
--   * evaluation_runs    — one row per run: the task snapshot plus the
--     ChatManage config snapshot (params) used to reproduce the run;
--   * evaluation_metrics — one row per run: the computed metric scores.
--     The cost / latency_ms columns are reserved for a later task and are
--     left NULL for now.
--
-- Task ids are generated in Go (utils.GenerateTaskID), so evaluation_runs.id
-- has no server-side default.

CREATE TABLE IF NOT EXISTS evaluation_runs (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id INTEGER NOT NULL,
    dataset_id VARCHAR(255) NOT NULL DEFAULT '',
    status INTEGER NOT NULL DEFAULT 0,
    err_msg TEXT NOT NULL DEFAULT '',
    total INTEGER NOT NULL DEFAULT 0,
    finished INTEGER NOT NULL DEFAULT 0,
    params JSONB,
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_evaluation_runs_tenant
    ON evaluation_runs (tenant_id);

CREATE TABLE IF NOT EXISTS evaluation_metrics (
    id VARCHAR(36) PRIMARY KEY,
    run_id VARCHAR(255) NOT NULL,
    tenant_id INTEGER NOT NULL,
    retrieval_metrics JSONB,
    generation_metrics JSONB,
    cost DOUBLE PRECISION,
    latency_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_evaluation_metrics_run
    ON evaluation_metrics (run_id);
CREATE INDEX IF NOT EXISTS idx_evaluation_metrics_tenant
    ON evaluation_metrics (tenant_id);
