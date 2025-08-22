package operator

import (
	"time"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

// BuildOperatorImage builds the Operator image using `make docker-build`
func BuildOperatorImage() {
	By("Building the Operator image")
	_, err := system.Run("make", "docker-build")
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
}

// DeployOperator deploys the Operator in the default namespace.
// It waits for the operator pod to be in 'Running' state.
// All the resources created using manifests in `config/` dir.
// To make the operator use Connect or Service Accounts, patch `config/manager/manager.yaml`
func DeployOperator() {
	By("Deploying the Operator")
	_, err := system.Run("make", "deploy")
	Expect(err).NotTo(HaveOccurred())
}

// WaitingForOperatorPod waits for the Operator pod to be in 'Running' state
func WaitingForOperatorPod() {
	By("Waiting for the Operator pod to be 'Running'")
	Eventually(func(g Gomega) {
		output, err := system.Run("kubectl", "get", "pods",
			"-l", "name=onepassword-connect-operator",
			"-o", "jsonpath={.items[0].status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("Running"))
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}

// WaitingForConnectPod waits for the Connect pod to be in 'Running' state
func WaitingForConnectPod() {
	By("Waiting for the Connect pod to be 'Running'")
	Eventually(func(g Gomega) {
		output, err := system.Run("kubectl", "get", "pods",
			"-l", "app=onepassword-connect",
			"-o", "jsonpath={.items[0].status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("Running"))
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}
