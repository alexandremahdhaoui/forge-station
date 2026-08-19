// Copyright 2024 Alexandre Mahdhaoui
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/alexandremahdhaoui/forge-station/internal/controller/service"
)

// runCLI parses subcommands and delegates to the workspace service.
func runCLI(svc service.WorkspaceService, namespace string) error {
	// Filter out --mcp and version flags (already handled by enginecli.Bootstrap).
	args := filterArgs(os.Args[1:])

	if len(args) == 0 {
		printUsage()
		return fmt.Errorf("no subcommand specified")
	}

	ctx := context.Background()
	subcmd := args[0]
	subArgs := args[1:]

	switch subcmd {
	case "create":
		return cliCreate(ctx, svc, namespace, subArgs)
	case "list":
		return cliList(ctx, svc, namespace)
	case "get":
		return cliGet(ctx, svc, namespace, subArgs)
	case "delete":
		return cliDelete(ctx, svc, namespace, subArgs)
	case "status":
		return cliStatus(ctx, svc, namespace, subArgs)
	case "suspend":
		return cliSuspend(ctx, svc, namespace, subArgs)
	case "resume":
		return cliResume(ctx, svc, namespace, subArgs)
	default:
		printUsage()
		return fmt.Errorf("unknown subcommand: %s", subcmd)
	}
}

func cliCreate(ctx context.Context, svc service.WorkspaceService, namespace string, args []string) error {
	fs := flag.NewFlagSet("create", flag.ContinueOnError)
	name := fs.String("name", "", "Workspace name (required)")
	image := fs.String("image", "", "Container image")
	storageClass := fs.String("storage-class", "", "Storage class (required)")
	storageSize := fs.String("storage-size", "", "Storage size, e.g. 20Gi (required)")
	repos := fs.String("repos", "", "Comma-separated list of git repository URLs")
	initScript := fs.String("init-script", "", "Post-start script")

	if err := fs.Parse(args); err != nil {
		return err
	}

	if *name == "" || *storageClass == "" || *storageSize == "" {
		fs.Usage()
		return fmt.Errorf("--name, --storage-class, and --storage-size are required")
	}

	var repoInputs []service.RepoInput
	if *repos != "" {
		for _, url := range strings.Split(*repos, ",") {
			url = strings.TrimSpace(url)
			if url != "" {
				repoInputs = append(repoInputs, service.RepoInput{URL: url})
			}
		}
	}

	detail, err := svc.Create(ctx, namespace, service.CreateWorkspaceRequest{
		Name:         *name,
		Image:        *image,
		Repos:        repoInputs,
		StorageClass: *storageClass,
		StorageSize:  *storageSize,
		InitScript:   *initScript,
	})
	if err != nil {
		return fmt.Errorf("creating workspace: %w", err)
	}

	return printJSON(detail)
}

func cliList(ctx context.Context, svc service.WorkspaceService, namespace string) error {
	summaries, err := svc.List(ctx, namespace)
	if err != nil {
		return fmt.Errorf("listing workspaces: %w", err)
	}

	return printJSON(summaries)
}

func cliGet(ctx context.Context, svc service.WorkspaceService, namespace string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: forge-ws get <name>")
	}

	detail, err := svc.Get(ctx, namespace, args[0])
	if err != nil {
		return fmt.Errorf("getting workspace: %w", err)
	}

	return printJSON(detail)
}

func cliDelete(ctx context.Context, svc service.WorkspaceService, namespace string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: forge-ws delete <name>")
	}

	if err := svc.Delete(ctx, namespace, args[0]); err != nil {
		return fmt.Errorf("deleting workspace: %w", err)
	}

	_, _ = fmt.Fprintf(os.Stdout, "Workspace %q deleted\n", args[0])
	return nil
}

func cliStatus(ctx context.Context, svc service.WorkspaceService, namespace string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: forge-ws status <name>")
	}

	detail, err := svc.Get(ctx, namespace, args[0])
	if err != nil {
		return fmt.Errorf("getting workspace status: %w", err)
	}

	return printJSON(detail)
}

func cliSuspend(ctx context.Context, svc service.WorkspaceService, namespace string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: forge-ws suspend <name>")
	}

	detail, err := svc.Suspend(ctx, namespace, args[0])
	if err != nil {
		return fmt.Errorf("suspending workspace: %w", err)
	}

	return printJSON(detail)
}

func cliResume(ctx context.Context, svc service.WorkspaceService, namespace string, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: forge-ws resume <name>")
	}

	detail, err := svc.Resume(ctx, namespace, args[0])
	if err != nil {
		return fmt.Errorf("resuming workspace: %w", err)
	}

	return printJSON(detail)
}

// filterArgs removes flags handled by enginecli.Bootstrap (--mcp, --version, -v, version).
func filterArgs(args []string) []string {
	var filtered []string
	for _, arg := range args {
		switch arg {
		case "--mcp", "--version", "-v", "version":
			continue
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func printUsage() {
	fmt.Fprintln(os.Stderr, "Usage: forge-ws <subcommand> [flags]")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Subcommands:")
	fmt.Fprintln(os.Stderr, "  create    Create a new workspace")
	fmt.Fprintln(os.Stderr, "  list      List all workspaces")
	fmt.Fprintln(os.Stderr, "  get       Get workspace details")
	fmt.Fprintln(os.Stderr, "  delete    Delete a workspace")
	fmt.Fprintln(os.Stderr, "  status    Get workspace status")
	fmt.Fprintln(os.Stderr, "  suspend   Suspend a workspace")
	fmt.Fprintln(os.Stderr, "  resume    Resume a suspended workspace")
	fmt.Fprintln(os.Stderr, "")
	fmt.Fprintln(os.Stderr, "Flags:")
	fmt.Fprintln(os.Stderr, "  --mcp     Run as MCP server on stdio")
	fmt.Fprintln(os.Stderr, "  --version Show version information")
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
