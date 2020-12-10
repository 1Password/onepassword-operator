package onepassword

import (
	"testing"
)

func TestAreVolmesUsingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
	}

	volumeSecretNames := []string{
		"onepassword-database-secret",
		"onepassword-api-key",
		"some_other_key",
	}

	volumes := generateVolumes(volumeSecretNames)

	if !AreVolumesUsingSecrets(volumes, secretNamesToSearch) {
		t.Errorf("Expected that volumes were using secrets but they were not detected.")
	}
}

func TestAreVolumesNotUsingSecrets(t *testing.T) {
	secretNamesToSearch := map[string]bool{
		"onepassword-database-secret": true,
		"onepassword-api-key":         true,
	}

	volumeSecretNames := []string{
		"some_other_key",
	}

	volumes := generateVolumes(volumeSecretNames)

	if AreVolumesUsingSecrets(volumes, secretNamesToSearch) {
		t.Errorf("Expected that volumes were not using secrets but they were detected.")
	}
}
