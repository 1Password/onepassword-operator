package e2e

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/testhelper/kind"
	"github.com/1Password/onepassword-operator/test/testhelper/kube"
	"github.com/1Password/onepassword-operator/test/testhelper/operator"
	"github.com/1Password/onepassword-operator/test/testhelper/system"
)

const (
	operatorImageName = "1password/onepassword-operator:latest"
	defaultInterval   = 1 * time.Second
	defaultTimeout    = 30 * time.Second
)

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	BeforeAll(func() {
		kube.SetContextNamespace("default")

		operator.BuildOperatorImage()
		kind.LoadImageToKind(operatorImageName)

		By("Create Connect credentials secret")
		kube.CreateOpCredentialsSecret()

		By("Create onepassword-token secret")
		kube.CreateSecretFromEnvVar("OP_CONNECT_TOKEN", "onepassword-token")

		By("Create onepassword-service-account-token secret")
		kube.CreateSecretFromEnvVar("OP_SERVICE_ACCOUNT_TOKEN", "onepassword-service-account-token")

		operator.DeployOperator()
	})

	Context("Use the operator with Connect", func() {
		runCommonTestCases()
	})

	Context("Use the operator with Service Account", func() {
		BeforeAll(func() {
			kube.PatchOperatorToUseServiceAccount()
			kube.DeleteSecret("login") // remove secret crated in previous test
		})

		runCommonTestCases()
	})
})

func runCommonTestCases() {
	It("Should create secret from manifest file", func() {
		By("Creating secret")
		wd, err := os.Getwd()
		Expect(err).NotTo(HaveOccurred())
		yamlPath := filepath.Join(wd, "manifests", "secret.yaml")
		_, err = system.Run("kubectl", "apply", "-f", yamlPath)
		Expect(err).NotTo(HaveOccurred())

		By("Waiting for secret to be created")
		Eventually(func(g Gomega) {
			output, err := system.Run("kubectl", "get", "secret", "login", "-o", "jsonpath={.metadata.name}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(Equal("login"))
		}, defaultTimeout, defaultInterval).Should(Succeed())
	})
}
