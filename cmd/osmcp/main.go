package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	"github.com/osmcp/osmcp/internal/audit"
	"github.com/osmcp/osmcp/internal/policy"
	"github.com/osmcp/osmcp/internal/response"
	"github.com/osmcp/osmcp/internal/tools"
)

var (
	versionFlag   = flag.Bool("version", false, "Print version string and exit")
	policyFlag    = flag.String("policy", ".osmcp/policy.toml", "Path to policy file")
	auditLogFlag  = flag.String("audit-log", "", "Path to audit log file")
	validateFlag  = flag.Bool("validate", false, "Validate policy and exit")
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Println("osmcp v0.1.0")
		os.Exit(0)
	}

	p, err := policy.LoadFromFile(*policyFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load policy: %v\n", err)
		os.Exit(1)
	}

	if *validateFlag {
		errs := policy.Validate(p)
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintf(os.Stderr, "policy validation error: %v\n", e)
			}
			os.Exit(1)
		}
		fmt.Println("policy is valid")
		os.Exit(0)
	}

	var auditLogger contracts.AuditLogger
	if *auditLogFlag != "" {
		l, err := audit.NewLogger("file", *auditLogFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to init audit logger: %v\n", err)
			os.Exit(1)
		}
		auditLogger = l
	} else if p.Audit.Destination == "stderr" {
        l, _ := audit.NewLogger("stderr", "")
        auditLogger = l
    } else if p.Audit.Destination == "file" && p.Audit.Path != "" {
        l, err := audit.NewLogger("file", p.Audit.Path)
        if err != nil {
            fmt.Fprintf(os.Stderr, "failed to init audit logger: %v\n", err)
            os.Exit(1)
        }
        auditLogger = l
    }

	policyEngine := policy.NewEngine(p, auditLogger)
	envelopeBuilder := response.NewBuilder()
	toolRegistry := tools.NewRegistry(policyEngine)

	toolRegistry.Register(tools.NewGrepTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewTreeTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewHeadTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewTailTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewDuTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewJqTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewSedTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewDiffTool(policyEngine, envelopeBuilder))

	lsTool := tools.NewLsTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(lsTool)

	catTool := tools.NewCatTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(catTool)

	statTool := tools.NewStatTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(statTool)

	wcTool := tools.NewWcTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(wcTool)

	gitStatusTool := tools.NewGitStatusTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitStatusTool)

	gitDiffTool := tools.NewGitDiffTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitDiffTool)

	gitLogTool := tools.NewGitLogTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitLogTool)

	findTool := tools.NewFindTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(findTool)

	gitAddTool := tools.NewGitAddTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitAddTool)

	gitCommitTool := tools.NewGitCommitTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitCommitTool)

	gitCheckoutTool := tools.NewGitCheckoutTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitCheckoutTool)

	gitBranchTool := tools.NewGitBranchTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitBranchTool)

	gitPullTool := tools.NewGitPullTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitPullTool)

	gitPushTool := tools.NewGitPushTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitPushTool)
	toolRegistry.Register(tools.NewPatchTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewTarTool(policyEngine, envelopeBuilder))

	s := server.NewMCPServer("osmcp", "0.1.0", server.WithToolCapabilities(true))

	visibleTools := toolRegistry.VisibleTools()
	for _, t := range visibleTools {
		t.RegisterMCP(s)
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
