# forge-workspace Design

**A Kubernetes operator and REST API that reconcile Workspace custom resources into running pods with persistent storage and cloned git repositories.**

## Problem Statement

Development teams need on-demand, isolated workspaces in Kubernetes. Each workspace requires a Pod, PersistentVolumeClaim, Service, and git clone setup. Without automation, provisioning one workspace demands 100+ lines of YAML, manual credential management, and no lifecycle tracking. Teams lack a unified API to create, suspend, resume, and delete workspaces. They also need to group workspaces into logical portfolios for organizational visibility.

forge-workspace solves this by defining 2 CRDs (Workspace, Portfolio) and reconciling them into Kubernetes-native resources. A REST API and CLI expose the same operations to humans and AI agents.

## Tenets

Listed in priority order. When tenets conflict, higher-ranked wins.

1. **Declarative first.** Workspace and Portfolio CRDs define desired state. The controller reconciles actual state.
2. **Credential safety.** Secret data never appears in CRD specs or API responses. The service creates Secrets separately and references them via `secretRef`.
3. **Single responsibility per layer.** Adapters wrap K8s clients. Services hold business logic. Reconcilers drive the control loop. REST handlers translate HTTP.
4. **Suspend without destroy.** Suspending a workspace pauses reconciliation. Existing Pod, PVC, and Service remain intact.
5. **Composability.** The CLI doubles as an MCP server. The REST API uses OpenAPI codegen. Any client can consume the workspace lifecycle.

## Requirements

1. A user creates a Workspace CR and the controller provisions a Pod, PVC, and Service within 30 seconds.
2. A user suspends a workspace. The controller stops reconciliation but preserves resources.
3. A user resumes a workspace. The controller re-enters the reconciliation loop.
4. A user deletes a workspace. Kubernetes garbage-collects owned sub-resources via ownerReferences.
5. A user creates a workspace with git repository credentials. The service creates a Secret and references it on the CR.
6. A user queries the REST API. Responses never expose secret data, only `hasCredential: true/false`.
7. A user creates a Portfolio with a label selector. The controller tracks matching workspace count and names.
8. A user runs `forge-ws --mcp`. The CLI exposes workspace operations as MCP tools over stdio.

## Out of Scope

- Workspace networking policies or ingress configuration.
- Multi-cluster workspace federation.
- Git clone execution inside the pod (init containers or sidecars). The CRD stores repo refs; the pod image handles cloning.
- User authentication and authorization for the REST API.
- Workspace resource quotas or limits (delegated to Kubernetes LimitRange/ResourceQuota).

## Success Criteria

| Metric | Target |
|--------|--------|
| Workspace reconciliation time (CR to Running pod) | < 30 seconds (excluding image pull) |
| Sub-resource cleanup on deletion | 100% via ownerReferences |
| Secret data exposure in API responses | 0 fields |
| CRD condition accuracy | Phase matches pod status within 1 reconciliation cycle |
| E2e test pass rate | 100% on Kind cluster |

## Proposed Design

### Architecture

```
+--------------------------------------------------+
|                  Kubernetes Cluster               |
|                                                   |
|  +--------------------+  +---------------------+ |
|  | forge-ws-controller|  |   forge-ws-api      | |
|  |                    |  |                      | |
|  | WorkspaceReconciler|  |  REST Handler (:8080)| |
|  | PortfolioReconciler|  |    |                 | |
|  |   |                |  |    v                 | |
|  |   v                |  | WorkspaceService    | |
|  | K8s API Server <---|--|--PortfolioService    | |
|  +--------------------+  +---------------------+ |
|                                                   |
|  +------+ +-----+ +---------+ +---------------+  |
|  | Pod  | | PVC | | Service | | Workspace CR  |  |
|  +------+ +-----+ +---------+ +---------------+  |
|  +---------------+                                |
|  | Portfolio CR  |                                |
|  +---------------+                                |
+--------------------------------------------------+

    +------------------+
    |    forge-ws      |
    | CLI + MCP server |
    |   |              |
    |   v              |
    | WorkspaceService |
    +------------------+
```

