package kube

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/test/testhelper/system"
)

func CreateSecretFromEnvVar(envVar, secretName string) {
	value, _ := os.LookupEnv(envVar)
	Expect(value).NotTo(BeEmpty())

	_, err := system.Run("kubectl", "create", "secret", "generic", secretName, "--from-literal=token="+value)
	Expect(err).NotTo(HaveOccurred())
}

func CreateSecretFromFile(fileName, secretName string) {
	_, err := system.Run("kubectl", "create", "secret", "generic", secretName, "--from-file="+fileName)
	Expect(err).NotTo(HaveOccurred())
}

func CreateOpCredentialsSecret() {
	rootDir, err := system.GetProjectRoot()
	Expect(err).NotTo(HaveOccurred())

	credentialsFilePath := filepath.Join(rootDir, "1password-credentials.json")
	data, err := os.ReadFile(credentialsFilePath)
	Expect(err).NotTo(HaveOccurred())

	encoded := base64.RawURLEncoding.EncodeToString(data)

	// create op-session file in project root
	sessionFilePath := filepath.Join(rootDir, "op-session")
	err = os.WriteFile(sessionFilePath, []byte(encoded), 0o600)
	Expect(err).NotTo(HaveOccurred())

	CreateSecretFromFile("op-session", "op-credentials")
}

func DeleteSecret(name string) {
	_, err := system.Run("kubectl", "delete", "secret", name, "--ignore-not-found=true")
	Expect(err).NotTo(HaveOccurred())
}

func SetContextNamespace(namespace string) {
	By("Set namespace to " + namespace)
	_, err := system.Run("kubectl", "config", "set-context", "--current", "--namespace="+namespace)
	Expect(err).NotTo(HaveOccurred())
}

// PatchOperatorToUseServiceAccount sets `OP_SERVICE_ACCOUNT_TOKEN` env variable
var PatchOperatorToUseServiceAccount = WithOperatorRestart(func() {
	By("patching the operator deployment with service account token")
	_, err := system.Run(
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
})

func WithOperatorRestart(operation func()) func() {
	return func() {
		operation()

		_, err := system.Run("kubectl", "rollout", "status",
			"deployment/onepassword-connect-operator", "-n", "default", "--timeout=120s")
		Expect(err).NotTo(HaveOccurred())

		By("waiting for the operator pod to be 'Running'")
		Eventually(func(g Gomega) {
			output, err := system.Run("kubectl", "get", "pods",
				"-l", "name=onepassword-connect-operator",
				"-o", "jsonpath={.items[0].status.phase}")
			g.Expect(err).NotTo(HaveOccurred())
			g.Expect(output).To(ContainSubstring("Running"))
		}, 120*time.Second, 1*time.Second).Should(Succeed())
	}
}
