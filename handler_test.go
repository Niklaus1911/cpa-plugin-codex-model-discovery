package main

import (
	"encoding/json"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestPluginRegistrationDeclaresOAuthModelProvider(t *testing.T) {
	registration := pluginRegistration()
	if registration.SchemaVersion != abiVersion || registration.Metadata.Version != version {
		t.Fatalf("registration = %#v", registration)
	}
	if !registration.Capabilities.ModelProvider || !registration.Capabilities.Executor {
		t.Fatalf("capabilities = %#v", registration.Capabilities)
	}
	if registration.Capabilities.ExecutorModelScope != pluginapi.ExecutorModelScopeOAuth {
		t.Fatalf("executor scope = %q", registration.Capabilities.ExecutorModelScope)
	}
}

func TestHandleMethodConfiguresAndIdentifiesCodex(t *testing.T) {
	previous := globalRuntime
	globalRuntime = newPluginRuntime(nil)
	defer func() { globalRuntime = previous }()

	lifecycleRaw, _ := json.Marshal(lifecycleRequest{ConfigYAML: []byte("client_version: 9.9.9\n")})
	raw, ok := handleMethod("plugin.register", lifecycleRaw)
	if !ok {
		t.Fatalf("plugin.register failed: %s", raw)
	}
	globalRuntime.mu.Lock()
	clientVersion := globalRuntime.config.ClientVersion
	globalRuntime.mu.Unlock()
	if clientVersion != "9.9.9" {
		t.Fatalf("client version = %q", clientVersion)
	}

	raw, ok = handleMethod("executor.identifier", nil)
	if !ok || !contains(string(raw), providerCodex) {
		t.Fatalf("executor.identifier = %s, %v", raw, ok)
	}
	if raw, ok = handleMethod("executor.execute", nil); ok || !contains(string(raw), "native_executor_required") {
		t.Fatalf("executor.execute = %s, %v", raw, ok)
	}
}
