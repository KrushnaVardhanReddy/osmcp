# Contributing to `osmcp`

Thank you for your interest in contributing! `osmcp` relies heavily on an AI-assisted open source development workflow, which means we have a strict "Spec and Contract First" philosophy.

## Spec and Contract First

We **never** write the Go implementation of a tool without first writing the Specification and the Go Interface Contract. This separates the design phase from the implementation phase and ensures that when AI agents write the code, they know exactly what the boundaries and data shapes should be.

### 1. Write the Spec
Create a markdown file in `docs/specs/` detailing the exact parameters, policy constraints, execution behavior, and JSON output shape for the new tool.

### 2. Write the Contract
Create a pure Go interface file in `docs/contracts/` that defines the parameter structs (`Args`), the output structs (`Data`), and the `Tool` interface.

### 3. Implement
Only after the spec and contract are approved can you create the implementation in `internal/tools/` and its accompanying unit tests.

### 4. Register
All new tools must be registered with the central registry in `cmd/osmcp/main.go` by calling `toolRegistry.Register(...)`.

### 5. E2E Testing
Add end-to-end testing in `e2e/` to ensure the tool interacts correctly with the `PolicyEngine` over the JSON-RPC stdio layer.
