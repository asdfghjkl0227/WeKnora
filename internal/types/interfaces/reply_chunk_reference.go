package interfaces

import (
	"context"

	"github.com/Tencent/WeKnora/internal/types"
)

// ReplyChunkReferenceRepository persists the association between an AI-generated
// answer message and the knowledge base chunks it cited (issue #1248, task 2).
type ReplyChunkReferenceRepository interface {
	// CreateBatch persists a batch of reply-chunk reference rows.
	CreateBatch(ctx context.Context, refs []*types.ReplyChunkReference) error
}
