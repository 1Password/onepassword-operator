package onepassword

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestAreContainersUsingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": &corev1.Secret{},
		"onepassword-api-key":         &corev1.Secret{},
	}

	containerSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	containers := generateContainers(containerSecretNames)

	if !AreContainersUsingSecrets(containers, secretNamesToSearch) {
		t.Errorf("Expected that containers were using secrets but they were not detected.")
	}
}

func TestAreContainersNotUsingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": &corev1.Secret{},
		"onepassword-api-key":         &corev1.Secret{},
	}

	containerSecretNames := []string{
		"some_other_key",
	}

	containers := generateContainers(containerSecretNames)

	if AreContainersUsingSecrets(containers, secretNamesToSearch) {
		t.Errorf("Expected that containers were not using secrets but they were detected.")
	}
}
