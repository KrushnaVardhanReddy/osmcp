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

	toolRegistry.Register(tools.NewGrepTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewTreeTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewHeadTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewTailTool(policyEngine, envelopeBuilder))
	toolRegistry.Register(tools.NewDuTool(policyEngine, envelopeBuilder))

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
		} else if t.Name() == "tree" {
			tt := t.(interface {
				Execute(context.Context, contracts_phase1.TreeArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("tree",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the root directory."),
				),
				mcp.WithNumber("max_depth",
					mcp.Description("Maximum depth to display."),
				),
				mcp.WithBoolean("show_hidden",
					mcp.Description("Include files and dirs starting with a dot."),
				),
				mcp.WithBoolean("dirs_only",
					mcp.Description("If true, only show directories."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.TreeArgs{
					MaxDepth: 3,
					ShowHidden: false,
					DirsOnly: false,
				}

				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if maxDepth, ok := argsMap["max_depth"].(float64); ok {
						args.MaxDepth = int(maxDepth)
					}
					if showHidden, ok := argsMap["show_hidden"].(bool); ok {
						args.ShowHidden = showHidden
					}
					if dirsOnly, ok := argsMap["dirs_only"].(bool); ok {
						args.DirsOnly = dirsOnly
					}
				}

				envelope := tt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}
				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "head" {
			ht := t.(interface {
				Execute(context.Context, contracts_phase1.HeadArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("head",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the file."),
				),
				mcp.WithNumber("lines",
					mcp.Description("Number of lines to return from the top."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.HeadArgs{
					Lines: 10,
				}
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if lines, ok := argsMap["lines"].(float64); ok {
						args.Lines = int(lines)
					}
				}
				envelope := ht.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}
				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "tail" {
			tt := t.(interface {
				Execute(context.Context, contracts_phase1.TailArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("tail",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the file."),
				),
				mcp.WithNumber("lines",
					mcp.Description("Number of lines to return from the end."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.TailArgs{
					Lines: 10,
				}
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if lines, ok := argsMap["lines"].(float64); ok {
						args.Lines = int(lines)
					}
				}
				envelope := tt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}
				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "du" {
			dt := t.(interface {
				Execute(context.Context, contracts_phase1.DuArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("du",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the directory or file."),
				),
				mcp.WithNumber("max_depth",
					mcp.Description("How deep to break down usage by subdirectory."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.DuArgs{
					MaxDepth: 1,
				}
				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if maxDepth, ok := argsMap["max_depth"].(float64); ok {
						args.MaxDepth = int(maxDepth)
					}
				}
				envelope := dt.Execute(ctx, args)
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
