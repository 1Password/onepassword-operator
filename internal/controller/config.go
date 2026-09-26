package controller

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type ReconcilerConfig struct {
	EnableAnnotations bool
	AllowEmptyValues  bool
	// LabelSelector, when non-nil, restricts reconciliation to OnePasswordItem
	// resources whose labels match the selector. A nil selector reconciles all
	// items (the default).
	LabelSelector *metav1.LabelSelector
}
