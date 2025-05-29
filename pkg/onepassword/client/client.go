package client

import (
	"errors"

	"github.com/1Password/onepassword-operator/pkg/onepassword/client/connect"
	"github.com/1Password/onepassword-operator/pkg/onepassword/client/sdk"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

// Client is an interface for interacting with 1Password items and vaults.
type Client interface {
	GetItemByID(vaultID, itemID string) (*model.Item, error)
	GetItemsByTitle(vaultID, itemTitle string) ([]model.Item, error)
	GetFileContent(vaultID, itemID, fileID string) ([]byte, error)
	GetVaultsByTitle(title string) ([]model.Vault, error)
}

// Config holds the configuration for creating a new 1Password client.
type Config struct {
	ConnectHost         string
	ConnectToken        string
	UserAgent           string
	ServiceAccountToken string
	IntegrationName     string
	IntegrationVersion  string
}

// NewClient creates a new 1Password client based on the provided configuration.
func NewClient(config Config) (Client, error) {
	if config.ServiceAccountToken != "" {
		return sdk.NewClient(sdk.Config{
			ServiceAccountToken: config.ServiceAccountToken,
			IntegrationName:     config.IntegrationName,
			IntegrationVersion:  config.IntegrationVersion,
		})
	} else if config.ConnectHost != "" && config.ConnectToken != "" {
		return connect.NewClient(connect.Config{
			ConnectHost:  config.ConnectHost,
			ConnectToken: config.ConnectToken,
		}), nil
	}
	return nil, errors.New("invalid configuration. Either Connect or Service Account credentials should be set")
}
