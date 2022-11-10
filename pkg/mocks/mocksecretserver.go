package mocks

import (
	"github.com/1Password/connect-sdk-go/onepassword"
)

type TestClient struct {
	GetVaultsFunc                 func() ([]onepassword.Vault, error)
	GetVaultsByTitleFunc          func(title string) ([]onepassword.Vault, error)
	GetVaultFunc                  func(uuid string) (*onepassword.Vault, error)
	GetVaultByUUIDFunc            func(uuid string) (*onepassword.Vault, error)
	GetVaultByTitleFunc           func(title string) (*onepassword.Vault, error)
	GetItemFunc                   func(itemQuery string, vaultQuery string) (*onepassword.Item, error)
	GetItemByUUIDFunc             func(uuid string, vaultQuery string) (*onepassword.Item, error)
	GetItemByTitleFunc            func(title string, vaultQuery string) (*onepassword.Item, error)
	GetItemsFunc                  func(vaultQuery string) ([]onepassword.Item, error)
	GetItemsByTitleFunc           func(title string, vaultQuery string) ([]onepassword.Item, error)
	CreateItemFunc                func(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error)
	UpdateItemFunc                func(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error)
	DeleteItemFunc                func(item *onepassword.Item, vaultQuery string) error
	DeleteItemByIDFunc            func(itemUUID string, vaultQuery string) error
	DeleteItemByTitleFunc         func(title string, vaultQuery string) error
	GetFilesFunc                  func(itemQuery string, vaultQuery string) ([]onepassword.File, error)
	GetFileFunc                   func(uuid string, itemQuery string, vaultQuery string) (*onepassword.File, error)
	GetFileContentFunc            func(file *onepassword.File) ([]byte, error)
	DownloadFileFunc              func(file *onepassword.File, targetDirectory string, overwrite bool) (string, error)
	LoadStructFromItemByUUIDFunc  func(config interface{}, itemUUID string, vaultQuery string) error
	LoadStructFromItemByTitleFunc func(config interface{}, itemTitle string, vaultQuery string) error
	LoadStructFromItemFunc        func(config interface{}, itemQuery string, vaultQuery string) error
	LoadStructFunc                func(config interface{}) error
}

var (
	DoGetVaultsFunc                 func() ([]onepassword.Vault, error)
	DoGetVaultsByTitleFunc          func(title string) ([]onepassword.Vault, error)
	DoGetVaultFunc                  func(uuid string) (*onepassword.Vault, error)
	DoGetVaultByUUIDFunc            func(uuid string) (*onepassword.Vault, error)
	DoGetVaultByTitleFunc           func(title string) (*onepassword.Vault, error)
	DoGetItemFunc                   func(itemQuery string, vaultQuery string) (*onepassword.Item, error)
	DoGetItemByUUIDFunc             func(uuid string, vaultQuery string) (*onepassword.Item, error)
	DoGetItemByTitleFunc            func(title string, vaultQuery string) (*onepassword.Item, error)
	DoGetItemsFunc                  func(vaultQuery string) ([]onepassword.Item, error)
	DoGetItemsByTitleFunc           func(title string, vaultQuery string) ([]onepassword.Item, error)
	DoCreateItemFunc                func(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error)
	DoUpdateItemFunc                func(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error)
	DoDeleteItemFunc                func(item *onepassword.Item, vaultQuery string) error
	DoDeleteItemByIDFunc            func(itemUUID string, vaultQuery string) error
	DoDeleteItemByTitleFunc         func(title string, vaultQuery string) error
	DoGetFilesFunc                  func(itemQuery string, vaultQuery string) ([]onepassword.File, error)
	DoGetFileFunc                   func(uuid string, itemQuery string, vaultQuery string) (*onepassword.File, error)
	DoGetFileContentFunc            func(file *onepassword.File) ([]byte, error)
	DoDownloadFileFunc              func(file *onepassword.File, targetDirectory string, overwrite bool) (string, error)
	DoLoadStructFromItemByUUIDFunc  func(config interface{}, itemUUID string, vaultQuery string) error
	DoLoadStructFromItemByTitleFunc func(config interface{}, itemTitle string, vaultQuery string) error
	DoLoadStructFromItemFunc        func(config interface{}, itemQuery string, vaultQuery string) error
	DoLoadStructFunc                func(config interface{}) error
)

