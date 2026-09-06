package types

import "time"

// ModelUsageRecord is one persisted row of a single model call's token usage
// and estimated cost. Rows are written asynchronously from the chat/embedding
// layer so they never block the request path.
type ModelUsageRecord struct {
	ID               string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	TenantID         uint64    `json:"tenant_id" gorm:"column:tenant_id;not null"`
	ModelID          string    `json:"model_id" gorm:"column:model_id;type:varchar(255);not null"`
	Purpose          string    `json:"purpose" gorm:"column:purpose;type:varchar(64)"`
	PromptTokens     int       `json:"prompt_tokens" gorm:"column:prompt_tokens;not null;default:0"`
	CompletionTokens int       `json:"completion_tokens" gorm:"column:completion_tokens;not null;default:0"`
	TotalTokens      int       `json:"total_tokens" gorm:"column:total_tokens;not null;default:0"`
	CachedTokens     int       `json:"cached_tokens" gorm:"column:cached_tokens;not null;default:0"`
	CacheReadTokens  int       `json:"cache_read_tokens" gorm:"column:cache_read_tokens;not null;default:0"`
	Cost             float64   `json:"cost" gorm:"column:cost;not null;default:0"`
	CreatedAt        time.Time `json:"created_at"`
}

// TableName returns the database table name for ModelUsageRecord.
func (ModelUsageRecord) TableName() string { return "model_usage_records" }

// ModelUsageAggregate is the aggregated usage of one model over a time window,
// returned by the model usage query.
type ModelUsageAggregate struct {
	ModelID          string  `json:"model_id"`
	CallCount        int64   `json:"call_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	CachedTokens     int64   `json:"cached_tokens"`
	Cost             float64 `json:"cost"`
	// CacheHitRate is cached_tokens / prompt_tokens (0 when prompt_tokens is 0).
	CacheHitRate float64 `json:"cache_hit_rate"`
}
