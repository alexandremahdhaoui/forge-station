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

package reconciler

import (
	"context"
	"testing"

	"github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func newReconciler(cl client.Client, scheme *runtime.Scheme) *WorkspaceReconciler {
	return &WorkspaceReconciler{
		Client: cl,
		Scheme: scheme,
		Log:    logr.Discard(),
	}
}

func TestReconcile_CreatesSubResources(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "my-ws")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		WithStatusSubresource(&v1alpha1.Workspace{}).
		Build()

	r := newReconciler(cl, scheme)
	ctx := context.Background()

	result, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "my-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	// Verify PVC was created
	var pvc corev1.PersistentVolumeClaim
	if err := cl.Get(ctx, types.NamespacedName{Name: "ws-my-ws-pvc", Namespace: "default"}, &pvc); err != nil {
		t.Fatalf("expected PVC ws-my-ws-pvc to exist: %v", err)
	}

	// Verify Pod was created
	var pod corev1.Pod
	if err := cl.Get(ctx, types.NamespacedName{Name: "ws-my-ws-pod", Namespace: "default"}, &pod); err != nil {
		t.Fatalf("expected Pod ws-my-ws-pod to exist: %v", err)
	}
	if pod.Spec.Containers[0].Image != "test-image:latest" {
		t.Errorf("expected image test-image:latest, got %s", pod.Spec.Containers[0].Image)
	}

	// Verify Service was created
	var svc corev1.Service
	if err := cl.Get(ctx, types.NamespacedName{Name: "ws-my-ws-svc", Namespace: "default"}, &svc); err != nil {
		t.Fatalf("expected Service ws-my-ws-svc to exist: %v", err)
	}

	// Verify status was updated
	var updated v1alpha1.Workspace
	if err := cl.Get(ctx, types.NamespacedName{Name: "my-ws", Namespace: "default"}, &updated); err != nil {
		t.Fatalf("failed to get updated workspace: %v", err)
	}
	if updated.Status.PodName != "ws-my-ws-pod" {
		t.Errorf("expected podName ws-my-ws-pod, got %s", updated.Status.PodName)
	}
	if updated.Status.PVCName != "ws-my-ws-pvc" {
		t.Errorf("expected pvcName ws-my-ws-pvc, got %s", updated.Status.PVCName)
	}
	if updated.Status.ServiceURL != "ws-my-ws-svc.default.svc.cluster.local" {
		t.Errorf("expected serviceURL ws-my-ws-svc.default.svc.cluster.local, got %s", updated.Status.ServiceURL)
	}
}

func TestReconcile_SuspendedWorkspace(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "suspended-ws")
	ws.Spec.Suspend = true

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		WithStatusSubresource(&v1alpha1.Workspace{}).
		Build()

	r := newReconciler(cl, scheme)
	ctx := context.Background()

	result, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "suspended-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}

	// Verify status
	var updated v1alpha1.Workspace
	if err := cl.Get(ctx, types.NamespacedName{Name: "suspended-ws", Namespace: "default"}, &updated); err != nil {
		t.Fatalf("failed to get workspace: %v", err)
	}
	if updated.Status.Phase != v1alpha1.WorkspaceStopped {
		t.Errorf("expected phase Stopped, got %s", updated.Status.Phase)
	}

	// Verify Stalled condition is True
	foundStalled := false
	for _, c := range updated.Status.Conditions {
		if c.Type == ConditionStalled && c.Status == metav1.ConditionTrue {
			foundStalled = true
		}
	}
	if !foundStalled {
		t.Error("expected Stalled condition to be True")
	}

	// Verify no sub-resources were created
	var pod corev1.Pod
	err = cl.Get(ctx, types.NamespacedName{Name: "ws-suspended-ws-pod", Namespace: "default"}, &pod)
	if err == nil {
		t.Error("expected no pod to be created for suspended workspace")
	}
}

func TestReconcile_DeletedWorkspace(t *testing.T) {
	scheme := testScheme()

	// No workspace in the store — simulates a deleted workspace
	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		Build()

	r := newReconciler(cl, scheme)
	ctx := context.Background()

	result, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "deleted-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("expected no error for deleted workspace, got: %v", err)
	}
	if result.Requeue {
		t.Error("expected no requeue")
	}
}

func TestReconcile_UpdatesStatusWithPodPhase(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "running-ws")

	// Pre-create the sub-resources so the reconciler skips creation
	// and focuses on status update.
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-running-ws-pod",
			Namespace: "default",
			Labels: map[string]string{
				labelAppName:     appNameValue,
				labelAppInstance: "running-ws",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "workspace", Image: "test-image:latest"},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{
					Type:   corev1.PodReady,
					Status: corev1.ConditionTrue,
				},
			},
		},
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-running-ws-pvc",
			Namespace: "default",
		},
	}

	svc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-running-ws-svc",
			Namespace: "default",
		},
	}

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws, pod, pvc, svc).
		WithStatusSubresource(&v1alpha1.Workspace{}).
		Build()

	r := newReconciler(cl, scheme)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "running-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	var updated v1alpha1.Workspace
	if err := cl.Get(ctx, types.NamespacedName{Name: "running-ws", Namespace: "default"}, &updated); err != nil {
		t.Fatalf("failed to get workspace: %v", err)
	}
	if updated.Status.Phase != v1alpha1.WorkspaceRunning {
		t.Errorf("expected phase Running, got %s", updated.Status.Phase)
	}

	// Verify Ready condition is True
	foundReady := false
	for _, c := range updated.Status.Conditions {
		if c.Type == ConditionReady && c.Status == metav1.ConditionTrue {
			foundReady = true
		}
	}
	if !foundReady {
		t.Error("expected Ready condition to be True")
	}
}

func TestReconcile_DefaultImage(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "default-img-ws")
	ws.Spec.Image = "" // empty image should use default

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		WithStatusSubresource(&v1alpha1.Workspace{}).
		Build()

	r := newReconciler(cl, scheme)
	ctx := context.Background()

	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "default-img-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("Reconcile failed: %v", err)
	}

	var pod corev1.Pod
	if err := cl.Get(ctx, types.NamespacedName{Name: "ws-default-img-ws-pod", Namespace: "default"}, &pod); err != nil {
		t.Fatalf("expected pod to exist: %v", err)
	}
	if pod.Spec.Containers[0].Image != defaultImage {
		t.Errorf("expected default image %s, got %s", defaultImage, pod.Spec.Containers[0].Image)
	}
}

func TestReconcile_IdempotentSubResourceCreation(t *testing.T) {
	scheme := testScheme()
	ws := newTestWorkspace("default", "idempotent-ws")

	cl := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(ws).
		WithStatusSubresource(&v1alpha1.Workspace{}).
		Build()

	r := newReconciler(cl, scheme)
	ctx := context.Background()

	// First reconcile
	_, err := r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "idempotent-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("first Reconcile failed: %v", err)
	}

	// Second reconcile should not fail (sub-resources already exist)
	_, err = r.Reconcile(ctx, ctrl.Request{
		NamespacedName: types.NamespacedName{Name: "idempotent-ws", Namespace: "default"},
	})
	if err != nil {
		t.Fatalf("second Reconcile failed: %v", err)
	}
}
