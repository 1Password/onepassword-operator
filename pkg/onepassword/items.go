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
	vaultNameOrID, itemNameOrID, err := ParseVaultAndItemFromPath(path)
	if err != nil {
		return nil, err
	}
	vaultID, err := getVaultID(ctx, opClient, vaultNameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to 'getVaultID' for vaultNameOrID='%s': %w", vaultNameOrID, err)
	}

	var item *model.Item
	// If it looks like a UUID, try fetching by ID first
	if IsValidClientUUID(itemNameOrID) {
		item, err = opClient.GetItemByID(ctx, vaultID, itemNameOrID)
		if err == nil {
			// Success, load files and return
			err = loadItemFiles(ctx, opClient, vaultID, item)
			if err != nil {
				return nil, fmt.Errorf("failed to load item files for vaultID='%s' and itemNameOrID='%s': %w",
					vaultID, itemNameOrID, err)
			}
			return item, nil
		}
		// If UUID lookup failed, fallback to title lookup
	}

	// Try to fetch item by title to get the ID
	itemID, err := getItemIDByTitle(ctx, opClient, vaultID, itemNameOrID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item for vaultID='%s' and itemNameOrID='%s': %w", vaultID, itemNameOrID, err)
	}

	item, err = opClient.GetItemByID(ctx, vaultID, itemID)
	if err != nil {
		return nil, fmt.Errorf("failed to get item by ID for vaultID='%s' and itemID='%s': %w", vaultID, itemID, err)
	}

	err = loadItemFiles(ctx, opClient, vaultID, item)
	if err != nil {
		return nil, fmt.Errorf("failed to load item files for vaultID='%s' and itemID='%s': %w", vaultID, itemID, err)
	}

	return item, nil
}

func ParseVaultAndItemFromPath(path string) (string, string, error) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 4 && splitPath[0] == "vaults" && splitPath[2] == "items" {
		return splitPath[1], splitPath[3], nil
	}
	return "", "", fmt.Errorf(
		"%q is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`",
		path,
	)
}

func getVaultID(ctx context.Context, client opclient.Client, vaultNameOrID string) (string, error) {
	// First try to get vault by title
	vaults, err := client.GetVaultsByTitle(ctx, vaultNameOrID)
	if err == nil && len(vaults) > 0 {
		// Found vault by title use the oldest one
		oldestVault := vaults[0]
		if len(vaults) > 1 {
			for _, returnedVault := range vaults {
				if returnedVault.CreatedAt.Before(oldestVault.CreatedAt) {
					oldestVault = returnedVault
				}
			}

			logger.Info(fmt.Sprintf("%v 1Password vaults found with the title %q. Will use vault %q as it is the oldest.",
				len(vaults), vaultNameOrID, oldestVault.ID,
			))
		}
		return oldestVault.ID, nil
	}

	// Title lookup failed or returned no results so try to use it as a UUID if it looks like one
	if IsValidClientUUID(vaultNameOrID) {
		return vaultNameOrID, nil
	}

	// Not found by title and doesn't look like a UUID
	if err != nil {
		return "", fmt.Errorf("failed to get vault by title %q: %w", vaultNameOrID, err)
	}
	return "", fmt.Errorf("no vaults found with identifier %q", vaultNameOrID)
}

func getItemIDByTitle(ctx context.Context, client opclient.Client, vaultId, itemNameOrID string) (string, error) {
	items, err := client.GetItemsByTitle(ctx, vaultId, itemNameOrID)
	if err != nil {
		return "", fmt.Errorf("failed to GetItemsByTitle for vaultID='%s' and itemTitle='%s': %w", vaultId, itemNameOrID, err)
	}

	if len(items) == 0 {
		return "", fmt.Errorf("no items found with identifier %q in vault %q", itemNameOrID, vaultId)
	}

	oldestItem := items[0]
	if len(items) > 1 {
		for i := range items {
			returnedItem := items[i]
			if returnedItem.CreatedAt.Before(oldestItem.CreatedAt) {
				oldestItem = returnedItem
			}
		}
		logger.Info(fmt.Sprintf("%v 1Password items found with the title %q. Will use item %q as it is the oldest.",
			len(items), itemNameOrID, oldestItem.ID,
		))
	}

	return oldestItem.ID, nil
}

func loadItemFiles(ctx context.Context, client opclient.Client, vaultID string,
	item *model.Item) error {
	for i, file := range item.Files {
		content, err := client.GetFileContent(ctx, vaultID, item.ID, file.ID)
		if err != nil {
			return err
		}
		item.Files[i].SetContent(content)
	}
	return nil
}
