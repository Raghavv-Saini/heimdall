package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	agentv1 "github.com/raghavdev/heimdall/proto/v1"
	"github.com/raghavdev/heimdall/services/go-gateway/permissions"
)

// Registry maintains an in-memory catalog of available diagnostic tools gated by the permission engine.
type Registry struct {
	mu          sync.RWMutex
	tools       map[string]Tool
	permissions *permissions.Engine
}

// NewRegistry constructs an empty tool Registry with a default permission engine.
func NewRegistry() *Registry {
	return NewRegistryWithPermissions(permissions.NewEngine())
}

// NewRegistryWithPermissions constructs an empty tool Registry with a custom permission engine.
func NewRegistryWithPermissions(engine *permissions.Engine) *Registry {
	if engine == nil {
		engine = permissions.NewEngine()
	}
	return &Registry{
		tools:       make(map[string]Tool),
		permissions: engine,
	}
}

// NewDefaultRegistry constructs a Registry initialized with all standard diagnostic tools.
func NewDefaultRegistry() *Registry {
	r := NewRegistry()
	r.Register(NewDockerInspectTool())
	r.Register(NewDockerLogsTool())
	r.Register(NewSystemdStatusTool())
	r.Register(NewNetHTTPProbeTool())
	return r
}

// Permissions returns the permission engine bound to this registry.
func (r *Registry) Permissions() *permissions.Engine {
	return r.permissions
}

// Register adds or updates a tool in the registry and syncs its risk tier with the permission engine.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
	if r.permissions != nil {
		r.permissions.RegisterTool(tool.Name(), tool.RiskLevel())
	}
}

// Get looks up a tool by its unique name.
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tool, exists := r.tools[name]
	return tool, exists
}

// List returns a sorted slice of all registered tools.
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]Tool, 0, len(names))
	for _, name := range names {
		result = append(result, r.tools[name])
	}
	return result
}

// ToProtobuf converts all registered tools into a list of Protobuf ToolDefinitions for the AI Worker.
func (r *Registry) ToProtobuf() []*agentv1.ToolDefinition {
	tools := r.List()
	defs := make([]*agentv1.ToolDefinition, 0, len(tools))
	for _, tool := range tools {
		defs = append(defs, &agentv1.ToolDefinition{
			Name:        tool.Name(),
			Description: tool.Description(),
			JsonSchema:  tool.JSONSchema(),
			RiskLevel:   string(tool.RiskLevel()),
		})
	}
	return defs
}

// Execute performs deterministic permission gating and dispatches execution to the registered tool.
// Tools with risk levels requiring approval (SAFE_WRITE, DANGEROUS) are blocked from direct execution.
func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	tool, exists := r.Get(name)
	if !exists {
		return "", fmt.Errorf("tool '%s' not registered in gateway", name)
	}

	// Gate execution through the deterministic safety permission engine (AGENTS.md Invariant §1)
	if r.permissions != nil {
		decision, err := r.permissions.Evaluate(name, argsJSON)
		if err != nil {
			return "", fmt.Errorf("permission evaluation failed for tool '%s': %w", name, err)
		}
		if decision.RequiresApproval {
			return "", fmt.Errorf("execution blocked: tool '%s' has risk level '%s' and requires operator approval", name, decision.RiskLevel)
		}
	}

	return tool.Execute(ctx, argsJSON)
}

// ExecuteAuthorized dispatches execution to a registered tool after explicit operator approval has been granted.
// It still strictly enforces shell injection scanning and argument validation via the permission engine.
func (r *Registry) ExecuteAuthorized(ctx context.Context, name string, argsJSON string) (string, error) {
	tool, exists := r.Get(name)
	if !exists {
		return "", fmt.Errorf("tool '%s' not registered in gateway", name)
	}

	if r.permissions != nil {
		_, err := r.permissions.Evaluate(name, argsJSON)
		if err != nil {
			return "", fmt.Errorf("permission validation failed for tool '%s': %w", name, err)
		}
	}

	return tool.Execute(ctx, argsJSON)
}
