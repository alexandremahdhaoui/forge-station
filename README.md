# forge-workspace

**Kubernetes operator and REST API that manage development workspace pods with persistent storage and git repository cloning.**

> "I need on-demand development environments in Kubernetes. Each workspace needs its own pod, persistent volume, and cloned repositories. I don't want to write YAML for every workspace -- I want a single CR, a REST call, or a CLI command." -- Platform Engineer

## What problem does forge-workspace solve?

Development teams need reproducible, isolated workspaces running in Kubernetes. Creating a workspace requires coordinating a Pod, PersistentVolumeClaim, Service, and git clone operations. Without automation, each workspace demands 100+ lines of YAML and manual credential management. forge-workspace reduces this to a single Workspace custom resource. The controller reconciles the CR into a running pod with mounted storage and cloned repositories. A REST API and CLI provide programmatic access.

## Quick Start

Install CRDs, deploy the controller, and create a workspace:

```bash
# 1. Install CRDs
helm install forge-workspace-crds ./charts/forge-workspace-crds

# 2. Deploy the controller
helm install forge-workspace-controller ./charts/forge-workspace-controller \
  -n forge-workspace-system --create-namespace

# 3. Create a workspace
cat <<EOF | kubectl apply -f -
apiVersion: forge.amahdha.com/v1alpha1
kind: Workspace
metadata:
  name: my-workspace
spec:
  image: ubuntu:24.04
  storage:
    storageClassName: standard
    size: 20Gi
  repos:
    - url: https://github.com/example/repo.git
      ref:
        branch: main
      path: repo
EOF

# 4. Check status
kubectl get workspace my-workspace -o yaml
```

## How does it work?

```
                      +------------------+
                      |  Workspace CR    |
                      +--------+---------+
                               |
                    +----------v-----------+
                    | WorkspaceReconciler   |
                    +--+-------+--------+--+
                       |       |        |
              +--------v+  +---v---+  +-v--------+
              |   PVC    |  |  Pod  |  | Service  |
              | (storage)|  | (dev) |  |  (SSH)   |
              +----------+  +-------+  +----------+

  REST API (forge-ws-api)          CLI (forge-ws)
       :8080                     create|list|get|delete
         |                       status|suspend|resume
         v                              |
  +------+-------+                      v
  | WorkspaceService <------------------+
  +------+-------+
         |
  +------v-------+
  | K8s Adapters |
  +--------------+
```

The controller watches Workspace CRs. On each reconciliation, it creates a PVC, a Pod (mounting the PVC at `/workspace`), and a ClusterIP Service (port 22). The REST API and CLI call the same WorkspaceService layer. Portfolios group workspaces by label selector.

See [DESIGN.md](DESIGN.md) for the full technical design.

## Table of Contents

