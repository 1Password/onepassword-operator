package e2e

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// Run e2e tests using the Ginkgo runner.
func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "onepassword-operator e2e suite")
}

//By("create onepassword-token secret")
//connectToken, _ := os.LookupEnv("OP_CONNECT_TOKEN")
//Expect(connectToken).NotTo(BeEmpty())
//output := exec.Command("kubectl", "-n", namespace, "create", "secret", "generic", "onepassword-token", "--from-literal=token="+connectToken)
//_, err = utils.Run(output)
//ExpectWithOffset(1, err).NotTo(HaveOccurred())

//It("Secret is updated after POOLING_INTERVAL", func() {
//	// TODO: implement
//})
//
//It("Secret with `ignore-secret` annotation is not updated", func() {
//	// TODO: implement
//})
//
//It("Deployment not auto restarts when ", func() {
//	// TODO: implement
//})
