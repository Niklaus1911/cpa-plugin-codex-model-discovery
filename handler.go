package main

import (
	"encoding/json"
	"strings"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func handleMethod(method string, raw []byte) ([]byte, bool) {
	result, rpcErr := dispatchMethod(strings.TrimSpace(method), raw)
	if rpcErr != nil {
		return errorEnvelope(rpcErr.Code, rpcErr.Message), false
	}
	return okEnvelope(result), true
}

func dispatchMethod(method string, raw []byte) (any, *envelopeError) {
	switch method {
	case "plugin.register", "plugin.reconfigure":
		var request lifecycleRequest
		if len(raw) > 0 {
			if errUnmarshal := json.Unmarshal(raw, &request); errUnmarshal != nil {
				return nil, &envelopeError{Code: "invalid_config", Message: "invalid lifecycle request"}
			}
		}
		if errConfigure := globalRuntime.configure(request.ConfigYAML); errConfigure != nil {
			return nil, &envelopeError{Code: "invalid_config", Message: errConfigure.Error()}
		}
		return pluginRegistration(), nil
	case "model.static":
		return unhandledModelResponse(), nil
	case "model.for_auth":
		var request authModelRPCRequest
		if errUnmarshal := json.Unmarshal(raw, &request); errUnmarshal != nil {
			return nil, &envelopeError{Code: "invalid_auth_model_request", Message: "invalid auth model request"}
		}
		return globalRuntime.modelsForAuth(request), nil
	case "executor.identifier":
		return identifierResponse{Identifier: providerCodex}, nil
	case "executor.execute", "executor.execute_stream", "executor.count_tokens", "executor.http_request":
		return nil, &envelopeError{
			Code:    "native_executor_required",
			Message: "codex-model-discovery requires CLIProxyAPI's native Codex executor",
		}
	default:
		return nil, &envelopeError{Code: "unknown_method", Message: "unknown method: " + method}
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: abiVersion,
		Metadata: pluginapi.Metadata{
			Name:             "Codex Model Discovery",
			Version:          version,
			Author:           "Niklaus1911",
			GitHubRepository: "https://github.com/Niklaus1911/cpa-plugin-codex-model-discovery",
			ConfigFields: []pluginapi.ConfigField{
				{Name: "base_url", Type: pluginapi.ConfigFieldTypeString, Description: "Default Codex API base URL; account-specific base_url values take precedence."},
				{Name: "client_version", Type: pluginapi.ConfigFieldTypeString, Description: "Codex client version sent to the authenticated models endpoint."},
				{Name: "user_agent", Type: pluginapi.ConfigFieldTypeString, Description: "User-Agent sent to the authenticated models endpoint."},
				{Name: "originator", Type: pluginapi.ConfigFieldTypeString, Description: "Originator header sent to the authenticated models endpoint."},
			},
		},
		Capabilities: registrationCapabilities{
			ModelProvider:         true,
			Executor:              true,
			ExecutorModelScope:    pluginapi.ExecutorModelScopeOAuth,
			ExecutorInputFormats:  []string{"codex"},
			ExecutorOutputFormats: []string{"codex"},
		},
	}
}
