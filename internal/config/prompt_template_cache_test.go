package config

import (
	"strings"
	"testing"
)

// TestPromptTemplatesPrefixCacheOrdering 验证「固定内容前置、可变内容后置」的
// 模板顺序：默认 system prompt 不再内嵌检索结果（{{contexts}}），检索结果改由
// 默认 context template（user message）携带。这样 system prompt 成为字节稳定的
// 前缀，多轮对话能命中厂商前缀缓存。
func TestPromptTemplatesPrefixCacheOrdering(t *testing.T) {
	cfg, err := loadPromptTemplates("../../config")
	if err != nil {
		t.Fatalf("loadPromptTemplates: %v", err)
	}
	if cfg == nil {
		t.Fatal("expected templates to load")
	}

	sys := DefaultTemplate(cfg.SystemPrompt)
	if sys == nil {
		t.Fatal("no default system prompt")
	}
	if strings.Contains(sys.Content, "{{contexts}}") {
		t.Fatalf("default system prompt %q still embeds retrieved contexts (breaks prefix cache)", sys.ID)
	}

	ctxTpl := DefaultTemplate(cfg.ContextTemplate)
	if ctxTpl == nil {
		t.Fatal("no default context template")
	}
	if !strings.Contains(ctxTpl.Content, "{{contexts}}") {
		t.Fatalf("default context template %q should carry retrieved contexts", ctxTpl.ID)
	}
}
