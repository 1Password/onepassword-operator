package onepassword

import (
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestIsDeploymentUsingSecretsUsingVolumes(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {},
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
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {},
	}

	containerSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	deployment := &appsv1.Deployment{}
	deployment.Spec.Template.Spec.Containers = generateContainersWithSecretRefsFromEnv(containerSecretNames)
	if !IsDeploymentUsingSecrets(deployment, secretNamesToSearch) {
		t.Errorf("Expected that deployment was using secrets but they were not detected.")
	}
}

func TestIsDeploymentNotUSingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {},
	}

	deployment := &appsv1.Deployment{}
	if IsDeploymentUsingSecrets(deployment, secretNamesToSearch) {
		t.Errorf("Expected that deployment was using not secrets but they were detected.")
	}
}
