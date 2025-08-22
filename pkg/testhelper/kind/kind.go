package kind

import (
	"os"

	//nolint:staticcheck // ST1001
	. "github.com/onsi/ginkgo/v2"
	//nolint:staticcheck // ST1001
	. "github.com/onsi/gomega"

	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

// LoadImageToKind loads a local docker image to the Kind cluster
func LoadImageToKind(imageName string) {
	By("Loading the operator image on Kind")
	clusterName := "kind"
	if value, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		clusterName = value
	}
	_, err := system.Run("kind", "load", "docker-image", imageName, "--name", clusterName)
	Expect(err).NotTo(HaveOccurred())
}
