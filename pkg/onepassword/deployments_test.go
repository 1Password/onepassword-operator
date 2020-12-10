package onepassword

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
)

func TestIsDeploymentUsingSecretsUsingVolumes(t *testing.T) {
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
	}

	volumeSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	deployment := &appsv1.Deployment{}
	deployment.Spec.Template.Spec.Volumes = generateVolumes(volumeSecretNames)
	if !IsDeploymentUsingSecrets(deployment, secretNamesToSearch) {
		t.Errorf("Expected that deployment was using secrets but they were not detected.")
	}
}

func TestIsDeploymentUsingSecretsUsingContainers(t *testing.T) {
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
	}

	containerSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	deployment := &appsv1.Deployment{}
	deployment.Spec.Template.Spec.Containers = generateContainers(containerSecretNames)
	if !IsDeploymentUsingSecrets(deployment, secretNamesToSearch) {
		t.Errorf("Expected that deployment was using secrets but they were not detected.")
	}
}

func TestIsDeploymentNotUSingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
	}

	deployment := &appsv1.Deployment{}
	if IsDeploymentUsingSecrets(deployment, secretNamesToSearch) {
		t.Errorf("Expected that deployment was using not secrets but they were detected.")
	}
}
