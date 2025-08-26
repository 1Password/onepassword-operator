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

	"github.com/1Password/onepassword-operator/pkg/testhelper/defaults"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kind"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kube"
	"github.com/1Password/onepassword-operator/pkg/testhelper/op"
	"github.com/1Password/onepassword-operator/pkg/testhelper/operator"
)

const (
	operatorImageName = "1password/onepassword-operator:latest"
	vaultName         = "operator-acceptance-tests"
)

var kubeClient *kube.Kube

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	ctx := context.Background()

	BeforeAll(func() {
		kubeClient = kube.NewKubeClient(&kube.ClusterConfig{
			Namespace:    "default",
			ManifestsDir: filepath.Join("manifests"),
		})
		kube.SetContextNamespace("default")

		operator.BuildOperatorImage()
		kind.LoadImageToKind(operatorImageName)

		kubeClient.Secret("op-credentials").CreateOpCredentials(ctx)
		kubeClient.Secret("op-credentials").CheckIfExists(ctx)

		kubeClient.Secret("onepassword-token").CreateFromEnvVar(ctx, "OP_CONNECT_TOKEN")
		kubeClient.Secret("onepassword-token").CheckIfExists(ctx)

		kubeClient.Secret("onepassword-service-account-token").CreateFromEnvVar(ctx, "OP_SERVICE_ACCOUNT_TOKEN")
		kubeClient.Secret("onepassword-service-account-token").CheckIfExists(ctx)

		operator.DeployOperator()
		operator.WaitingForOperatorPod()
	})

	Context("Use the operator with Connect", func() {
		BeforeAll(func() {
			operator.WaitingForConnectPod()
		})

		runCommonTestCases(ctx)
	})

	//Context("Use the operator with Service Account", func() {
	//	BeforeAll(func() {
	//		kube.PatchOperatorToUseServiceAccount(struct{}{})
	//		kubeClient.DeleteSecret(ctx, "login") // remove secret crated in previous test
	//	})
	//
	//	runCommonTestCases(ctx)
	//})
})

// runCommonTestCases contains test cases that are common to both Connect and Service Account authentication methods.
func runCommonTestCases(ctx context.Context) {
	It("Should create secret from manifest file", func() {
		By("Creating secret `login` from 1Password item")
		kubeClient.ApplyOnePasswordItem(ctx, "secret.yaml")
		kubeClient.Secret("login").CheckIfExists(ctx)
	})

	It("Secret is updated after POOLING_INTERVAL", func() {
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

	It("1Password item with `ignore-secret` doesn't pull updates to kubernetes secret", func() {
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

		newPassword, err := op.ReadItemPassword(itemName, vaultName)
		Expect(newPassword).NotTo(Equal(oldPassword))

		// checking that password was NOT updated
		Eventually(func(g Gomega) {
			// Derive a short-lived context so this API call won't hang indefinitely.
			attemptCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()

			intervalStr := kubeClient.Deployment("onepassword-connect-operator").ReadEnvVar(attemptCtx, "POLLING_INTERVAL")
			Expect(intervalStr).NotTo(BeEmpty())

			i, err := strconv.Atoi(intervalStr)
			Expect(err).NotTo(HaveOccurred())

			interval := time.Duration(i) * time.Second // convert to duration in seconds
			time.Sleep(interval + 2*time.Second)       // wait for one polling interval + 2 seconds to make sure updated secret is pulled

			secret = kubeClient.Secret(secretName).Get(attemptCtx)
			g.Expect(err).NotTo(HaveOccurred())

			currentPassword, ok := secret.Data["password"]
			Expect(ok).To(BeTrue())
			Expect(currentPassword).To(Equal(oldPassword))
			Expect(currentPassword).NotTo(Equal(newPassword))
		}, defaults.E2ETimeout, defaults.E2EInterval).Should(Succeed())
	})
}
