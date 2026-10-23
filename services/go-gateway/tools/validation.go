package tools

import (
	"fmt"
	"strings"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// ValidateSafeArgument validates that an argument string contains no shell metacharacters.
func ValidateSafeArgument(fieldName, value string) error {
	for _, pattern := range permissions.ForbiddenShellPatterns {
		if strings.Contains(value, pattern) {
			return fmt.Errorf("dangerous shell metacharacter '%s' detected in '%s'", pattern, fieldName)
		}
	}
	if strings.Contains(value, "\n") {
		return fmt.Errorf("dangerous shell metacharacter '\\n' detected in '%s'", fieldName)
	}
	if strings.Contains(value, "\r") {
		return fmt.Errorf("dangerous shell metacharacter '\\r' detected in '%s'", fieldName)
	}
	return nil
}
