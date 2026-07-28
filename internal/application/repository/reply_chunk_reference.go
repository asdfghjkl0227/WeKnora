package repository

import (
	"context"

	"gorm.io/gorm"

	"github.com/Tencent/WeKnora/internal/types"
	"github.com/Tencent/WeKnora/internal/types/interfaces"
)

// replyChunkReferenceRepository implements the reply chunk reference repository interface.
// It persists the association between an assistant reply (message) and the knowledge
// chunks it cited, enabling downstream feedback analysis (like/dislike) without
// leaking the chunk-association internals to end users.
type replyChunkReferenceRepository struct {
	db *gorm.DB
}

// NewReplyChunkReferenceRepository creates a new reply chunk reference repository.
func NewReplyChunkReferenceRepository(db *gorm.DB) interfaces.ReplyChunkReferenceRepository {
	return &replyChunkReferenceRepository{
		db: db,
	}
}

// CreateBatch inserts a batch of reply chunk references in a single statement group.
func (r *replyChunkReferenceRepository) CreateBatch(ctx context.Context, refs []*types.ReplyChunkReference) error {
	if len(refs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(refs, 100).Error
}
