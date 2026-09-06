package interfaces

import (
	"context"
	"time"

	"github.com/Tencent/WeKnora/internal/types"
)

// ModelUsageRepository is the storage contract for per-call model usage.
type ModelUsageRepository interface {
	// CreateUsageRecord inserts one model call's usage row.
	CreateUsageRecord(ctx context.Context, record *types.ModelUsageRecord) error
	// AggregateUsage returns per-model aggregated usage over a time window.
	// An empty modelID aggregates across all models; a zero start/end means no
	// lower/upper bound.
	AggregateUsage(
		ctx context.Context, tenantID uint64, modelID string, start, end time.Time,
	) ([]*types.ModelUsageAggregate, error)
}
