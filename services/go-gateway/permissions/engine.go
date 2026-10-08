package permissions

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// RiskLevel represents the deterministic risk classification tier for a tool.
type RiskLevel string

const (
	// RiskLevelReadOnly indicates zero operational impact; safe for auto-approval.
	RiskLevelReadOnly RiskLevel = "READ_ONLY"

	// RiskLevelSafeWrite indicates non-destructive remediation; requires operator confirmation [y/N].
	RiskLevelSafeWrite RiskLevel = "SAFE_WRITE"

	// RiskLevelDangerous indicates potentially destructive or downtime-inducing operations;
	// requires explicit confirmation with resource verification.
	RiskLevelDangerous RiskLevel = "DANGEROUS"
)

// Sentinel errors returned by the permission engine for validation failures.
var (
	ErrMalformedJSON           = errors.New("malformed JSON arguments: must be a valid JSON object")
	ErrShellInjectionDetected  = errors.New("dangerous shell injection sequence detected in arguments")
	ErrMissingRequiredArgument = errors.New("missing required tool argument")
	ErrInvalidArgumentType     = errors.New("invalid argument type: expected string")
)

// PermissionDecision encapsulates the deterministic safety evaluation result.
type PermissionDecision struct {
	ToolName         string         `json:"tool_name"`
	RiskLevel        RiskLevel      `json:"risk_level"`
	RequiresApproval bool           `json:"requires_approval"`
	Summary          string         `json:"summary"`
	ParsedArguments  map[string]any `json:"-"`
}

// Engine implements deterministic safety gating and risk classification.
type Engine struct {
	riskMatrix map[string]RiskLevel
}

// forbiddenShellPatterns contains dangerous shell metacharacters and command injection sequences.
var forbiddenShellPatterns = []string{
	";",
	"&&",
	"||",
	"`",
	"$(",
	"rm -rf",
}

// NewEngine constructs a new permission Engine initialized with the static MVP risk matrix.
func NewEngine() *Engine {
	return &Engine{
		riskMatrix: map[string]RiskLevel{
			"docker_list_containers": RiskLevelReadOnly,
			"docker_inspect":         RiskLevelReadOnly,
			"docker_logs":            RiskLevelReadOnly,
			"systemd_status":         RiskLevelReadOnly,
			"net_http_probe":         RiskLevelReadOnly,
			"docker_restart":         RiskLevelSafeWrite,
			"docker_stop":            RiskLevelDangerous,
			"docker_remove":          RiskLevelDangerous,
			"systemd_restart":        RiskLevelDangerous,
		},
	}
}

// GetRiskLevel returns the static risk tier for the given tool name.
// Any unrecognized tool defaults to DANGEROUS under the zero-trust policy.
func (e *Engine) GetRiskLevel(toolName string) RiskLevel {
	if level, exists := e.riskMatrix[toolName]; exists {
		return level
	}
	return RiskLevelDangerous
}

// Evaluate performs deterministic safety gating on a tool request.
// It parses and validates JSON arguments, inspects for shell injections,
// checks required fields, and applies the static risk matrix.
func (e *Engine) Evaluate(toolName string, argsJSON string) (*PermissionDecision, error) {
	// 1. Scan toolName for shell injection sequences
	if bad, token := containsShellInjection(toolName); bad {
		return nil, fmt.Errorf("%w: '%s' in tool name '%s'", ErrShellInjectionDetected, token, toolName)
	}

	// 2. Normalize empty arguments
	trimmed := strings.TrimSpace(argsJSON)
	if trimmed == "" {
		trimmed = "{}"
	}

	// 3. Parse JSON arguments into map[string]any
	var parsed map[string]any
	if err := json.Unmarshal([]byte(trimmed), &parsed); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrMalformedJSON, err)
	}
	if parsed == nil {
		parsed = make(map[string]any)
	}

	// 4. Scan parsed arguments recursively for shell injection sequences
	if bad, token := scanForInjection(parsed); bad {
		return nil, fmt.Errorf("%w: '%s'", ErrShellInjectionDetected, token)
	}

	// 5. Validate tool-specific required arguments
	if err := e.validateToolArguments(toolName, parsed); err != nil {
		return nil, err
	}

	// 6. Look up risk level (zero-trust defaults unknown tools to DANGEROUS)
	riskLevel := e.GetRiskLevel(toolName)
	requiresApproval := riskLevel != RiskLevelReadOnly

	// 7. Generate human-readable impact summary
	summary := e.generateSummary(toolName, riskLevel, parsed)

	return &PermissionDecision{
		ToolName:         toolName,
		RiskLevel:        riskLevel,
		RequiresApproval: requiresApproval,
		Summary:          summary,
		ParsedArguments:  parsed,
	}, nil
}

