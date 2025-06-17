package sdk

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	clienttesting "github.com/1Password/onepassword-operator/pkg/onepassword/client/testing"
	clientmock "github.com/1Password/onepassword-operator/pkg/onepassword/client/testing/mock"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	sdk "github.com/1password/onepassword-sdk-go"
)

const VaultTitleEmployee = "Employee"

func TestSDK_GetItemByID(t *testing.T) {
	sdkItem := clienttesting.CreateSDKItem()

	testCases := map[string]struct {
		mockItemAPI func() *clientmock.ItemAPIMock
		check       func(t *testing.T, item *model.Item, err error)
	}{
		"should return a single item": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				m := &clientmock.ItemAPIMock{}
				m.On("Get", context.Background(), "vault-id", "item-id").Return(*sdkItem, nil)
				return m
			},
			check: func(t *testing.T, item *model.Item, err error) {
				require.NoError(t, err)
				clienttesting.CheckSDKItemMapping(t, sdkItem, item)
			},
		},
		"should return an error": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				m := &clientmock.ItemAPIMock{}
				m.On("Get", context.Background(), "vault-id", "item-id").Return(sdk.Item{}, errors.New("error"))
				return m
			},
			check: func(t *testing.T, item *model.Item, err error) {
				require.Error(t, err)
				require.Empty(t, item)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &SDK{
				client: &sdk.Client{
					ItemsAPI: tc.mockItemAPI(),
				},
			}
			item, err := client.GetItemByID("vault-id", "item-id")
			tc.check(t, item, err)
		})
	}
}

func TestSDK_GetItemsByTitle(t *testing.T) {
	sdkItem1 := clienttesting.CreateSDKItemOverview()
	sdkItem2 := clienttesting.CreateSDKItemOverview()

	testCases := map[string]struct {
		mockItemAPI func() *clientmock.ItemAPIMock
		check       func(t *testing.T, items []model.Item, err error)
	}{
		"should return a single item": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				m := &clientmock.ItemAPIMock{}

				copySDKItem2 := *sdkItem2
				copySDKItem2.Title = "Some other item"

				m.On("List", context.Background(), "vault-id", mock.Anything).Return([]sdk.ItemOverview{
					*sdkItem1,
					copySDKItem2,
				}, nil)
				return m
			},
			check: func(t *testing.T, items []model.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 1)
				clienttesting.CheckSDKItemOverviewMapping(t, sdkItem1, &items[0])
			},
		},
		"should return a two items": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				m := &clientmock.ItemAPIMock{}
				m.On("List", context.Background(), "vault-id", mock.Anything).Return([]sdk.ItemOverview{
					*sdkItem1,
					*sdkItem2,
				}, nil)
				return m
			},
			check: func(t *testing.T, items []model.Item, err error) {
				require.NoError(t, err)
				require.Len(t, items, 2)
				clienttesting.CheckSDKItemOverviewMapping(t, sdkItem1, &items[0])
				clienttesting.CheckSDKItemOverviewMapping(t, sdkItem2, &items[1])
			},
		},
		"should return an error": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				m := &clientmock.ItemAPIMock{}
				m.On("List", context.Background(), "vault-id", mock.Anything).Return([]sdk.ItemOverview{}, errors.New("error"))
				return m
			},
			check: func(t *testing.T, items []model.Item, err error) {
				require.Error(t, err)
				require.Empty(t, items)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &SDK{
				client: &sdk.Client{
					ItemsAPI: tc.mockItemAPI(),
				},
			}
			items, err := client.GetItemsByTitle("vault-id", "item-title")
			tc.check(t, items, err)
		})
	}
}

