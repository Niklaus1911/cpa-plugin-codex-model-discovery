package main

import (
	"encoding/hex"
	"encoding/json"
	"net/url"
	"os"
	"strings"
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
				Type      string `json:"type"`
				Artifacts []struct {
					GOOS   string `json:"goos"`
					GOARCH string `json:"goarch"`
					URL    string `json:"url"`
					SHA256 string `json:"sha256"`
					Size   int64  `json:"size"`
				} `json:"artifacts"`
			} `json:"install"`
		} `json:"plugins"`
	}
	if errUnmarshal := json.Unmarshal(raw, &registry); errUnmarshal != nil {
		t.Fatalf("decode registry.json: %v", errUnmarshal)
	}
	if registry.SchemaVersion != 2 || len(registry.Plugins) != 1 {
		t.Fatalf("registry = %#v", registry)
	}
	plugin := registry.Plugins[0]
	if plugin.ID != pluginID || plugin.Version != version || plugin.Repository != pluginRegistration().Metadata.GitHubRepository || plugin.Install.Type != "direct" || len(plugin.Install.Artifacts) != 4 {
		t.Fatalf("registry plugin = %#v", plugin)
	}

	expectedPlatforms := map[string]bool{
		"darwin/arm64":  false,
		"linux/amd64":   false,
		"linux/arm64":   false,
		"windows/amd64": false,
	}
	for _, artifact := range plugin.Install.Artifacts {
		platform := artifact.GOOS + "/" + artifact.GOARCH
		seen, expected := expectedPlatforms[platform]
		parsedURL, errParse := url.Parse(artifact.URL)
		_, errSHA := hex.DecodeString(artifact.SHA256)
		if !expected || seen || errParse != nil || parsedURL.Scheme != "https" || parsedURL.Host != "github.com" ||
			!strings.HasPrefix(artifact.URL, plugin.Repository+"/releases/download/v"+version+"/") ||
			len(artifact.SHA256) != 64 || errSHA != nil || artifact.Size <= 0 {
			t.Errorf("invalid registry artifact: %#v", artifact)
			continue
		}
		expectedPlatforms[platform] = true
	}
}
