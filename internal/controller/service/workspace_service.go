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
	"fmt"

	"github.com/alexandremahdhaoui/forge-workspace/internal/adapter"
	v1alpha1 "github.com/alexandremahdhaoui/forge-workspace/pkg/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// --- Request types ---

// CreateWorkspaceRequest is the user-facing workspace creation request.
type CreateWorkspaceRequest struct {
	Name         string            `json:"name"`
	Image        string            `json:"image,omitempty"`
	Repos        []RepoInput       `json:"repos"`
	StorageClass string            `json:"storageClass"`
	StorageSize  string            `json:"storageSize"`
	InitScript   string            `json:"initScript,omitempty"`
	Compo        *CompoInput       `json:"compo,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
}

// RepoInput is the user-facing repo config. Credentials are provided inline;
// the service creates a Secret and sets secretRef on the CRD.
type RepoInput struct {
	URL      string      `json:"url"`
	Ref      GitRefInput `json:"ref,omitempty"`
	Path     string      `json:"path,omitempty"`
	Provider string      `json:"provider,omitempty"`
	Auth     *AuthInput  `json:"auth,omitempty"`
}

// GitRefInput specifies a git revision in a request.
type GitRefInput struct {
	Branch string `json:"branch,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// AuthInput accepts credentials from the user. The service creates a Secret
// from this data. Only one auth method should be set.
type AuthInput struct {
	SSHPrivateKey      string `json:"sshPrivateKey,omitempty"`
	KnownHosts         string `json:"knownHosts,omitempty"`
	Username           string `json:"username,omitempty"`
	Password           string `json:"password,omitempty"`
	BearerToken        string `json:"bearerToken,omitempty"`
	CAFile             string `json:"caFile,omitempty"`
	ExistingSecretName string `json:"existingSecretName,omitempty"`
}

// CompoInput is the user-facing composition repo config.
type CompoInput struct {
	URL      string      `json:"url"`
	Ref      GitRefInput `json:"ref,omitempty"`
	Provider string      `json:"provider,omitempty"`
	Auth     *AuthInput  `json:"auth,omitempty"`
}

// --- Response types (NEVER expose secret data) ---

// WorkspaceDetail is the full workspace representation returned by the API.
type WorkspaceDetail struct {
	Name       string             `json:"name"`
	Namespace  string             `json:"namespace"`
	Image      string             `json:"image"`
	Repos      []RepoDetail       `json:"repos"`
	Storage    StorageDetail      `json:"storage"`
	Compo      *CompoDetail       `json:"compo,omitempty"`
	Suspended  bool               `json:"suspended"`
	Phase      string             `json:"phase"`
	Conditions []ConditionSummary `json:"conditions,omitempty"`
	Labels     map[string]string  `json:"labels,omitempty"`
}

// RepoDetail is the sanitized repo representation. HasCredential indicates
// whether a secretRef is configured, without exposing secret contents.
type RepoDetail struct {
	URL           string `json:"url"`
	Branch        string `json:"branch"`
	Path          string `json:"path"`
	Provider      string `json:"provider"`
	HasCredential bool   `json:"hasCredential"`
}

// StorageDetail describes the workspace storage configuration.
type StorageDetail struct {
	StorageClassName string `json:"storageClassName"`
	Size             string `json:"size"`
}

// CompoDetail is the sanitized composition repo representation.
type CompoDetail struct {
	URL           string `json:"url"`
	Branch        string `json:"branch"`
	Provider      string `json:"provider"`
	HasCredential bool   `json:"hasCredential"`
}

// ConditionSummary is a simplified representation of a metav1.Condition.
type ConditionSummary struct {
	Type    string `json:"type"`
	Status  string `json:"status"`
	Reason  string `json:"reason"`
	Message string `json:"message"`
}

// WorkspaceSummary is a lightweight workspace representation for list responses.
type WorkspaceSummary struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Phase     string `json:"phase"`
	Suspended bool   `json:"suspended"`
	Image     string `json:"image"`
}

// --- Service interface + implementation ---

// WorkspaceService provides business logic for workspace operations.
type WorkspaceService interface {
	Create(ctx context.Context, namespace string, req CreateWorkspaceRequest) (*WorkspaceDetail, error)
	Get(ctx context.Context, namespace, name string) (*WorkspaceDetail, error)
	List(ctx context.Context, namespace string) ([]WorkspaceSummary, error)
	Delete(ctx context.Context, namespace, name string) error
	Suspend(ctx context.Context, namespace, name string) (*WorkspaceDetail, error)
	Resume(ctx context.Context, namespace, name string) (*WorkspaceDetail, error)
}

