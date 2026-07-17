package main

import (
	"encoding/json"
	"errors"
	gort "runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func codexAuthRequest(token, accountID string) authModelRPCRequest {
	return authModelRPCRequest{
		AuthModelRequest: pluginapi.AuthModelRequest{
			AuthID:       "auth-1",
			AuthProvider: providerCodex,
			Metadata: map[string]any{
				"access_token": token,
				"account_id":   accountID,
			},
		},
		HostCallbackID: "callback-1",
	}
}

func TestModelsForAuthSuccessThenCachedFallback(t *testing.T) {
	runtime := newPluginRuntime(nil)
	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		return []modelCatalogEntry{{Slug: "gpt-5.6-sol"}}, nil
	}
	request := codexAuthRequest("secret-token", "account-1")
	first := runtime.modelsForAuth(request)
	if first.Provider != providerCodex || !hasModel(first.Models, "gpt-5.6-sol") || !hasModel(first.Models, codexImage2ModelID) {
		t.Fatalf("first response = %#v", first)
	}

	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		return nil, &discoveryFailure{category: "request_failed", cause: errors.New("token must not be logged: secret-token")}
	}
	second := runtime.modelsForAuth(request)
	if second.Provider != providerCodex || !hasModel(second.Models, "gpt-5.6-sol") {
		t.Fatalf("cached response = %#v", second)
	}

	otherAccount := codexAuthRequest("other-token", "account-2")
	otherAccount.AuthID = "auth-2"
	third := runtime.modelsForAuth(otherAccount)
	if third.Provider != unhandledProvider || len(third.Models) != 0 {
		t.Fatalf("uncached failure response = %#v", third)
	}
}

func TestModelsForAuthSkipsNonOAuthAndMissingToken(t *testing.T) {
	runtime := newPluginRuntime(nil)
	var calls atomic.Int32
	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		calls.Add(1)
		return nil, nil
	}

	nonOAuth := codexAuthRequest("token", "account")
	nonOAuth.Attributes = map[string]string{"auth_kind": "apikey"}
	missingToken := codexAuthRequest("", "account")
	otherProvider := codexAuthRequest("token", "account")
	otherProvider.AuthProvider = "claude"
	for _, request := range []authModelRPCRequest{nonOAuth, missingToken, otherProvider} {
		if response := runtime.modelsForAuth(request); response.Provider != unhandledProvider {
			t.Fatalf("response = %#v", response)
		}
	}
	if calls.Load() != 0 {
		t.Fatalf("fetch calls = %d, want 0", calls.Load())
	}
}

func TestModelsForAuthCoalescesConcurrentFetches(t *testing.T) {
	runtime := newPluginRuntime(nil)
	started := make(chan struct{})
	release := make(chan struct{})
	var calls atomic.Int32
	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		if calls.Add(1) == 1 {
			close(started)
		}
		<-release
		return []modelCatalogEntry{{Slug: "gpt-5.6-sol"}}, nil
	}

	request := codexAuthRequest("token", "account")
	const workers = 12
	responses := make(chan pluginapi.ModelResponse, workers)
	var group sync.WaitGroup
	for range workers {
		group.Add(1)
		go func() {
			defer group.Done()
			responses <- runtime.modelsForAuth(request)
		}()
	}
	<-started
	deadline := time.Now().Add(time.Second)
	for {
		runtime.mu.Lock()
		waiters := 0
		for _, current := range runtime.inflight {
			waiters += current.waiters
		}
		runtime.mu.Unlock()
		if waiters == workers-1 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("inflight waiters = %d, want %d", waiters, workers-1)
		}
		gort.Gosched()
	}
	close(release)
	group.Wait()
	close(responses)
	if calls.Load() != 1 {
		t.Fatalf("fetch calls = %d, want 1", calls.Load())
	}
	for response := range responses {
		if response.Provider != providerCodex || !hasModel(response.Models, "gpt-5.6-sol") {
			t.Fatalf("response = %#v", response)
		}
	}
}

func TestDiscoveryFailureLogDoesNotContainCredentials(t *testing.T) {
	var logPayload []byte
	runtime := newPluginRuntime(func(method string, payload []byte) (json.RawMessage, error) {
		if method == "host.log" {
			logPayload = append([]byte(nil), payload...)
		}
		return json.RawMessage(`{}`), nil
	})
	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		return nil, &discoveryFailure{category: "request_failed", cause: errors.New("secret-token")}
	}
	runtime.modelsForAuth(codexAuthRequest("secret-token", "account"))
	if len(logPayload) == 0 {
		t.Fatal("host.log was not called")
	}
	if contains(string(logPayload), "secret-token") || contains(string(logPayload), "Authorization") {
		t.Fatalf("log payload leaked credentials: %s", logPayload)
	}
}

func TestConfigureInvalidatesCache(t *testing.T) {
	runtime := newPluginRuntime(nil)
	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		return []modelCatalogEntry{{Slug: "old-model"}}, nil
	}
	request := codexAuthRequest("token", "account")
	if response := runtime.modelsForAuth(request); !hasModel(response.Models, "old-model") {
		t.Fatalf("initial response = %#v", response)
	}
	if errConfigure := runtime.configure([]byte("client_version: 9.9.9\n")); errConfigure != nil {
		t.Fatalf("configure() error = %v", errConfigure)
	}
	runtime.fetch = func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error) {
		return nil, &discoveryFailure{category: "request_failed", cause: errors.New("failed")}
	}
	if response := runtime.modelsForAuth(request); response.Provider != unhandledProvider || len(response.Models) != 0 {
		t.Fatalf("response after reconfigure = %#v", response)
	}
}

func hasModel(models []pluginapi.ModelInfo, id string) bool {
	for _, model := range models {
		if model.ID == id {
			return true
		}
	}
	return false
}
