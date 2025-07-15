/*
MIT License

Copyright (c) 2020-2024 1Password

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnePasswordItemSpec defines the desired state of OnePasswordItem
type OnePasswordItemSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	ItemPath string `json:"itemPath,omitempty"`
}

type OnePasswordItemConditionType string

const (
	// OnePasswordItemReady means the Kubernetes secret is ready for use.
	OnePasswordItemReady OnePasswordItemConditionType = "Ready"
)

type OnePasswordItemCondition struct {
	// Type of job condition, Completed.
	Type OnePasswordItemConditionType `json:"type"`
	// Status of the condition, one of True, False, Unknown.
	Status metav1.ConditionStatus `json:"status"`
	// Last time the condition transit from one status to another.
	// +optional
	LastTransitionTime metav1.Time `json:"lastTransitionTime,omitempty"`
	// Human-readable message indicating details about last transition.
	// +optional
	Message string `json:"message,omitempty"`
}

// OnePasswordItemStatus defines the observed state of OnePasswordItem
type OnePasswordItemStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Conditions []OnePasswordItemCondition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// OnePasswordItem is the Schema for the onepassworditems API
type OnePasswordItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Kubernetes secret type. More info: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types
	Type   string                `json:"type,omitempty"`
	Spec   OnePasswordItemSpec   `json:"spec,omitempty"`
	Status OnePasswordItemStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// OnePasswordItemList contains a list of OnePasswordItem
type OnePasswordItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OnePasswordItem `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OnePasswordItem{}, &OnePasswordItemList{})
}
