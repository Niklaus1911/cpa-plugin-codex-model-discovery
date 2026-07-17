package main

import (
	"encoding/base64"
	"fmt"
	"testing"

	"github.com/router-for-me/CLIProxyAPI/v7/sdk/pluginapi"
)

func TestAuthValuesFromRequestPrecedenceAndJWT(t *testing.T) {
	payload := base64.RawURLEncoding.EncodeToString([]byte(`{"https://api.openai.com/auth":{"chatgpt_account_id":"jwt-account"}}`))
	idToken := "header." + payload + ".signature"
	request := authModelRPCRequest{
		AuthModelRequest: pluginapi.AuthModelRequest{
			StorageJSON: []byte(fmt.Sprintf(`{"access_token":"storage-token","id_token":%q,"base_url":"https://storage.test"}`, idToken)),
			Metadata: map[string]any{
				"access_token": "metadata-token",
				"email":        " Person@Example.com ",
				"base_url":     "https://metadata.test",
			},
			Attributes: map[string]string{
				"base_url":          "https://attribute.test",
				"header:X-Test":     " value ",
				"header:User-Agent": "account-agent",
			},
		},
	}
	values := authValuesFromRequest(request, defaultConfig())
	if values.AccessToken != "metadata-token" || values.AccountID != "jwt-account" || values.Email != "Person@Example.com" || values.BaseURL != "https://attribute.test" {
		t.Fatalf("authValuesFromRequest() = %#v", values)
	}
	if values.Headers["X-Test"] != "value" || values.Headers["User-Agent"] != "account-agent" {
		t.Fatalf("headers = %#v", values.Headers)
	}
}

func TestCacheIdentityDoesNotExposeToken(t *testing.T) {
	request := authModelRPCRequest{AuthModelRequest: pluginapi.AuthModelRequest{AuthID: "auth-1"}}
	identity := cacheIdentity(request, authValues{AccessToken: "super-secret-token", BaseURL: defaultBaseURL})
	if identity == "" || identity == "super-secret-token" {
		t.Fatalf("cacheIdentity() = %q", identity)
	}
	if contains(identity, "super-secret-token") {
		t.Fatalf("cacheIdentity() leaked token: %q", identity)
	}
}

func contains(value, fragment string) bool {
	return len(fragment) > 0 && len(value) >= len(fragment) && stringContains(value, fragment)
}

func stringContains(value, fragment string) bool {
	for index := 0; index+len(fragment) <= len(value); index++ {
		if value[index:index+len(fragment)] == fragment {
			return true
		}
	}
	return false
}
