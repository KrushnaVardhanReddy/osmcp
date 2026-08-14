.PHONY: build test lint e2e validate-policy

build:
	go build -o bin/osmcp ./cmd/osmcp

test:
	go test -race ./internal/...

e2e: build
	go test -race ./e2e/...

lint:
	go vet ./...

validate-policy:
	./bin/osmcp --validate --policy .osmcp/policy.toml
