package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const maxModelsCatalogSize = 8 << 20

type hostCallFunc func(string, []byte) (json.RawMessage, error)

type catalogRequest struct {
	BaseURL       string
	ClientVersion string
	AccessToken   string
	AccountID     string
	UserAgent     string
	Originator    string
	Headers       map[string]string
}

type discoveryFailure struct {
	category string
	cause    error
}

func (e *discoveryFailure) Error() string {
	if e == nil || e.cause == nil {
		return "model discovery failed"
	}
	return e.cause.Error()
}

func fetchModelsCatalog(call hostCallFunc, callbackID string, request catalogRequest) ([]modelCatalogEntry, error) {
	if call == nil {
		return nil, &discoveryFailure{category: "host_unavailable", cause: fmt.Errorf("host callback is unavailable")}
	}
	endpoint, errURL := modelsURL(request.BaseURL, request.ClientVersion)
	if errURL != nil {
		return nil, &discoveryFailure{category: "invalid_url", cause: errURL}
	}

	headers := make(http.Header)
	headers.Set("Accept", "application/json")
	headers.Set("Authorization", "Bearer "+strings.TrimSpace(request.AccessToken))
	headers.Set("Originator", strings.TrimSpace(request.Originator))
	headers.Set("User-Agent", strings.TrimSpace(request.UserAgent))
	if accountID := strings.TrimSpace(request.AccountID); accountID != "" {
		headers.Set("Chatgpt-Account-Id", accountID)
	}
	for name, value := range request.Headers {
		name = strings.TrimSpace(name)
		value = strings.TrimSpace(value)
		if name != "" && value != "" {
			headers.Set(name, value)
		}
	}

	payload, errMarshal := json.Marshal(hostHTTPRequest{
		HostCallbackID: callbackID,
		Request: hostRequestBody{
			Method:  http.MethodGet,
			URL:     endpoint,
			Headers: headers,
		},
	})
	if errMarshal != nil {
		return nil, &discoveryFailure{category: "request_encoding", cause: errMarshal}
	}
	rawResponse, errOpen := call("host.http.do_stream", payload)
	if errOpen != nil {
		return nil, &discoveryFailure{category: "request_failed", cause: errOpen}
	}
	var response hostHTTPStreamResponse
	if errUnmarshal := json.Unmarshal(rawResponse, &response); errUnmarshal != nil {
		return nil, &discoveryFailure{category: "response_encoding", cause: errUnmarshal}
	}
	if strings.TrimSpace(response.StreamID) == "" {
		return nil, &discoveryFailure{category: "response_stream", cause: fmt.Errorf("host returned no stream")}
	}
	defer closeHostHTTPStream(call, response.StreamID)

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, &discoveryFailure{
			category: fmt.Sprintf("upstream_status_%d", response.StatusCode),
			cause:    fmt.Errorf("Codex models request failed with status %d", response.StatusCode),
		}
	}
	body, errRead := readHostHTTPStream(call, response.StreamID)
	if errRead != nil {
		return nil, errRead
	}
	models, errParse := parseModelsCatalog(body)
	if errParse != nil {
		return nil, &discoveryFailure{category: "invalid_catalog", cause: errParse}
	}
	return models, nil
}

func readHostHTTPStream(call hostCallFunc, streamID string) ([]byte, error) {
	body := make([]byte, 0, 64*1024)
	for {
		payload, _ := json.Marshal(hostHTTPStreamReadRequest{StreamID: streamID})
		rawChunk, errRead := call("host.http.stream_read", payload)
		if errRead != nil {
			return nil, &discoveryFailure{category: "stream_read", cause: errRead}
		}
		var chunk hostHTTPStreamReadResponse
		if errUnmarshal := json.Unmarshal(rawChunk, &chunk); errUnmarshal != nil {
			return nil, &discoveryFailure{category: "stream_encoding", cause: errUnmarshal}
		}
		if chunk.Error != "" {
			return nil, &discoveryFailure{category: "stream_read", cause: fmt.Errorf("host stream read failed")}
		}
		if len(body)+len(chunk.Payload) > maxModelsCatalogSize {
			return nil, &discoveryFailure{
				category: "response_too_large",
				cause:    fmt.Errorf("Codex models response exceeded %d bytes", maxModelsCatalogSize),
			}
		}
		body = append(body, chunk.Payload...)
		if chunk.Done {
			return body, nil
		}
	}
}

func closeHostHTTPStream(call hostCallFunc, streamID string) {
	payload, _ := json.Marshal(hostHTTPStreamCloseRequest{StreamID: streamID})
	_, _ = call("host.http.stream_close", payload)
}
