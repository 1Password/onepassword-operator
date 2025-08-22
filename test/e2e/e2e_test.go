package e2e

import (
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/pkg/testhelper/kind"
	"github.com/1Password/onepassword-operator/pkg/testhelper/kube"
	"github.com/1Password/onepassword-operator/pkg/testhelper/op"
	"github.com/1Password/onepassword-operator/pkg/testhelper/operator"
	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

const (
	operatorImageName = "1password/onepassword-operator:latest"
)

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	BeforeAll(func() {
		kube.SetContextNamespace("default")
		operator.BuildOperatorImage()
		kind.LoadImageToKind(operatorImageName)

		kube.CreateOpCredentialsSecret()
		kube.CheckSecretExists("op-credentials")

		kube.CreateSecretFromEnvVar("OP_CONNECT_TOKEN", "onepassword-token")
		kube.CheckSecretExists("onepassword-token")

		kube.CreateSecretFromEnvVar("OP_SERVICE_ACCOUNT_TOKEN", "onepassword-service-account-token")
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

// runCommonTestCases contains test cases that are common to both Connect and Service Account authentication methods.
func runCommonTestCases() {
	It("Should create secret from manifest file", func() {
		By("Creating secret `login` from 1Password item")
		root, err := system.GetProjectRoot()
		Expect(err).NotTo(HaveOccurred())

		yamlPath := filepath.Join(root, "test", "e2e", "manifests", "secret.yaml")
		kube.Apply(yamlPath)
		kube.CheckSecretExists("login")
	})

	It("Secret is updated after POOLING_INTERVAL", func() {
		itemName := "secret-for-update"
		secretName := itemName

		By("Creating secret `" + secretName + "` from 1Password item")
		root, err := system.GetProjectRoot()
		Expect(err).NotTo(HaveOccurred())

		yamlPath := filepath.Join(root, "test", "e2e", "manifests", secretName+".yaml")
		kube.Apply(yamlPath)
		kube.CheckSecretExists(secretName)

		By("Reading old password")
		oldPassword, err := kube.ReadingSecretData(secretName, "password")
		Expect(err).NotTo(HaveOccurred())

		By("Updating `" + secretName + "` 1Password item")
		err = op.UpdateItemPassword(itemName)
		Expect(err).NotTo(HaveOccurred())

		kube.CheckSecretPasswordWasUpdated(secretName, oldPassword)
	})
}
