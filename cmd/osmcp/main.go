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

	lsTool := tools.NewLsTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(lsTool)

	catTool := tools.NewCatTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(catTool)

	statTool := tools.NewStatTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(statTool)

	wcTool := tools.NewWcTool(policyEngine, envelopeBuilder)
	toolRegistry.Register(wcTool)

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
		} else if t.Name() == "ls" {
			lt := t.(interface {
				Execute(context.Context, contracts_phase1.LsArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("ls",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the directory to list."),
				),
				mcp.WithBoolean("recursive",
					mcp.Description("If true, walks subdirectories."),
				),
				mcp.WithNumber("max_depth",
					mcp.Description("Maximum depth for recursive listing."),
				),
				mcp.WithBoolean("show_hidden",
					mcp.Description("If true, includes files and directories starting with a dot (.)."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.LsArgs{
					Recursive:  false,
					MaxDepth:   1,
					ShowHidden: false,
				}

				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if rec, ok := argsMap["recursive"].(bool); ok {
						args.Recursive = rec
					}
					if maxDepth, ok := argsMap["max_depth"].(float64); ok {
						args.MaxDepth = int(maxDepth)
					}
					if showHidden, ok := argsMap["show_hidden"].(bool); ok {
						args.ShowHidden = showHidden
					}
				}

				envelope := lt.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}

				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "cat" {
			ct := t.(interface {
				Execute(context.Context, contracts_phase1.CatArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("cat",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the file to read."),
				),
				mcp.WithNumber("start_line",
					mcp.Description("Line number to start reading from (1-indexed)."),
				),
				mcp.WithNumber("end_line",
					mcp.Description("Line number to stop reading at (inclusive). If omitted, reads to EOF."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.CatArgs{
					StartLine: 1,
				}

				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
					if sl, ok := argsMap["start_line"].(float64); ok {
						args.StartLine = int(sl)
					}
					if el, ok := argsMap["end_line"].(float64); ok {
						val := int(el)
						args.EndLine = &val
					}
				}

				envelope := ct.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}

				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "stat" {
			st := t.(interface {
				Execute(context.Context, contracts_phase1.StatArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("stat",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the file or directory."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.StatArgs{}

				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
				}

				envelope := st.Execute(ctx, args)
				resBytes, err := json.Marshal(envelope)
				if err != nil {
					return mcp.NewToolResultError(fmt.Sprintf("failed to serialize response: %v", err)), nil
				}

				return mcp.NewToolResultText(string(resBytes)), nil
			})
		} else if t.Name() == "wc" {
			wt := t.(interface {
				Execute(context.Context, contracts_phase1.WcArgs) contracts.Envelope
			})

			mcpTool := mcp.NewTool("wc",
				mcp.WithDescription(t.Description()),
				mcp.WithString("path",
					mcp.Required(),
					mcp.Description("Absolute path to the file."),
				),
			)

			s.AddTool(mcpTool, func(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
				args := contracts_phase1.WcArgs{}

				argsMap, ok := request.Params.Arguments.(map[string]interface{})
				if ok {
					if path, ok := argsMap["path"].(string); ok {
						args.Path = path
					}
				}

				envelope := wt.Execute(ctx, args)
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
