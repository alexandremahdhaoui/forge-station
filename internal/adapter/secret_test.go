//go:build unit

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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func newTestSecret(namespace, name string) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"identity": []byte("ssh-key-data"),
		},
	}
}

func TestSecretAdapter_CreateAndGet(t *testing.T) {
	scheme := testScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	adapter := NewSecretAdapter(cl)

	ctx := context.Background()
	secret := newTestSecret("default", "test-secret")

	if err := adapter.Create(ctx, secret); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	got, err := adapter.Get(ctx, "default", "test-secret")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "test-secret" {
		t.Errorf("expected name test-secret, got %s", got.Name)
	}
	if string(got.Data["identity"]) != "ssh-key-data" {
		t.Errorf("expected identity ssh-key-data, got %s", string(got.Data["identity"]))
	}
}

func TestSecretAdapter_GetNotFound(t *testing.T) {
	scheme := testScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	adapter := NewSecretAdapter(cl)

	ctx := context.Background()
	_, err := adapter.Get(ctx, "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent secret, got nil")
	}
}

func TestSecretAdapter_Delete(t *testing.T) {
	scheme := testScheme()
	secret := newTestSecret("default", "test-secret")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(secret).
		Build()
	adapter := NewSecretAdapter(cl)

	ctx := context.Background()

	if err := adapter.Delete(ctx, "default", "test-secret"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, err := adapter.Get(ctx, "default", "test-secret")
	if err == nil {
		t.Fatal("expected error after deletion, got nil")
	}
}

func TestSecretAdapter_DeleteNotFound(t *testing.T) {
	scheme := testScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	adapter := NewSecretAdapter(cl)

	ctx := context.Background()
	err := adapter.Delete(ctx, "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent secret, got nil")
	}
}
