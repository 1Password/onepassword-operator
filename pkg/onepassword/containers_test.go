package onepassword

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAreContainersUsingSecretsFromEnv(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {},
	}

	containerSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	containers := generateContainersWithSecretRefsFromEnv(containerSecretNames)

	if !AreContainersUsingSecrets(containers, secretNamesToSearch) {
		t.Errorf("Expected that containers were using secrets but they were not detected.")
	}
}

func TestAreContainersUsingSecretsFromEnvFrom(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {},
	}

	containerSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	containers := generateContainersWithSecretRefsFromEnvFrom(containerSecretNames)

	if !AreContainersUsingSecrets(containers, secretNamesToSearch) {
		t.Errorf("Expected that containers were using secrets but they were not detected.")
	}
}

func TestAreContainersNotUsingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {},
	}

	containerSecretNames := []string{
		"some_other_key",
	}

	containers := generateContainersWithSecretRefsFromEnv(containerSecretNames)

	if AreContainersUsingSecrets(containers, secretNamesToSearch) {
		t.Errorf("Expected that containers were not using secrets but they were detected.")
	}
}

func TestAppendUpdatedContainerSecretsParsesEnvFromEnv(t *testing.T) {
	secretNamesToSearch := map[string]*corev1.Secret{
		"onepassword-database-secret": {},
		"onepassword-api-key":         {ObjectMeta: metav1.ObjectMeta{Name: "onepassword-api-key"}},
	}

	containerSecretNames := []string{
		"onepassword-api-key",
	}

	containers := generateContainersWithSecretRefsFromEnvFrom(containerSecretNames)

	updatedDeploymentSecrets := map[string]*corev1.Secret{}
	updatedDeploymentSecrets = AppendUpdatedContainerSecrets(containers, secretNamesToSearch, updatedDeploymentSecrets)

	secretKeyName := "onepassword-api-key"

	if updatedDeploymentSecrets[secretKeyName] != secretNamesToSearch[secretKeyName] {
		t.Errorf("Expected that updated Secret from envfrom is found.")
	}
}
