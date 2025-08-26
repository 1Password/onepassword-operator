package kube

import (
	"context"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
)

type Deployment struct {
	client client.Client
	config *ClusterConfig
	name   string
}

func (d *Deployment) ReadEnvVar(ctx context.Context, envVarName string) string {
	By("Reading " + envVarName + " value from deployment/" + d.name)

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	deployment := &appsv1.Deployment{}
	err := d.client.Get(c, client.ObjectKey{Name: d.name, Namespace: d.config.Namespace}, deployment)
	Expect(err).ToNot(HaveOccurred())

	// Search env across all containers
	found := ""
	for _, container := range deployment.Spec.Template.Spec.Containers {
		for _, env := range container.Env {
			if env.Name == envVarName && env.Value != "" {
				found = env.Value
				break
			}
		}
	}

	Expect(found).NotTo(BeEmpty())
	return found
}
