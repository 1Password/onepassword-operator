package kube

import (
	"context"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	admissionregistrationv1 "k8s.io/api/admissionregistration/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Webhook struct {
	client client.Client
	config *Config
	name   string
}

func (w *Webhook) WaitForWebhookToBeRegistered(ctx context.Context) {
	By("Waiting for webhook " + w.name + " to be registered")

	Eventually(func(g Gomega) {
		// short per-attempt timeout to avoid hanging calls while Eventually polls
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		webhookConfig := &admissionregistrationv1.MutatingWebhookConfiguration{}
		err := w.client.Get(attemptCtx, client.ObjectKey{Name: w.name}, webhookConfig)
		g.Expect(err).ToNot(HaveOccurred())
	}, w.config.TestConfig.Timeout, w.config.TestConfig.Interval).Should(Succeed())
}
