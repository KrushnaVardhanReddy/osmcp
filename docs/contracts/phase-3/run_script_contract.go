package contracts_phase3

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

type RunScriptArgs struct {
	Interpreter string `json:"interpreter"`
	Script      string `json:"script"`
	WorkingDir  string `json:"working_dir,omitempty"`
	TimeoutMs   int    `json:"timeout_ms,omitempty"`
}

type RunScriptData struct {
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	ExitCode int    `json:"exit_code"`
	TimedOut bool   `json:"timed_out"`
}

type RunScriptTool interface {
	contracts.Tool
}
