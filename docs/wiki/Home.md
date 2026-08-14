# osmcp Wiki

A typed, policy-controlled OS capability layer for AI agents.

## Pages

- [Architecture](./Architecture.md) — Layers, two-tier tool model, MCP wiring
- [Policy Engine](./Policy.md) — TOML config format, evaluation order, scope hierarchy
- [Tools](./Tools.md) — Per-tool argument schemas and structured output shapes
- [Threat Model](./ThreatModel.md) — Trust boundary, attack surface, scope of protection
- [Audit Log](./AuditLog.md) — Log format, destinations, what is and isn't logged
- [Roadmap](./Roadmap.md) — v1 / v2 / v3 feature breakdown

## Quick Start

```bash
brew install osmcp  # (once published)

# or build from source
go build ./cmd/osmcp

# validate policy and start
osmcp --policy ~/.config/osmcp/policy.toml
```

## Design Principles

1. **Typed over raw** — structured tool arguments, not shell strings
2. **Capability-based visibility** — agents only see what policy grants
3. **Pure Go Tier 1** — typed tools are built in pure Go (stdlib/libs) for zero dependencies and absolute cross-platform portability. `os/exec` is reserved for v2 `run_script`.
4. **Uniform response** — one JSON envelope shape for every tool, always
5. **Fail closed** — deny by default, explicit grants required
6. **Spec-first** — contracts and specs written before implementation
