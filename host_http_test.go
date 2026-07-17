package main

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
)

func TestFetchModelsCatalogUsesHostStreamAndClosesIt(t *testing.T) {
	var opened hostHTTPRequest
	readCalls := 0
	closed := false
	call := func(method string, payload []byte) (json.RawMessage, error) {
		switch method {
		case "host.http.do_stream":
			if errUnmarshal := json.Unmarshal(payload, &opened); errUnmarshal != nil {
				t.Fatalf("decode open request: %v", errUnmarshal)
			}
			return mustJSON(t, hostHTTPStreamResponse{StatusCode: http.StatusOK, StreamID: "stream-1"}), nil
		case "host.http.stream_read":
			readCalls++
			return mustJSON(t, hostHTTPStreamReadResponse{Payload: []byte(`{"models":[{"slug":"gpt-5.6-sol"}]}`), Done: true}), nil
		case "host.http.stream_close":
			closed = true
			return mustJSON(t, struct{}{}), nil
		default:
			t.Fatalf("unexpected host method %q", method)
			return nil, nil
		}
	}

	models, errFetch := fetchModelsCatalog(call, "callback-1", catalogRequest{
		BaseURL:       defaultBaseURL,
		ClientVersion: defaultClientVersion,
		AccessToken:   "secret-token",
		AccountID:     "account-1",
		UserAgent:     "configured-agent",
		Originator:    "configured-origin",
		Headers:       map[string]string{"X-Custom": "custom", "User-Agent": "account-agent"},
	})
	if errFetch != nil {
		t.Fatalf("fetchModelsCatalog() error = %v", errFetch)
	}
	if len(models) != 1 || models[0].Slug != "gpt-5.6-sol" || readCalls != 1 || !closed {
		t.Fatalf("models/read/closed = %#v/%d/%v", models, readCalls, closed)
	}
	if opened.HostCallbackID != "callback-1" || !strings.Contains(opened.Request.URL, "/models?client_version=0.144.1") {
		t.Fatalf("open request = %#v", opened)
	}
	if opened.Request.Headers.Get("Authorization") != "Bearer secret-token" || opened.Request.Headers.Get("Chatgpt-Account-Id") != "account-1" {
		t.Fatalf("authorization headers = %#v", opened.Request.Headers)
	}
	if opened.Request.Headers.Get("User-Agent") != "account-agent" || opened.Request.Headers.Get("X-Custom") != "custom" {
		t.Fatalf("custom headers = %#v", opened.Request.Headers)
	}
}

func TestFetchModelsCatalogRejectsStatusWithoutReadingBody(t *testing.T) {
	read := false
	closed := false
	call := func(method string, payload []byte) (json.RawMessage, error) {
		switch method {
		case "host.http.do_stream":
			return mustJSON(t, hostHTTPStreamResponse{StatusCode: http.StatusUnauthorized, StreamID: "stream-1"}), nil
		case "host.http.stream_read":
			read = true
			return nil, nil
		case "host.http.stream_close":
			closed = true
			return mustJSON(t, struct{}{}), nil
		}
		return nil, nil
	}
	_, errFetch := fetchModelsCatalog(call, "callback", catalogRequest{BaseURL: defaultBaseURL, AccessToken: "token"})
	if errFetch == nil || failureCategory(errFetch) != "upstream_status_401" || read || !closed {
		t.Fatalf("error/read/closed = %v/%v/%v", errFetch, read, closed)
	}
}

func TestReadHostHTTPStreamEnforcesLimit(t *testing.T) {
	call := func(method string, payload []byte) (json.RawMessage, error) {
		return mustJSON(t, hostHTTPStreamReadResponse{Payload: make([]byte, maxModelsCatalogSize+1), Done: true}), nil
	}
	_, errRead := readHostHTTPStream(call, "stream")
	if errRead == nil || failureCategory(errRead) != "response_too_large" {
		t.Fatalf("readHostHTTPStream() error = %v", errRead)
	}
}

func mustJSON(t *testing.T, value any) json.RawMessage {
	t.Helper()
	raw, errMarshal := json.Marshal(value)
	if errMarshal != nil {
		t.Fatalf("json.Marshal() error = %v", errMarshal)
	}
	return raw
}
