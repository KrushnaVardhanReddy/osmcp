package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
	"github.com/osmcp/osmcp/docs/contracts/cross_cutting"
	contracts_phase1 "github.com/osmcp/osmcp/docs/contracts/phase-1"
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

	grepTool := tools.NewGrepTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(grepTool)

	gitStatusTool := tools.NewGitStatusTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitStatusTool)

	gitDiffTool := tools.NewGitDiffTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitDiffTool)

	gitLogTool := tools.NewGitLogTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(gitLogTool)

	s := server.NewMCPServer("osmcp", "0.1.0", server.WithToolCapabilities(true))

	visibleTools := toolRegistry.VisibleTools()
	for _, t := range visibleTools {
		if t.Name() == "grep" {
			gt := t.(interface {
				Execute(context.Context, contracts_phase1.GrepArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("grep",
				mcp.WithDescription(t.Description()),
				mcp.WithString("pattern",
					mcp.Required(),
					mcp.Description("Search pattern. Treated as a regex unless literal=true."),
				),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to file or directory."),
				),
				mcp.WithBoolean("recursive",
					mcp.Description("Search subdirectories recursively."),
				),
				mcp.WithBoolean("case_sensitive",
					mcp.Description("Case-sensitive search."),
				),
				mcp.WithBoolean("literal",
					mcp.Description("Treat pattern as a fixed string (not regex)."),
				),
				mcp.WithNumber("context_lines",
					mcp.Description("Lines of context around each match."),
				),
				mcp.WithString("include",
					mcp.Description("Glob to restrict searched files. e.g. '*.go'"),
				),
				mcp.WithString("exclude",
					mcp.Description("Glob to exclude files from search."),
				),
				mcp.WithNumber("max_matches",
					mcp.Description("Override policy max_matches for this call."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.GrepArgs{
                    Recursive: true,
                    CaseSensitive: true,
                    Literal: false,
                    ContextLines: 0,
                }

                // Parse arguments manually
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
                if ok {
                    if pat, ok := argsMap["pattern"].(string); ok {
                        args.Pattern = pat
                    }
                    if path, ok := argsMap["path"].(string); ok {
                        args.Path = path
                    }
                    if rec, ok := argsMap["recursive"].(bool); ok {
                        args.Recursive = rec
                    }
                    if caseSens, ok := argsMap["case_sensitive"].(bool); ok {
                        args.CaseSensitive = caseSens
                    }
                    if literal, ok := argsMap["literal"].(bool); ok {
                        args.Literal = literal
                    }
                    if cl, ok := argsMap["context_lines"].(float64); ok {
                        args.ContextLines = int(cl)
                    }
                    if inc, ok := argsMap["include"].(string); ok {
                        args.Include = inc
                    }
                    if exc, ok := argsMap["exclude"].(string); ok {
                        args.Exclude = exc
                    }
                    if max, ok := argsMap["max_matches"].(float64); ok {
                        args.MaxMatches = int(max)
                    }
                }

				envelope := gt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}

				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "git_status" {
			gt := t.(interface {
				Execute(context.Context, contracts_phase1.GitStatusArgs) contracts.Envelope
			})
			mcpTool := mcp.NewTool("git_status",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the repository root (or any subdirectory within it)."),
				),
			)
			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.GitStatusArgs{}
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
				}
				envelope := gt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}
				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "git_diff" {
			gt := t.(interface {
				Execute(context.Context, contracts_phase1.GitDiffArgs) contracts.Envelope
			})
			mcpTool := mcp.NewTool("git_diff",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the repository root."),
				),
				mcp.WithString("file",
					mcp.Description("Optional. Relative file path to restrict the diff to a single file."),
				),
				mcp.WithString("from_commit",
					mcp.Description("Optional. Starting commit hash (defaults to HEAD~1)."),
				),
				mcp.WithString("to_commit",
					mcp.Description("Optional. Ending commit hash (defaults to HEAD)."),
				),
			)
			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.GitDiffArgs{}
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if file, ok := argsMap["file"].(string); ok {
						args.File = &file
					}
					if fromCommit, ok := argsMap["from_commit"].(string); ok {
						args.FromCommit = &fromCommit
					}
					if toCommit, ok := argsMap["to_commit"].(string); ok {
						args.ToCommit = &toCommit
					}
				}
				envelope := gt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}
				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "git_log" {
			gt := t.(interface {
				Execute(context.Context, contracts_phase1.GitLogArgs) contracts.Envelope
			})
			mcpTool := mcp.NewTool("git_log",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the repository root."),
				),
				mcp.WithNumber("max_commits",
					mcp.Description("Maximum number of commits to return. (default: 20)"),
				),
				mcp.WithString("file",
					mcp.Description("Optional. If set, only return commits that touched this file path."),
				),
				mcp.WithString("branch",
					mcp.Description("Optional. Branch to read history from (defaults to current HEAD)."),
				),
			)
			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.GitLogArgs{
					MaxCommits: 20,
				}
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if maxCommits, ok := argsMap["max_commits"].(float64); ok {
						args.MaxCommits = int(maxCommits)
					}
					if file, ok := argsMap["file"].(string); ok {
						args.File = &file
					}
					if branch, ok := argsMap["branch"].(string); ok {
						args.Branch = &branch
					}
				}
				envelope := gt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}
				return mcp.NewToolResultText(string(resBytes)), nil
			})
		}
	}

	if err := server.ServeStdio(s); err != nil {
		fmt.Fprintf(os.Stderr, "server error: %v\n", err)
		os.Exit(1)
	}
}
