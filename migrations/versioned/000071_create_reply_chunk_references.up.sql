CREATE TABLE IF NOT EXISTS reply_chunk_references (
    id VARCHAR(36) NOT NULL PRIMARY KEY,
    tenant_id BIGINT NOT NULL,
    message_id VARCHAR(36) NOT NULL,
    chunk_id VARCHAR(36) NOT NULL,
    knowledge_id VARCHAR(36) NOT NULL DEFAULT '',
    knowledge_base_id VARCHAR(36) NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reply_chunk_references_message ON reply_chunk_references(message_id);
CREATE INDEX IF NOT EXISTS idx_reply_chunk_references_chunk ON reply_chunk_references(chunk_id);
CREATE INDEX IF NOT EXISTS idx_reply_chunk_references_tenant ON reply_chunk_references(tenant_id);
