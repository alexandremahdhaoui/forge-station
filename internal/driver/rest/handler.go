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

package rest

import (
	"context"

	"github.com/alexandremahdhaoui/forge-station/internal/controller/service"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// APIHandler implements the generated StrictServerInterface, delegating to
// controller services and returning typed JSON responses.
type APIHandler struct {
	WorkspaceService service.WorkspaceService
	PortfolioService service.PortfolioService
}

// NewAPIHandler creates an APIHandler wired to the given controller services.
func NewAPIHandler(ws service.WorkspaceService, ps service.PortfolioService) *APIHandler {
	return &APIHandler{WorkspaceService: ws, PortfolioService: ps}
}

// Compile-time check that APIHandler satisfies StrictServerInterface.
var _ StrictServerInterface = (*APIHandler)(nil)

// namespaceOrDefault returns the namespace string from an optional pointer,
// defaulting to "default" when nil.
func namespaceOrDefault(ns *string) string {
	if ns != nil {
		return *ns
	}
	return "default"
}

// ListWorkspaces handles GET /api/v1/workspaces.
func (h *APIHandler) ListWorkspaces(ctx context.Context, request ListWorkspacesRequestObject) (ListWorkspacesResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	summaries, err := h.WorkspaceService.List(ctx, ns)
	if err != nil {
		return ListWorkspaces500JSONResponse{Error: err.Error()}, nil
	}
	return ListWorkspaces200JSONResponse(summaries), nil
}

// CreateWorkspace handles POST /api/v1/workspaces.
func (h *APIHandler) CreateWorkspace(ctx context.Context, request CreateWorkspaceRequestObject) (CreateWorkspaceResponseObject, error) {
	if request.Body == nil {
		return CreateWorkspace400JSONResponse{Error: "request body is required"}, nil
	}
	ns := namespaceOrDefault(request.Params.Namespace)
	detail, err := h.WorkspaceService.Create(ctx, ns, *request.Body)
	if err != nil {
		return CreateWorkspace500JSONResponse{Error: err.Error()}, nil
	}
	return CreateWorkspace201JSONResponse(*detail), nil
}

// GetWorkspace handles GET /api/v1/workspaces/{name}.
func (h *APIHandler) GetWorkspace(ctx context.Context, request GetWorkspaceRequestObject) (GetWorkspaceResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	detail, err := h.WorkspaceService.Get(ctx, ns, request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return GetWorkspace404JSONResponse{Error: err.Error()}, nil
		}
		return GetWorkspace500JSONResponse{Error: err.Error()}, nil
	}
	return GetWorkspace200JSONResponse(*detail), nil
}

// DeleteWorkspace handles DELETE /api/v1/workspaces/{name}.
func (h *APIHandler) DeleteWorkspace(ctx context.Context, request DeleteWorkspaceRequestObject) (DeleteWorkspaceResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	err := h.WorkspaceService.Delete(ctx, ns, request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return DeleteWorkspace404JSONResponse{Error: err.Error()}, nil
		}
		return DeleteWorkspace500JSONResponse{Error: err.Error()}, nil
	}
	return DeleteWorkspace204Response{}, nil
}

// SuspendWorkspace handles PUT /api/v1/workspaces/{name}/suspend.
func (h *APIHandler) SuspendWorkspace(ctx context.Context, request SuspendWorkspaceRequestObject) (SuspendWorkspaceResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	detail, err := h.WorkspaceService.Suspend(ctx, ns, request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return SuspendWorkspace404JSONResponse{Error: err.Error()}, nil
		}
		return SuspendWorkspace500JSONResponse{Error: err.Error()}, nil
	}
	return SuspendWorkspace200JSONResponse(*detail), nil
}

// ResumeWorkspace handles PUT /api/v1/workspaces/{name}/resume.
func (h *APIHandler) ResumeWorkspace(ctx context.Context, request ResumeWorkspaceRequestObject) (ResumeWorkspaceResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	detail, err := h.WorkspaceService.Resume(ctx, ns, request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return ResumeWorkspace404JSONResponse{Error: err.Error()}, nil
		}
		return ResumeWorkspace500JSONResponse{Error: err.Error()}, nil
	}
	return ResumeWorkspace200JSONResponse(*detail), nil
}

// ListPortfolios handles GET /api/v1/portfolios.
func (h *APIHandler) ListPortfolios(ctx context.Context, request ListPortfoliosRequestObject) (ListPortfoliosResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	list, err := h.PortfolioService.List(ctx, ns)
	if err != nil {
		return ListPortfolios500JSONResponse{Error: err.Error()}, nil
	}
	return ListPortfolios200JSONResponse(list.Items), nil
}

// GetPortfolio handles GET /api/v1/portfolios/{name}.
func (h *APIHandler) GetPortfolio(ctx context.Context, request GetPortfolioRequestObject) (GetPortfolioResponseObject, error) {
	ns := namespaceOrDefault(request.Params.Namespace)
	portfolio, err := h.PortfolioService.Get(ctx, ns, request.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return GetPortfolio404JSONResponse{Error: err.Error()}, nil
		}
		return GetPortfolio500JSONResponse{Error: err.Error()}, nil
	}
	return GetPortfolio200JSONResponse(*portfolio), nil
}
