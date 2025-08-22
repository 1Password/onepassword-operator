package op

import (
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
