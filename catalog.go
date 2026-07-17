package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

const (
	codexImage15ModelID = "gpt-image-1.5"
	codexImage2ModelID  = "gpt-image-2"
)

type modelCatalogReasoningLevel struct {
	Effort      string `json:"effort"`
	Description string `json:"description"`
}

type modelCatalogEntry struct {
	Slug                     string                       `json:"slug"`
	DisplayName              string                       `json:"display_name"`
	Description              string                       `json:"description"`
	ContextWindow            int64                        `json:"context_window"`
	MaxContextWindow         int64                        `json:"max_context_window"`
	SupportedReasoningLevels []modelCatalogReasoningLevel `json:"supported_reasoning_levels"`
}

func modelsURL(baseURL, clientVersion string) (string, error) {
	parsed, errURL := validatedBaseURL(baseURL)
	if errURL != nil {
		return "", errURL
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/models"
	if clientVersion = strings.TrimSpace(clientVersion); clientVersion != "" {
		query := parsed.Query()
		query.Set("client_version", clientVersion)
		parsed.RawQuery = query.Encode()
	}
	return parsed.String(), nil
}

func parseModelsCatalog(raw []byte) ([]modelCatalogEntry, error) {
	var payload struct {
		Models []modelCatalogEntry `json:"models"`
	}
	if errUnmarshal := json.Unmarshal(raw, &payload); errUnmarshal != nil {
		return nil, fmt.Errorf("decode catalog: %w", errUnmarshal)
	}
	if len(payload.Models) == 0 {
		return nil, fmt.Errorf("catalog has no models")
	}

	models := make([]modelCatalogEntry, 0, len(payload.Models))
	seen := make(map[string]struct{}, len(payload.Models))
	for index := range payload.Models {
		model := payload.Models[index]
		model.Slug = strings.TrimSpace(model.Slug)
		if model.Slug == "" {
			return nil, fmt.Errorf("catalog models[%d] has empty slug", index)
		}
		key := strings.ToLower(model.Slug)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		model.DisplayName = strings.TrimSpace(model.DisplayName)
		model.Description = strings.TrimSpace(model.Description)
		model.SupportedReasoningLevels = normalizedReasoningLevels(model.SupportedReasoningLevels)
		models = append(models, model)
	}
	return models, nil
}

func normalizedReasoningLevels(levels []modelCatalogReasoningLevel) []modelCatalogReasoningLevel {
	normalized := make([]modelCatalogReasoningLevel, 0, len(levels))
	seen := make(map[string]struct{}, len(levels))
	for _, level := range levels {
		level.Effort = strings.ToLower(strings.TrimSpace(level.Effort))
		if level.Effort == "" {
			continue
		}
		if _, exists := seen[level.Effort]; exists {
			continue
		}
		seen[level.Effort] = struct{}{}
		level.Description = strings.TrimSpace(level.Description)
		normalized = append(normalized, level)
	}
	return normalized
}

func catalogModelInfos(entries []modelCatalogEntry) []pluginapi.ModelInfo {
	models := make([]pluginapi.ModelInfo, 0, len(entries)+2)
	for _, entry := range entries {
		displayName := strings.TrimSpace(entry.DisplayName)
		if displayName == "" {
			displayName = entry.Slug
		}
		description := strings.TrimSpace(entry.Description)
		if description == "" {
			description = displayName
		}
		contextLength := entry.ContextWindow
		if contextLength <= 0 {
			contextLength = entry.MaxContextWindow
		}
		model := pluginapi.ModelInfo{
			ID:                  entry.Slug,
			Object:              "model",
			OwnedBy:             "openai",
			Type:                "openai",
			DisplayName:         displayName,
			Version:             entry.Slug,
			Description:         description,
			ContextLength:       contextLength,
			SupportedParameters: []string{"tools"},
		}
		levels := make([]string, 0, len(entry.SupportedReasoningLevels))
		for _, level := range entry.SupportedReasoningLevels {
			levels = append(levels, level.Effort)
		}
		if len(levels) > 0 {
			model.Thinking = &pluginapi.ThinkingSupport{Levels: levels}
		}
		models = append(models, model)
	}
	return withCodexImageModels(models)
}

func withCodexImageModels(models []pluginapi.ModelInfo) []pluginapi.ModelInfo {
	builtins := []pluginapi.ModelInfo{
		{ID: codexImage15ModelID, Object: "model", Created: 1704067200, OwnedBy: "openai", Type: "openai", DisplayName: "GPT Image 1.5", Version: codexImage15ModelID},
		{ID: codexImage2ModelID, Object: "model", Created: 1704067200, OwnedBy: "openai", Type: "openai", DisplayName: "GPT Image 2", Version: codexImage2ModelID},
	}
	for _, builtin := range builtins {
		replaced := false
		for index := range models {
			if strings.EqualFold(models[index].ID, builtin.ID) {
				models[index] = builtin
				replaced = true
				break
			}
		}
		if !replaced {
			models = append(models, builtin)
		}
	}
	return models
}

func cloneModelInfos(models []pluginapi.ModelInfo) []pluginapi.ModelInfo {
	if len(models) == 0 {
		return nil
	}
	cloned := make([]pluginapi.ModelInfo, len(models))
	copy(cloned, models)
	for index := range cloned {
		cloned[index].SupportedGenerationMethods = append([]string(nil), models[index].SupportedGenerationMethods...)
		cloned[index].SupportedParameters = append([]string(nil), models[index].SupportedParameters...)
		cloned[index].SupportedInputModalities = append([]string(nil), models[index].SupportedInputModalities...)
		cloned[index].SupportedOutputModalities = append([]string(nil), models[index].SupportedOutputModalities...)
		if models[index].Thinking != nil {
			thinking := *models[index].Thinking
			thinking.Levels = append([]string(nil), models[index].Thinking.Levels...)
			cloned[index].Thinking = &thinking
		}
	}
	return cloned
}
