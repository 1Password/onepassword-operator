package onepassword

import (
	"fmt"
	"strings"

	"github.com/1Password/connect-sdk-go/connect"
	"github.com/1Password/connect-sdk-go/onepassword"
)

func GetOnePasswordItemByPath(opConnectClient connect.Client, path string) (*onepassword.Item, error) {
	vaultId, itemId, err := ParseVaultIdAndItemIdFromPath(path)
	if err != nil {
		return nil, err
	}
	item, err := opConnectClient.GetItem(itemId, vaultId)
	if err != nil {
		return nil, err
	}
	return item, nil
}

func ParseVaultIdAndItemIdFromPath(path string) (string, string, error) {
	splitPath := strings.Split(path, "/")
	if len(splitPath) == 4 && splitPath[0] == "vaults" && splitPath[2] == "items" {
		return splitPath[1], splitPath[3], nil
	}
	return "", "", fmt.Errorf("%q is not an acceptable path for One Password item. Must be of the format: `vaults/{vault_id}/items/{item_id}`", path)
}
