package contracts

// WriteFileRequest defines the parameters for the write_file tool.
type WriteFileRequest struct {
	Path      string `json:"path"`
	Content   string `json:"content"`
	Overwrite bool   `json:"overwrite"` // Default: false
}

// AppendFileRequest defines the parameters for the append_file tool.
type AppendFileRequest struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

// MkdirRequest defines the parameters for the mkdir tool.
type MkdirRequest struct {
	Path string `json:"path"`
}

// RmRequest defines the parameters for the rm tool.
type RmRequest struct {
	Path      string `json:"path"`
	Recursive bool   `json:"recursive"` // Default: false
}

// MvRequest defines the parameters for the mv tool.
type MvRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// CpRequest defines the parameters for the cp tool.
type CpRequest struct {
	Source      string `json:"source"`
	Destination string `json:"destination"`
}

// FSMutationTool implementations must embed the policy engine and response builder.
type FSMutationTool interface {
	// WriteFile executes the write_file tool.
	WriteFile(req WriteFileRequest) Envelope
	// AppendFile executes the append_file tool.
	AppendFile(req AppendFileRequest) Envelope
	// Mkdir executes the mkdir tool.
	Mkdir(req MkdirRequest) Envelope
	// Rm executes the rm tool.
	Rm(req RmRequest) Envelope
	// Mv executes the mv tool.
	Mv(req MvRequest) Envelope
	// Cp executes the cp tool.
	Cp(req CpRequest) Envelope
}
