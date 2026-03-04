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

package service

import (
	"context"
	"testing"

	"github.com/alexandremahdhaoui/forge-workspace/internal/adapter"
	v1alpha1 "github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
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

func newService(scheme *runtime.Scheme, objs ...runtime.Object) (WorkspaceService, *runtime.Scheme) {
	builder := fake.NewClientBuilder().WithScheme(scheme)
	for _, obj := range objs {
		builder = builder.WithRuntimeObjects(obj)
	}
	cl := builder.Build()
	ws := adapter.NewWorkspaceAdapter(cl)
	secrets := adapter.NewSecretAdapter(cl)
	return NewWorkspaceService(ws, secrets), scheme
}

func TestWorkspaceService_CreateWithSSHAuth(t *testing.T) {
	scheme := testScheme()
	svc, _ := newService(scheme)

	ctx := context.Background()
	req := CreateWorkspaceRequest{
		Name:         "test-ws",
		Image:        "my-image:latest",
		StorageClass: "standard",
		StorageSize:  "20Gi",
		Repos: []RepoInput{
			{
				URL: "git@github.com:org/repo.git",
				Ref: GitRefInput{Branch: "main"},
				Auth: &AuthInput{
					SSHPrivateKey: "fake-ssh-key",
					KnownHosts:    "github.com ssh-rsa AAAA...",
				},
			},
		},
	}

	detail, err := svc.Create(ctx, "default", req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if detail.Name != "test-ws" {
		t.Errorf("expected name test-ws, got %s", detail.Name)
	}
	if detail.Namespace != "default" {
		t.Errorf("expected namespace default, got %s", detail.Namespace)
	}
	if len(detail.Repos) != 1 {
		t.Fatalf("expected 1 repo, got %d", len(detail.Repos))
	}
	if !detail.Repos[0].HasCredential {
		t.Error("expected HasCredential=true for repo with SSH auth")
	}
}

func TestWorkspaceService_CreateWithoutAuth(t *testing.T) {
	scheme := testScheme()
	svc, _ := newService(scheme)

	ctx := context.Background()
	req := CreateWorkspaceRequest{
		Name:         "no-auth-ws",
		Image:        "my-image:latest",
		StorageClass: "standard",
		StorageSize:  "10Gi",
		Repos: []RepoInput{
			{
				URL: "https://github.com/org/public-repo.git",
				Ref: GitRefInput{Branch: "main"},
			},
		},
	}

	detail, err := svc.Create(ctx, "default", req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if detail.Repos[0].HasCredential {
		t.Error("expected HasCredential=false for repo without auth")
	}
}

func TestWorkspaceService_Get(t *testing.T) {
	scheme := testScheme()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "existing-ws",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "my-image:latest",
			Repos: []v1alpha1.RepoRef{
				{
					URL:       "git@github.com:org/repo.git",
					Ref:       v1alpha1.GitRef{Branch: "main"},
					SecretRef: &v1alpha1.SecretRef{Name: "ws-existing-ws-repo-0-creds"},
				},
			},
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "20Gi",
			},
		},
	}

	svc, _ := newService(scheme, ws)
	ctx := context.Background()

	detail, err := svc.Get(ctx, "default", "existing-ws")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if detail.Name != "existing-ws" {
		t.Errorf("expected name existing-ws, got %s", detail.Name)
	}
	if !detail.Repos[0].HasCredential {
		t.Error("expected HasCredential=true when secretRef is set")
	}
}

func TestWorkspaceService_GetNotFound(t *testing.T) {
	scheme := testScheme()
	svc, _ := newService(scheme)

	ctx := context.Background()
	_, err := svc.Get(ctx, "default", "nonexistent")
	if err == nil {
		t.Fatal("expected error for nonexistent workspace, got nil")
	}
}

func TestWorkspaceService_List(t *testing.T) {
	scheme := testScheme()
	ws1 := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-1",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "img:1",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "10Gi",
			},
		},
	}
	ws2 := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ws-2",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "img:2",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "10Gi",
			},
		},
	}

	svc, _ := newService(scheme, ws1, ws2)
	ctx := context.Background()

	summaries, err := svc.List(ctx, "default")
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(summaries) != 2 {
		t.Errorf("expected 2 summaries, got %d", len(summaries))
	}
}