3 binaries:
- `forge-ws-controller` -- Kubernetes controller manager. Runs WorkspaceReconciler and PortfolioReconciler.
- `forge-ws-api` -- REST API server on port 8080. Delegates to WorkspaceService and PortfolioService.
- `forge-ws` -- CLI and MCP server. 7 subcommands: create, list, get, delete, status, suspend, resume.

### Workspace Lifecycle State Diagram

```
                    create
                      |
                      v
  +----------+   pod ready   +----------+
  | Pending  |-------------->| Running  |
  +----------+               +----------+
       |                          |
       | pod failed               | suspend
       v                          v
  +----------+               +----------+
  | Failed   |               | Stopped  |
  +----------+               +----------+
                                  |
                                  | resume
                                  v
                             +----------+
                             | Pending  |
                             +----------+
```

Phase transitions:
- `Pending` -- CR created, waiting for pod readiness.
- `Running` -- Pod is ready (all containers pass readiness check).
- `Failed` -- Pod status is `Failed`.
- `Stopped` -- `spec.suspend` is true. Reconciler skips processing.

### Reconciliation Sequence (Workspace)

```
  Controller Manager          WorkspaceReconciler          K8s API
       |                            |                        |
       |--- watch event ----------->|                        |
       |                            |-- GET Workspace CR --->|
       |                            |<-- Workspace ----------|
       |                            |                        |
       |                            |-- check spec.suspend   |
       |                            |   (if true: update     |
       |                            |    status, return)     |
       |                            |                        |
       |                            |-- ensurePVC ---------->|
       |                            |   (create if missing)  |
       |                            |                        |
       |                            |-- ensurePod ---------->|
       |                            |   (create if missing)  |
       |                            |                        |
       |                            |-- ensureService ------>|
       |                            |   (create if missing)  |
       |                            |                        |
       |                            |-- GET Pod ------------>|
       |                            |<-- Pod status ---------|
       |                            |                        |
       |                            |-- UPDATE status ------>|
       |                            |   (phase, conditions,  |
       |                            |    podName, pvcName,   |
       |                            |    serviceURL)         |
       |                            |                        |
```

### Credential Flow

```
  Client (REST/CLI)          WorkspaceService          SecretAdapter      K8s API
       |                          |                         |                |
       |-- CreateWorkspace ------>|                         |                |
       |   (AuthInput inline)     |                         |                |
       |                          |-- buildSecret --------->|                |
       |                          |   (from AuthInput)      |-- CREATE ----->|
       |                          |                         |<-- Secret -----|
       |                          |                         |                |
       |                          |-- set secretRef on CR   |                |
       |                          |                         |                |
       |                          |-- CREATE Workspace CR --|--------------->|
       |                          |                         |                |
       |<-- WorkspaceDetail ------|                         |                |
       |   (hasCredential: true,  |                         |                |
       |    no secret data)       |                         |                |
```

Secret naming: `ws-{name}-repo-{index}-creds` for repos, `ws-{name}-compo-creds` for compositions. If `auth.existingSecretName` is set, the service skips creation.

## Technical Design

### CRD Types (pkg/v1alpha1)

**API Group:** `forge.amahdha.com/v1alpha1`

```go
type Workspace struct {
    Spec   WorkspaceSpec   // suspend, image, initScript, repos[], storage, compoRef, serviceAccountName
    Status WorkspaceStatus // phase, podName, pvcName, serviceURL, observedGeneration, conditions[]
}

type WorkspaceSpec struct {
    Suspend            bool
    Image              string
    InitScript         string
    Repos              []RepoRef    // url, ref{branch,tag,commit}, path, provider, secretRef
    Storage            StorageSpec  // storageClassName, size
    CompoRef           *CompoRef   // url, ref, provider, secretRef
    ServiceAccountName string
}

type Portfolio struct {
    Spec   PortfolioSpec   // suspend, selector, description
    Status PortfolioStatus // workspaceCount, workspaceNames[], observedGeneration, conditions[]
}
```

**WorkspacePhase values:** `Pending`, `Running`, `Stopped`, `Failed`

**Condition types:** `Ready`, `Reconciling`, `Stalled`

### REST API

9 endpoints defined in [api/forge-workspace.v1.yaml](api/forge-workspace.v1.yaml). Code generated with oapi-codegen into `internal/driver/rest/zz_generated.oapi-codegen.go`.

