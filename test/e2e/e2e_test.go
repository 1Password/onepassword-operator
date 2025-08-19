package e2e

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/cmd"
	"github.com/1Password/onepassword-operator/test/kind"
	"github.com/1Password/onepassword-operator/test/kube"
)

const (
	operatorImage   = "1password/onepassword-operator:latest"
	defaultInterval = 1 * time.Second
	defaultTimeout  = 30 * time.Second
)

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	BeforeAll(func() {
		By("building the operator image")
		_, err := cmd.Run("make", "docker-build")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("loading the operator image on Kind")
		err = kind.LoadImageToKind(operatorImage)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("create onepassword-token secret")
		kube.CreateSecretFromEnvVar("OP_CONNECT_TOKEN", "onepassword-token")

		By("create onepassword-service-account-token secret")
		kube.CreateSecretFromEnvVar("OP_SERVICE_ACCOUNT_TOKEN", "onepassword-service-account-token")

		kube.DeployOperator()
		kube.PathOperatorToUseServiceAccount()
	})

	Describe("Deployment annotations", func() {
		It("Should create secret from manifest file", func() {
			By("creating secret")
			wd, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			yamlPath := filepath.Join(wd, "manifests", "secret.yaml")
			_, err = cmd.Run("kubectl", "apply", "-f", yamlPath)
			Expect(err).NotTo(HaveOccurred())

			By("waiting for secret to be created")
			Eventually(func(g Gomega) {
				output, err := cmd.Run("kubectl", "get", "secret", "login", "-o", "jsonpath={.metadata.name}")
				g.Expect(err).NotTo(HaveOccurred())
				g.Expect(output).To(Equal("login"))
			}, defaultTimeout, defaultInterval).Should(Succeed())
		})
	})
})
