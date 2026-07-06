package audit

import (
	"encoding/json"
	"strings"
)

var sensitiveFields = []string{
	"password",
	"app_secret",
	"token",
	"refresh_token",
	"access_token",
	"secret",
}

// SanitizeBody recebe um JSON em string e substitui campos sensíveis por "***REDACTED***".
// Se o body não for JSON válido, retorna o original sem modificação.
func SanitizeBody(body string) string {
	body = strings.TrimSpace(body)
	if body == "" || body[0] != '{' {
		return body
	}

	var m map[string]any
	if err := json.Unmarshal([]byte(body), &m); err != nil {
		return body
	}

	redactMap(m)

	out, err := json.Marshal(m)
	if err != nil {
		return body
	}
	return string(out)
}

func redactMap(m map[string]any) {
	for k, v := range m {
		if isSensitive(k) {
			m[k] = "***REDACTED***"
			continue
		}
		switch val := v.(type) {
		case map[string]any:
			redactMap(val)
		case []any:
			redactSlice(val)
		}
	}
}

func redactSlice(s []any) {
	for _, item := range s {
		if m, ok := item.(map[string]any); ok {
			redactMap(m)
		}
	}
}

func isSensitive(key string) bool {
	lower := strings.ToLower(key)
	for _, f := range sensitiveFields {
		if lower == f {
			return true
		}
	}
	return false
}
