package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnePasswordItemSpec defines the desired state of OnePasswordItem
type OnePasswordItemSpec struct {
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
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	Conditions []OnePasswordItemCondition `json:"conditions"`

	// True when the Kubernetes secret is ready for use.
	Ready *bool `json:"ready,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OnePasswordItem is the Schema for the onepassworditems API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=onepassworditems,scope=Namespaced
type OnePasswordItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Kubernetes secret type. More info: https://kubernetes.io/docs/concepts/configuration/secret/#secret-types
	Type              string `json:"type,omitempty"`

	Spec   OnePasswordItemSpec   `json:"spec,omitempty"`
	Status OnePasswordItemStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OnePasswordItemList contains a list of OnePasswordItem
type OnePasswordItemList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []OnePasswordItem `json:"items"`
}

func init() {
	SchemeBuilder.Register(&OnePasswordItem{}, &OnePasswordItemList{})
}
