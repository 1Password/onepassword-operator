package onepassword

import (
	"fmt"
	"strings"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var logger = logf.Log.WithName("retrieve_item")

const (
	secretReferencePrefix = "op://"
)

func GetOnePasswordItemByReference(opConnectClient connect.Client, reference string) (*onepassword.Item, error) {
	vaultValue, itemValue, err := ParseReference(reference)
	if err != nil {
		return nil, err
	}

	vaultId, err := getVaultId(opConnectClient, vaultValue)
	if err != nil {
		return nil, err
	}

	itemId, err := getItemId(opConnectClient, itemValue, vaultId)
	if err != nil {
		return nil, err
	}

	item, err := opConnectClient.GetItem(itemId, vaultId)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func ParseReference(reference string) (string, string, error) {
	if !strings.HasPrefix(reference, secretReferencePrefix) {
		return "", "", fmt.Errorf("secret reference should start with `op://`")
	}
	path := strings.TrimPrefix(reference, secretReferencePrefix)

	splitPath := strings.Split(path, "/")
	if len(splitPath) != 2 {
		return "", "", fmt.Errorf("Invalid secret reference : %s. Secret references should match op://<vault>/<item>", reference)
	}

	vault := splitPath[0]
	if vault == "" {
		return "", "", fmt.Errorf("Invalid secret reference : %s. Vault can't be empty.", reference)
	}

	item := splitPath[1]
	if item == "" {
		return "", "", fmt.Errorf("Invalid secret reference : %s. Item can't be empty.", reference)
	}

	return vault, item, nil
}

func getVaultId(client connect.Client, vaultIdentifier string) (string, error) {
	if !IsValidClientUUID(vaultIdentifier) {
		vaults, err := client.GetVaultsByTitle(vaultIdentifier)
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

func getItemId(client connect.Client, itemIdentifier string, vaultId string) (string, error) {
	if !IsValidClientUUID(itemIdentifier) {
		items, err := client.GetItemsByTitle(itemIdentifier, vaultId)
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
