package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// evaluationRepository persists evaluation runs and their metric results.
// It backs the EvaluationService so results survive a process restart,
// replacing the previous in-memory storage.
type evaluationRepository struct {
	db *gorm.DB
}

// NewEvaluationRepository creates the evaluation repository.
func NewEvaluationRepository(db *gorm.DB) interfaces.EvaluationRepository {
	return &evaluationRepository{db: db}
}

// SaveRun upserts the full evaluation snapshot: the task row in
// evaluation_runs plus (when metrics are present) the metric row in
// evaluation_metrics. It is safe to call repeatedly as the run progresses.
func (r *evaluationRepository) SaveRun(ctx context.Context, detail *types.EvaluationDetail) error {
	if detail == nil || detail.Task == nil {
		return nil
	}

	run := &types.EvaluationRun{
		ID:        detail.Task.ID,
		TenantID:  detail.Task.TenantID,
		DatasetID: detail.Task.DatasetID,
		Status:    int(detail.Task.Status),
		ErrMsg:    detail.Task.ErrMsg,
		Total:     detail.Task.Total,
		Finished:  detail.Task.Finished,
		StartTime: detail.Task.StartTime,
		EndTime:   detail.Task.EndTime,
	}

	// Serialize the config snapshot (ChatManage) so the same run can be
	// reproduced from a clean environment.
	if detail.Params != nil {
		raw, err := json.Marshal(detail.Params)
		if err != nil {
			return err
		}
		run.Params = types.JSON(raw)
	}

	if err := r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"dataset_id", "status", "err_msg", "total", "finished",
				"params", "end_time", "updated_at",
			}),
		}).
		Create(run).Error; err != nil {
		return err
	}

	if detail.Metric == nil {
		return nil
	}

	metric := &types.EvaluationMetric{
		ID:        uuid.New().String(),
		RunID:     detail.Task.ID,
		TenantID:  detail.Task.TenantID,
		Cost:      detail.Cost,
		LatencyMS: detail.LatencyMS,
	}
	retrievalRaw, err := json.Marshal(detail.Metric.RetrievalMetrics)
	if err != nil {
		return err
	}
	generationRaw, err := json.Marshal(detail.Metric.GenerationMetrics)
	if err != nil {
		return err
	}
	metric.RetrievalMetrics = types.JSON(retrievalRaw)
	metric.GenerationMetrics = types.JSON(generationRaw)

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "run_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"retrieval_metrics", "generation_metrics", "cost", "latency_ms", "updated_at",
			}),
		}).
		Create(metric).Error
}

// GetRun reads the snapshot for one task and reassembles it into an
// EvaluationDetail. Returns (nil, nil) when the task does not exist.
func (r *evaluationRepository) GetRun(ctx context.Context, taskID string) (*types.EvaluationDetail, error) {
	var run types.EvaluationRun
	if err := r.db.WithContext(ctx).Where("id = ?", taskID).First(&run).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	detail := &types.EvaluationDetail{
		Task: &types.EvaluationTask{
			ID:        run.ID,
			TenantID:  run.TenantID,
			DatasetID: run.DatasetID,
			StartTime: run.StartTime,
			Status:    types.EvaluationStatue(run.Status),
			ErrMsg:    run.ErrMsg,
			Total:     run.Total,
			Finished:  run.Finished,
			EndTime:   run.EndTime,
		},
	}

	if len(run.Params) > 0 {
		var params types.ChatManage
		if err := json.Unmarshal(run.Params, &params); err == nil {
			detail.Params = &params
		}
	}

	var metric types.EvaluationMetric
	if err := r.db.WithContext(ctx).Where("run_id = ?", taskID).First(&metric).Error; err == nil {
		result := &types.MetricResult{}
		if len(metric.RetrievalMetrics) > 0 {
			_ = json.Unmarshal(metric.RetrievalMetrics, &result.RetrievalMetrics)
		}
		if len(metric.GenerationMetrics) > 0 {
			_ = json.Unmarshal(metric.GenerationMetrics, &result.GenerationMetrics)
		}
		detail.Metric = result
		detail.Cost = metric.Cost
		detail.LatencyMS = metric.LatencyMS
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	return detail, nil
}
