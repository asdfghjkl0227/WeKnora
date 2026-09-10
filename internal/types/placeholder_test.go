package types

import (
	"regexp"
	"testing"
)

// TestRenderPromptPlaceholdersCurrentTimeMinutePrecision 验证 {{current_time}}
// 的自动填充是「分钟精度」（不含秒）。秒级时间戳会让含它的 system prompt
// 每秒都变、彻底破坏厂商前缀缓存；降到分钟可让同一分钟内的请求共享稳定前缀。
func TestRenderPromptPlaceholdersCurrentTimeMinutePrecision(t *testing.T) {
	result := RenderPromptPlaceholders(
		"Current time: {{current_time}}",
		PlaceholderValues{"language": "Chinese"},
	)
	// 分钟精度：YYYY-MM-DD HH:MM，结尾没有 :SS
	re := regexp.MustCompile(`^Current time: \d{4}-\d{2}-\d{2} \d{2}:\d{2}$`)
	if !re.MatchString(result) {
		t.Fatalf("current_time should be minute precision (no seconds), got %q", result)
	}
}

// TestRenderPromptPlaceholdersCurrentWeekStillFilled 验证降频不影响其它自动填充项。
func TestRenderPromptPlaceholdersCurrentWeekStillFilled(t *testing.T) {
	result := RenderPromptPlaceholders(
		"Week: {{current_week}}",
		PlaceholderValues{"language": "Chinese"},
	)
	if result == "Week: " || result == "Week: {{current_week}}" {
		t.Fatalf("current_week should be auto-filled, got %q", result)
	}
}
