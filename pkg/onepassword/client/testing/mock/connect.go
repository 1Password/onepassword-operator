package mock

import (
	"github.com/stretchr/testify/mock"

	"github.com/1Password/connect-sdk-go/onepassword"
)

// ConnectClientMock is a mock implementation of the ConnectClient interface
type ConnectClientMock struct {
	mock.Mock
}

func (c *ConnectClientMock) GetVaults() ([]onepassword.Vault, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetVault(uuid string) (*onepassword.Vault, error) {
	args := c.Called(uuid)
	return args.Get(0).(*onepassword.Vault), args.Error(1)
}

func (c *ConnectClientMock) GetVaultByUUID(uuid string) (*onepassword.Vault, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetVaultByTitle(title string) (*onepassword.Vault, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetVaultsByTitle(title string) ([]onepassword.Vault, error) {
	args := c.Called(title)
	return args.Get(0).([]onepassword.Vault), args.Error(1)
}

func (c *ConnectClientMock) GetItems(vaultQuery string) ([]onepassword.Item, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetItem(itemQuery, vaultQuery string) (*onepassword.Item, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetItemByUUID(uuid string, vaultQuery string) (*onepassword.Item, error) {
	args := c.Called(uuid, vaultQuery)
	return args.Get(0).(*onepassword.Item), args.Error(1)
}

func (c *ConnectClientMock) GetItemByTitle(title string, vaultQuery string) (*onepassword.Item, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetItemsByTitle(title string, vaultQuery string) ([]onepassword.Item, error) {
	args := c.Called(title, vaultQuery)
	return args.Get(0).([]onepassword.Item), args.Error(1)
}

func (c *ConnectClientMock) CreateItem(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) UpdateItem(item *onepassword.Item, vaultQuery string) (*onepassword.Item, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) DeleteItem(item *onepassword.Item, vaultQuery string) error {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) DeleteItemByID(itemUUID string, vaultQuery string) error {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) DeleteItemByTitle(title string, vaultQuery string) error {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetFiles(itemQuery string, vaultQuery string) ([]onepassword.File, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetFile(uuid string, itemQuery string, vaultQuery string) (*onepassword.File, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) GetFileContent(file *onepassword.File) ([]byte, error) {
	args := c.Called(file)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]byte), args.Error(1)
}

func (c *ConnectClientMock) DownloadFile(file *onepassword.File, targetDirectory string, overwrite bool) (string, error) {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) LoadStructFromItemByUUID(config interface{}, itemUUID string, vaultQuery string) error {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) LoadStructFromItemByTitle(config interface{}, itemTitle string, vaultQuery string) error {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) LoadStructFromItem(config interface{}, itemQuery string, vaultQuery string) error {
	// implement when need to mock this method
	panic("implement me")
}

func (c *ConnectClientMock) LoadStruct(config interface{}) error {
	// implement when need to mock this method
	panic("implement me")
}
