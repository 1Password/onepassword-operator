package client

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/1Password/onepassword-operator/pkg/onepassword/client/connect"
	"github.com/1Password/onepassword-operator/pkg/onepassword/client/sdk"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

// Client is an interface for interacting with 1Password items and vaults.
type Client interface {
	GetItemByID(ctx context.Context, vaultID, itemID string) (*model.Item, error)
	GetItemsByTitle(ctx context.Context, vaultID, itemTitle string) ([]model.Item, error)
	GetFileContent(ctx context.Context, vaultID, itemID, fileID string) ([]byte, error)
	GetVaultsByTitle(ctx context.Context, title string) ([]model.Vault, error)
}

// NewClient creates a new 1Password client based on the provided configuration.
func NewClient(ctx context.Context, integrationVersion string) (Client, error) {
	connectHost, _ := os.LookupEnv("OP_CONNECT_HOST")
	connectToken, _ := os.LookupEnv("OP_CONNECT_TOKEN")
	serviceAccountToken, _ := os.LookupEnv("OP_SERVICE_ACCOUNT_TOKEN")

	if connectHost != "" && connectToken != "" && serviceAccountToken != "" {
		return nil, errors.New("invalid configuration. Either Connect or Service Account credentials should be set, not both")
	}

	if serviceAccountToken != "" {
		fmt.Printf("Using Service Account Token")
		return sdk.NewClient(ctx, sdk.Config{
			ServiceAccountToken: serviceAccountToken,
			IntegrationName:     "1password-operator",
			IntegrationVersion:  integrationVersion,
		})
	}

	if connectHost != "" && connectToken != "" {
		fmt.Printf("Using Connect")
		return connect.NewClient(connect.Config{
			ConnectHost:  connectHost,
			ConnectToken: connectToken,
		}), nil
	}

	return nil, errors.New("invalid configuration. Connect or Service Account credentials should be set")
}
