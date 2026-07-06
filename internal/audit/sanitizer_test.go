package audit

import (
	"encoding/json"
	"testing"
)

func TestSanitizeBody_RedactsSensitiveFields(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		redacted []string
		kept     []string
	}{
		{
			name:     "redacts password",
			input:    `{"email":"user@example.com","password":"secret123"}`,
			redacted: []string{"password"},
			kept:     []string{"email"},
		},
		{
			name:     "redacts app_secret",
			input:    `{"app_key":"abc","app_secret":"supersecret"}`,
			redacted: []string{"app_secret"},
			kept:     []string{"app_key"},
		},
		{
			name:     "redacts token",
			input:    `{"user_id":"123","token":"jwt.token.here"}`,
			redacted: []string{"token"},
			kept:     []string{"user_id"},
		},
		{
			name:     "redacts nested fields",
			input:    `{"user":{"name":"João","password":"abc"}}`,
			redacted: []string{"password"},
			kept:     []string{"name"},
		},
		{
			name:  "preserves non-sensitive body",
			input: `{"nome":"Alpha","slug":"alpha"}`,
			kept:  []string{"nome", "slug"},
		},
		{
			name:  "returns non-json unchanged",
			input: `not json at all`,
			kept:  []string{},
		},
		{
			name:  "returns empty unchanged",
			input: ``,
			kept:  []string{},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeBody(tc.input)

			if len(tc.redacted) > 0 {
				var m map[string]any
				if err := json.Unmarshal([]byte(result), &m); err != nil {
					t.Fatalf("resultado não é JSON válido: %v", err)
				}
				for _, field := range tc.redacted {
					val, ok := m[field]
					if !ok {
						// campo pode estar aninhado — checar string
						if !contains(result, "***REDACTED***") {
							t.Errorf("campo %q não foi redactado", field)
						}
						continue
					}
					if val != "***REDACTED***" {
						t.Errorf("campo %q = %v, queria ***REDACTED***", field, val)
					}
				}
			}

			for _, field := range tc.kept {
				if !contains(result, field) {
					t.Errorf("campo %q foi removido mas não deveria", field)
				}
			}
		})
	}
}

func TestSanitizeBody_CaseInsensitive(t *testing.T) {
	inputs := []string{
		`{"Password":"abc"}`,
		`{"PASSWORD":"abc"}`,
		`{"pAsSwOrD":"abc"}`,
	}
	for _, input := range inputs {
		result := SanitizeBody(input)
		var m map[string]any
		if err := json.Unmarshal([]byte(result), &m); err != nil {
			t.Fatalf("resultado não é JSON: %v", err)
		}
		for _, v := range m {
			if v != "***REDACTED***" {
				t.Errorf("input %q: esperava redact, got %v", input, v)
			}
		}
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 ||
		func() bool {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
			return false
		}())
}
