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

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	errSecretCreate = errors.New("creating secret")
	errSecretGet    = errors.New("getting secret")
	errSecretDelete = errors.New("deleting secret")
)

var _ SecretAdapter = (*kubeSecretAdapter)(nil)

type kubeSecretAdapter struct {
	client client.Client
}

// NewSecretAdapter returns a new SecretAdapter backed by a controller-runtime client.
func NewSecretAdapter(c client.Client) SecretAdapter {
	return &kubeSecretAdapter{client: c}
}

func (r *kubeSecretAdapter) Create(ctx context.Context, secret *corev1.Secret) error {
	if err := r.client.Create(ctx, secret); err != nil {
		return errors.Join(err, errSecretCreate)
	}

	return nil
}

func (r *kubeSecretAdapter) Get(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret := new(corev1.Secret)
	if err := r.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, secret); err != nil {
		return nil, errors.Join(err, errSecretGet)
	}

	return secret, nil
}

func (r *kubeSecretAdapter) Delete(ctx context.Context, namespace, name string) error {
	secret, err := r.Get(ctx, namespace, name)
	if err != nil {
		return errors.Join(err, errSecretDelete)
	}

	if err := r.client.Delete(ctx, secret); err != nil {
		return errors.Join(err, errSecretDelete)
	}

	return nil
}
