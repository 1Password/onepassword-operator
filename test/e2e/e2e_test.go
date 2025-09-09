package e2e

import (
	"context"
	"path/filepath"
	"strconv"
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	"github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kind"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kube"
	"github.com/1Password/onepassword-operator/pkg/testhelper/op"
	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

const (
	operatorImageName = "1password/onepassword-operator:latest"
	vaultName         = "operator-acceptance-tests"
)

var kubeClient *kube.Kube

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	ctx := context.Background()

	BeforeAll(func() {
		rootDir, err := system.GetProjectRoot()
		Expect(err).NotTo(HaveOccurred())

		kubeClient = kube.NewKubeClient(&kube.Config{
			Namespace:    "default",
			ManifestsDir: filepath.Join("manifests"),
			TestConfig: &kube.TestConfig{
				Timeout:  defaults.E2ETimeout,
				Interval: defaults.E2EInterval,
			},
			CRDs: []string{
				filepath.Join(rootDir, "config", "crd", "bases", "onepassword.com_onepassworditems.yaml"),
			},
		})

		By("Building the Operator image")
		_, err = system.Run("make", "docker-build")
		Expect(err).NotTo(HaveOccurred())

		kind.LoadImageToKind(operatorImageName)

		kubeClient.Secret("op-credentials").CreateOpCredentials(ctx)
		kubeClient.Secret("op-credentials").CheckIfExists(ctx)

		kubeClient.Secret("onepassword-token").CreateFromEnvVar(ctx, "OP_CONNECT_TOKEN")
		kubeClient.Secret("onepassword-token").CheckIfExists(ctx)

		kubeClient.Secret("onepassword-service-account-token").CreateFromEnvVar(ctx, "OP_SERVICE_ACCOUNT_TOKEN")
		kubeClient.Secret("onepassword-service-account-token").CheckIfExists(ctx)

		By("Replace manager.yaml")
		err = system.ReplaceFile("test/e2e/manifests/manager.yaml", "config/manager/manager.yaml")
		Expect(err).NotTo(HaveOccurred())

		_, err = system.Run("make", "deploy")
		Expect(err).NotTo(HaveOccurred())
		kubeClient.Pod(map[string]string{"name": "onepassword-connect-operator"}).WaitingForRunningPod(ctx)
	})

	Context("Use the operator with Service Account", func() {
		runCommonTestCases(ctx)
	})

	Context("Use the operator with Connect", func() {
		BeforeAll(func() {
			kubeClient.Deployment("onepassword-connect-operator").PatchEnvVars(ctx, []corev1.EnvVar{
				{Name: "MANAGE_CONNECT", Value: "true"},
				{Name: "OP_CONNECT_HOST", Value: "http://onepassword-connect:8080"},
				{
					Name: "OP_CONNECT_TOKEN",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "onepassword-token",
							},
							Key: "token",
						},
					},
				},
			}, []string{"OP_SERVICE_ACCOUNT_TOKEN"})

			kubeClient.Secret("login").Delete(ctx)             // remove secret crated in previous test
			kubeClient.Secret("secret-ignored").Delete(ctx)    // remove secret crated in previous test
			kubeClient.Secret("secret-for-update").Delete(ctx) // remove secret crated in previous test

			kubeClient.Pod(map[string]string{"app": "onepassword-connect"}).WaitingForRunningPod(ctx)
		})

		runCommonTestCases(ctx)
	})
})

