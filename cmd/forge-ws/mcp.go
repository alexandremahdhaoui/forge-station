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
	"fmt"
	"log"

	"github.com/alexandremahdhaoui/forge-station/internal/controller/service"
	"github.com/alexandremahdhaoui/forge/pkg/mcpserver"
	"github.com/alexandremahdhaoui/forge/pkg/mcputil"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// --- MCP input types ---

type createWorkspaceInput struct {
	Name         string              `json:"name" jsonschema:"Workspace name (required)"`
	Image        string              `json:"image,omitempty" jsonschema:"Container image for the workspace pod"`
	Repos        []service.RepoInput `json:"repos,omitempty" jsonschema:"Git repositories to clone"`
	StorageClass string              `json:"storageClass" jsonschema:"Storage class for the PVC (required)"`
	StorageSize  string              `json:"storageSize" jsonschema:"Storage size for the PVC, e.g. 20Gi (required)"`
	InitScript   string              `json:"initScript,omitempty" jsonschema:"Post-start script executed in the workspace"`
}

type nameInput struct {
	Name string `json:"name" jsonschema:"Workspace name (required)"`
}

type emptyInput struct{}

// runMCP starts the MCP server on stdio with all workspace tools registered.
func runMCP(svc service.WorkspaceService, namespace string) error {
	server := mcpserver.New("forge-ws", Version)

	registerMCPTools(server, svc, namespace)

	log.Println("Starting forge-ws MCP server on stdio")
	return server.RunDefault()
}

func registerMCPTools(server *mcpserver.Server, svc service.WorkspaceService, namespace string) {
	// create-workspace
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "create-workspace",
		Description: "Create a new development workspace with git repositories and persistent storage.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input createWorkspaceInput) (*mcp.CallToolResult, any, error) {
		if input.Name == "" {
			return mcputil.ErrorResult("name is required"), nil, nil
		}
		if input.StorageClass == "" {
			return mcputil.ErrorResult("storageClass is required"), nil, nil
		}
		if input.StorageSize == "" {
			return mcputil.ErrorResult("storageSize is required"), nil, nil
		}

		detail, err := svc.Create(ctx, namespace, service.CreateWorkspaceRequest{
			Name:         input.Name,
			Image:        input.Image,
			Repos:        input.Repos,
			StorageClass: input.StorageClass,
			StorageSize:  input.StorageSize,
			InitScript:   input.InitScript,
		})
		if err != nil {
			return mcputil.ErrorResult(fmt.Sprintf("failed to create workspace: %v", err)), nil, nil
		}

		result, artifact := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Workspace %q created in namespace %q", detail.Name, detail.Namespace),
			detail,
		)
		return result, artifact, nil
	})

	// list-workspaces
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "list-workspaces",
		Description: "List all workspaces in the configured namespace.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input emptyInput) (*mcp.CallToolResult, any, error) {
		summaries, err := svc.List(ctx, namespace)
		if err != nil {
			return mcputil.ErrorResult(fmt.Sprintf("failed to list workspaces: %v", err)), nil, nil
		}

		result, artifact := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Found %d workspace(s)", len(summaries)),
			summaries,
		)
		return result, artifact, nil
	})

	// get-workspace
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "get-workspace",
		Description: "Get detailed information about a workspace by name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input nameInput) (*mcp.CallToolResult, any, error) {
		if input.Name == "" {
			return mcputil.ErrorResult("name is required"), nil, nil
		}

		detail, err := svc.Get(ctx, namespace, input.Name)
		if err != nil {
			return mcputil.ErrorResult(fmt.Sprintf("failed to get workspace: %v", err)), nil, nil
		}

		result, artifact := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Workspace %q in namespace %q", detail.Name, detail.Namespace),
			detail,
		)
		return result, artifact, nil
	})

	// delete-workspace
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "delete-workspace",
		Description: "Delete a workspace by name.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input nameInput) (*mcp.CallToolResult, any, error) {
		if input.Name == "" {
			return mcputil.ErrorResult("name is required"), nil, nil
		}

		if err := svc.Delete(ctx, namespace, input.Name); err != nil {
			return mcputil.ErrorResult(fmt.Sprintf("failed to delete workspace: %v", err)), nil, nil
		}

		return mcputil.SuccessResult(fmt.Sprintf("Workspace %q deleted", input.Name)), nil, nil
	})

	// suspend-workspace
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "suspend-workspace",
		Description: "Suspend a workspace, pausing reconciliation while keeping existing resources.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input nameInput) (*mcp.CallToolResult, any, error) {
		if input.Name == "" {
			return mcputil.ErrorResult("name is required"), nil, nil
		}

		detail, err := svc.Suspend(ctx, namespace, input.Name)
		if err != nil {
			return mcputil.ErrorResult(fmt.Sprintf("failed to suspend workspace: %v", err)), nil, nil
		}

		result, artifact := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Workspace %q suspended", detail.Name),
			detail,
		)
		return result, artifact, nil
	})

	// resume-workspace
	mcpserver.RegisterTool(server, &mcp.Tool{
		Name:        "resume-workspace",
		Description: "Resume a suspended workspace, re-enabling reconciliation.",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input nameInput) (*mcp.CallToolResult, any, error) {
		if input.Name == "" {
			return mcputil.ErrorResult("name is required"), nil, nil
		}

		detail, err := svc.Resume(ctx, namespace, input.Name)
		if err != nil {
			return mcputil.ErrorResult(fmt.Sprintf("failed to resume workspace: %v", err)), nil, nil
		}

		result, artifact := mcputil.SuccessResultWithArtifact(
			fmt.Sprintf("Workspace %q resumed", detail.Name),
			detail,
		)
		return result, artifact, nil
	})
}
