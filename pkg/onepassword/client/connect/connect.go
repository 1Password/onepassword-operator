package connect

import (
	"context"
	"fmt"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

// Config holds the configuration for the Connect client.
type Config struct {
	ConnectHost  string
	ConnectToken string
}

// Connect is a client for interacting with 1Password using the Connect API.
type Connect struct {
	client connect.Client
}

// NewClient creates a new Connect client using provided configuration.
func NewClient(config Config) *Connect {
	return &Connect{
		client: connect.NewClient(config.ConnectHost, config.ConnectToken),
	}
}

func (c *Connect) GetItemByID(ctx context.Context, vaultID, itemID string) (*model.Item, error) {
	connectItem, err := c.client.GetItemByUUID(itemID, vaultID)
	if err != nil {
		return nil, fmt.Errorf("1Password Connect error: %w", err)
	}

	var item model.Item
	item.FromConnectItem(connectItem)
	return &item, nil
}

func (c *Connect) GetItemsByTitle(ctx context.Context, vaultID, itemTitle string) ([]model.Item, error) {
	// Get all items in the vault with the specified title
	connectItems, err := c.client.GetItemsByTitle(itemTitle, vaultID)
	if err != nil {
		return nil, fmt.Errorf("1Password Connect error: %w", err)
	}

	items := make([]model.Item, len(connectItems))
	for i, connectItem := range connectItems {
		var item model.Item
		item.FromConnectItem(&connectItem)
		items[i] = item
	}

	return items, nil
}

func (c *Connect) GetFileContent(ctx context.Context, vaultID, itemID, fileID string) ([]byte, error) {
	bytes, err := c.client.GetFileContent(&onepassword.File{
		ContentPath: fmt.Sprintf("/v1/vaults/%s/items/%s/files/%s/content", vaultID, itemID, fileID),
	})
	if err != nil {
		return nil, fmt.Errorf("1Password Connect error: %w", err)
	}

	return bytes, nil
}

func (c *Connect) GetVaultsByTitle(ctx context.Context, vaultQuery string) ([]model.Vault, error) {
	connectVaults, err := c.client.GetVaultsByTitle(vaultQuery)
	if err != nil {
		return nil, fmt.Errorf("1Password Connect error: %w", err)
	}

	var vaults []model.Vault
	for _, connectVault := range connectVaults {
		if vaultQuery == connectVault.Name {
			var vault model.Vault
			vault.FromConnectVault(&connectVault)
			vaults = append(vaults, vault)
		}
	}
	return vaults, nil
}
