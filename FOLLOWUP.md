# FOLLOWUP

## Mission

On demand development workspaces in Kubernetes. One custom resource becomes a
pod with storage and cloned repositories, reachable by CRD, REST or CLI.

## Now

- [ ] Nothing in flight.

## Next

- [ ] **The name is inconsistent in four places.** The repo was renamed from
      forge-workspace to forge-station on 2026-08-19, because forge-workspace
      was needed for a different concern and is now forge-factory. The Go module
      path followed. These did not, deliberately, because each one breaks
      something already deployed:

      | Still says workspace | Renaming it breaks |
      |---|---|
      | Helm charts `forge-workspace-crds`, `-controller`, `-api` | `helm upgrade` for any existing release |
      | Binaries `forge-ws`, `forge-ws-api`, `forge-ws-controller` | Anyone with them installed, and `forge ws` in forge |
      | `api/forge-workspace.v1.yaml` | Nothing, it is just a filename |
      | The Kubernetes API group and `Workspace` kind | Every custom resource already applied to a cluster |

      The API group and kind should probably never change. The charts and
      binaries can, with a deprecation cycle.

- [ ] **`forge ws` in the forge repo points here.** `cmd/forge/ws.go` is a
      passthrough. Its subcommand name is unrelated to this rename but reads
      oddly now.

## Deferred

- Nothing.

## Decided

- The Go module path is `github.com/alexandremahdhaoui/forge-station` as of
  2026-08-19. GitHub redirects the old repository URL, so the old module path
  still resolves, but nothing should rely on that.
