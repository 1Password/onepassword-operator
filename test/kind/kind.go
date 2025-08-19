package kind

import (
	"github.com/1Password/onepassword-operator/test/cmd"
	"os"
)

// LoadImageToKind loads a local docker image to the Kind cluster
func LoadImageToKind(imageName string) error {
	clusterName := "kind"
	if value, ok := os.LookupEnv("KIND_CLUSTER"); ok {
		clusterName = value
	}
	_, err := cmd.Run("kind", "load", "docker-image", imageName, "--name", clusterName)
	return err
}
