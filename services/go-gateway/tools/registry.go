package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	agentv1 "github.com/raghavdev/heimdall/proto/v1"
)

// Registry maintains an in-memory catalog of available diagnostic tools.
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry constructs an empty tool Registry.
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
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

// Register adds or updates a tool in the registry.
func (r *Registry) Register(tool Tool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tools[tool.Name()] = tool
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

// Execute dispatches execution to the registered tool by name.
func (r *Registry) Execute(ctx context.Context, name string, argsJSON string) (string, error) {
	tool, exists := r.Get(name)
	if !exists {
		return "", fmt.Errorf("tool '%s' not registered in gateway", name)
	}
	return tool.Execute(ctx, argsJSON)
}
