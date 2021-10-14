package controller

import (
	"github.com/1Password/connect-sdk-go/connect"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

// AddToManagerFuncs is a list of functions to add all Controllers to the Manager
var AddToManagerFuncs []func(manager.Manager, connect.Client) error

// AddToManager adds all Controllers to the Manager
func AddToManager(m manager.Manager, opConnectClient connect.Client) error {
	for _, f := range AddToManagerFuncs {
		if err := f(m, opConnectClient); err != nil {
			return err
		}
	}
	return nil
}