| Method | Path | Handler | Service Method |
|--------|------|---------|----------------|
| POST | `/api/v1/workspaces` | CreateWorkspace | WorkspaceService.Create |
| GET | `/api/v1/workspaces` | ListWorkspaces | WorkspaceService.List |
| GET | `/api/v1/workspaces/{name}` | GetWorkspace | WorkspaceService.Get |
| DELETE | `/api/v1/workspaces/{name}` | DeleteWorkspace | WorkspaceService.Delete |
| PUT | `/api/v1/workspaces/{name}/suspend` | SuspendWorkspace | WorkspaceService.Suspend |
| PUT | `/api/v1/workspaces/{name}/resume` | ResumeWorkspace | WorkspaceService.Resume |
| GET | `/api/v1/portfolios` | ListPortfolios | PortfolioService.List |
| GET | `/api/v1/portfolios/{name}` | GetPortfolio | PortfolioService.Get |

Health endpoints: `GET /healthz`, `GET /readyz`.

### Adapter Interfaces (internal/adapter)

```go
type WorkspaceAdapter interface {
    Create(ctx, ws)           error
    Get(ctx, namespace, name) (*Workspace, error)
    List(ctx, namespace)      (*WorkspaceList, error)
    Update(ctx, ws)           error
    Delete(ctx, namespace, name) error
    UpdateStatus(ctx, ws)     error
}

type PortfolioAdapter interface {
    Get(ctx, namespace, name)             (*Portfolio, error)
    List(ctx, namespace)                  (*PortfolioList, error)
    UpdateStatus(ctx, p)                  error
    ListWorkspacesBySelector(ctx, ns, sel) (*WorkspaceList, error)
}

type SecretAdapter interface {
    Create(ctx, secret)          error
    Get(ctx, namespace, name)    (*Secret, error)
    Delete(ctx, namespace, name) error
}
```

Each adapter wraps a `controller-runtime/pkg/client.Client`.

### Services (internal/controller/service)

**WorkspaceService** -- Create, Get, List, Delete, Suspend, Resume. Create resolves inline AuthInput into Secrets. All responses use sanitized types (`WorkspaceDetail`, `WorkspaceSummary`). Secret data is masked as `hasCredential: bool`.

**PortfolioService** -- Get, List. Thin pass-through to PortfolioAdapter.

### Reconcilers (internal/controller/reconciler)

**WorkspaceReconciler** -- Watches Workspace CRs. Owns Pod, PVC, Service. Steps: fetch CR, check suspend, set Reconciling condition, ensurePVC, ensurePod, ensureService, read pod status, update workspace status.

**PortfolioReconciler** -- Watches Portfolio CRs. Steps: fetch CR, check suspend, convert label selector, list matching workspaces, update status with count and sorted names.

Both reconcilers set 3 conditions: `Ready`, `Reconciling`, `Stalled`.

### Package Catalog

**Public packages (pkg/):**

| Package | Purpose |
|---------|---------|
| `pkg/v1alpha1` | CRD types, GroupVersion, SchemeBuilder, deepcopy |

**Internal packages (internal/):**

| Package | Purpose |
|---------|---------|
| `internal/adapter` | WorkspaceAdapter, PortfolioAdapter, SecretAdapter interfaces + K8s implementations |
| `internal/controller/service` | WorkspaceService, PortfolioService business logic |
| `internal/controller/reconciler` | WorkspaceReconciler, PortfolioReconciler, condition constants |
| `internal/driver/rest` | APIHandler (REST), generated server code |
| `internal/types` | WorkspaceState, PodStatus, StorageInfo, CredentialType |
| `internal/util/mocks/mockadapter` | Generated mocks for adapter interfaces |
| `internal/util/mocks/mockcontroller` | Generated mocks for service interfaces |
| `internal/util/mocks/mockclient` | Generated mocks for K8s client |

## Design Patterns

**Adapter pattern.** Each K8s resource type has a dedicated adapter interface. Adapters encapsulate `client.Client` calls. Services depend on adapter interfaces, not concrete clients. This enables unit testing with mockery-generated mocks.

**Credential separation (FluxCD-inspired).** CRD specs reference Secrets via `secretRef.name`. The service layer creates Secrets from inline auth input and sets the reference. This ensures no credential data enters the CRD spec. API responses expose only `hasCredential: bool`.

