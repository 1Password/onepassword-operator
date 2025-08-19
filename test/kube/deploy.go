package kube

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/cmd"
)

// DeployOperator deploys the Onepassword Operator in the default namespace.
// It waits for the operator pod to be in 'Running' state.
// All the resources created using manifests in `config/` dir.
// To make the operator use Connect or Service Accounts, patch `config/manager/manager.yaml`
func DeployOperator() {
	By("deploying the operator")
	_, err := cmd.Run("make", "deploy")
	Expect(err).NotTo(HaveOccurred())

	By("waiting for the operator pod to be 'Running'")
	Eventually(func(g Gomega) {
		output, err := cmd.Run("kubectl", "get", "pods",
			"-l", "name=onepassword-connect-operator",
			"-o", "jsonpath={.items[0].status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("Running"))
	}, 30*time.Second, 1*time.Second).Should(Succeed())
}

func UndeployOperator() {
	Delete("secret", "onepassword-connect-token")
	Delete("secret", "onepassword-service-account-token")

	By("undeploying the operator")
	_, err := cmd.Run("make", "undeploy", "ignore-not-found")
	Expect(err).NotTo(HaveOccurred())
}

func PathOperatorToUseServiceAccount() {
	By("patching the operator deployment with service account token")
	_, err := cmd.Run(
		"kubectl", "patch", "deployment", "onepassword-connect-operator",
		"--type=json",
		`-p=[{"op":"replace","path":"/spec/template/spec/containers/0/env","value":[
    {"name":"OPERATOR_NAME","value":"onepassword-connect-operator"},
    {"name":"POD_NAME","valueFrom":{"fieldRef":{"fieldPath":"metadata.name"}}},
    {"name":"WATCH_NAMESPACE","value":"default"},
    {"name":"POLLING_INTERVAL","value":"10"},
    {"name":"AUTO_RESTART","value":"false"},
    {"name":"OP_SERVICE_ACCOUNT_TOKEN","valueFrom":{"secretKeyRef":{"name":"onepassword-service-account-token","key":"token"}}},
    {"name":"MANAGE_CONNECT","value":"false"}
  ]}]`,
	)
	Expect(err).NotTo(HaveOccurred())

	_, err = cmd.Run("kubectl", "rollout", "status",
		"deployment/onepassword-connect-operator", "-n", "default", "--timeout=120s")
	Expect(err).NotTo(HaveOccurred())

	By("waiting for the operator pod to be 'Running'")
	Eventually(func(g Gomega) {
		output, err := cmd.Run("kubectl", "get", "pods",
			"-l", "name=onepassword-connect-operator",
			"-o", "jsonpath={.items[0].status.phase}")
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(output).To(ContainSubstring("Running"))
	}, 120*time.Second, 1*time.Second).Should(Succeed())
}
