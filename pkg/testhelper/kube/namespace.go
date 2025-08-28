package kube

import (
	"context"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Namespace struct {
	client client.Client
	config *Config
	name   string
}

// LabelNamespace applies the given labels to the specified namespace
func (n *Namespace) LabelNamespace(ctx context.Context, labelsMap map[string]string) {
	if len(labelsMap) == 0 {
		return
	}

	By("Setting labelsMap " + labels.Set(labelsMap).String() + " to namespace/" + n.name)
	ns := &corev1.Namespace{}
	err := n.client.Get(ctx, client.ObjectKey{Name: n.name}, ns)
	Expect(err).NotTo(HaveOccurred())

	if ns.Labels == nil {
		ns.Labels = map[string]string{}
	}

	for k, v := range labelsMap {
		ns.Labels[k] = v
	}

	err = n.client.Update(ctx, ns)
	Expect(err).NotTo(HaveOccurred())
}
