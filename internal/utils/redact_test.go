package utils

import (
	"testing"
	"github.com/stretchr/testify/assert"
)

func TestRedact(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "OpenAI Key",
			input:    "my key is sk-XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
			expected: "my key is [REDACTED]",
		},
		{
			name:     "GitHub Token",
			input:    "token: ghp_XXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXXX",
			expected: "token: [REDACTED]",
		},
		{
			name:     "AWS Access Key",
			input:    "AKIAXXXXXXXXXXXXXXXX",
			expected: "[REDACTED]",
		},
		{
			name:     "Slack Webhook",
			input:    "https://hooks.slack.com/services/TXXXXX/BXXXXX/XXXXXXXXXXXX",
			expected: "[REDACTED]",
		},
		{
			name:     "Password assignment",
			input:    "password: some-secret-password-123",
			expected: "[REDACTED]",
		},
		{
			name:     "No secrets",
			input:    "ls -la /tmp",
			expected: "ls -la /tmp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, Redact(tt.input))
		})
	}
}
