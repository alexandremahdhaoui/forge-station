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

	v1alpha1 "github.com/alexandremahdhaoui/forge-station/pkg/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// WorkspaceAdapter provides CRUD operations for Workspace custom resources.
type WorkspaceAdapter interface {
	Create(ctx context.Context, ws *v1alpha1.Workspace) error
	Get(ctx context.Context, namespace, name string) (*v1alpha1.Workspace, error)
	List(ctx context.Context, namespace string, opts ...client.ListOption) (*v1alpha1.WorkspaceList, error)
	Update(ctx context.Context, ws *v1alpha1.Workspace) error
	Delete(ctx context.Context, namespace, name string) error
	UpdateStatus(ctx context.Context, ws *v1alpha1.Workspace) error
}

// PortfolioAdapter provides operations for Portfolio custom resources.
type PortfolioAdapter interface {
	Get(ctx context.Context, namespace, name string) (*v1alpha1.Portfolio, error)
	List(ctx context.Context, namespace string) (*v1alpha1.PortfolioList, error)
	UpdateStatus(ctx context.Context, p *v1alpha1.Portfolio) error
	ListWorkspacesBySelector(ctx context.Context, namespace string, selector *metav1.LabelSelector) (*v1alpha1.WorkspaceList, error)
}

// SecretAdapter provides CRUD operations for K8s Secrets with ownerReferences.
type SecretAdapter interface {
	Create(ctx context.Context, secret *corev1.Secret) error
	Get(ctx context.Context, namespace, name string) (*corev1.Secret, error)
	Delete(ctx context.Context, namespace, name string) error
}
