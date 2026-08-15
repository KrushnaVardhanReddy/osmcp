package contracts_phase4

import "github.com/osmcp/osmcp/docs/contracts/cross_cutting"

type HashFileArgs struct {
	Path      string `json:"path"`
	Algorithm string `json:"algorithm,omitempty"` // "sha256" (default), "md5", "sha1"
}

type HashFileData struct {
	Hash      string `json:"hash"`
	Algorithm string `json:"algorithm"`
}

type HashFileTool interface {
	contracts.Tool
}