- [What CRDs does forge-workspace define?](#what-crds-does-forge-workspace-define)
- [What REST API endpoints exist?](#what-rest-api-endpoints-exist)
- [How do I use the CLI?](#how-do-i-use-the-cli)
- [How do I configure git credentials?](#how-do-i-configure-git-credentials)
- [How do I build and test?](#how-do-i-build-and-test)
- [FAQ](#faq)
- [Documentation](#documentation)
- [Contributing](#contributing)
- [License](#license)

## What CRDs does forge-workspace define?

**API Group:** `forge.amahdha.com/v1alpha1`

| CRD | Purpose |
|-----|---------|
| `Workspace` | Declares a development workspace with image, storage, and git repos |
| `Portfolio` | Groups workspaces by label selector, tracks count and names |

**Workspace phases:** `Pending`, `Running`, `Stopped`, `Failed`

**Conditions:** `Ready`, `Reconciling`, `Stalled`

**Workspace spec fields:**

| Field | Type | Description |
|-------|------|-------------|
| `suspend` | bool | Pause reconciliation. Default: false |
| `image` | string | Container image for the workspace pod |
| `initScript` | string | Post-start script |
| `repos[]` | RepoRef | Git repositories to clone |
| `storage` | StorageSpec | PVC storage class and size |
| `compoRef` | CompoRef | Composition repository reference |
| `serviceAccountName` | string | Workload identity |

## What REST API endpoints exist?

The API server (`forge-ws-api`) listens on port 8080. All endpoints accept an optional `?namespace=` query parameter (default: `default`).

| Method | Path | Description |
|--------|------|-------------|
| POST | `/api/v1/workspaces` | Create a workspace |
| GET | `/api/v1/workspaces` | List all workspaces |
| GET | `/api/v1/workspaces/{name}` | Get workspace details |
| DELETE | `/api/v1/workspaces/{name}` | Delete a workspace |
| PUT | `/api/v1/workspaces/{name}/suspend` | Suspend a workspace |
| PUT | `/api/v1/workspaces/{name}/resume` | Resume a workspace |
| GET | `/api/v1/portfolios` | List all portfolios |
| GET | `/api/v1/portfolios/{name}` | Get portfolio details |

OpenAPI spec: [api/forge-workspace.v1.yaml](api/forge-workspace.v1.yaml)

## How do I use the CLI?

`forge-ws` runs as a CLI or Model Context Protocol (MCP) server.

```bash
# CLI mode
forge-ws create --name dev --storage-class standard --storage-size 20Gi --repos https://github.com/example/repo.git
forge-ws list
forge-ws get dev
forge-ws status dev
forge-ws suspend dev
forge-ws resume dev
forge-ws delete dev

# MCP server mode (stdio transport)
forge-ws --mcp
```

**MCP tools:** `create-workspace`, `list-workspaces`, `get-workspace`, `delete-workspace`, `suspend-workspace`, `resume-workspace`

Set `NAMESPACE` env var to target a specific namespace (default: `default`).

## How do I configure git credentials?

forge-workspace uses a FluxCD-inspired credential pattern. Repository credentials reference Kubernetes Secrets via `secretRef`.

**Supported auth methods:** SSH key, HTTPS basic (username/password), HTTPS bearer token.

```yaml
# Create a Secret with SSH credentials
apiVersion: v1
kind: Secret
metadata:
  name: repo-ssh-creds
data:
  identity: <base64-encoded-private-key>
  known_hosts: <base64-encoded-known-hosts>

# Reference it in the Workspace
spec:
  repos:
    - url: git@github.com:example/repo.git
      secretRef:
        name: repo-ssh-creds
```

The REST API and CLI accept inline `AuthInput`. The service creates Secrets automatically and sets `secretRef` on the CR. Responses expose `hasCredential: true/false` without leaking secret data.

**Secret data keys:** `identity`, `known_hosts`, `username`, `password`, `bearerToken`, `caFile`

## How do I build and test?

Requires: Go 1.25+, [forge](https://github.com/alexandremahdhaoui/forge) CLI, Docker, Kind (for e2e).

```bash
# Generate code (CRDs, deepcopy, mocks, OpenAPI)
forge build generate-all
forge build generate-openapi

# Build binaries
forge build forge-ws-controller
forge build forge-ws-api
forge build forge-ws

# Run all tests
forge test-all

# Run individual test stages
forge test run lint
forge test run unit
forge test run e2e
```

**Helm charts (3):**

| Chart | Content |
|-------|---------|
| `forge-workspace-crds` | CRD definitions |
| `forge-workspace-controller` | Controller manager deployment |
| `forge-workspace-api` | REST API server deployment |

## FAQ

**Q: What happens when I suspend a workspace?**
The reconciler stops processing the Workspace CR. Existing Pod, PVC, and Service remain. The phase changes to `Stopped` and the `Stalled` condition becomes true.

**Q: How does the controller track owned resources?**
The reconciler sets `ownerReferences` on Pods, PVCs, and Services via `controllerutil.SetControllerReference`. Kubernetes garbage-collects them on Workspace deletion.

**Q: Can I use an existing Secret for credentials?**
Yes. Set `auth.existingSecretName` in the REST/CLI request. The service skips Secret creation and uses the existing Secret name as `secretRef`.

**Q: What image does the workspace pod use by default?**
`forge-ws-dev:latest`. Override with `spec.image` on the Workspace CR.

**Q: Does the controller support leader election?**
Yes. The controller manager uses `LeaderElection: true` with ID `forge-ws-controller`.

**Q: How does Portfolio grouping work?**
A Portfolio CR defines a label selector. The PortfolioReconciler lists matching Workspaces and updates `status.workspaceCount` and `status.workspaceNames`.

## Documentation

| Document | Audience | Content |
|----------|----------|---------|
| [README.md](README.md) | Users | Quick start, API reference, CLI usage |
| [DESIGN.md](DESIGN.md) | Developers | Architecture, data model, design decisions |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Contributors | Build, test, commit conventions |
| [api/forge-workspace.v1.yaml](api/forge-workspace.v1.yaml) | API consumers | OpenAPI 3.0 spec |

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

Apache License 2.0. See [LICENSE](LICENSE).
