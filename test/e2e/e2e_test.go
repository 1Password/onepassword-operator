package e2e

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/testhelper/kind"
	"github.com/1Password/onepassword-operator/test/testhelper/kube"
	"github.com/1Password/onepassword-operator/test/testhelper/operator"
	"github.com/1Password/onepassword-operator/test/testhelper/system"
)

const (
	operatorImageName = "1password/onepassword-operator:latest"
)

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	BeforeAll(func() {
		kube.SetContextNamespace("default")

		operator.BuildOperatorImage()
		kind.LoadImageToKind(operatorImageName)

		By("Create Connect 'op-credentials' credentials secret")
		kube.CreateOpCredentialsSecret()

		By("Checking Connect 'op-credentials' secret is created")
		kube.CheckSecretExists("op-credentials")

		By("Create 'onepassword-token' secret")
		kube.CreateSecretFromEnvVar("OP_CONNECT_TOKEN", "onepassword-token")

		By("Checking 'onepassword-token' secret is created")
		kube.CheckSecretExists("onepassword-token")

		By("Create 'onepassword-service-account-token' secret")
		kube.CreateSecretFromEnvVar("OP_SERVICE_ACCOUNT_TOKEN", "onepassword-service-account-token")

		By("Checking 'onepassword-service-account-token' secret is created")
		kube.CheckSecretExists("onepassword-service-account-token")

		operator.DeployOperator()
		operator.WaitingForOperatorPod()
	})

	Context("Use the operator with Connect", func() {
		BeforeAll(func() {
			operator.WaitingForConnectPod()
		})

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
		root, err := system.GetProjectRoot()
		Expect(err).NotTo(HaveOccurred())

		yamlPath := filepath.Join(root, "test", "e2e", "manifests", "secret.yaml")
		kube.Apply(yamlPath)

		By("Checking for secret to be created")
		kube.CheckSecretExists("login")
	})
}
