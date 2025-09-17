package kube

import (
	"context"
	"io"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Pod struct {
	client    client.Client
	clientset kubernetes.Interface
	config    *Config
	selector  map[string]string
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

func (p *Pod) GetPodLogs(ctx context.Context) string {
	// First find the pod by label selector
	var pods corev1.PodList
	listOpts := []client.ListOption{
		client.InNamespace(p.config.Namespace),
		client.MatchingLabels(p.selector),
	}
	err := p.client.List(ctx, &pods, listOpts...)
	Expect(err).NotTo(HaveOccurred())
	Expect(pods.Items).NotTo(BeEmpty(), "no pods found with selector %q", labels.Set(p.selector).String())

	// Use the first pod found
	pod := pods.Items[0]
	podName := pod.Name

	// Verify pod is running before getting logs
	Expect(pod.Status.Phase).To(Equal(corev1.PodRunning), "pod %s is not running (status: %s)", podName, pod.Status.Phase)

	// Get logs using the Kubernetes clientset
	req := p.clientset.CoreV1().Pods(p.config.Namespace).GetLogs(podName, &corev1.PodLogOptions{})
	stream, err := req.Stream(context.TODO())
	Expect(err).NotTo(HaveOccurred(), "failed to stream logs for pod %s", podName)
	defer stream.Close()

	// Read all logs from the stream
	logs, err := io.ReadAll(stream)
	Expect(err).NotTo(HaveOccurred(), "failed to read logs for pod %s", podName)

	return string(logs)
}

func (p *Pod) VerifyWebhookInjection(ctx context.Context) {
	By("Verifying webhook injection for pod with selector " + labels.Set(p.selector).String())

	Eventually(func(g Gomega) {
		// short per-attempt timeout to avoid hanging calls while Eventually polls
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		// First find the pod by label selector
		var pods corev1.PodList
		listOpts := []client.ListOption{
			client.InNamespace(p.config.Namespace),
			client.MatchingLabels(p.selector),
		}
		g.Expect(p.client.List(attemptCtx, &pods, listOpts...)).To(Succeed())
		g.Expect(pods.Items).NotTo(BeEmpty(), "no pods found with selector %q", labels.Set(p.selector).String())

		// Use the first pod found
		pod := pods.Items[0]

		// Check injection status annotation
		g.Expect(pod.Annotations).To(HaveKey("operator.1password.io/status"))
		g.Expect(pod.Annotations["operator.1password.io/status"]).To(Equal("injected"))

		// Check command was modified to use op run
		if len(pod.Spec.Containers) > 0 {
			container := pod.Spec.Containers[0]
			g.Expect(container.Command).To(HaveLen(4))
			g.Expect(container.Command[0]).To(Equal("/op/bin/op"))
			g.Expect(container.Command[1]).To(Equal("run"))
			g.Expect(container.Command[2]).To(Equal("--"))
		}

		// Check init container was added
		g.Expect(pod.Spec.InitContainers).To(HaveLen(1))
		g.Expect(pod.Spec.InitContainers[0].Name).To(Equal("copy-op-bin"))

		// Check volume mount was added
		g.Expect(pod.Spec.Containers[0].VolumeMounts).To(ContainElement(HaveField("Name", "op-bin")))
	}, p.config.TestConfig.Timeout, p.config.TestConfig.Interval).Should(Succeed())
}

func (p *Pod) VerifySecretsInjected(ctx context.Context) {
	By("Verifying secrets are injected and concealed in pod with selector " + labels.Set(p.selector).String())

	Eventually(func(g Gomega) {
		// short per-attempt timeout to avoid hanging calls while Eventually polls
		attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()

		logs := p.GetPodLogs(attemptCtx)
		// Check that secrets are concealed in the application logs
		g.Expect(logs).To(ContainSubstring("SECRET: '<concealed by 1Password>'"))
	}, p.config.TestConfig.Timeout, p.config.TestConfig.Interval).Should(Succeed())
}
