package tools

import (
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
)

type registry struct {
	tools  map[string]contracts.Tool
	policy contracts.PolicyEngine
}

// NewRegistry creates a new tool registry.
func NewRegistry(policy contracts.PolicyEngine) contracts.ToolRegistry {
	return &registry{
		tools:  make(map[string]contracts.Tool),
		policy: policy,
	}
}

// Register adds a tool to the registry.
func (r *registry) Register(tool contracts.Tool) {
	r.tools[tool.Name()] = tool
}

// VisibleTools returns all tools permitted by the active policy.
func (r *registry) VisibleTools() []contracts.Tool {
	var visible []contracts.Tool
	for name, tool := range r.tools {
		if r.policy.IsToolVisible(name) {
			visible = append(visible, tool)
		}
	}
	return visible
}
