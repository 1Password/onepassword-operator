package onepassword

import (
	"testing"
)

func TestAreContainersUsingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
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
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
	}

	containerSecretNames := []string{
		"some_other_key",
	}

	containers := generateContainers(containerSecretNames)

	if AreContainersUsingSecrets(containers, secretNamesToSearch) {
		t.Errorf("Expected that containers were not using secrets but they were detected.")
	}
}
