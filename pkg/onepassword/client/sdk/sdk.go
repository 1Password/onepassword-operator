package sdk

import (
	"context"
	"fmt"
	"strings"

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

func NewClient(ctx context.Context, config Config) (*SDK, error) {
	client, err := sdk.NewClient(ctx,
		sdk.WithServiceAccountToken(config.ServiceAccountToken),
		sdk.WithIntegrationInfo(config.IntegrationName, config.IntegrationVersion),
	)
	if err != nil {
		return nil, fmt.Errorf("1Password sdk error: %w", err)
	}

	return &SDK{
		client: client,
	}, nil
}

func (s *SDK) GetItemByID(ctx context.Context, vaultID, itemID string) (*model.Item, error) {
	sdkItem, err := s.client.Items().Get(ctx, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to GetItemsByTitle using 1Password SDK: %w", err)
	}

	var item model.Item
	item.FromSDKItem(&sdkItem)
	return &item, nil
}

func (s *SDK) GetItemsByTitle(ctx context.Context, vaultID, itemTitle string) ([]model.Item, error) {
	// Get all items in the vault
	sdkItems, err := s.client.Items().List(ctx, vaultID)
	if err != nil {
		return nil, fmt.Errorf("failed to GetItemsByTitle using 1Password SDK: %w", err)
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

func (s *SDK) GetFileContent(ctx context.Context, vaultID, itemID, fileID string) ([]byte, error) {
	bytes, err := s.client.Items().Files().Read(ctx, vaultID, itemID, sdk.FileAttributes{
		ID: fileID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to GetFileContent using 1Password SDK: %w", err)
	}

	return bytes, nil
}

func (s *SDK) GetVaultsByTitle(ctx context.Context, title string) ([]model.Vault, error) {
	// List all vaults
	sdkVaults, err := s.client.Vaults().List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to GetVaultsByTitle using 1Password SDK: %w", err)
	}

	// Filter vaults by title
	var vaults []model.Vault
	for _, sdkVault := range sdkVaults {
		if strings.EqualFold(sdkVault.Title, title) {
			var vault model.Vault
			vault.FromSDKVault(&sdkVault)
			vaults = append(vaults, vault)
		}
	}

	return vaults, nil
}
