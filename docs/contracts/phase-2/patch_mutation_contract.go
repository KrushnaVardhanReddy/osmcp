package phase2

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

// PatchRequest defines the parameters for the patch tool.
type PatchRequest struct {
	Path string `json:"path"`
	Diff string `json:"diff"`
}

// PatchMutationTool implementations must embed the policy engine and response builder.
type PatchMutationTool interface {
	Patch(req PatchRequest) contracts.Envelope
}