func TestWorkspaceService_Delete(t *testing.T) {
	scheme := testScheme()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-delete",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "img:1",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "10Gi",
			},
		},
	}

	svc, _ := newService(scheme, ws)
	ctx := context.Background()

	if err := svc.Delete(ctx, "default", "to-delete"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deletion
	_, err := svc.Get(ctx, "default", "to-delete")
	if err == nil {
		t.Fatal("expected error after deletion, got nil")
	}
}

func TestWorkspaceService_Suspend(t *testing.T) {
	scheme := testScheme()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-suspend",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image: "img:1",
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "10Gi",
			},
		},
	}

	svc, _ := newService(scheme, ws)
	ctx := context.Background()

	detail, err := svc.Suspend(ctx, "default", "to-suspend")
	if err != nil {
		t.Fatalf("Suspend failed: %v", err)
	}
	if !detail.Suspended {
		t.Error("expected Suspended=true after Suspend call")
	}

	// Verify via Get
	got, err := svc.Get(ctx, "default", "to-suspend")
	if err != nil {
		t.Fatalf("Get after suspend failed: %v", err)
	}
	if !got.Suspended {
		t.Error("expected Suspended=true when fetched after Suspend")
	}
}

func TestWorkspaceService_Resume(t *testing.T) {
	scheme := testScheme()
	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "to-resume",
			Namespace: "default",
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image:   "img:1",
			Suspend: true,
			Storage: v1alpha1.StorageSpec{
				StorageClassName: "standard",
				Size:             "10Gi",
			},
		},
	}

	svc, _ := newService(scheme, ws)
	ctx := context.Background()

	detail, err := svc.Resume(ctx, "default", "to-resume")
	if err != nil {
		t.Fatalf("Resume failed: %v", err)
	}
	if detail.Suspended {
		t.Error("expected Suspended=false after Resume call")
	}

	// Verify via Get
	got, err := svc.Get(ctx, "default", "to-resume")
	if err != nil {
		t.Fatalf("Get after resume failed: %v", err)
	}
	if got.Suspended {
		t.Error("expected Suspended=false when fetched after Resume")
	}
}

func TestWorkspaceService_CreateWithCompoAuth(t *testing.T) {
	scheme := testScheme()
	svc, _ := newService(scheme)

	ctx := context.Background()
	req := CreateWorkspaceRequest{
		Name:         "compo-ws",
		Image:        "my-image:latest",
		StorageClass: "standard",
		StorageSize:  "10Gi",
		Repos: []RepoInput{
			{URL: "https://github.com/org/repo.git"},
		},
		Compo: &CompoInput{
			URL: "git@github.com:org/compo.git",
			Ref: GitRefInput{Branch: "main"},
			Auth: &AuthInput{
				SSHPrivateKey: "compo-ssh-key",
			},
		},
	}

	detail, err := svc.Create(ctx, "default", req)
	if err != nil {
		t.Fatalf("Create with compo failed: %v", err)
	}
	if detail.Compo == nil {
		t.Fatal("expected Compo to be set")
	}
	if !detail.Compo.HasCredential {
		t.Error("expected compo HasCredential=true")
	}
}

func TestWorkspaceService_CreateWithExistingSecret(t *testing.T) {
	scheme := testScheme()
	svc, _ := newService(scheme)

	ctx := context.Background()
	req := CreateWorkspaceRequest{
		Name:         "existing-secret-ws",
		Image:        "my-image:latest",
		StorageClass: "standard",
		StorageSize:  "10Gi",
		Repos: []RepoInput{
			{
				URL: "git@github.com:org/repo.git",
				Auth: &AuthInput{
					ExistingSecretName: "my-pre-created-secret",
				},
			},
		},
	}

	detail, err := svc.Create(ctx, "default", req)
	if err != nil {
		t.Fatalf("Create with existing secret failed: %v", err)
	}
	if !detail.Repos[0].HasCredential {
		t.Error("expected HasCredential=true when using existing secret")
	}
}
