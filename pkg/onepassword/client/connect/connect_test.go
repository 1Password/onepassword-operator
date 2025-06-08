package connect

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/1Password/connect-sdk-go/onepassword"
	clienttesting "github.com/1Password/onepassword-operator/pkg/onepassword/client/testing"
	"github.com/1Password/onepassword-operator/pkg/onepassword/client/testing/mock"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

const VaultTitleEmployee = "Employee"

func TestConnect_GetItemByID(t *testing.T) {
	connectItem := clienttesting.CreateConnectItem()

	testCases := map[string]struct {
		mockClient func() *mock.ConnectClientMock
		check      func(t *testing.T, item *model.Item, err error)
	}{
		"should return an item": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetItemByUUID", "item-id", "vault-id").Return(connectItem, nil)
				return mockConnectClient
			},
			check: func(t *testing.T, item *model.Item, err error) {
				require.NoError(t, err)
				clienttesting.CheckConnectItemMapping(t, connectItem, item)
			},
		},
		"should return an error": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetItemByUUID", "item-id", "vault-id").Return((*onepassword.Item)(nil), errors.New("error"))
				return mockConnectClient
			},
			check: func(t *testing.T, item *model.Item, err error) {
				require.Error(t, err)
				require.Nil(t, item)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &Connect{client: tc.mockClient()}
			item, err := client.GetItemByID(context.Background(), "vault-id", "item-id")
			tc.check(t, item, err)
		})
	}
}

func TestConnect_GetItemsByTitle(t *testing.T) {
	connectItem1 := clienttesting.CreateConnectItem()
	connectItem2 := clienttesting.CreateConnectItem()

	testCases := map[string]struct {
		mockClient func() *mock.ConnectClientMock
		check      func(t *testing.T, items []model.Item, err error)
	}{
		"should return a single item": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetItemsByTitle", "item-title", "vault-id").Return(
					[]onepassword.Item{
						*connectItem1,
					}, nil)
				return mockConnectClient
			},
			check: func(t *testing.T, items []model.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 1)
				require.Equal(t, connectItem1.ID, items[0].ID)
			},
		},
		"should return two items": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetItemsByTitle", "item-title", "vault-id").Return(
					[]onepassword.Item{
						*connectItem1,
						*connectItem2,
					}, nil)
				return mockConnectClient
			},
			check: func(t *testing.T, items []model.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 2)
				clienttesting.CheckConnectItemMapping(t, connectItem1, &items[0])
				clienttesting.CheckConnectItemMapping(t, connectItem2, &items[1])
			},
		},
		"should return an error": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetItemsByTitle", "item-title", "vault-id").Return([]onepassword.Item{}, errors.New("error"))
				return mockConnectClient
			},
			check: func(t *testing.T, items []model.Item, err error) {
				require.Error(t, err)
				require.Nil(t, items)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &Connect{client: tc.mockClient()}
			items, err := client.GetItemsByTitle(context.Background(), "vault-id", "item-title")
			tc.check(t, items, err)
		})
	}
}

func TestConnect_GetFileContent(t *testing.T) {
	testCases := map[string]struct {
		mockClient func() *mock.ConnectClientMock
		check      func(t *testing.T, content []byte, err error)
	}{
		"should return file content": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetFileContent", &onepassword.File{
					ContentPath: "/v1/vaults/vault-id/items/item-id/files/file-id/content",
				}).Return([]byte("file content"), nil)
				return mockConnectClient
			},
			check: func(t *testing.T, content []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("file content"), content)
			},
		},
		"should return an error": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetFileContent", &onepassword.File{
					ContentPath: "/v1/vaults/vault-id/items/item-id/files/file-id/content",
				}).Return(nil, errors.New("error"))
				return mockConnectClient
			},
			check: func(t *testing.T, content []byte, err error) {
				require.Error(t, err)
				require.Nil(t, content)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &Connect{client: tc.mockClient()}
			content, err := client.GetFileContent(context.Background(), "vault-id", "item-id", "file-id")
			tc.check(t, content, err)
		})
	}
}

func TestConnect_GetVaultsByTitle(t *testing.T) {
	testCases := map[string]struct {
		mockClient func() *mock.ConnectClientMock
		check      func(t *testing.T, vaults []model.Vault, err error)
	}{
		"should return a single vault": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetVaultsByTitle", VaultTitleEmployee).Return([]onepassword.Vault{
					{
						ID:   "test-id",
						Name: VaultTitleEmployee,
					},
					{
						ID:   "test-id-2",
						Name: "Some other vault",
					},
				}, nil)
				return mockConnectClient
			},
			check: func(t *testing.T, vaults []model.Vault, err error) {
				require.NoError(t, err)
				require.Len(t, vaults, 1)
				require.Equal(t, "test-id", vaults[0].ID)
			},
		},
		"should return a two vaults": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetVaultsByTitle", VaultTitleEmployee).Return([]onepassword.Vault{
					{
						ID:   "test-id",
						Name: VaultTitleEmployee,
					},
					{
						ID:   "test-id-2",
						Name: VaultTitleEmployee,
					},
				}, nil)
				return mockConnectClient
			},
			check: func(t *testing.T, vaults []model.Vault, err error) {
				require.NoError(t, err)
				require.Len(t, vaults, 2)
				// Check the first vault
				require.Equal(t, "test-id", vaults[0].ID)
				// Check the second vault
				require.Equal(t, "test-id-2", vaults[1].ID)
			},
		},
		"should return an error": {
			mockClient: func() *mock.ConnectClientMock {
				mockConnectClient := &mock.ConnectClientMock{}
				mockConnectClient.On("GetVaultsByTitle", VaultTitleEmployee).Return([]onepassword.Vault{}, errors.New("error"))
				return mockConnectClient
			},
			check: func(t *testing.T, vaults []model.Vault, err error) {
				require.Error(t, err)
				require.Empty(t, vaults)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &Connect{client: tc.mockClient()}
			vault, err := client.GetVaultsByTitle(context.Background(), VaultTitleEmployee)
			tc.check(t, vault, err)
		})
	}
}