type workspaceService struct {
	workspaces adapter.WorkspaceAdapter
	secrets    adapter.SecretAdapter
}

// Compile-time interface check.
var _ WorkspaceService = (*workspaceService)(nil)

// NewWorkspaceService creates a WorkspaceService.
func NewWorkspaceService(ws adapter.WorkspaceAdapter, secrets adapter.SecretAdapter) WorkspaceService {
	return &workspaceService{workspaces: ws, secrets: secrets}
}

// Create builds credential Secrets, creates the Workspace CR, and returns a WorkspaceDetail.
func (s *workspaceService) Create(ctx context.Context, namespace string, req CreateWorkspaceRequest) (*WorkspaceDetail, error) {
	// Build repo refs and create secrets for repos that provide inline credentials.
	repos := make([]v1alpha1.RepoRef, len(req.Repos))
	for i, ri := range req.Repos {
		repos[i] = v1alpha1.RepoRef{
			URL:      ri.URL,
			Ref:      toGitRef(ri.Ref),
			Path:     ri.Path,
			Provider: ri.Provider,
		}

		secretName, err := s.resolveRepoAuth(ctx, namespace, req.Name, i, ri.Auth)
		if err != nil {
			return nil, fmt.Errorf("creating secret for repo %d: %w", i, err)
		}
		if secretName != "" {
			repos[i].SecretRef = &v1alpha1.SecretRef{Name: secretName}
		}
	}

	// Build compo ref and create secret if needed.
	var compoRef *v1alpha1.CompoRef
	if req.Compo != nil {
		compoRef = &v1alpha1.CompoRef{
			URL:      req.Compo.URL,
			Ref:      toGitRef(req.Compo.Ref),
			Provider: req.Compo.Provider,
		}

		secretName, err := s.resolveCompoAuth(ctx, namespace, req.Name, req.Compo.Auth)
		if err != nil {
			return nil, fmt.Errorf("creating secret for compo: %w", err)
		}
		if secretName != "" {
			compoRef.SecretRef = &v1alpha1.SecretRef{Name: secretName}
		}
	}

	ws := &v1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      req.Name,
			Namespace: namespace,
			Labels:    req.Labels,
		},
		Spec: v1alpha1.WorkspaceSpec{
			Image:      req.Image,
			InitScript: req.InitScript,
			Repos:      repos,
			Storage: v1alpha1.StorageSpec{
				StorageClassName: req.StorageClass,
				Size:             req.StorageSize,
			},
			CompoRef: compoRef,
		},
	}

	if err := s.workspaces.Create(ctx, ws); err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	detail := toWorkspaceDetail(ws)
	return &detail, nil
}

