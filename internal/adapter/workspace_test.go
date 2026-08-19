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

	v1alpha1 "github.com/alexandremahdhaoui/forge-station/pkg/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testScheme() *runtime.Scheme {
	s := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(s))
	utilruntime.Must(v1alpha1.AddToScheme(s))
	return s
}

func newTestWorkspace(namespace, name string) *v1alpha1.Workspace {
	return &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "test-image:latest",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "10Gi",
			},
		},
	}
}

func TestWorkspaceAdapter_CreateAndGet(t *testing.T) {
	scheme := testScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()
	ws := newTestWorkspace("default", "test-ws")

	// Create
	if err := adapter.Create(ctx, ws); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Get
	got, err := adapter.Get(ctx, "default", "test-ws")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.Name != "test-ws" {
		t.Errorf("expected name test-ws, got %s", got.Name)
	}
	if got.Spec.Image != "test-image:latest" {
		t.Errorf("expected image test-image:latest, got %s", got.Spec.Image)
	}
}

func TestWorkspaceAdapter_GetNotFound(t *testing.T) {
	scheme := testScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()
	_, err := adapter.Get(ctx, "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workspace, got nil")
	}
}

func TestWorkspaceAdapter_List(t *testing.T) {
	scheme := testScheme()
	ws1 := newTestWorkspace("default", "ws-1")
	ws2 := newTestWorkspace("default", "ws-2")
	ws3 := newTestWorkspace("other-ns", "ws-3")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws1, ws2, ws3).
		Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()
	list, err := adapter.List(ctx, "default")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(list.Items) != 2 {
		t.Errorf("expected 2 workspaces in default namespace, got %d", len(list.Items))
	}
}

func TestWorkspaceAdapter_Update(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "test-ws")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()

	// Fetch, modify, update
	got, err := adapter.Get(ctx, "default", "test-ws")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	got.Spec.Image = "updated-image:v2"
	if err := adapter.Update(ctx, got); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Verify
	updated, err := adapter.Get(ctx, "default", "test-ws")
	if err != nil {
		t.Fatalf("Get after update failed: %v", err)
	}
	if updated.Spec.Image != "updated-image:v2" {
		t.Errorf("expected image updated-image:v2, got %s", updated.Spec.Image)
	}
}

func TestWorkspaceAdapter_Delete(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "test-ws")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()

	if err := adapter.Delete(ctx, "default", "test-ws"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err := adapter.Get(ctx, "default", "test-ws")
	if err == nil {
		t.Fatal("expected error after deletion, got nil")
	}
}

func TestWorkspaceAdapter_DeleteNotFound(t *testing.T) {
	scheme := testScheme()
	cl := fake.NewClientBuilder().WithScheme(scheme).Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()
	err := adapter.Delete(ctx, "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent workspace, got nil")
	}
}

func TestWorkspaceAdapter_UpdateStatus(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "test-ws")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		WithStatusSubresource(&v1alpha1.Workspace{}).
		Build()
	adapter := NewWorkspaceAdapter(cl)

	ctx := context.Background()

	// Fetch, set status, update
	got, err := adapter.Get(ctx, "default", "test-ws")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	got.Status.Phase = v1alpha1.WorkspaceRunning
	got.Status.PodName = "ws-test-ws-pod"
	if err := adapter.UpdateStatus(ctx, got); err != nil {
		t.Fatalf("UpdateStatus failed: %v", err)
	}

	// Verify
	updated, err := adapter.Get(ctx, "default", "test-ws")
	if err != nil {
		t.Fatalf("Get after status update failed: %v", err)
	}
	if updated.Status.Phase != v1alpha1.WorkspaceRunning {
		t.Errorf("expected phase Running, got %s", updated.Status.Phase)
	}
	if updated.Status.PodName != "ws-test-ws-pod" {
		t.Errorf("expected podName ws-test-ws-pod, got %s", updated.Status.PodName)
	}
}
