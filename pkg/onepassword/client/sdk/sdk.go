package sdk

import (
	"context"
	"fmt"

	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	sdk "github.com/1password/onepassword-sdk-go"
)

// Config holds the configuration for the 1Password SDK client.
type Config struct {
	ServiceAccountToken string
	IntegrationName     string
	IntegrationVersion  string
}

// SDK is a client for interacting with 1Password using the SDK.
type SDK struct {
	client *sdk.Client
}

func NewClient(config Config) (*SDK, error) {
	client, err := sdk.NewClient(context.Background(),
		sdk.WithServiceAccountToken(config.ServiceAccountToken),
		sdk.WithIntegrationInfo(config.IntegrationName, config.IntegrationVersion),
	)
	if err != nil {
		return nil, fmt.Errorf("1password sdk error: %w", err)
	}

	return &SDK{
		client: client,
	}, nil
}

func (s *SDK) GetItemByID(vaultID, itemID string) (*model.Item, error) {
	sdkItem, err := s.client.Items().Get(context.Background(), vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("1password sdk error: %w", err)
	}

	var item model.Item
	item.FromSDKItem(&sdkItem)
	return &item, nil
}

func (s *SDK) GetItemsByTitle(vaultID, itemTitle string) ([]model.Item, error) {
	// Get all items in the vault
	sdkItems, err := s.client.Items().List(context.Background(), vaultID)
	if err != nil {
		return nil, fmt.Errorf("1password sdk error: %w", err)
	}

	// Filter items by title
	var items []model.Item
	for _, sdkItem := range sdkItems {
		if sdkItem.Title == itemTitle {
			var item model.Item
			item.FromSDKItemOverview(&sdkItem)
			items = append(items, item)
		}
	}

	return items, nil
}

func (s *SDK) GetFileContent(vaultID, itemID, fileID string) ([]byte, error) {
	bytes, err := s.client.Items().Files().Read(context.Background(), vaultID, itemID, sdk.FileAttributes{
		ID: fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("1password sdk error: %w", err)
	}

	return bytes, nil
}

func (s *SDK) GetVaultsByTitle(title string) ([]model.Vault, error) {
	// List all vaults
	sdkVaults, err := s.client.Vaults().List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("1password sdk error: %w", err)
	}

	// Filter vaults by title
	var vaults []model.Vault
	for _, sdkVault := range sdkVaults {
		if sdkVault.Title == title {
			var vault model.Vault
			vault.FromSDKVault(&sdkVault)
			vaults = append(vaults, vault)
		}
	}

	return vaults, nil
}
