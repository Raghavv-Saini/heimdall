package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// DockerInspectTool retrieves container state, exit codes, OOMKilled flag, and mounts telemetry.
type DockerInspectTool struct {
	runner CommandRunner
}

// NewDockerInspectTool constructs a DockerInspectTool using host exec.CommandContext.
func NewDockerInspectTool() *DockerInspectTool {
	return NewDockerInspectToolWithRunner(defaultCommandRunner)
}

// NewDockerInspectToolWithRunner constructs a DockerInspectTool with a custom CommandRunner.
func NewDockerInspectToolWithRunner(runner CommandRunner) *DockerInspectTool {
	return &DockerInspectTool{
		runner: runner,
	}
}

// Name returns the unique tool identifier matching permission matrix.
func (t *DockerInspectTool) Name() string {
	return "docker_inspect"
}

// Description explains the tool's diagnostic purpose to the LLM and operator.
func (t *DockerInspectTool) Description() string {
	return "Inspect detailed low-level metadata of a container including state, exit code, OOMKilled flag, mounts, and error state."
}

// JSONSchema specifies the expected JSON schema for input arguments.
func (t *DockerInspectTool) JSONSchema() string {
	return `{
  "type": "object",
  "properties": {
    "container_id": {
      "type": "string",
      "description": "The name or ID of the Docker container to inspect."
    }
  },
  "required": ["container_id"]
}`
}

// RiskLevel returns the deterministic safety classification.
func (t *DockerInspectTool) RiskLevel() permissions.RiskLevel {
	return permissions.RiskLevelReadOnly
}

// Execute executes docker inspect using structured argument slices and distills golden signals.
func (t *DockerInspectTool) Execute(ctx context.Context, argsJSON string) (string, error) {
	var args struct {
		ContainerID string `json:"container_id"`
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

	output, err := t.runner(ctx, "docker", "inspect", containerID)
	if err != nil {
		// Return structured observation instead of failing with raw Go error
		notFoundResult := map[string]any{
			"container_id": containerID,
			"found":        false,
			"error":        strings.TrimSpace(string(output)),
		}
		bytes, _ := json.MarshalIndent(notFoundResult, "", "  ")
		return string(bytes), nil
	}

	var rawInspect []struct {
		ID    string `json:"Id"`
		Name  string `json:"Name"`
		State struct {
			Status     string `json:"Status"`
			Running    bool   `json:"Running"`
			ExitCode   int    `json:"ExitCode"`
			OOMKilled  bool   `json:"OOMKilled"`
			Error      string `json:"Error"`
			FinishedAt string `json:"FinishedAt"`
		} `json:"State"`
		RestartCount int `json:"RestartCount"`
		Mounts       []struct {
			Source      string `json:"Source"`
			Destination string `json:"Destination"`
			Mode        string `json:"Mode"`
		} `json:"Mounts"`
	}

	if err := json.Unmarshal(output, &rawInspect); err != nil {
		return "", fmt.Errorf("failed to parse docker inspect output: %w", err)
	}
	if len(rawInspect) == 0 {
		return "", fmt.Errorf("docker inspect returned empty inspection array for '%s'", containerID)
	}

	item := rawInspect[0]
	shortID := item.ID
	if len(shortID) > 12 {
		shortID = shortID[:12]
	}

	cleanResult := map[string]any{
		"id":            shortID,
		"name":          strings.TrimPrefix(item.Name, "/"),
		"status":        item.State.Status,
		"running":       item.State.Running,
		"exit_code":     item.State.ExitCode,
		"oom_killed":    item.State.OOMKilled,
		"error":         item.State.Error,
		"restart_count": item.RestartCount,
		"finished_at":   item.State.FinishedAt,
		"mounts":        item.Mounts,
	}

	bytes, err := json.MarshalIndent(cleanResult, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to format distilled inspect result: %w", err)
	}

	return string(bytes), nil
}
