package templates

import "embed"

//go:embed policies/*.toml
var PolicyTemplates embed.FS
