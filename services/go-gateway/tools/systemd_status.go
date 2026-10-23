package tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// SystemdStatusTool queries host systemd service status via systemctl.
type SystemdStatusTool struct {
	runner CommandRunner
}

// NewSystemdStatusTool constructs a SystemdStatusTool using host exec.CommandContext.
func NewSystemdStatusTool() *SystemdStatusTool {
	return NewSystemdStatusToolWithRunner(defaultCommandRunner)
}

// NewSystemdStatusToolWithRunner constructs a SystemdStatusTool with a custom CommandRunner.
func NewSystemdStatusToolWithRunner(runner CommandRunner) *SystemdStatusTool {
	return &SystemdStatusTool{
		runner: runner,
	}
}

// Name returns the unique tool identifier matching permission matrix.
func (t *SystemdStatusTool) Name() string {
	return "systemd_status"
}

// Description explains the tool's diagnostic purpose.
func (t *SystemdStatusTool) Description() string {
	return "Query host systemd service unit status and recent journal logs with timeout protection."
}

// JSONSchema specifies the expected JSON schema for input arguments.
func (t *SystemdStatusTool) JSONSchema() string {
	return `{
  "type": "object",
  "properties": {
    "service_name": {
      "type": "string",
      "description": "The name of the systemd service unit (e.g. 'nginx' or 'docker.service')."
    }
  },
  "required": ["service_name"]
}`
}

// RiskLevel returns the deterministic safety classification.
func (t *SystemdStatusTool) RiskLevel() permissions.RiskLevel {
	return permissions.RiskLevelReadOnly
}

// Execute queries systemctl status and handles non-zero inactive/dead exit codes gracefully.
func (t *SystemdStatusTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		ServiceName string `json:"service_name"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("malformed JSON arguments: %w", err)
	}
	if strings.TrimSpace(args.ServiceName) == "" {
		return "", fmt.Errorf("missing required argument: 'service_name'")
	}

	output, err := t.runner(ctx, "systemctl", "status", args.ServiceName, "--no-pager")
	exitCode := 0
	if err != nil {
		if ctx.Err() != nil {
			return "", fmt.Errorf("systemd status timed out: %w", ctx.Err())
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			exitCode = exitErr.ExitCode()
			// Exit code 3 (inactive/dead) and code 4 (unit not found) still contain diagnostic output
		} else {
			return "", fmt.Errorf("failed to execute systemctl: %w", err)
		}
	}

	res := map[string]any{
		"service_name": args.ServiceName,
		"exit_code":    exitCode,
		"active":       exitCode == 0,
		"output":       string(output),
	}

	bytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format systemd status result: %w", err)
	}

	return string(bytes), nil
}
