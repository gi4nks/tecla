package utils

import (
	"strings"
)

// Redact is a simple placeholder implementation to ensure the project builds.
// In a real scenario, this would use regex or a more sophisticated matching logic.
func Redact(input string) string {
	// Very basic implementation for demonstration purposes
	secrets := []string{"sk-", "ghp_", "AKIA", "https://hooks.slack.com", "password:"}
	for _, s := range secrets {
		if strings.Contains(input, s) {
			return "[REDACTED]"
		}
	}
	return input
}
