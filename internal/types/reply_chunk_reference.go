package types

import "time"

// ReplyChunkReference records the association between an AI-generated answer
// message and a knowledge base chunk it cited.
//
// It is the persistence layer for issue #1248, task 2:
// "AI 生成问答回复时，自动记录该回复所引用的所有知识库片段 ID，
// 并持久化存储'问答回复-知识库片段'关联关系".
//
// The cited chunk list comes from the assistant message's
// KnowledgeReferences (type References == []*SearchResult), each element
// carrying the chunk ID (SearchResult.ID) plus its KnowledgeID / KnowledgeBaseID.
type ReplyChunkReference struct {
	// ID is the primary key (UUID).
	ID string `json:"id" gorm:"type:varchar(36);primaryKey"`
	// TenantID enables multi-tenant isolation.
	TenantID uint64 `json:"tenant_id"`
	// MessageID is the AI answer message that cited the chunk.
	MessageID string `json:"message_id" gorm:"type:varchar(36);index"`
	// ChunkID is the cited knowledge base chunk (== SearchResult.ID).
	ChunkID string `json:"chunk_id" gorm:"type:varchar(36);index"`
	// KnowledgeID is the knowledge document the chunk belongs to.
	KnowledgeID string `json:"knowledge_id"`
	// KnowledgeBaseID is the knowledge base the chunk belongs to.
	KnowledgeBaseID string `json:"knowledge_base_id"`
	// CreatedAt is the association creation time.
	CreatedAt time.Time `json:"created_at"`
}

// TableName overrides the default GORM table name.
func (ReplyChunkReference) TableName() string {
	return "reply_chunk_references"
}
