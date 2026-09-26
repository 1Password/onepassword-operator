package template

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildDockerConfigJSON(t *testing.T) {
	tests := []struct {
		name     string
		registry string
		username string
		password string
		email    string
		wantErr  bool
		validate func(t *testing.T, result []byte)
	}{
		{
			name:     "basic docker config",
			registry: "docker.io",
			username: "testuser",
			password: "testpass",
			email:    "",
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				var config DockerConfigJSON
				err := json.Unmarshal(result, &config)
				require.NoError(t, err)

				assert.Contains(t, config.Auths, "docker.io")
				entry := config.Auths["docker.io"]
				assert.Equal(t, "testuser", entry.Username)
				assert.Equal(t, "testpass", entry.Password)
				assert.Empty(t, entry.Email)

				// Verify auth field is base64-encoded "username:password"
				expectedAuth := base64.StdEncoding.EncodeToString([]byte("testuser:testpass"))
				assert.Equal(t, expectedAuth, entry.Auth)
			},
		},
		{
			name:     "with email",
			registry: "ghcr.io",
			username: "ghuser",
			password: "ghpass",
			email:    "user@example.com",
			wantErr:  false,
			validate: func(t *testing.T, result []byte) {
				var config DockerConfigJSON
				err := json.Unmarshal(result, &config)
				require.NoError(t, err)

				entry := config.Auths["ghcr.io"]
				assert.Equal(t, "user@example.com", entry.Email)
			},
		},
		{
			name:     "empty registry",
			registry: "",
			username: "user",
			password: "pass",
			email:    "",
			wantErr:  true,
		},
		{
			name:     "empty username",
			registry: "docker.io",
			username: "",
			password: "pass",
			email:    "",
			wantErr:  true,
		},
		{
			name:     "empty password",
			registry: "docker.io",
			username: "user",
			password: "",
			email:    "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := BuildDockerConfigJSON(tt.registry, tt.username, tt.password, tt.email)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if tt.validate != nil {
					tt.validate(t, result)
				}
			}
		})
	}
}

func TestBuildDockerConfigJSON_ValidJSON(t *testing.T) {
	result, err := BuildDockerConfigJSON("registry.example.com", "user", "pass", "user@example.com")
	require.NoError(t, err)

	// Verify it's valid JSON
	var config DockerConfigJSON
	err = json.Unmarshal(result, &config)
	assert.NoError(t, err)

	// Verify structure
	assert.Len(t, config.Auths, 1)
	assert.Contains(t, config.Auths, "registry.example.com")
}