// Do is the mock client's `Do` func

func (m *TestClient) GetVaults() ([]onepassword.Vault, error) {
	return DoGetVaultsFunc()
}

func (m *TestClient) GetVaultsByTitle(title string) ([]onepassword.Vault, error) {
	return DoGetVaultsByTitleFunc(title)
}

func (m *TestClient) GetVault(vaultQuery string) (*onepassword.Vault, error) {
	return DoGetVaultFunc(vaultQuery)
}

func (m *TestClient) GetVaultByUUID(uuid string) (*onepassword.Vault, error) {
	return DoGetVaultByUUIDFunc(uuid)
}

func (m *TestClient) GetVaultByTitle(title string) (*onepassword.Vault, error) {
	return DoGetVaultByTitleFunc(title)
}

func (m *TestClient) GetItem(itemQuery string, vaultQuery string) (*onepassword.Item, error) {
	return DoGetItemFunc(itemQuery, vaultQuery)
}

func (m *TestClient) GetItemByUUID(uuid string, vaultQuery string) (*onepassword.Item, error) {
	return DoGetItemByUUIDFunc(uuid, vaultQuery)
}

func (m *TestClient) GetItemByTitle(title string, vaultQuery string) (*onepassword.Item, error) {
	return DoGetItemByTitleFunc(title, vaultQuery)
}

func (m *TestClient) GetItems(vaultQuery string) ([]onepassword.Item, error) {
	return DoGetItemsFunc(vaultQuery)
}

func (m *TestClient) GetItemsByTitle(title string, vaultQuery string) ([]onepassword.Item, error) {
	return DoGetItemsByTitleFunc(title, vaultQuery)
}

func (m *TestClient) CreateItem(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error) {
	return DoCreateItemFunc(item, vaultQuery)
}

func (m *TestClient) UpdateItem(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error) {
	return DoUpdateItemFunc(item, vaultQuery)
}

func (m *TestClient) DeleteItem(item *onepassword.Item, vaultQuery string) error {
	return DoDeleteItemFunc(item, vaultQuery)
}

func (m *TestClient) DeleteItemByID(itemUUID string, vaultQuery string) error {
	return DoDeleteItemByIDFunc(itemUUID, vaultQuery)
}

func (m *TestClient) DeleteItemByTitle(title string, vaultQuery string) error {
	return DoDeleteItemByTitleFunc(title, vaultQuery)
}

func (m *TestClient) GetFiles(itemQuery string, vaultQuery string) ([]onepassword.File, error) {
	return DoGetFilesFunc(itemQuery, vaultQuery)
}

func (m *TestClient) GetFile(uuid string, itemQuery string, vaultQuery string) (*onepassword.File, error) {
	return DoGetFileFunc(uuid, itemQuery, vaultQuery)
}

func (m *TestClient) GetFileContent(file *onepassword.File) ([]byte, error) {
	return DoGetFileContentFunc(file)
}

func (m *TestClient) DownloadFile(file *onepassword.File, targetDirectory string, overwrite bool) (string, error) {
	return DoDownloadFileFunc(file, targetDirectory, overwrite)
}

func (m *TestClient) LoadStructFromItemByUUID(config interface{}, itemUUID string, vaultQuery string) error {
	return DoLoadStructFromItemByUUIDFunc(config, itemUUID, vaultQuery)
}

func (m *TestClient) LoadStructFromItemByTitle(config interface{}, itemTitle string, vaultQuery string) error {
	return DoLoadStructFromItemByTitleFunc(config, itemTitle, vaultQuery)
}

func (m *TestClient) LoadStructFromItem(config interface{}, itemQuery string, vaultQuery string) error {
	return DoLoadStructFromItemFunc(config, itemQuery, vaultQuery)
}

func (m *TestClient) LoadStruct(config interface{}) error {
	return DoLoadStructFunc(config)
}
