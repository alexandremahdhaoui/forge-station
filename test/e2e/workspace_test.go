//go:build e2e

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

package e2e

import (
	"context"
	"fmt"
	"testing"
	"time"

	v1alpha1 "github.com/alexandremahdhaoui/forge-station/pkg/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestWorkspaceCreation(t *testing.T) {
	ctx := context.Background()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ws-create",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "registry.k8s.io/pause:3.10",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "1Gi",
			},
		},
	}

	// Create
	if err := k8sClient.Create(ctx, ws); err != nil {
		t.Fatalf("failed to create workspace: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, ws)
	}()

	// Wait for reconciliation — poll for PVC, Pod, Service
	var pvc corev1.PersistentVolumeClaim
	if err := waitFor(t, 30*time.Second, func() error {
		return k8sClient.Get(ctx, types.NamespacedName{
			Name:      "ws-test-ws-create-pvc",
			Namespace: "default",
		}, &pvc)
	}); err != nil {
		t.Fatalf("PVC not created: %v", err)
	}

	var pod corev1.Pod
	if err := waitFor(t, 30*time.Second, func() error {
		return k8sClient.Get(ctx, types.NamespacedName{
			Name:      "ws-test-ws-create-pod",
			Namespace: "default",
		}, &pod)
	}); err != nil {
		t.Fatalf("Pod not created: %v", err)
	}

	var svc corev1.Service
	if err := waitFor(t, 30*time.Second, func() error {
		return k8sClient.Get(ctx, types.NamespacedName{
			Name:      "ws-test-ws-create-svc",
			Namespace: "default",
		}, &svc)
	}); err != nil {
		t.Fatalf("Service not created: %v", err)
	}
}

func TestWorkspaceDeletion(t *testing.T) {
	ctx := context.Background()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ws-delete",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "registry.k8s.io/pause:3.10",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "1Gi",
			},
		},
	}

	// Create and wait for sub-resources
	if err := k8sClient.Create(ctx, ws); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := waitFor(t, 30*time.Second, func() error {
		return k8sClient.Get(ctx, types.NamespacedName{
			Name: "ws-test-ws-delete-pod", Namespace: "default",
		}, &corev1.Pod{})
	}); err != nil {
		t.Fatalf("pod not created before deletion test: %v", err)
	}

	// Delete
	if err := k8sClient.Delete(ctx, ws); err != nil {
		t.Fatalf("delete: %v", err)
	}

	// Wait for cleanup via ownerReferences GC
	if err := waitFor(t, 60*time.Second, func() error {
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name: "ws-test-ws-delete-pod", Namespace: "default",
		}, &corev1.Pod{})
		if apierrors.IsNotFound(err) {
			return nil
		}
		if err == nil {
			return fmt.Errorf("pod still exists")
		}
		return err
	}); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}
}

func TestWorkspaceSuspendResume(t *testing.T) {
	ctx := context.Background()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-ws-suspend",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "registry.k8s.io/pause:3.10",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "1Gi",
			},
		},
	}

	// Create and wait for pod
	if err := k8sClient.Create(ctx, ws); err != nil {
		t.Fatalf("create: %v", err)
	}
	defer func() {
		_ = k8sClient.Delete(ctx, ws)
	}()

	if err := waitFor(t, 60*time.Second, func() error {
		var updated v1alpha1.Workspace
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name: "test-ws-suspend", Namespace: "default",
		}, &updated); err != nil {
			return err
		}
		if updated.Status.Phase != v1alpha1.WorkspaceRunning {
			return fmt.Errorf("expected phase Running, got %s", updated.Status.Phase)
		}
		return nil
	}); err != nil {
		t.Fatalf("workspace not running: %v", err)
	}

	// Suspend
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name: "test-ws-suspend", Namespace: "default",
	}, ws); err != nil {
		t.Fatalf("get workspace for suspend: %v", err)
	}
	ws.Spec.Suspend = true
	if err := k8sClient.Update(ctx, ws); err != nil {
		t.Fatalf("suspend: %v", err)
	}

	// Check that the workspace enters Stopped phase
	if err := waitFor(t, 30*time.Second, func() error {
		var updated v1alpha1.Workspace
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name: "test-ws-suspend", Namespace: "default",
		}, &updated); err != nil {
			return err
		}
		if updated.Status.Phase != v1alpha1.WorkspaceStopped {
			return fmt.Errorf("expected phase Stopped, got %s", updated.Status.Phase)
		}
		return nil
	}); err != nil {
		t.Fatalf("suspend phase: %v", err)
	}

	// Resume
	if err := k8sClient.Get(ctx, types.NamespacedName{
		Name: "test-ws-suspend", Namespace: "default",
	}, ws); err != nil {
		t.Fatalf("get workspace for resume: %v", err)
	}
	ws.Spec.Suspend = false
	if err := k8sClient.Update(ctx, ws); err != nil {
		t.Fatalf("resume: %v", err)
	}

	// Check that the workspace returns to Running phase
	if err := waitFor(t, 60*time.Second, func() error {
		var updated v1alpha1.Workspace
		if err := k8sClient.Get(ctx, types.NamespacedName{
			Name: "test-ws-suspend", Namespace: "default",
		}, &updated); err != nil {
			return err
		}
		if updated.Status.Phase != v1alpha1.WorkspaceRunning {
			return fmt.Errorf("expected phase Running, got %s", updated.Status.Phase)
		}
		return nil
	}); err != nil {
		t.Fatalf("resume phase: %v", err)
	}
}

func waitFor(t *testing.T, timeout time.Duration, fn func() error) error {
	t.Helper()
	deadline := time.Now().Add(timeout)
	var lastErr error
	for time.Now().Before(deadline) {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		time.Sleep(time.Second)
	}
	return lastErr
}
