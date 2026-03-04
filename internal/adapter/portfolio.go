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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errPortfolioGet          = errors.New("getting portfolio")
	errPortfolioList         = errors.New("listing portfolios")
	errPortfolioUpdateStatus = errors.New("updating portfolio status")

	errListWorkspacesBySelector        = errors.New("listing workspaces by selector")
	errConvertingLabelSelectorToLabels = errors.New("converting label selector to labels")
)

var _ PortfolioAdapter = (*kubePortfolioAdapter)(nil)

type kubePortfolioAdapter struct {
	client client.Client
}

// NewPortfolioAdapter returns a new PortfolioAdapter backed by a controller-runtime client.
func NewPortfolioAdapter(c client.Client) PortfolioAdapter {
	return &kubePortfolioAdapter{client: c}
}

func (r *kubePortfolioAdapter) Get(ctx context.Context, namespace, name string) (*v1alpha1.Portfolio, error) {
	p := new(v1alpha1.Portfolio)
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, p); err != nil {
		return nil, errors.Join(err, errPortfolioGet)
	}

	return p, nil
}

func (r *kubePortfolioAdapter) List(ctx context.Context, namespace string) (*v1alpha1.PortfolioList, error) {
	list := new(v1alpha1.PortfolioList)
	if err := r.client.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return nil, errors.Join(err, errPortfolioList)
	}

	return list, nil
}

func (r *kubePortfolioAdapter) UpdateStatus(ctx context.Context, p *v1alpha1.Portfolio) error {
	if err := r.client.Status().Update(ctx, p); err != nil {
		return errors.Join(err, errPortfolioUpdateStatus)
	}

	return nil
}

func (r *kubePortfolioAdapter) ListWorkspacesBySelector(
	ctx context.Context,
	namespace string,
	selector *metav1.LabelSelector,
) (*v1alpha1.WorkspaceList, error) {
	sel, err := metav1.LabelSelectorAsSelector(selector)
	if err != nil {
		return nil, errors.Join(err, errConvertingLabelSelectorToLabels, errListWorkspacesBySelector)
	}

	list := new(v1alpha1.WorkspaceList)
	if err := r.client.List(ctx, list,
		client.InNamespace(namespace),
		client.MatchingLabelsSelector{Selector: sel},
	); err != nil {
		return nil, errors.Join(err, errListWorkspacesBySelector)
	}

	return list, nil
}
