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

package types

// ------------------------------------------------ WORKSPACE STATE -------------------------------------------------- //

// WorkspaceState is an enriched view of a Workspace with computed fields.
type WorkspaceState struct {
	// Name is the name of the workspace.
	Name string
	// Namespace is the namespace of the workspace.
	Namespace string
	// Phase is the current phase of the workspace lifecycle.
	Phase string
	// Suspended indicates whether the workspace is suspended.
	Suspended bool
	// PodReady indicates whether the workspace pod is ready.
	PodReady bool
	// PVCBound indicates whether the workspace PVC is bound.
	PVCBound bool
	// ServiceURL is the URL of the workspace service.
	ServiceURL string
	// CredentialType is the auth method used for the repository.
	CredentialType CredentialType
}

// -------------------------------------------------- POD STATUS ---------------------------------------------------- //

// PodStatus represents the runtime state of a workspace pod.
type PodStatus struct {
	// Phase is the current phase of the pod.
	Phase string
	// Ready indicates whether the pod is ready.
	Ready bool
	// Conditions is a list of pod condition types.
	Conditions []string
}

// ------------------------------------------------- STORAGE INFO --------------------------------------------------- //

// StorageInfo represents the state of workspace storage.
type StorageInfo struct {
	// Bound indicates whether the PVC is bound.
	Bound bool
	// Size is the requested storage size.
	Size string
}

// ----------------------------------------------- CREDENTIAL TYPE -------------------------------------------------- //

// CredentialType identifies the auth method used for a repository.
type CredentialType int

const (
	// CredentialNone indicates no credentials are configured.
	CredentialNone CredentialType = iota
	// CredentialSSH indicates SSH key-based authentication.
	CredentialSSH
	// CredentialHTTPSBasic indicates HTTPS basic authentication.
	CredentialHTTPSBasic
	// CredentialHTTPSToken indicates HTTPS token-based authentication.
	CredentialHTTPSToken
)

// String returns the string representation of a CredentialType.
func (c CredentialType) String() string {
	switch c {
	case CredentialSSH:
		return "ssh"
	case CredentialHTTPSBasic:
		return "https-basic"
	case CredentialHTTPSToken:
		return "https-token"
	default:
		return "none"
	}
}
