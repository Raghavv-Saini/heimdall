package tools

import (
	"context"
	"os/exec"
)

// CommandRunner represents a function signature for executing host commands via structured argument slices.
type CommandRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

// defaultCommandRunner executes commands directly using exec.CommandContext without shell interpolation.
func defaultCommandRunner(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}
