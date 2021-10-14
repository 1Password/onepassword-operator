package controller

import (
	"github.com/1Password/onepassword-operator/operator/pkg/controller/onepassworditem"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, onepassworditem.Add)
}
