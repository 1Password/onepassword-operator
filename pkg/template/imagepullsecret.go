package template

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
)

// DockerConfigJSON represents the structure of a .dockerconfigjson file.
type DockerConfigJSON struct {
	Auths map[string]DockerConfigEntry `json:"auths"`
}

// DockerConfigEntry represents a single registry entry in dockerconfigjson.
type DockerConfigEntry struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Email    string `json:"email,omitempty"`
	Auth     string `json:"auth,omitempty"`
}

// BuildDockerConfigJSON generates the proper .dockerconfigjson structure for Kubernetes image pull secrets.
// The auth field is base64-encoded "username:password".
func BuildDockerConfigJSON(registry, username, password, email string) ([]byte, error) {
	if registry == "" {
		return nil, fmt.Errorf("registry cannot be empty")
	}
	if username == "" {
		return nil, fmt.Errorf("username cannot be empty")
	}
	if password == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	// Create base64-encoded auth string
	authString := fmt.Sprintf("%s:%s", username, password)
	auth := base64.StdEncoding.EncodeToString([]byte(authString))

	entry := DockerConfigEntry{
		Username: username,
		Password: password,
		Auth:     auth,
	}

	if email != "" {
		entry.Email = email
	}

	config := DockerConfigJSON{
		Auths: map[string]DockerConfigEntry{
			registry: entry,
		},
	}

	jsonBytes, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal docker config json: %w", err)
	}

	return jsonBytes, nil
}
