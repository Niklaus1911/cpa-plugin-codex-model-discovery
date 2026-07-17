package main

import (
	"encoding/json"
	"strconv"
	"strings"
	"sync"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

type catalogFetcher func(hostCallFunc, string, catalogRequest) ([]modelCatalogEntry, error)

type inflightDiscovery struct {
	done    chan struct{}
	waiters int
}

type pluginRuntime struct {
	mu         sync.Mutex
	config     pluginConfig
	generation uint64
	cache      map[string][]pluginapi.ModelInfo
	inflight   map[string]*inflightDiscovery
	call       hostCallFunc
	fetch      catalogFetcher
}

var globalRuntime = newPluginRuntime(callHost)

func newPluginRuntime(call hostCallFunc) *pluginRuntime {
	return &pluginRuntime{
		config:     defaultConfig(),
		generation: 1,
		cache:      make(map[string][]pluginapi.ModelInfo),
		inflight:   make(map[string]*inflightDiscovery),
		call:       call,
		fetch:      fetchModelsCatalog,
	}
}

func (r *pluginRuntime) configure(raw []byte) error {
	cfg, errDecode := decodeConfig(raw)
	if errDecode != nil {
		return errDecode
	}
	r.mu.Lock()
	if r.config != cfg {
		r.cache = make(map[string][]pluginapi.ModelInfo)
		r.generation++
	}
	r.config = cfg
	r.mu.Unlock()
	return nil
}

func (r *pluginRuntime) shutdown() {
	if r == nil {
		return
	}
	r.mu.Lock()
	r.generation++
	r.cache = make(map[string][]pluginapi.ModelInfo)
	r.inflight = make(map[string]*inflightDiscovery)
	r.mu.Unlock()
}

func (r *pluginRuntime) modelsForAuth(request authModelRPCRequest) pluginapi.ModelResponse {
	if r == nil || !strings.EqualFold(strings.TrimSpace(request.AuthProvider), providerCodex) {
		return unhandledModelResponse()
	}
	if kind := explicitAuthKind(request); kind != "" && kind != "oauth" && kind != "oauth2" {
		return unhandledModelResponse()
	}

	r.mu.Lock()
	cfg := r.config
	generation := r.generation
	r.mu.Unlock()
	values := authValuesFromRequest(request, cfg)
	if strings.TrimSpace(values.AccessToken) == "" || strings.TrimSpace(request.HostCallbackID) == "" {
		return unhandledModelResponse()
	}
	identity := cacheIdentity(request, values) + "|config:" + strconv.FormatUint(generation, 10)

	r.mu.Lock()
	if current := r.inflight[identity]; current != nil {
		current.waiters++
		done := current.done
		r.mu.Unlock()
		<-done
		return r.cachedResponse(identity)
	}
	current := &inflightDiscovery{done: make(chan struct{})}
	r.inflight[identity] = current
	r.mu.Unlock()

	models, errFetch := r.fetch(r.call, request.HostCallbackID, catalogRequest{
		BaseURL:       values.BaseURL,
		ClientVersion: cfg.ClientVersion,
		AccessToken:   values.AccessToken,
		AccountID:     values.AccountID,
		UserAgent:     cfg.UserAgent,
		Originator:    cfg.Originator,
		Headers:       values.Headers,
	})

	r.mu.Lock()
	if errFetch == nil && len(models) > 0 && r.generation == generation {
		r.cache[identity] = cloneModelInfos(catalogModelInfos(models))
	}
	cached := cloneModelInfos(r.cache[identity])
	delete(r.inflight, identity)
	close(current.done)
	r.mu.Unlock()

	if errFetch != nil {
		r.logFailure(request.HostCallbackID, request.AuthID, failureCategory(errFetch), len(cached) > 0)
	}
	if len(cached) == 0 {
		return unhandledModelResponse()
	}
	return pluginapi.ModelResponse{Provider: providerCodex, Models: cached}
}

func (r *pluginRuntime) cachedResponse(identity string) pluginapi.ModelResponse {
	r.mu.Lock()
	cached := cloneModelInfos(r.cache[identity])
	r.mu.Unlock()
	if len(cached) == 0 {
		return unhandledModelResponse()
	}
	return pluginapi.ModelResponse{Provider: providerCodex, Models: cached}
}

func (r *pluginRuntime) logFailure(callbackID, authID, category string, cached bool) {
	if r == nil || r.call == nil {
		return
	}
	payload, errMarshal := json.Marshal(hostLogRequest{
		HostCallbackID: callbackID,
		Level:          "debug",
		Message:        "Codex model discovery failed; using fallback catalog",
		Fields: map[string]any{
			"auth_id":              strings.TrimSpace(authID),
			"provider":             providerCodex,
			"category":             category,
			"using_cached_catalog": cached,
		},
	})
	if errMarshal == nil {
		_, _ = r.call("host.log", payload)
	}
}

func failureCategory(err error) string {
	if failure, ok := err.(*discoveryFailure); ok && strings.TrimSpace(failure.category) != "" {
		return failure.category
	}
	return "unexpected_failure"
}
