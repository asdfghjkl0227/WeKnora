package repository

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// modelUsageRepository persists per-call model usage records.
type modelUsageRepository struct {
	db *gorm.DB
}

// NewModelUsageRepository creates the model usage repository.
func NewModelUsageRepository(db *gorm.DB) interfaces.ModelUsageRepository {
	return &modelUsageRepository{db: db}
}

// CreateUsageRecord inserts one model call's usage row.
func (r *modelUsageRepository) CreateUsageRecord(
	ctx context.Context, record *types.ModelUsageRecord,
) error {
	if record == nil {
		return nil
	}
	if record.ID == "" {
		record.ID = uuid.New().String()
	}
	return r.db.WithContext(ctx).Create(record).Error
}

// usageRow is the intermediate scan target for the aggregate query.
type usageRow struct {
	ModelID          string
	CallCount        int64
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
	CachedTokens     int64
	Cost             float64
}

// AggregateUsage returns per-model aggregated usage over a time window.
func (r *modelUsageRepository) AggregateUsage(
	ctx context.Context, tenantID uint64, modelID string, start, end time.Time,
) ([]*types.ModelUsageAggregate, error) {
	query := r.db.WithContext(ctx).Model(&types.ModelUsageRecord{}).
		Where("tenant_id = ?", tenantID)
	if modelID != "" {
		query = query.Where("model_id = ?", modelID)
	}
	if !start.IsZero() {
		query = query.Where("created_at >= ?", start)
	}
	if !end.IsZero() {
		query = query.Where("created_at <= ?", end)
	}

	var rows []usageRow
	err := query.
		Select(`model_id,
			COUNT(*) AS call_count,
			COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens,
			COALESCE(SUM(completion_tokens), 0) AS completion_tokens,
			COALESCE(SUM(total_tokens), 0) AS total_tokens,
			COALESCE(SUM(cached_tokens), 0) AS cached_tokens,
			COALESCE(SUM(cost), 0) AS cost`).
		Group("model_id").
		Order("cost DESC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	aggregates := make([]*types.ModelUsageAggregate, 0, len(rows))
	for _, row := range rows {
		hitRate := 0.0
		if row.PromptTokens > 0 {
			hitRate = float64(row.CachedTokens) / float64(row.PromptTokens)
		}
		aggregates = append(aggregates, &types.ModelUsageAggregate{
			ModelID:          row.ModelID,
			CallCount:        row.CallCount,
			PromptTokens:     row.PromptTokens,
			CompletionTokens: row.CompletionTokens,
			TotalTokens:      row.TotalTokens,
			CachedTokens:     row.CachedTokens,
			Cost:             row.Cost,
			CacheHitRate:     hitRate,
		})
	}
	return aggregates, nil
}
