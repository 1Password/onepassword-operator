package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// OnePasswordItemSpec defines the desired state of OnePasswordItem
type OnePasswordItemSpec struct {
	ItemReference string `json:"itemReference,omitempty"`
}

// OnePasswordItemStatus defines the observed state of OnePasswordItem
type OnePasswordItemStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// OnePasswordItem is the Schema for the onepassworditems API
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=onepassworditems,scope=Namespaced
type OnePasswordItem struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

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
