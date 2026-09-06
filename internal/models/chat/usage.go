package chat

import (
	"context"
	"strings"

	"github.com/Tencent/WeKnora/internal/logger"
	"github.com/Tencent/WeKnora/internal/types"
)

// UsageRecorder is an optional hook that persists each model call's usage to
// storage. The container injects it at startup; the chat package itself has no
// storage dependency. Calls fire asynchronously so accounting never blocks the
// request path.
var UsageRecorder func(ctx context.Context, record types.ModelUsageRecord)

// logUsage emits the standard "[LLM Usage]" line shared by every Chat
// implementation. It is a no-op when usage is nil so callers can pass through
// optional usage blocks without guarding at each call site.
func logUsage(ctx context.Context, model string, u *types.TokenUsage) {
	if u == nil {
		return
	}
	// Fold this call into the caller's usage accumulator (if any) so that
	// evaluation and other orchestrators can sum token usage across a whole
	// pipeline run. The log line below is unchanged.
	if acc := types.UsageAccumulatorFromContext(ctx); acc != nil {
		acc.Add(*u)
	}
	purpose, prefixFingerprint := types.LLMCallMetadataFromContext(ctx)
	if UsageRecorder != nil {
		tenantID, _ := types.TenantIDFromContext(ctx)
		record := types.ModelUsageRecord{
			TenantID:         tenantID,
			ModelID:          model,
			Purpose:          purpose,
			PromptTokens:     u.PromptTokens,
			CompletionTokens: u.CompletionTokens,
			TotalTokens:      u.TotalTokens,
			CachedTokens:     u.CachedTokens,
			CacheReadTokens:  u.CacheReadTokens,
			Cost:             types.ComputeTokenCost(u.PromptTokens, u.CompletionTokens),
		}
		go func() {
			UsageRecorder(logger.CloneContext(ctx), record)
		}()
	}
	logger.Infof(ctx,
		"[LLM Usage] model=%s, purpose=%s, prompt_prefix=%s, prompt_tokens=%d, completion_tokens=%d, "+
			"total_tokens=%d, cached_tokens=%d, cache_read_tokens=%d, cache_write_tokens=%d, "+
			"cache_miss_tokens=%d, cache_reported=%t, cache_status=%s%s",
		model, purpose, prefixFingerprint, u.PromptTokens, u.CompletionTokens, u.TotalTokens,
		u.CachedTokens, u.CacheReadTokens, u.CacheWriteTokens, u.CacheMissTokens,
		u.CacheReported, u.CacheStatus, usageAttribution(ctx))
}

// usageAttribution renders the ", session_id=…, principal=…" suffix that
// attributes a usage line to the session and terminal principal that
// triggered the call. Calls that run outside a session or without a resolved
// principal (document parsing, title generation, background jobs) render an
// empty suffix, keeping their lines byte-identical to before.
func usageAttribution(ctx context.Context) string {
	var b strings.Builder
	if sessionID, ok := types.SessionIDFromContext(ctx); ok && sessionID != "" {
		b.WriteString(", session_id=")
		b.WriteString(sessionID)
	}
	if principal, ok := types.PrincipalFromContext(ctx); ok {
		b.WriteString(", principal=")
		b.WriteString(principal.Type)
		b.WriteString(":")
		b.WriteString(principal.ID)
	}
	return b.String()
}
