package main

import (
	"encoding/json"
	"os"
	"testing"
)

func TestCommunityRegistryMetadata(t *testing.T) {
	raw, errRead := os.ReadFile("registry.json")
	if errRead != nil {
		t.Fatalf("read registry.json: %v", errRead)
	}
	var registry struct {
		SchemaVersion int `json:"schema_version"`
		Plugins       []struct {
			ID         string `json:"id"`
			Version    string `json:"version"`
			Repository string `json:"repository"`
			Install    struct {
				Type string `json:"type"`
			} `json:"install"`
		} `json:"plugins"`
	}
	if errUnmarshal := json.Unmarshal(raw, &registry); errUnmarshal != nil {
		t.Fatalf("decode registry.json: %v", errUnmarshal)
	}
	if registry.SchemaVersion != 1 || len(registry.Plugins) != 1 {
		t.Fatalf("registry = %#v", registry)
	}
	plugin := registry.Plugins[0]
	if plugin.ID != pluginID || plugin.Version != version || plugin.Repository != pluginRegistration().Metadata.GitHubRepository || plugin.Install.Type != "github-release" {
		t.Fatalf("registry plugin = %#v", plugin)
	}
}
