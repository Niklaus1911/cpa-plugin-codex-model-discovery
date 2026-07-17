package main

import "testing"

func TestDecodeConfigDefaults(t *testing.T) {
	cfg, errDecode := decodeConfig(nil)
	if errDecode != nil {
		t.Fatalf("decodeConfig() error = %v", errDecode)
	}
	if cfg != defaultConfig() {
		t.Fatalf("decodeConfig() = %#v, want %#v", cfg, defaultConfig())
	}
}

func TestDecodeConfigOverrides(t *testing.T) {
	cfg, errDecode := decodeConfig([]byte(`
base_url: https://example.test/codex/
client_version: "9.9.9"
user_agent: test-agent
originator: test-origin
`))
	if errDecode != nil {
		t.Fatalf("decodeConfig() error = %v", errDecode)
	}
	if cfg.BaseURL != "https://example.test/codex/" || cfg.ClientVersion != "9.9.9" || cfg.UserAgent != "test-agent" || cfg.Originator != "test-origin" {
		t.Fatalf("decodeConfig() = %#v", cfg)
	}
}

func TestDecodeConfigRejectsUnsafeBaseURL(t *testing.T) {
	for _, raw := range []string{"base_url: file:///tmp/catalog", "base_url: example.test/codex", "base_url: https://user:secret@example.test/codex"} {
		if _, errDecode := decodeConfig([]byte(raw)); errDecode == nil {
			t.Fatalf("decodeConfig(%q) error = nil", raw)
		}
	}
}
