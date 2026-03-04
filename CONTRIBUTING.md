# Contributing to forge-workspace

**Build, test, and contribute to the Kubernetes workspace operator.**

## Quick Start

```bash
# Clone
git clone https://github.com/alexandremahdhaoui/forge-workspace.git
cd forge-workspace

# Generate code (CRDs, deepcopy, mocks, OpenAPI)
forge build generate-all
forge build generate-openapi

# Build all binaries
forge build forge-ws-controller
forge build forge-ws-api
forge build forge-ws

# Run all tests
forge test-all
```

Requires: Go 1.25+, [forge](https://github.com/alexandremahdhaoui/forge) v0.38.0+, Docker, Kind (for e2e tests).

## How do I structure commits?

Each commit uses an emoji prefix and a structured body.

| Emoji | Meaning |
|---|---|
| `✨` | New feature (feat:) |
| `🐛` | Bug fix (fix:) |
| `📖` | Documentation (docs:) |
| `🌱` | Misc (chore:, test:, and others) |
| `⚠` | Breaking changes -- never use without maintainer approval |

**Commit body format:**

```
✨ Short imperative summary (50 chars or less)

Why: Explain the motivation. What problem exists?

How: Describe the approach. What strategy did you choose?

What:

- pkg/foo/bar.go: description of change
- cmd/baz/main.go: description of change

How changes were verified:

- Unit tests for new logic (go test)
- forge test-all: all stages passed

Signed-off-by: Your Name <your@email.com>
```

Every commit requires `Signed-off-by`. Use `git commit -s` to add it automatically.

## How do I submit a pull request?

1. Create a feature branch from `main`.
2. Make changes. Run `forge test-all` before pushing.
3. Push and open a PR with this format:

```
## Summary
- 1-3 bullet points describing the change

## Test plan
- [ ] forge test-all passes
- [ ] Relevant new tests added
```

## How do I run tests?

| Stage | Command | What it checks |
|-------|---------|---------------|
| All stages | `forge test-all` | Build + lint + unit + e2e (full validation) |
| Lint (tags) | `forge test run lint-tags` | Test files have build tags |
| Lint (licenses) | `forge test run lint-licenses` | Apache 2.0 headers present |
| Lint (code) | `forge test run lint` | golangci-lint |
| Unit tests | `forge test run unit` | Adapter, service, reconciler logic |
| E2e tests | `forge test run e2e` | Kind cluster + Helm + full lifecycle |

E2e tests create a Kind cluster, push container images to a local registry, install 3 Helm charts, and run workspace lifecycle tests. Timeout: 20 minutes.

**List available targets:**

```bash
forge list build   # Build targets
forge list test    # Test stages
```

## How is the project structured?

```
forge-workspace/
  api/
    forge-workspace.v1.yaml        # OpenAPI 3.0 spec
  charts/
    forge-workspace-crds/          # CRD Helm chart
    forge-workspace-controller/    # Controller Helm chart
    forge-workspace-api/           # REST API Helm chart
  cmd/
    forge-ws/                      # CLI + MCP server binary
    forge-ws-api/                  # REST API server binary
    forge-ws-controller/           # Controller manager binary
  containers/
    forge-ws-api/                  # Containerfile for API
    forge-ws-controller/           # Containerfile for controller
  internal/
    adapter/                       # K8s client wrappers
    controller/
      reconciler/                  # Workspace + Portfolio reconcilers
      service/                     # Business logic services
    driver/
      rest/                        # REST handler + generated server
    types/                         # Internal domain types
    util/mocks/                    # Generated mocks (mockery)
  pkg/
    v1alpha1/                      # CRD types (public API)
  test/
    e2e/                           # End-to-end test suite
  hack/                            # Build scripts
  forge.yaml                       # Build + test configuration
```

## What does each CLI binary do?

| Binary | Purpose | Entry point |
|--------|---------|-------------|
| `forge-ws` | CLI (7 subcommands) + MCP server (`--mcp`) | `cmd/forge-ws/main.go` |
| `forge-ws-api` | REST API server on port 8080 | `cmd/forge-ws-api/main.go` |
| `forge-ws-controller` | Kubernetes controller manager | `cmd/forge-ws-controller/main.go` |

## What does each package do?

**Public packages (pkg/):**

| Package | Purpose |
|---------|---------|
| `pkg/v1alpha1` | Workspace, Portfolio CRD types. GroupVersion `forge.amahdha.com/v1alpha1`. SchemeBuilder. Deepcopy. |

**Internal packages (internal/):**

| Package | Purpose |
|---------|---------|
| `internal/adapter` | WorkspaceAdapter, PortfolioAdapter, SecretAdapter interfaces + `kubeXxxAdapter` implementations |
| `internal/controller/service` | WorkspaceService (create, get, list, delete, suspend, resume). PortfolioService (get, list). Request/response types. |
| `internal/controller/reconciler` | WorkspaceReconciler (PVC + Pod + Service). PortfolioReconciler (label selector matching). Condition constants. |
| `internal/driver/rest` | APIHandler implements `StrictServerInterface`. Generated server code from OpenAPI. |
| `internal/types` | WorkspaceState, PodStatus, StorageInfo, CredentialType domain types |
| `internal/util/mocks` | mockery-generated mocks for adapter, service, and client interfaces |

## What conventions must I follow?

**Build tags.** Every test file requires a build tag. The `lint-tags` stage enforces this.

**License headers.** Every `.go` file starts with the Apache 2.0 license header. See `hack/boilerplate.go.txt`. The `lint-licenses` stage enforces this.

**Generated files.** Files prefixed with `zz_generated.` are generated. Do not edit them. Regenerate with:

```bash
forge build generate-all       # deepcopy, CRDs, mocks
forge build generate-openapi   # REST server code
```

**Code formatting.** Run `forge build format` before committing.

**Mocks.** Generated by mockery into `internal/util/mocks/`. Regenerated as part of `generate-all`. Do not commit hand-written mocks.

**CRD changes.** Edit types in `pkg/v1alpha1/`. Run `forge build generate-all` to regenerate deepcopy methods and CRD YAML in `charts/forge-workspace-crds/templates/crds/`.

**Dependencies:** controller-runtime v0.22.4, k8s.io/api v0.34.2, go-sdk v1.4.0, forge v0.38.0.
