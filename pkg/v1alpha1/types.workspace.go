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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&Workspace{}, &WorkspaceList{})
}

// WorkspacePhase represents the lifecycle phase of a Workspace.
type WorkspacePhase string

const (
	WorkspacePending WorkspacePhase = "Pending"
	WorkspaceRunning WorkspacePhase = "Running"
	WorkspaceStopped WorkspacePhase = "Stopped"
	WorkspaceFailed  WorkspacePhase = "Failed"
)

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Workspace is the Schema for the workspaces API
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WorkspaceSpec   `json:"spec,omitempty"`
	Status WorkspaceStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// WorkspaceList contains a list of Workspace
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Workspace `json:"items"`
}

// WorkspaceSpec defines the desired state of Workspace
type WorkspaceSpec struct {
	// Suspend pauses reconciliation. Existing resources stay. Default: false.
	Suspend bool `json:"suspend,omitempty"`
	// Image is the container image for the workspace pod.
	Image string `json:"image,omitempty"`
	// InitScript is a post-start script executed in the workspace.
	InitScript string `json:"initScript,omitempty"`
	// Repos is a list of git repositories to clone into the workspace.
	Repos []RepoRef `json:"repos,omitempty"`
	// Storage configures the PVC for the workspace.
	Storage StorageSpec `json:"storage"`
	// CompoRef references a forge-cu composition repository.
	CompoRef *CompoRef `json:"compoRef,omitempty"`
	// ServiceAccountName for workload identity.
	ServiceAccountName string `json:"serviceAccountName,omitempty"`
}

// RepoRef specifies a git repository to clone into the workspace.
type RepoRef struct {
	// URL of the git repository.
	URL string `json:"url"`
	// Ref specifies which revision to check out.
	Ref GitRef `json:"ref,omitempty"`
	// Path is the mount path inside /workspace.
	Path string `json:"path,omitempty"`
	// Provider is the git provider (generic, github, gitlab, azure).
	Provider string `json:"provider,omitempty"`
	// SecretRef references credentials for the repository.
	SecretRef *SecretRef `json:"secretRef,omitempty"`
}

// GitRef specifies which revision to check out.
// Only one field should be set. Priority: commit > tag > branch.
type GitRef struct {
	Branch string `json:"branch,omitempty"`
	Tag    string `json:"tag,omitempty"`
	Commit string `json:"commit,omitempty"`
}

// StorageSpec configures the PVC for the workspace.
type StorageSpec struct {
	// StorageClassName is the storage class to use.
	StorageClassName string `json:"storageClassName"`
	// Size is the requested storage size (e.g., 20Gi).
	Size string `json:"size"`
}

// CompoRef references a forge-cu composition repository.
type CompoRef struct {
	// URL of the composition repository.
	URL string `json:"url"`
	// Ref specifies which revision to check out.
	Ref GitRef `json:"ref,omitempty"`
	// Provider is the git provider (generic, github, gitlab, azure).
	Provider string `json:"provider,omitempty"`
	// SecretRef references credentials for the composition repository.
	SecretRef *SecretRef `json:"secretRef,omitempty"`
}

// WorkspaceStatus defines the observed state of Workspace
type WorkspaceStatus struct {
	Phase              WorkspacePhase     `json:"phase,omitempty"`
	PodName            string             `json:"podName,omitempty"`
	PVCName            string             `json:"pvcName,omitempty"`
	ServiceURL         string             `json:"serviceURL,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}
