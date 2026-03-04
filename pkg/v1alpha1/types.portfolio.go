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
	SchemeBuilder.Register(&Portfolio{}, &PortfolioList{})
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Portfolio is the Schema for the portfolios API
type Portfolio struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PortfolioSpec   `json:"spec,omitempty"`
	Status PortfolioStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// PortfolioList contains a list of Portfolio
type PortfolioList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []Portfolio `json:"items"`
}

// PortfolioSpec defines the desired state of Portfolio
type PortfolioSpec struct {
	// Suspend pauses reconciliation. Default: false.
	Suspend bool `json:"suspend,omitempty"`
	// Selector selects Workspaces by labels.
	Selector *metav1.LabelSelector `json:"selector,omitempty"`
	// Description is a human-readable description of the portfolio.
	Description string `json:"description,omitempty"`
}

// PortfolioStatus defines the observed state of Portfolio
type PortfolioStatus struct {
	WorkspaceCount     int                `json:"workspaceCount"`
	WorkspaceNames     []string           `json:"workspaceNames,omitempty"`
	ObservedGeneration int64              `json:"observedGeneration,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
}
