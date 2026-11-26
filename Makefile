# go-hamapping — make targets. `make help` lists them.
GO ?= go
GOFUMPT_VERSION ?= v0.12.0
GOLANGCI_LINT_VERSION ?= v2.14.0

.DEFAULT_GOAL := help

help: ## show this help
	@grep -E '^[a-zA-Z_-]+:.*?## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-14s %s\n", $$1, $$2}'

setup: ## install the pinned gofumpt and golangci-lint
	$(GO) install mvdan.cc/gofumpt@$(GOFUMPT_VERSION)
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)

build: ## go build ./...
	CGO_ENABLED=0 $(GO) build ./...

vet: ## go vet ./...
	$(GO) vet ./...

test: ## go test -count=1 ./...
	CGO_ENABLED=0 $(GO) test -count=1 ./...

race: ## go test -race (needs CGO_ENABLED=1)
	CGO_ENABLED=1 $(GO) test -race -count=1 ./...

fmt: ## gofumpt -w .
	gofumpt -w .

fmt-check: ## fail when a file is not gofumpt-clean
	@diff=$$(gofumpt -l .); if [ -n "$$diff" ]; then echo "gofumpt would rewrite:"; echo "$$diff"; exit 1; fi

lint: fmt-check ## gofumpt check + golangci-lint + go mod tidy -diff + go mod verify
	golangci-lint config verify
	golangci-lint run --timeout=5m ./...
	$(GO) mod tidy -diff
	$(GO) mod verify

# The main configuration excludes generated files leniently
# (exclusions.generated: lax); this run is SA1019 alone with that exclusion
# off, so a deprecated identifier in rows_gen.go or a test fails too.
deprecated-check: ## fail on a deprecated identifier, generated code and tests included (SA1019)
	golangci-lint config verify --config .golangci-deprecated.yaml
	golangci-lint run --config .golangci-deprecated.yaml --timeout=5m ./...

tidy-check: ## verify go.mod/go.sum are tidy and the module checksums verify
	$(GO) mod tidy -diff
	$(GO) mod verify

vuln: ## govulncheck ./...
	govulncheck ./...

# GOROOT explicit: go-licenses tells the standard library by its GOROOT,
# and a toolchain installed elsewhere than go-licenses was built for leaves
# every std package "without module info". modernc.org/mathutil's LICENSE
# (BSD-3-Clause, in go-fabric's graph) is not classified by its wording and
# is logged, without failing the check.
licenses: ## fail on a forbidden, restricted or reciprocal dependency licence (go-licenses)
	GOROOT=$$($(GO) env GOROOT) go-licenses check ./... --disallowed_types=forbidden,restricted,reciprocal

generate: ## regenerate rows_gen.go and the row table of docs/rows.md from rows/*.yaml
	$(GO) run ./internal/gen/cmd

cover: ## coverage profile of every package, as the floors count it
	CGO_ENABLED=0 $(GO) test -count=1 -timeout 300s -coverpkg=./... -coverprofile=coverage.out ./...

cover-check: cover ## fail when a package is below its floor (script/coverfloor/floors.txt)
	$(GO) run ./script/coverfloor coverage.out

FUZZTIME ?= 10s
fuzz-smoke: ## the fuzz target of the row loader, $(FUZZTIME)
	$(GO) test -run '^$$' -fuzz '^FuzzParse$$' -fuzztime $(FUZZTIME) ./internal/rowfile/

ci: build vet test lint deprecated-check ## everything the pipeline runs

.PHONY: help setup build vet test race fmt fmt-check lint deprecated-check tidy-check vuln licenses generate cover cover-check fuzz-smoke ci
