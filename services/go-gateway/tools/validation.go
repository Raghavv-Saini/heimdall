package tools

import (
	"fmt"
	"strings"
)

// forbiddenShellPatterns contains dangerous shell metacharacters and command injection sequences.
var forbiddenShellPatterns = []string{
	";",
	"&&",
	"||",
	"`",
	"$(",
	"rm -rf",
	"\n",
	"\r",
}

// ValidateSafeArgument validates that an argument string contains no shell metacharacters.
func ValidateSafeArgument(fieldName, value string) error {
	for _, pattern := range forbiddenShellPatterns {
		if strings.Contains(value, pattern) {
			return fmt.Errorf("dangerous shell metacharacter '%s' detected in '%s'", pattern, fieldName)
		}
	}
	return nil
}