// runCommonTestCases contains test cases that are common to both Connect and Service Account authentication methods.
func runCommonTestCases(ctx context.Context) {
	It("Should create kubernetes secret from manifest file", func() {
		By("Creating secret `login` from 1Password item")
		kubeClient.ApplyOnePasswordItem(ctx, "secret.yaml")
		kubeClient.Secret("login").CheckIfExists(ctx)
	})

	It("Kubernetes secret is updated after POOLING_INTERVAL, when updating item in 1Password", func() {
		itemName := "secret-for-update"
		secretName := itemName

		By("Creating secret `" + secretName + "` from 1Password item")
		kubeClient.ApplyOnePasswordItem(ctx, secretName+".yaml")
		kubeClient.Secret(secretName).CheckIfExists(ctx)

		By("Reading old password")
		secret := kubeClient.Secret(secretName).Get(ctx)
		oldPassword, ok := secret.Data["password"]
		Expect(ok).To(BeTrue())

		By("Updating `" + secretName + "` 1Password item")
		err := op.UpdateItemPassword(itemName)
		Expect(err).NotTo(HaveOccurred())

		// checking that password was updated
		Eventually(func(g Gomega) {
			// Derive a short-lived context so this API call won't hang indefinitely.
			attemptCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			secret = kubeClient.Secret(secretName).Get(attemptCtx)
			g.Expect(err).NotTo(HaveOccurred())

			newPassword, ok := secret.Data["password"]
			g.Expect(ok).To(BeTrue())
			g.Expect(newPassword).NotTo(Equal(oldPassword))
		}, defaults.E2ETimeout, defaults.E2EInterval).Should(Succeed())
	})

	It("1Password item with `ignore-secret` tag doesn't pull updates to kubernetes secret", func() {
		itemName := "secret-ignored"
		secretName := itemName

		By("Creating secret `" + secretName + "` from 1Password item")
		kubeClient.ApplyOnePasswordItem(ctx, secretName+".yaml")
		kubeClient.Secret(secretName).CheckIfExists(ctx)

		By("Reading old password")
		secret := kubeClient.Secret(secretName).Get(ctx)
		oldPassword, ok := secret.Data["password"]
		Expect(ok).To(BeTrue())

		By("Updating `" + secretName + "` 1Password item")
		err := op.UpdateItemPassword(itemName)
		Expect(err).NotTo(HaveOccurred())

		newPassword, err := op.ReadItemField(itemName, vaultName, op.FieldPassword)
		Expect(err).NotTo(HaveOccurred())
		Expect(newPassword).NotTo(BeEquivalentTo(oldPassword))

		// checking that password was NOT updated
		Eventually(func(g Gomega) {
			// Derive a short-lived context so this API call won't hang indefinitely.
			attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			intervalStr := kubeClient.Deployment("onepassword-connect-operator").ReadEnvVar(attemptCtx, "POLLING_INTERVAL")
			Expect(intervalStr).NotTo(BeEmpty())

			i, err := strconv.Atoi(intervalStr)
			Expect(err).NotTo(HaveOccurred())

			// convert to duration in seconds
			interval := time.Duration(i) * time.Second
			// wait for one polling interval + 2 seconds to make sure updated secret is pulled
			time.Sleep(interval + 2*time.Second)

			secret = kubeClient.Secret(secretName).Get(attemptCtx)
			g.Expect(err).NotTo(HaveOccurred())

			currentPassword, ok := secret.Data["password"]
			Expect(ok).To(BeTrue())
			Expect(currentPassword).To(BeEquivalentTo(oldPassword))
			Expect(currentPassword).NotTo(BeEquivalentTo(newPassword))
		}, defaults.E2ETimeout, defaults.E2EInterval).Should(Succeed())
	})

	It("AUTO_RESTART restarts deployments using 1Password secrets after item update", func() {
		By("Enabling AUTO_RESTART")
		kubeClient.Deployment("onepassword-connect-operator").PatchEnvVars(ctx, []corev1.EnvVar{
			{Name: "AUTO_RESTART", Value: "true"},
		}, nil)

		DeferCleanup(func() {
			By("Disabling AUTO_RESTART")
			// disable AUTO_RESTART after test
			kubeClient.Deployment("onepassword-connect-operator").PatchEnvVars(ctx, []corev1.EnvVar{
				{Name: "AUTO_RESTART", Value: "false"},
			}, nil)
		})

		// Ensure the secret exists (created in earlier test), but apply again safely just in case
		kubeClient.ApplyOnePasswordItem(ctx, "secret-for-update.yaml")
		kubeClient.Secret("secret-for-update").CheckIfExists(ctx)

		// add custom secret to the operator
		kubeClient.Deployment("onepassword-connect-operator").PatchEnvVars(ctx, []corev1.EnvVar{
			{
				Name: "CUSTOM_SECRET",
				ValueFrom: &corev1.EnvVarSource{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: "secret-for-update",
						},
						Key: "password",
					},
				},
			},
		}, nil)

		By("Updating `secret-for-update` 1Password item")
		err := op.UpdateItemPassword("secret-for-update")
		Expect(err).NotTo(HaveOccurred())

		By("Checking the operator is restarted")
		kubeClient.Deployment("onepassword-connect-operator").WaitDeploymentRolledOut(ctx)
	})
}
