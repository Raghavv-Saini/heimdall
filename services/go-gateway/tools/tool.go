package tools

import (
	"context"

	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// Tool represents a uniform diagnostic tool executable by the Heimdall Gateway.
type Tool interface {
	// Name returns the unique tool identifier matching permission matrix (e.g. "docker_inspect").
	Name() string

	// Description returns a human and LLM-readable summary of the tool's diagnostic capability.
	Description() string

	// JSONSchema returns the stringified JSON schema specifying expected input arguments.
	JSONSchema() string

	// RiskLevel returns the deterministic safety classification (READ_ONLY, SAFE_WRITE, DANGEROUS).
	RiskLevel() permissions.RiskLevel

	// Execute runs the diagnostic check with timeout protection and returns formatted JSON output.
	Execute(ctx context.Context, argsJSON string) (string, error)
}