// validateToolArguments verifies that known tools have their required fields populated with valid types.
func (e *Engine) validateToolArguments(toolName string, args map[string]any) error {
	switch toolName {
	case "docker_inspect", "docker_logs", "docker_restart", "docker_stop", "docker_remove":
		return requireNonEmptyString(args, "container_id")
	case "systemd_status", "systemd_restart":
		return requireNonEmptyString(args, "service_name")
	case "net_http_probe":
		return requireNonEmptyString(args, "url")
	case "docker_list_containers":
		return nil
	default:
		// Unknown tools have no predefined schema in the static matrix;
		// they are permitted to proceed to operator approval under DANGEROUS classification.
		return nil
	}
}

// requireNonEmptyString asserts that a field exists, is a string, and is non-empty.
func requireNonEmptyString(args map[string]any, field string) error {
	val, exists := args[field]
	if !exists {
		return fmt.Errorf("%w: '%s'", ErrMissingRequiredArgument, field)
	}
	strVal, ok := val.(string)
	if !ok {
		return fmt.Errorf("%w: '%s' must be a string", ErrInvalidArgumentType, field)
	}
	if strings.TrimSpace(strVal) == "" {
		return fmt.Errorf("%w: '%s' cannot be empty", ErrMissingRequiredArgument, field)
	}
	return nil
}

// containsShellInjection checks if a single string contains any forbidden shell sequence.
func containsShellInjection(s string) (bool, string) {
	for _, pattern := range forbiddenShellPatterns {
		if strings.Contains(s, pattern) {
			return true, pattern
		}
	}
	if strings.Contains(s, "\n") {
		return true, "\\n"
	}
	if strings.Contains(s, "\r") {
		return true, "\\r"
	}
	return false, ""
}

// scanForInjection recursively walks JSON objects, arrays, and string leaves to detect injection tokens.
func scanForInjection(val any) (bool, string) {
	switch v := val.(type) {
	case string:
		if bad, token := containsShellInjection(v); bad {
			return true, token
		}
	case map[string]any:
		for k, child := range v {
			if bad, token := containsShellInjection(k); bad {
				return true, token
			}
			if bad, token := scanForInjection(child); bad {
				return true, token
			}
		}
	case []any:
		for _, item := range v {
			if bad, token := scanForInjection(item); bad {
				return true, token
			}
		}
	}
	return false, ""
}

// generateSummary produces a human-readable impact description for the operator.
func (e *Engine) generateSummary(toolName string, riskLevel RiskLevel, args map[string]any) string {
	switch toolName {
	case "docker_list_containers":
		return "List all active and inactive containers (READ_ONLY)"
	case "docker_inspect":
		return fmt.Sprintf("Inspect container '%s' state and metadata (READ_ONLY)", args["container_id"])
	case "docker_logs":
		return fmt.Sprintf("Fetch stdout/stderr logs for container '%s' (READ_ONLY)", args["container_id"])
	case "systemd_status":
		return fmt.Sprintf("Check systemd service health for '%s' (READ_ONLY)", args["service_name"])
	case "net_http_probe":
		return fmt.Sprintf("Probe HTTP endpoint '%s' (READ_ONLY)", args["url"])
	case "docker_restart":
		return fmt.Sprintf("Restart container '%s' (SAFE_WRITE - non-destructive remediation)", args["container_id"])
	case "docker_stop":
		return fmt.Sprintf("Stop container '%s' (DANGEROUS - service downtime)", args["container_id"])
	case "docker_remove":
		return fmt.Sprintf("Remove container '%s' (DANGEROUS - permanent deletion)", args["container_id"])
	case "systemd_restart":
		return fmt.Sprintf("Restart host systemd service '%s' (DANGEROUS - host impact)", args["service_name"])
	default:
		return fmt.Sprintf("Unknown tool '%s' requested (DANGEROUS - zero-trust policy requires operator confirmation)", toolName)
	}
}