// Get retrieves a single workspace and returns a WorkspaceDetail.
func (s *workspaceService) Get(ctx context.Context, namespace, name string) (*WorkspaceDetail, error) {
	ws, err := s.workspaces.Get(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	detail := toWorkspaceDetail(ws)
	return &detail, nil
}

// List returns all workspaces in a namespace as summaries.
func (s *workspaceService) List(ctx context.Context, namespace string) ([]WorkspaceSummary, error) {
	list, err := s.workspaces.List(ctx, namespace)
	if err != nil {
		return nil, err
	}
	summaries := make([]WorkspaceSummary, len(list.Items))
	for i := range list.Items {
		summaries[i] = toWorkspaceSummary(&list.Items[i])
	}
	return summaries, nil
}

// Delete removes a workspace CR. Secrets without ownerReferences may be
// leaked; the reconciler handles garbage collection in a future enhancement.
func (s *workspaceService) Delete(ctx context.Context, namespace, name string) error {
	return s.workspaces.Delete(ctx, namespace, name)
}

// Suspend sets spec.suspend = true on the workspace.
func (s *workspaceService) Suspend(ctx context.Context, namespace, name string) (*WorkspaceDetail, error) {
	return s.setSuspend(ctx, namespace, name, true)
}

// Resume sets spec.suspend = false on the workspace.
func (s *workspaceService) Resume(ctx context.Context, namespace, name string) (*WorkspaceDetail, error) {
	return s.setSuspend(ctx, namespace, name, false)
}

func (s *workspaceService) setSuspend(ctx context.Context, namespace, name string, suspend bool) (*WorkspaceDetail, error) {
	ws, err := s.workspaces.Get(ctx, namespace, name)
	if err != nil {
		return nil, err
	}
	ws.Spec.Suspend = suspend
	if err := s.workspaces.Update(ctx, ws); err != nil {
		return nil, err
	}
	detail := toWorkspaceDetail(ws)
	return &detail, nil
}

// --- Auth helpers ---

// resolveRepoAuth creates a Secret for inline credentials or returns the
// existing secret name. Returns empty string if no auth is provided.
func (s *workspaceService) resolveRepoAuth(ctx context.Context, namespace, wsName string, idx int, auth *AuthInput) (string, error) {
	if auth == nil {
		return "", nil
	}
	if auth.ExistingSecretName != "" {
		return auth.ExistingSecretName, nil
	}
	secretName := fmt.Sprintf("ws-%s-repo-%d-creds", wsName, idx)
	secret := buildSecret(namespace, secretName, auth)
	if err := s.secrets.Create(ctx, secret); err != nil {
		return "", err
	}
	return secretName, nil
}

// resolveCompoAuth creates a Secret for inline compo credentials or returns
// the existing secret name.
func (s *workspaceService) resolveCompoAuth(ctx context.Context, namespace, wsName string, auth *AuthInput) (string, error) {
	if auth == nil {
		return "", nil
	}
	if auth.ExistingSecretName != "" {
		return auth.ExistingSecretName, nil
	}
	secretName := fmt.Sprintf("ws-%s-compo-creds", wsName)
	secret := buildSecret(namespace, secretName, auth)
	if err := s.secrets.Create(ctx, secret); err != nil {
		return "", err
	}
	return secretName, nil
}

// --- Conversion helpers ---

// toWorkspaceDetail converts a Workspace CR to a WorkspaceDetail, masking secrets.
func toWorkspaceDetail(ws *v1alpha1.Workspace) WorkspaceDetail {
	repos := make([]RepoDetail, len(ws.Spec.Repos))
	for i, r := range ws.Spec.Repos {
		repos[i] = RepoDetail{
			URL:           r.URL,
			Branch:        r.Ref.Branch,
			Path:          r.Path,
			Provider:      r.Provider,
			HasCredential: r.SecretRef != nil,
		}
	}

	var compo *CompoDetail
	if ws.Spec.CompoRef != nil {
		compo = &CompoDetail{
			URL:           ws.Spec.CompoRef.URL,
			Branch:        ws.Spec.CompoRef.Ref.Branch,
			Provider:      ws.Spec.CompoRef.Provider,
			HasCredential: ws.Spec.CompoRef.SecretRef != nil,
		}
	}

	conditions := make([]ConditionSummary, len(ws.Status.Conditions))
	for i, c := range ws.Status.Conditions {
		conditions[i] = ConditionSummary{
			Type:    c.Type,
			Status:  string(c.Status),
			Reason:  c.Reason,
			Message: c.Message,
		}
	}

	return WorkspaceDetail{
		Name:      ws.Name,
		Namespace: ws.Namespace,
		Image:     ws.Spec.Image,
		Repos:     repos,
		Storage: StorageDetail{
			StorageClassName: ws.Spec.Storage.StorageClassName,
			Size:             ws.Spec.Storage.Size,
		},
		Compo:      compo,
		Suspended:  ws.Spec.Suspend,
		Phase:      string(ws.Status.Phase),
		Conditions: conditions,
		Labels:     ws.Labels,
	}
}

// toWorkspaceSummary converts a Workspace CR to a lightweight WorkspaceSummary.
func toWorkspaceSummary(ws *v1alpha1.Workspace) WorkspaceSummary {
	return WorkspaceSummary{
		Name:      ws.Name,
		Namespace: ws.Namespace,
		Phase:     string(ws.Status.Phase),
		Suspended: ws.Spec.Suspend,
		Image:     ws.Spec.Image,
	}
}

// buildSecret creates a corev1.Secret from an AuthInput.
func buildSecret(namespace, name string, auth *AuthInput) *corev1.Secret {
	data := make(map[string][]byte)

	if auth.SSHPrivateKey != "" {
		data["identity"] = []byte(auth.SSHPrivateKey)
	}
	if auth.KnownHosts != "" {
		data["known_hosts"] = []byte(auth.KnownHosts)
	}
	if auth.Username != "" {
		data["username"] = []byte(auth.Username)
	}
	if auth.Password != "" {
		data["password"] = []byte(auth.Password)
	}
	if auth.BearerToken != "" {
		data["bearerToken"] = []byte(auth.BearerToken)
	}
	if auth.CAFile != "" {
		data["caFile"] = []byte(auth.CAFile)
	}

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: data,
	}
}

// toGitRef converts a GitRefInput to a v1alpha1.GitRef.
func toGitRef(ref GitRefInput) v1alpha1.GitRef {
	return v1alpha1.GitRef{
		Branch: ref.Branch,
		Tag:    ref.Tag,
		Commit: ref.Commit,
	}
}
