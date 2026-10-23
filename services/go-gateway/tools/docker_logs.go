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
		Tail        int    `json:"tail"`
	}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return "", fmt.Errorf("malformed JSON arguments: %w", err)
	}
	if strings.TrimSpace(args.ContainerID) == "" {
		return "", fmt.Errorf("missing required argument: 'container_id'")
	}

	tailLimit := args.Tail
	if tailLimit <= 0 {
		tailLimit = 100
	}
	if tailLimit > 1000 {
		tailLimit = 1000
	}

	tailStr := strconv.Itoa(tailLimit)
	output, err := t.runner(ctx, "docker", "logs", "--tail", tailStr, args.ContainerID)
	if err != nil {
		return "", fmt.Errorf("docker logs failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	res := map[string]any{
		"container_id": args.ContainerID,
		"tail":         tailLimit,
		"logs":         string(output),
	}

	bytes, err := json.MarshalIndent(res, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format docker logs result: %w", err)
	}

	return string(bytes), nil
}