**Owner reference garbage collection.** The reconciler sets `controllerutil.SetControllerReference` on every sub-resource (Pod, PVC, Service). Kubernetes cascading deletion removes sub-resources when the Workspace CR is deleted.

**Strict OpenAPI codegen.** The REST handler implements `StrictServerInterface` generated by oapi-codegen. Type-safe request/response structs eliminate manual JSON marshaling.

## Alternatives Considered

### Do nothing

Developers write raw Pod, PVC, and Service YAML for each workspace. This works for 1-2 workspaces. At 10+ workspaces, manual management becomes error-prone. Credential handling is ad-hoc. No lifecycle tracking exists. Rejected because it does not scale.

### Operator without REST API

A controller-only approach requires `kubectl` for all operations. This blocks non-cluster-admin users and AI agents from workspace management. Rejected because programmatic access is a core requirement.

### Helm-only approach (no operator)

Helm charts can template Pod/PVC/Service. But Helm does not track workspace phase, does not reconcile drift, and does not support suspend/resume. Rejected because lifecycle management requires a control loop.

## Risks and Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Secret leak in CRD spec | Credential exposure | Service creates Secrets separately; `secretRef` is the only CRD field |
| Orphaned Secrets on workspace deletion | Resource leak | Secrets created by the service lack ownerReferences today; tracked for future enhancement |
| Pod stuck in Pending (image pull, scheduling) | Workspace stays in Pending phase | Condition messages surface the root cause; users can inspect pod events |
| Stale portfolio status | Incorrect workspace count | PortfolioReconciler runs on every Portfolio event; workspace changes trigger re-reconciliation via watches |

## Testing Strategy

| Stage | Tool | Scope |
|-------|------|-------|
| `lint-tags` | go-lint-tags | All test files have build tags |
| `lint-licenses` | go-lint-licenses | All files have Apache 2.0 headers |
| `lint` | golangci-lint | Code quality, style |
| `unit` | go test | Adapter, service, reconciler logic with generated mocks |
| `e2e` | Kind + Helm | Full cluster: install CRDs, deploy controller/API, create/get/delete workspaces |

E2e test environment: Kind cluster, local container registry, 3 Helm chart installations. Timeout: 20 minutes.

Run all stages: `forge test-all`

## FAQ

**Q: Why 2 separate binaries for the controller and API?**
The controller manager uses leader election and runs reconciliation loops. The API server is stateless and horizontally scalable. Separating them allows independent scaling and deployment.

**Q: Why not use Deployments instead of bare Pods for workspaces?**
Workspaces are single-instance, stateful environments. A Deployment would add restart semantics that conflict with suspend/resume behavior. A bare Pod with ownerReferences provides the right abstraction.

**Q: How does the MCP server relate to the CLI?**
`forge-ws` uses `enginecli.Bootstrap`. With `--mcp`, it starts an MCP server over stdio. Without `--mcp`, it runs CLI subcommands. Both modes call the same `WorkspaceService`.

**Q: Why generate REST server code from OpenAPI?**
Manual HTTP handlers are error-prone. oapi-codegen produces a `StrictServerInterface` with typed request/response structs. The handler implements the interface, and the compiler catches mismatches.

## Appendix: Workspace CR Example

```yaml
apiVersion: forge.amahdha.com/v1alpha1
kind: Workspace
metadata:
  name: dev-workspace
  labels:
    team: platform
spec:
  image: ubuntu:24.04
  storage:
    storageClassName: standard
    size: 20Gi
  repos:
    - url: https://github.com/example/app.git
      ref:
        branch: main
      path: app
      secretRef:
        name: app-repo-creds
    - url: https://github.com/example/config.git
      ref:
        tag: v1.2.0
      path: config
  initScript: |
    apt-get update && apt-get install -y vim
  serviceAccountName: dev-sa
status:
  phase: Running
  podName: ws-dev-workspace-pod
  pvcName: ws-dev-workspace-pvc
  serviceURL: ws-dev-workspace-svc.default.svc.cluster.local
  conditions:
    - type: Ready
      status: "True"
      reason: PodReady
      message: Pod is running and ready
```
