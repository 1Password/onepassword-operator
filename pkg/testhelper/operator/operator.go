package operator

import (
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
