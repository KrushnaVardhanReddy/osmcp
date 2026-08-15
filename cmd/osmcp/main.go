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
	"github.com/osmcp/osmcp/templates"
)

var (
	versionFlag  = flag.Bool("version", false, "Print version string and exit")
	policyFlag   = flag.String("policy", ".osmcp/policy.toml", "Path to policy file")
	auditLogFlag = flag.String("audit-log", "", "Path to audit log file")
	validateFlag = flag.Bool("validate", false, "Validate policy and exit")
	initFlag     = flag.Bool("init", false, "Scaffold a starter configuration")
	profileFlag  = flag.String("profile", "dev-agent", "Profile to use for initialization (read-only, dev-agent, ci-agent, review-agent)")

	// version is injected by GoReleaser via ldflags during build.
	version = "dev"
)

func main() {
	flag.Parse()

	if *versionFlag {
		fmt.Printf("osmcp %s\n", version)
		os.Exit(0)
	}

	if *initFlag {
		profile := *profileFlag
		validProfiles := map[string]bool{"read-only": true, "dev-agent": true, "ci-agent": true, "review-agent": true}
		if !validProfiles[profile] {
			fmt.Fprintf(os.Stderr, "invalid profile: %s. Valid profiles are read-only, dev-agent, ci-agent, review-agent\n", profile)
			os.Exit(1)
		}

		if _, err := os.Stat(".osmcp/policy.toml"); err == nil {
			fmt.Fprintf(os.Stderr, "error: .osmcp/policy.toml already exists. Will not overwrite.\n")
			os.Exit(1)
		}

		if err := os.MkdirAll(".osmcp", 0755); err != nil {
			fmt.Fprintf(os.Stderr, "failed to create .osmcp directory: %v\n", err)
			os.Exit(1)
		}

		content, err := templates.PolicyTemplates.ReadFile("policies/" + profile + ".toml")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read embedded template: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(".osmcp/policy.toml", content, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "failed to write policy file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("✅ Initialized osmcp with profile: %s\n", profile)
		fmt.Printf("   Config: .osmcp/policy.toml\n")
		fmt.Printf("   Run:    osmcp --policy .osmcp/policy.toml\n")
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
	switch {
	case *auditLogFlag != "":
		auditLogger, err = audit.NewLogger("file", *auditLogFlag)
	case p.Audit.Destination == "stderr":
		auditLogger, err = audit.NewLogger("stderr", "")
	case p.Audit.Destination == "file" && p.Audit.Path != "":
		auditLogger, err = audit.NewLogger("file", p.Audit.Path)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to init audit logger: %v\n", err)
		os.Exit(1)
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
	toolRegistry.Register(tools.NewWriteFileTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewAppendFileTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewMkdirTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewRmTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewMvTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewCpTool(policyEngine, envelopeBuilder))

	gitPushTool := tools.NewGitPushTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitPushTool)
	toolRegistry.Register(tools.NewPatchTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewSortTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewAwkTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewTarTool(policyEngine, envelopeBuilder))

	runScriptTool := tools.NewRunScriptTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(runScriptTool)

	s := server.NewMCPServer("osmcp", version, server.WithToolCapabilities(true))

	visibleTools := toolRegistry.VisibleTools()
	for _, t := range visibleTools {
		t.RegisterMCP(s)
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
