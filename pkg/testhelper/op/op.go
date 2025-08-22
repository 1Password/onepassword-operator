package op

import (
	"fmt"
	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

// UpdateItemPassword updates the password of an item in 1Password
func UpdateItemPassword(item string) error {
	_, err := system.Run("op", "item", "edit", item, "--generate-password=letters,digits,symbols,32")
	if err != nil {
		return err
	}
	return nil
}

// ReadItemPassword reads the password of an item in 1Password
func ReadItemPassword(item, vault string) (string, error) {
	output, err := system.Run("op", "read", fmt.Sprintf("op://%s/%s/password", vault, item))
	if err != nil {
		return "", err
	}
	return output, nil
}
