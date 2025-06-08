package onepassword

import (
	"context"
	"fmt"
	"strings"

	logf "sigs.k8s.io/controller-runtime/pkg/log"

	opclient "github.com/1Password/onepassword-operator/pkg/onepassword/client"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

var logger = logf.Log.WithName("retrieve_item")

func GetOnePasswordItemByPath(ctx context.Context, opClient opclient.Client, path string) (*model.Item, error) {
	vaultIdentifier, itemIdentifier, err := ParseVaultAndItemFromPath(path)
	if err != nil {
		return nil, err
	}
	vaultID, err := getVaultID(ctx, opClient, vaultIdentifier)
	if err != nil {
		return nil, fmt.Errorf("failed to 'getVaultID' for vaultIdentifier='%s': %w", vaultIdentifier, err)
	}

	itemID, err := getItemID(ctx, opClient, vaultID, itemIdentifier)
	if err != nil {
		return nil, fmt.Errorf("faild to 'getItemID' for vaultID='%s' and itemIdentifier='%s': %w", vaultID, itemIdentifier, err)
	}

	item, err := opClient.GetItemByID(ctx, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("faield to 'GetItemByID' for vaultID='%s' and itemID='%s': %w", vaultID, itemID, err)
	}

	for _, file := range item.Files {
		_, err := opClient.GetFileContent(ctx, vaultID, itemID, file.ID)
		if err != nil {
			return nil, err
		}
	}

	return item, nil
}

func ParseVaultAndItemFromPath(path string) (string, string, error) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 4 && splitPath[0] == "vaults" && splitPath[2] == "items" {
		return splitPath[1], splitPath[3], nil
	}
	return "", "", fmt.Errorf("%q is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`", path)
}

func getVaultID(ctx context.Context, client opclient.Client, vaultIdentifier string) (string, error) {
	if !IsValidClientUUID(vaultIdentifier) {
		vaults, err := client.GetVaultsByTitle(ctx, vaultIdentifier)
		if err != nil {
			return "", err
		}

		if len(vaults) == 0 {
			return "", fmt.Errorf("No vaults found with identifier %q", vaultIdentifier)
		}

		oldestVault := vaults[0]
		if len(vaults) > 1 {
			for _, returnedVault := range vaults {
				if returnedVault.CreatedAt.Before(oldestVault.CreatedAt) {
					oldestVault = returnedVault
				}
			}
			logger.Info(fmt.Sprintf("%v 1Password vaults found with the title %q. Will use vault %q as it is the oldest.", len(vaults), vaultIdentifier, oldestVault.ID))
		}
		vaultIdentifier = oldestVault.ID
	}
	return vaultIdentifier, nil
}

func getItemID(ctx context.Context, client opclient.Client, vaultId, itemIdentifier string) (string, error) {
	if !IsValidClientUUID(itemIdentifier) {
		items, err := client.GetItemsByTitle(ctx, vaultId, itemIdentifier)
		if err != nil {
			return "", err
		}

		if len(items) == 0 {
			return "", fmt.Errorf("No items found with identifier %q", itemIdentifier)
		}

		oldestItem := items[0]
		if len(items) > 1 {
			for _, returnedItem := range items {
				if returnedItem.CreatedAt.Before(oldestItem.CreatedAt) {
					oldestItem = returnedItem
				}
			}
			logger.Info(fmt.Sprintf("%v 1Password items found with the title %q. Will use item %q as it is the oldest.", len(items), itemIdentifier, oldestItem.ID))
		}
		itemIdentifier = oldestItem.ID
	}
	return itemIdentifier, nil
}
