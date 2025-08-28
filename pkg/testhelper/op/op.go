package op

import (
	"fmt"

	"github.com/1Password/onepassword-operator/pkg/testhelper/system"
)

type Field string

const (
	FieldUsername = "username"
	FieldPassword = "password"
)

// UpdateItemPassword updates the password of an item in 1Password
func UpdateItemPassword(item string) error {
	_, err := system.Run("op", "item", "edit", item, "--generate-password=letters,digits,symbols,32")
	if err != nil {
		return err
	}
	return nil
}

// ReadItemField reads the password of an item in 1Password
func ReadItemField(item, vault string, field Field) (string, error) {
	output, err := system.Run("op", "read", fmt.Sprintf("op://%s/%s/%s", vault, item, field))
	if err != nil {
		return "", err
	}
	return output, nil
}
