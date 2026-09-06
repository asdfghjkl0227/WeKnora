package types

// Default per-million-token prices (USD) used to estimate the cost of a model
// call when the model has no explicit pricing configured. Task 4 will
// introduce per-model pricing; until then these defaults give evaluation runs
// a consistent, reproducible cost figure.
const (
	// DefaultPromptPricePerMillionTokens is the price of 1M input tokens.
	DefaultPromptPricePerMillionTokens = 2.0
	// DefaultCompletionPricePerMillionTokens is the price of 1M output tokens.
	DefaultCompletionPricePerMillionTokens = 6.0
)

// ComputeTokenCost estimates the cost (USD) of a call from its input/output
// token counts using the default prices.
func ComputeTokenCost(promptTokens, completionTokens int) float64 {
	return float64(promptTokens)/1_000_000*DefaultPromptPricePerMillionTokens +
		float64(completionTokens)/1_000_000*DefaultCompletionPricePerMillionTokens
}