func TestSDK_GetFileContent(t *testing.T) {
	testCases := map[string]struct {
		mockItemAPI func() *clientmock.ItemAPIMock
		check       func(t *testing.T, content []byte, err error)
	}{
		"should return file content": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				fileMock := &clientmock.FileAPIMock{}
				fileMock.On("Read", mock.Anything, "vault-id", "item-id",
					mock.MatchedBy(func(attr sdk.FileAttributes) bool {
						return attr.ID == "file-id"
					}),
				).Return([]byte("file content"), nil)

				itemMock := &clientmock.ItemAPIMock{
					FilesAPI: fileMock,
				}
				itemMock.On("Files").Return(fileMock)

				return itemMock
			},
			check: func(t *testing.T, content []byte, err error) {
				require.NoError(t, err)
				require.Equal(t, []byte("file content"), content)
			},
		},
		"should return an error": {
			mockItemAPI: func() *clientmock.ItemAPIMock {
				fileMock := &clientmock.FileAPIMock{}
				fileMock.On("Read", mock.Anything, "vault-id", "item-id",
					mock.MatchedBy(func(attr sdk.FileAttributes) bool {
						return attr.ID == "file-id"
					}),
				).Return(nil, errors.New("error"))

				itemMock := &clientmock.ItemAPIMock{
					FilesAPI: fileMock,
				}
				itemMock.On("Files").Return(fileMock)

				return itemMock
			},
			check: func(t *testing.T, content []byte, err error) {
				require.Error(t, err)
				require.Nil(t, content)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &SDK{
				client: &sdk.Client{
					ItemsAPI: tc.mockItemAPI(),
				},
			}
			content, err := client.GetFileContent("vault-id", "item-id", "file-id")
			tc.check(t, content, err)
		})
	}
}

func TestSDK_GetVaultsByTitle(t *testing.T) {
	now := time.Now()
	testCases := map[string]struct {
		mockVaultAPI func() *clientmock.VaultAPIMock
		check        func(t *testing.T, vaults []model.Vault, err error)
	}{
		"should return a single vault": {
			mockVaultAPI: func() *clientmock.VaultAPIMock {
				m := &clientmock.VaultAPIMock{}
				m.On("List", context.Background()).Return([]sdk.VaultOverview{
					{
						ID:        "test-id",
						Title:     VaultTitleEmployee,
						CreatedAt: now,
					},
					{
						ID:        "test-id-2",
						Title:     "Some other vault",
						CreatedAt: now,
					},
				}, nil)
				return m
			},
			check: func(t *testing.T, vaults []model.Vault, err error) {
				require.NoError(t, err)
				require.Len(t, vaults, 1)
				require.Equal(t, "test-id", vaults[0].ID)
				require.Equal(t, now, vaults[0].CreatedAt)
			},
		},
		"should return a two vaults": {
			mockVaultAPI: func() *clientmock.VaultAPIMock {
				m := &clientmock.VaultAPIMock{}
				m.On("List", context.Background()).Return([]sdk.VaultOverview{
					{
						ID:        "test-id",
						Title:     VaultTitleEmployee,
						CreatedAt: now,
					},
					{
						ID:        "test-id-2",
						Title:     VaultTitleEmployee,
						CreatedAt: now,
					},
				}, nil)
				return m
			},
			check: func(t *testing.T, vaults []model.Vault, err error) {
				require.NoError(t, err)
				require.Len(t, vaults, 2)
				// Check the first vault
				require.Equal(t, "test-id", vaults[0].ID)
				require.Equal(t, now, vaults[0].CreatedAt)
				// Check the second vault
				require.Equal(t, "test-id-2", vaults[1].ID)
				require.Equal(t, now, vaults[1].CreatedAt)
			},
		},
		"should return an error": {
			mockVaultAPI: func() *clientmock.VaultAPIMock {
				m := &clientmock.VaultAPIMock{}
				m.On("List", context.Background()).Return([]sdk.VaultOverview{}, errors.New("error"))
				return m
			},
			check: func(t *testing.T, vaults []model.Vault, err error) {
				require.Error(t, err)
				require.Empty(t, vaults)
			},
		},
	}

	for description, tc := range testCases {
		t.Run(description, func(t *testing.T) {
			client := &SDK{
				client: &sdk.Client{
					VaultsAPI: tc.mockVaultAPI(),
				},
			}
			vault, err := client.GetVaultsByTitle(VaultTitleEmployee)
			tc.check(t, vault, err)
		})
	}
}
