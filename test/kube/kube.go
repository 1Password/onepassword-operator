package kube

import (
	"os"

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
