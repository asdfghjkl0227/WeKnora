package types

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/yanyiwu/gojieba"
)

// Jieba is a global instance of Chinese text segmentation tool
var Jieba *gojieba.Jieba = newJieba()

func newJieba() *gojieba.Jieba {
	dictDir := os.Getenv("JIEBA_DICT_DIR")
	if dictDir == "" {
		return gojieba.NewJieba()
	}

	return gojieba.NewJieba(
		filepath.Join(dictDir, "jieba.dict.utf8"),
		filepath.Join(dictDir, "hmm_model.utf8"),
		filepath.Join(dictDir, "user.dict.utf8"),
		filepath.Join(dictDir, "idf.utf8"),
		filepath.Join(dictDir, "stop_words.utf8"),
	)
}

// EvaluationStatue represents the status of an evaluation task
type EvaluationStatue int

const (
	EvaluationStatuePending EvaluationStatue = iota // Task is waiting to start
	EvaluationStatueRunning                         // Task is in progress
	EvaluationStatueSuccess                         // Task completed successfully
	EvaluationStatueFailed                          // Task failed
)

// EvaluationTask contains information about an evaluation task
type EvaluationTask struct {
	ID        string `json:"id"`         // Unique task ID
	TenantID  uint64 `json:"tenant_id"`  // Tenant/Organization ID
	DatasetID string `json:"dataset_id"` // Dataset ID for evaluation

	StartTime time.Time        `json:"start_time"`         // Task start time
	EndTime   *time.Time       `json:"end_time,omitempty"` // Task end time (set when the run finishes)
	Status    EvaluationStatue `json:"status"`             // Current task status
	ErrMsg    string           `json:"err_msg,omitempty"`  // Error message if failed

	Total    int `json:"total,omitempty"`    // Total items to evaluate
	Finished int `json:"finished,omitempty"` // Completed items count
}

// EvaluationDetail contains detailed evaluation information
type EvaluationDetail struct {
	Task   *EvaluationTask `json:"task"`             // Evaluation task info
	Params *ChatManage     `json:"params"`           // Evaluation parameters
	Metric *MetricResult   `json:"metric,omitempty"` // Evaluation metrics
	// Cost is the total estimated cost (USD) of all model calls in the run.
	Cost float64 `json:"cost,omitempty"`
	// LatencyMS is the total wall-clock time of the evaluation loop in ms.
	LatencyMS int64 `json:"latency_ms,omitempty"`
}

// String returns JSON representation of EvaluationTask
func (e *EvaluationTask) String() string {
	b, _ := json.Marshal(e)
	return string(b)
}

// MetricInput contains input data for metric calculation
type MetricInput struct {
	RetrievalGT  [][]int // Ground truth for retrieval
	RetrievalIDs []int   // Retrieved IDs

	GeneratedTexts string // Generated text for evaluation
	GeneratedGT    string // Ground truth text for comparison
}

// MetricResult contains evaluation metrics
type MetricResult struct {
	RetrievalMetrics  RetrievalMetrics  `json:"retrieval_metrics"`  // Retrieval performance metrics
	GenerationMetrics GenerationMetrics `json:"generation_metrics"` // Text generation quality metrics
}

// RetrievalMetrics contains metrics for retrieval evaluation
type RetrievalMetrics struct {
	Precision float64 `json:"precision"` // Precision score
	Recall    float64 `json:"recall"`    // Recall score

	NDCG3  float64 `json:"ndcg3"`  // Normalized Discounted Cumulative Gain at 3
	NDCG10 float64 `json:"ndcg10"` // Normalized Discounted Cumulative Gain at 10
	MRR    float64 `json:"mrr"`    // Mean Reciprocal Rank
	MAP    float64 `json:"map"`    // Mean Average Precision
}

// GenerationMetrics contains metrics for text generation evaluation
type GenerationMetrics struct {
	BLEU1 float64 `json:"bleu1"` // BLEU-1 score
	BLEU2 float64 `json:"bleu2"` // BLEU-2 score
	BLEU4 float64 `json:"bleu4"` // BLEU-4 score

	ROUGE1 float64 `json:"rouge1"` // ROUGE-1 score
	ROUGE2 float64 `json:"rouge2"` // ROUGE-2 score
	ROUGEL float64 `json:"rougel"` // ROUGE-L score
}

// EvalState represents different stages of evaluation process
type EvalState int

const (
	StateBegin             EvalState = iota // Evaluation started
	StateAfterQaPairs                       // After loading QA pairs
	StateAfterDataset                       // After processing dataset
	StateAfterEmbedding                     // After generating embeddings
	StateAfterVectorSearch                  // After vector search
	StateAfterRerank                        // After reranking
	StateAfterComplete                      // After completion
	StateEnd                                // Evaluation ended
)

// EvaluationRun is the persisted snapshot of one evaluation task. It backs the
// durable evaluation record so a run survives a process restart, replacing the
// previous in-memory storage.
type EvaluationRun struct {
	ID        string `json:"id" gorm:"primaryKey;type:varchar(255)"`
	TenantID  uint64 `json:"tenant_id" gorm:"column:tenant_id;not null"`
	DatasetID string `json:"dataset_id" gorm:"column:dataset_id;type:varchar(255)"`
	Status    int    `json:"status" gorm:"not null;default:0"`
	ErrMsg    string `json:"err_msg" gorm:"column:err_msg;type:text"`
	Total     int    `json:"total" gorm:"not null;default:0"`
	Finished  int    `json:"finished" gorm:"not null;default:0"`
	// Params is the serialized ChatManage config snapshot used to reproduce
	// the run from a clean environment.
	Params    JSON       `json:"params" gorm:"column:params;type:jsonb"`
	StartTime time.Time  `json:"start_time" gorm:"column:start_time"`
	EndTime   *time.Time `json:"end_time" gorm:"column:end_time"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// TableName returns the database table name for EvaluationRun.
func (EvaluationRun) TableName() string { return "evaluation_runs" }

// EvaluationMetric is the persisted metric result of one evaluation run. The
// cost / latency columns are reserved for a later task and are left empty for
// now.
type EvaluationMetric struct {
	ID                string    `json:"id" gorm:"primaryKey;type:varchar(36)"`
	RunID             string    `json:"run_id" gorm:"column:run_id;type:varchar(255);not null;uniqueIndex:idx_evaluation_metrics_run"`
	TenantID          uint64    `json:"tenant_id" gorm:"column:tenant_id;not null"`
	RetrievalMetrics  JSON      `json:"retrieval_metrics" gorm:"column:retrieval_metrics;type:jsonb"`
	GenerationMetrics JSON      `json:"generation_metrics" gorm:"column:generation_metrics;type:jsonb"`
	Cost              float64   `json:"cost" gorm:"column:cost"`
	LatencyMS         int64     `json:"latency_ms" gorm:"column:latency_ms"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// TableName returns the database table name for EvaluationMetric.
func (EvaluationMetric) TableName() string { return "evaluation_metrics" }
