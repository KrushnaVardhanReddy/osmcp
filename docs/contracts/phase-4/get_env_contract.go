package contracts_phase4

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

type GetEnvArgs struct {
	Keys []string `json:"keys,omitempty"` // If empty, return all explicitly allowed keys
}

type GetEnvData struct {
	Variables map[string]string `json:"variables"`
}

type GetEnvTool interface {
	contracts.Tool
}
