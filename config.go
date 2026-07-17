package main

import (
	"fmt"
	"net/url"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	defaultBaseURL       = "https://chatgpt.com/backend-api/codex"
	defaultClientVersion = "0.144.1"
	defaultUserAgent     = "codex_cli_rs/0.144.1 (Mac OS 26.3.1; arm64) iTerm.app/3.6.9"
	defaultOriginator    = "codex_cli_rs"
)

type pluginConfig struct {
	BaseURL       string `yaml:"base_url"`
	ClientVersion string `yaml:"client_version"`
	UserAgent     string `yaml:"user_agent"`
	Originator    string `yaml:"originator"`
}

func defaultConfig() pluginConfig {
	return pluginConfig{
		BaseURL:       defaultBaseURL,
		ClientVersion: defaultClientVersion,
		UserAgent:     defaultUserAgent,
		Originator:    defaultOriginator,
	}
}

func decodeConfig(raw []byte) (pluginConfig, error) {
	cfg := pluginConfig{}
	if len(raw) > 0 {
		if errUnmarshal := yaml.Unmarshal(raw, &cfg); errUnmarshal != nil {
			return pluginConfig{}, fmt.Errorf("decode plugin configuration: %w", errUnmarshal)
		}
	}
	cfg.BaseURL = strings.TrimSpace(cfg.BaseURL)
	cfg.ClientVersion = strings.TrimSpace(cfg.ClientVersion)
	cfg.UserAgent = strings.TrimSpace(cfg.UserAgent)
	cfg.Originator = strings.TrimSpace(cfg.Originator)
	defaults := defaultConfig()
	if cfg.BaseURL == "" {
		cfg.BaseURL = defaults.BaseURL
	}
	if cfg.ClientVersion == "" {
		cfg.ClientVersion = defaults.ClientVersion
	}
	if cfg.UserAgent == "" {
		cfg.UserAgent = defaults.UserAgent
	}
	if cfg.Originator == "" {
		cfg.Originator = defaults.Originator
	}
	if _, errURL := validatedBaseURL(cfg.BaseURL); errURL != nil {
		return pluginConfig{}, fmt.Errorf("invalid base_url: %w", errURL)
	}
	return cfg, nil
}

func validatedBaseURL(raw string) (*url.URL, error) {
	parsed, errParse := url.Parse(strings.TrimSpace(raw))
	if errParse != nil {
		return nil, errParse
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("scheme must be http or https")
	}
	if strings.TrimSpace(parsed.Host) == "" {
		return nil, fmt.Errorf("host is required")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("userinfo is not allowed")
	}
	return parsed, nil
}
