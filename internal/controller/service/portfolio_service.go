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

package service

import (
	"context"

	"github.com/alexandremahdhaoui/forge-station/internal/adapter"
	v1alpha1 "github.com/alexandremahdhaoui/forge-station/pkg/v1alpha1"
)

// PortfolioService provides business logic for portfolio operations.
type PortfolioService interface {
	Get(ctx context.Context, namespace, name string) (*v1alpha1.Portfolio, error)
	List(ctx context.Context, namespace string) (*v1alpha1.PortfolioList, error)
}

type portfolioService struct {
	portfolios adapter.PortfolioAdapter
}

// Compile-time interface check.
var _ PortfolioService = (*portfolioService)(nil)

// NewPortfolioService creates a PortfolioService.
func NewPortfolioService(p adapter.PortfolioAdapter) PortfolioService {
	return &portfolioService{portfolios: p}
}

// Get retrieves a single portfolio by namespace and name.
func (s *portfolioService) Get(ctx context.Context, namespace, name string) (*v1alpha1.Portfolio, error) {
	return s.portfolios.Get(ctx, namespace, name)
}

// List returns all portfolios in a namespace.
func (s *portfolioService) List(ctx context.Context, namespace string) (*v1alpha1.PortfolioList, error) {
	return s.portfolios.List(ctx, namespace)
}
