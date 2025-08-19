package e2e

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/cmd"
	"github.com/1Password/onepassword-operator/test/kind"
)

const (
	operatorImage = "1password/onepassword-operator:latest"
	e2eInterval   = 500 * time.Millisecond
)

var _ = Describe("Onepassword Operator e2e", Ordered, func() {
	BeforeAll(func() {
		By("building the operator image")
		_, err := cmd.Run("make", "docker-build")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("loading the operator image on Kind")
		err = kind.LoadImageToKind(operatorImage)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("create onepassword-service-account-token secret")
		serviceAccountTokenToken, _ := os.LookupEnv("OP_SERVICE_ACCOUNT_TOKEN")
		Expect(serviceAccountTokenToken).NotTo(BeEmpty())
		_, err = cmd.Run("kubectl", "create", "secret", "generic", "onepassword-service-account-token", "--from-literal=token="+serviceAccountTokenToken)
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("deploying the operator")
		_, err = cmd.Run("make", "deploy")
		ExpectWithOffset(1, err).NotTo(HaveOccurred())

		By("waiting for the operator pod to be 'Running'")
		Eventually(func(g Gomega) {
			output, err := cmd.Run("kubectl", "get", "pods",
				"-l", "name=onepassword-connect-operator",
				"-o", "jsonpath={.items[0].status.phase}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("Running"))
		}, 30*time.Second, 1*time.Second).Should(Succeed())
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
			}, 5*time.Second, e2eInterval).Should(Succeed())
		})
	})
})
