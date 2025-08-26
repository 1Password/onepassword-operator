package kube

import (
	"context"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pod struct {
	client   client.Client
	config   *Config
	selector map[string]string
}

func (p *Pod) WaitingForRunningPod(ctx context.Context) {
	By("Waiting for the pod " + labels.Set(p.selector).String() + " to be 'Running'")

	Eventually(func(g Gomega) {
		// short per-attempt timeout to avoid hanging calls while Eventually polls
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		var pods corev1.PodList
		listOpts := []client.ListOption{
			client.InNamespace(p.config.Namespace),
			client.MatchingLabels(p.selector),
		}
		g.Expect(p.client.List(attemptCtx, &pods, listOpts...)).To(Succeed())
		g.Expect(pods.Items).NotTo(BeEmpty(), "no pods found with selector %q", labels.Set(p.selector).String())

		foundRunning := false
		for _, p := range pods.Items {
			if p.Status.Phase == corev1.PodRunning {
				foundRunning = true
				break
			}
		}
		g.Expect(foundRunning).To(BeTrue(), "pod not Running yet")
	}, p.config.TestConfig.Timeout, p.config.TestConfig.Interval).Should(Succeed())
}
