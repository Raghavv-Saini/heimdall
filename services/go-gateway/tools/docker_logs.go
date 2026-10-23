package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// DockerLogsTool retrieves recent container logs with tail limits.
type DockerLogsTool struct {
	runner CommandRunner
}

// NewDockerLogsTool constructs a DockerLogsTool using host exec.CommandContext.
func NewDockerLogsTool() *DockerLogsTool {
	return NewDockerLogsToolWithRunner(defaultCommandRunner)
}

// NewDockerLogsToolWithRunner constructs a DockerLogsTool with a custom CommandRunner.
func NewDockerLogsToolWithRunner(runner CommandRunner) *DockerLogsTool {
	return &DockerLogsTool{
		runner: runner,
	}
}

// Name returns the unique tool identifier matching permission matrix.
func (t *DockerLogsTool) Name() string {
	return "docker_logs"
}

// Description explains the tool's diagnostic purpose.
func (t *DockerLogsTool) Description() string {
	return "Retrieve recent stdout and stderr logs for a Docker container with configurable line tailing."
}

// JSONSchema specifies the expected JSON schema for input arguments.
func (t *DockerLogsTool) JSONSchema() string {
	return `{
  "type": "object",
  "properties": {
    "container_id": {
      "type": "string",
      "description": "The name or ID of the Docker container."
    },
    "tail": {
      "type": "integer",
      "description": "Number of recent log lines to retrieve (default: 100)."
    }
  },
  "required": ["container_id"]
}`
}

// RiskLevel returns the deterministic safety classification.
func (t *DockerLogsTool) RiskLevel() permissions.RiskLevel {
	return permissions.RiskLevelReadOnly
}

// Execute retrieves recent logs using structured argument slices.
func (t *DockerLogsTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		ContainerID string `json:"container_id"`
		Tail        any    `json:"tail"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("malformed JSON arguments: %w", err)
	}
	containerID := strings.TrimSpace(args.ContainerID)
	if containerID == "" {
		return "", fmt.Errorf("missing required argument: 'container_id'")
	}
	if err := ValidateSafeArgument("container_id", containerID); err != nil {
		return "", err
	}

	tailLimit := parseTailLimit(args.Tail, 100, 1000)
	tailStr := strconv.Itoa(tailLimit)

	output, err := t.runner(ctx, "docker", "logs", "--tail", tailStr, containerID)
	if err != nil {
		// Return structured observation instead of failing with raw Go error
		notFoundResult := map[string]any{
			"container_id": containerID,
			"found":        false,
			"error":        strings.TrimSpace(string(output)),
			"logs":         "",
		}
		bytes, _ := json.MarshalIndent(notFoundResult, "", "  ")
		return string(bytes), nil
	}

	res := map[string]any{
		"container_id": containerID,
		"found":        true,
		"tail":         tailLimit,
		"logs":         string(output),
	}

	bytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format docker logs result: %w", err)
	}

	return string(bytes), nil
}

func clamp(n, defaultVal, maxVal int) int {
	if n <= 0 {
		return defaultVal
	}
	if n > maxVal {
		return maxVal
	}
	return n
}

// parseTailLimit converts int, float64, or string tail inputs into bounded int.
func parseTailLimit(val any, defaultVal, maxVal int) int {
	if val == nil {
		return defaultVal
	}
	switch v := val.(type) {
	case float64:
		return clamp(int(v), defaultVal, maxVal)
	case int:
		return clamp(v, defaultVal, maxVal)
	case string:
		n, err := strconv.Atoi(strings.TrimSpace(v))
		if err != nil {
			return defaultVal
		}
		return clamp(n, defaultVal, maxVal)
	default:
		return defaultVal
	}
}
