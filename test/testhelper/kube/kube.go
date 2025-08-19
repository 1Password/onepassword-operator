package kube

import (
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/cmd"
)

func CreateSecretFromEnvVar(envVar, secretName string) {
	serviceAccountTokenToken, _ := os.LookupEnv(envVar)
	Expect(serviceAccountTokenToken).NotTo(BeEmpty())
	_, err := cmd.Run("kubectl", "create", "secret", "generic", secretName, "--from-literal=token="+serviceAccountTokenToken)
	Expect(err).NotTo(HaveOccurred())
}

func Delete(kind, name string) {
	_, err := cmd.Run("kubectl", "delete", kind, name, "--ignore-not-found=true")
	Expect(err).NotTo(HaveOccurred())
}

func PatchOperatorToUseServiceAccount() {
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
