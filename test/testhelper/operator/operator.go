package operator

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/testhelper/system"
)

func BuildOperatorImage() {
	By("building the operator image")
	_, err := system.Run("make", "docker-build")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

// DeployOperator deploys the Onepassword Operator in the default namespace.
// It waits for the operator pod to be in 'Running' state.
// All the resources created using manifests in `config/` dir.
// To make the operator use Connect or Service Accounts, patch `config/manager/manager.yaml`
func DeployOperator() {
	By("deploying the operator")
	_, err := system.Run("make", "deploy")
	Expect(err).NotTo(HaveOccurred())

	By("waiting for the operator pod to be 'Running'")
	Eventually(func(g Gomega) {
		output, err := system.Run("kubectl", "get", "pods",
			"-l", "name=onepassword-connect-operator",
			"-o", "jsonpath={.items[0].status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("Running"))
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}
