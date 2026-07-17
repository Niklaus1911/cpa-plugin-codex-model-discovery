package main

import (
	"strings"
	"testing"
)

func TestModelsURLPreservesQueryAndSetsClientVersion(t *testing.T) {
	got, errURL := modelsURL("https://example.test/backend/codex/?gateway=one", "0.144.1")
	if errURL != nil {
		t.Fatalf("modelsURL() error = %v", errURL)
	}
	if got != "https://example.test/backend/codex/models?client_version=0.144.1&gateway=one" {
		t.Fatalf("modelsURL() = %q", got)
	}
}

func TestParseAndConvertModelsCatalog(t *testing.T) {
	entries, errParse := parseModelsCatalog([]byte(`{
  "models": [
    {
      "slug": " gpt-5.6-sol ",
      "display_name": "GPT 5.6 Sol",
      "description": "Fast model",
      "context_window": 400000,
      "supported_reasoning_levels": [
        {"effort":" HIGH ","description":"deep"},
        {"effort":"high"},
        {"effort":"medium"}
      ]
    },
    {"slug":"GPT-5.6-SOL"},
    {"slug":"unknown-model","max_context_window":12345},
    {"slug":"gpt-image-2","display_name":"wrong"}
  ]
}`))
	if errParse != nil {
		t.Fatalf("parseModelsCatalog() error = %v", errParse)
	}
	if len(entries) != 3 {
		t.Fatalf("entries = %d, want 3", len(entries))
	}
	models := catalogModelInfos(entries)
	if len(models) != 4 {
		t.Fatalf("models = %d, want 4", len(models))
	}
	if models[0].ID != "gpt-5.6-sol" || models[0].ContextLength != 400000 || models[0].Thinking == nil {
		t.Fatalf("first model = %#v", models[0])
	}
	if got := strings.Join(models[0].Thinking.Levels, ","); got != "high,medium" {
		t.Fatalf("reasoning levels = %q", got)
	}
	if models[1].DisplayName != "unknown-model" || models[1].Description != "unknown-model" || models[1].ContextLength != 12345 {
		t.Fatalf("unknown model = %#v", models[1])
	}
	if models[2].ID != codexImage2ModelID || models[2].DisplayName != "GPT Image 2" {
		t.Fatalf("image replacement = %#v", models[2])
	}
	if models[3].ID != codexImage15ModelID {
		t.Fatalf("last model = %#v", models[3])
	}
}

func TestParseModelsCatalogRejectsEmptyModels(t *testing.T) {
	for _, raw := range []string{`{"models":[]}`, `{"models":[{"slug":" "}]}`, `{not-json}`} {
		if _, errParse := parseModelsCatalog([]byte(raw)); errParse == nil {
			t.Fatalf("parseModelsCatalog(%q) error = nil", raw)
		}
	}
}

func TestCloneModelInfosIsDeep(t *testing.T) {
	models := catalogModelInfos([]modelCatalogEntry{{
		Slug:                     "model",
		SupportedReasoningLevels: []modelCatalogReasoningLevel{{Effort: "high"}},
	}})
	cloned := cloneModelInfos(models)
	cloned[0].SupportedParameters[0] = "changed"
	cloned[0].Thinking.Levels[0] = "changed"
	if models[0].SupportedParameters[0] == "changed" || models[0].Thinking.Levels[0] == "changed" {
		t.Fatal("cloneModelInfos() shared nested storage")
	}
}
