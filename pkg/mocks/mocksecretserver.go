package mocks

import (
	"github.com/1Password/connect-sdk-go/onepassword"
)

type TestClient struct {
	GetVaultsFunc        func() ([]onepassword.Vault, error)
	GetVaultsByTitleFunc func(title string) ([]onepassword.Vault, error)
	GetVaultFunc         func(uuid string) (*onepassword.Vault, error)
	GetItemFunc          func(uuid string, vaultUUID string) (*onepassword.Item, error)
	GetItemsFunc         func(vaultUUID string) ([]onepassword.Item, error)
	GetItemsByTitleFunc  func(title string, vaultUUID string) ([]onepassword.Item, error)
	GetItemByTitleFunc   func(title string, vaultUUID string) (*onepassword.Item, error)
	CreateItemFunc       func(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error)
	UpdateItemFunc       func(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error)
	DeleteItemFunc       func(item *onepassword.Item, vaultUUID string) error
	GetFileFunc          func(uuid string, itemUUID string, vaultUUID string) (*onepassword.File, error)
	GetFileContentFunc   func(file *onepassword.File) ([]byte, error)
}

var (
	GetGetVaultsFunc       func() ([]onepassword.Vault, error)
	DoGetVaultsByTitleFunc func(title string) ([]onepassword.Vault, error)
	DoGetVaultFunc         func(uuid string) (*onepassword.Vault, error)
	GetGetItemFunc         func(uuid string, vaultUUID string) (*onepassword.Item, error)
	DoGetItemsByTitleFunc  func(title string, vaultUUID string) ([]onepassword.Item, error)
	DoGetItemByTitleFunc   func(title string, vaultUUID string) (*onepassword.Item, error)
	DoCreateItemFunc       func(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error)
	DoDeleteItemFunc       func(item *onepassword.Item, vaultUUID string) error
	DoGetItemsFunc         func(vaultUUID string) ([]onepassword.Item, error)
	DoUpdateItemFunc       func(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error)
	DoGetFileFunc          func(uuid string, itemUUID string, vaultUUID string) (*onepassword.File, error)
	DoGetFileContentFunc   func(file *onepassword.File) ([]byte, error)
)

// Do is the mock client's `Do` func
func (m *TestClient) GetVaults() ([]onepassword.Vault, error) {
	return GetGetVaultsFunc()
}

func (m *TestClient) GetVaultsByTitle(title string) ([]onepassword.Vault, error) {
	return DoGetVaultsByTitleFunc(title)
}

func (m *TestClient) GetVault(uuid string) (*onepassword.Vault, error) {
	return DoGetVaultFunc(uuid)
}

func (m *TestClient) GetItem(uuid string, vaultUUID string) (*onepassword.Item, error) {
	return GetGetItemFunc(uuid, vaultUUID)
}

func (m *TestClient) GetItems(vaultUUID string) ([]onepassword.Item, error) {
	return DoGetItemsFunc(vaultUUID)
}

func (m *TestClient) GetItemsByTitle(title, vaultUUID string) ([]onepassword.Item, error) {
	return DoGetItemsByTitleFunc(title, vaultUUID)
}

func (m *TestClient) GetItemByTitle(title string, vaultUUID string) (*onepassword.Item, error) {
	return DoGetItemByTitleFunc(title, vaultUUID)
}

func (m *TestClient) CreateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error) {
	return DoCreateItemFunc(item, vaultUUID)
}

func (m *TestClient) DeleteItem(item *onepassword.Item, vaultUUID string) error {
	return DoDeleteItemFunc(item, vaultUUID)
}

func (m *TestClient) UpdateItem(item *onepassword.Item, vaultUUID string) (*onepassword.Item, error) {
	return DoUpdateItemFunc(item, vaultUUID)
}

func (m *TestClient) GetFile(uuid string, itemUUID string, vaultUUID string) (*onepassword.File, error) {
	return DoGetFileFunc(uuid, itemUUID, vaultUUID)
}

func (m *TestClient) GetFileContent(file *onepassword.File) ([]byte, error) {
	return DoGetFileContentFunc(file)
}
