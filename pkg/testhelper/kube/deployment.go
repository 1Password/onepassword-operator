package kube

import (
	"context"
	"fmt"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Deployment struct {
	client client.Client
	config *Config
	name   string
}

func (d *Deployment) Get(ctx context.Context) *appsv1.Deployment {
	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	deployment := &appsv1.Deployment{}
	err := d.client.Get(c, client.ObjectKey{Name: d.name, Namespace: d.config.Namespace}, deployment)
	Expect(err).ToNot(HaveOccurred())

	return deployment
}

func (d *Deployment) ReadEnvVar(ctx context.Context, envVarName string) string {
	By("Reading " + envVarName + " value from deployment/" + d.name)
	deployment := d.Get(ctx)

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

func (d *Deployment) PatchEnvVars(ctx context.Context, upsert []corev1.EnvVar, remove []string) {
	By("Patching env variables for deployment/" + d.name)
	deployment := d.Get(ctx)
	deploymentCopy := deployment.DeepCopy()
	container := &deployment.Spec.Template.Spec.Containers[0]

	// Build removal set for quick lookup
	toRemove := make(map[string]struct{}, len(remove))
	for _, n := range remove {
		toRemove[n] = struct{}{}
	}

	// Build upsert map for quick lookup
	upserts := make(map[string]corev1.EnvVar, len(upsert))
	for _, e := range upsert {
		upserts[e.Name] = e
	}

	// Filter existing envs: keep if not in remove and not being upserted
	filtered := make([]corev1.EnvVar, 0, len(container.Env))
	for _, e := range container.Env {
		if _, ok := toRemove[e.Name]; ok {
			continue
		}
		if newE, ok := upserts[e.Name]; ok {
			filtered = append(filtered, newE) // replace existing
			delete(upserts, e.Name)           // delete from map to not use once again
		} else {
			filtered = append(filtered, e)
		}
	}

	// Append any new envs that weren’t already in the container
	for _, e := range upserts {
		filtered = append(filtered, e)
	}

	container.Env = filtered

	// Derive a short-lived context so this API call won't hang indefinitely.
	c, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	err := d.client.Patch(c, deployment, client.MergeFrom(deploymentCopy))
	Expect(err).ToNot(HaveOccurred())

	// wait for new deployment to roll out
	d.WaitDeploymentRolledOut(ctx)
}

// WaitDeploymentRolledOut waits for deployment to finish a rollout.
func (d *Deployment) WaitDeploymentRolledOut(ctx context.Context) {
	By("Waiting for deployment/" + d.name + " to roll out")

	deployment := d.Get(ctx)
	targetGen := deployment.Generation

	Eventually(func(g Gomega) error {
		newDeployment := d.Get(ctx)
		// Has controller observed the new spec?
		if newDeployment.Status.ObservedGeneration < targetGen {
			return fmt.Errorf("observedGeneration %d < desired %d", newDeployment.Status.ObservedGeneration, targetGen)
		}
		g.Expect(newDeployment.Status.ObservedGeneration).To(BeNumerically(">=", targetGen))

		desired := int32(1)
		if newDeployment.Spec.Replicas != nil {
			desired = *newDeployment.Spec.Replicas
		}

		g.Expect(newDeployment.Status.UpdatedReplicas).To(Equal(desired))
		g.Expect(newDeployment.Status.AvailableReplicas).To(Equal(desired))
		g.Expect(newDeployment.Status.Replicas).To(Equal(desired))

		return nil
	}, d.config.TestConfig.Timeout, d.config.TestConfig.Interval).Should(Succeed())
}
