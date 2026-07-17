package main

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"strings"
)

type authValues struct {
	AccessToken string
	AccountID   string
	IDToken     string
	Email       string
	BaseURL     string
	Headers     map[string]string
}

func authValuesFromRequest(request authModelRPCRequest, cfg pluginConfig) authValues {
	storage := decodeStringMap(request.StorageJSON)
	value := func(key string) string {
		if current := anyMapString(request.Metadata, key); current != "" {
			return current
		}
		return anyMapString(storage, key)
	}

	values := authValues{
		AccessToken: value("access_token"),
		AccountID:   value("account_id"),
		IDToken:     value("id_token"),
		Email:       value("email"),
		BaseURL:     strings.TrimSpace(request.Attributes["base_url"]),
		Headers:     make(map[string]string),
	}
	if values.BaseURL == "" {
		values.BaseURL = value("base_url")
	}
	if values.BaseURL == "" {
		values.BaseURL = cfg.BaseURL
	}
	if values.AccountID == "" {
		values.AccountID = accountIDFromJWT(values.IDToken)
	}
	for key, headerValue := range request.Attributes {
		if !strings.HasPrefix(key, "header:") {
			continue
		}
		name := strings.TrimSpace(strings.TrimPrefix(key, "header:"))
		headerValue = strings.TrimSpace(headerValue)
		if name != "" && headerValue != "" {
			values.Headers[name] = headerValue
		}
	}
	return values
}

func decodeStringMap(raw []byte) map[string]any {
	if len(raw) == 0 {
		return nil
	}
	var decoded map[string]any
	if errUnmarshal := json.Unmarshal(raw, &decoded); errUnmarshal != nil {
		return nil
	}
	return decoded
}

func anyMapString(values map[string]any, key string) string {
	if values == nil {
		return ""
	}
	value, _ := values[key].(string)
	return strings.TrimSpace(value)
}

func explicitAuthKind(request authModelRPCRequest) string {
	if value := strings.TrimSpace(request.Attributes["auth_kind"]); value != "" {
		return strings.ToLower(value)
	}
	return strings.ToLower(anyMapString(request.Metadata, "auth_kind"))
}

func accountIDFromJWT(token string) string {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 3 {
		return ""
	}
	payload, errDecode := base64.RawURLEncoding.DecodeString(parts[1])
	if errDecode != nil {
		return ""
	}
	var claims struct {
		Auth struct {
			AccountID string `json:"chatgpt_account_id"`
		} `json:"https://api.openai.com/auth"`
	}
	if errUnmarshal := json.Unmarshal(payload, &claims); errUnmarshal != nil {
		return ""
	}
	return strings.TrimSpace(claims.Auth.AccountID)
}

func cacheIdentity(request authModelRPCRequest, values authValues) string {
	identity := ""
	switch {
	case strings.TrimSpace(values.AccountID) != "":
		identity = "account:" + strings.TrimSpace(values.AccountID)
	case strings.TrimSpace(values.Email) != "":
		identity = "email:" + strings.ToLower(strings.TrimSpace(values.Email))
	case strings.TrimSpace(values.AccessToken) != "":
		digest := sha256.Sum256([]byte(strings.TrimSpace(values.AccessToken)))
		identity = "token:" + hex.EncodeToString(digest[:8])
	default:
		identity = "auth:" + strings.TrimSpace(request.AuthID)
	}
	sourceDigest := sha256.Sum256([]byte(strings.TrimSpace(values.BaseURL)))
	return identity + "|source:" + hex.EncodeToString(sourceDigest[:6])
}
