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

package adapter

import (
	"context"
	"errors"

	v1alpha1 "github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errWorkspaceCreate       = errors.New("creating workspace")
	errWorkspaceGet          = errors.New("getting workspace")
	errWorkspaceList         = errors.New("listing workspaces")
	errWorkspaceUpdate       = errors.New("updating workspace")
	errWorkspaceDelete       = errors.New("deleting workspace")
	errWorkspaceUpdateStatus = errors.New("updating workspace status")
)

var _ WorkspaceAdapter = (*kubeWorkspaceAdapter)(nil)

type kubeWorkspaceAdapter struct {
	client client.Client
}

// NewWorkspaceAdapter returns a new WorkspaceAdapter backed by a controller-runtime client.
func NewWorkspaceAdapter(c client.Client) WorkspaceAdapter {
	return &kubeWorkspaceAdapter{client: c}
}

func (r *kubeWorkspaceAdapter) Create(ctx context.Context, ws *v1alpha1.Workspace) error {
	if err := r.client.Create(ctx, ws); err != nil {
		return errors.Join(err, errWorkspaceCreate)
	}

	return nil
}

func (r *kubeWorkspaceAdapter) Get(ctx context.Context, namespace, name string) (*v1alpha1.Workspace, error) {
	ws := new(v1alpha1.Workspace)
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, ws); err != nil {
		return nil, errors.Join(err, errWorkspaceGet)
	}

	return ws, nil
}

func (r *kubeWorkspaceAdapter) List(
	ctx context.Context,
	namespace string,
	opts ...client.ListOption,
) (*v1alpha1.WorkspaceList, error) {
	list := new(v1alpha1.WorkspaceList)

	allOpts := append([]client.ListOption{client.InNamespace(namespace)}, opts...)
	if err := r.client.List(ctx, list, allOpts...); err != nil {
		return nil, errors.Join(err, errWorkspaceList)
	}

	return list, nil
}

func (r *kubeWorkspaceAdapter) Update(ctx context.Context, ws *v1alpha1.Workspace) error {
	if err := r.client.Update(ctx, ws); err != nil {
		return errors.Join(err, errWorkspaceUpdate)
	}

	return nil
}

func (r *kubeWorkspaceAdapter) Delete(ctx context.Context, namespace, name string) error {
	ws, err := r.Get(ctx, namespace, name)
	if err != nil {
		return errors.Join(err, errWorkspaceDelete)
	}

	if err := r.client.Delete(ctx, ws); err != nil {
		return errors.Join(err, errWorkspaceDelete)
	}

	return nil
}

func (r *kubeWorkspaceAdapter) UpdateStatus(ctx context.Context, ws *v1alpha1.Workspace) error {
	if err := r.client.Status().Update(ctx, ws); err != nil {
		return errors.Join(err, errWorkspaceUpdateStatus)
	}

	return nil
}
