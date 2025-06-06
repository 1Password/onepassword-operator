package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	connect "github.com/1Password/connect-sdk-go/onepassword"
	sdk "github.com/1password/onepassword-sdk-go"
)

func TestItem_FromConnectItem(t *testing.T) {
	connectItem := &connect.Item{
		ID: "test-item-id",
		Vault: connect.ItemVault{
			ID: "test-vault-id",
		},
		Version: 1,
		Tags:    []string{"tag1", "tag2"},
		Fields: []*connect.ItemField{
			{Label: "field1", Value: "value1"},
			{Label: "field2", Value: "value2"},
		},
		Files: []*connect.File{
			{ID: "file1", Name: "file1.txt", Size: 1234},
			{ID: "file2", Name: "file2.txt", Size: 1234},
		},
		CreatedAt: time.Now(),
	}

	item := &Item{}
	item.FromConnectItem(connectItem)

	require.Equal(t, connectItem.ID, item.ID)
	require.Equal(t, connectItem.Vault.ID, item.VaultID)
	require.Equal(t, connectItem.Version, item.Version)
	require.ElementsMatch(t, connectItem.Tags, item.Tags)

	for i, field := range connectItem.Fields {
		require.Equal(t, field.Label, item.Fields[i].Label)
		require.Equal(t, field.Value, item.Fields[i].Value)
	}

	for i, file := range connectItem.Files {
		require.Equal(t, file.ID, item.Files[i].ID)
		require.Equal(t, file.Name, item.Files[i].Name)
		require.Equal(t, file.Size, item.Files[i].Size)
	}

	require.Equal(t, connectItem.CreatedAt, item.CreatedAt)
}

func TestItem_FromSDKItem(t *testing.T) {
	sdkItem := &sdk.Item{
		ID:      "test-item-id",
		VaultID: "test-vault-id",
		Version: 1,
		Tags:    []string{"tag1", "tag2"},
		Fields: []sdk.ItemField{
			{ID: "1", Title: "field1", Value: "value1"},
			{ID: "2", Title: "field2", Value: "value2"},
		},
		Files: []sdk.ItemFile{
			{Attributes: sdk.FileAttributes{Name: "file1.txt", Size: 1234}, FieldID: "file1"},
			{Attributes: sdk.FileAttributes{Name: "file2.txt", Size: 1234}, FieldID: "file2"},
		},
		CreatedAt: time.Now(),
	}

	item := &Item{}
	item.FromSDKItem(sdkItem)

	require.Equal(t, sdkItem.ID, item.ID)
	require.Equal(t, sdkItem.VaultID, item.VaultID)
	require.Equal(t, int(sdkItem.Version), item.Version)
	require.ElementsMatch(t, sdkItem.Tags, item.Tags)

	for i, field := range sdkItem.Fields {
		require.Equal(t, field.Title, item.Fields[i].Label)
		require.Equal(t, field.Value, item.Fields[i].Value)
	}

	for i, file := range sdkItem.Files {
		require.Equal(t, file.Attributes.ID, item.Files[i].ID)
		require.Equal(t, file.Attributes.Name, item.Files[i].Name)
		require.Equal(t, int(file.Attributes.Size), item.Files[i].Size)
	}

	require.Equal(t, sdkItem.CreatedAt, item.CreatedAt)
}

func TestItem_FromSDKItemOverview(t *testing.T) {
	sdkItemOverview := &sdk.ItemOverview{
		ID:        "test-item-id",
		VaultID:   "test-vault-id",
		Tags:      []string{"tag1", "tag2"},
		CreatedAt: time.Now(),
	}

	item := &Item{}
	item.FromSDKItemOverview(sdkItemOverview)

	require.Equal(t, sdkItemOverview.ID, item.ID)
	require.Equal(t, sdkItemOverview.VaultID, item.VaultID)
	require.ElementsMatch(t, sdkItemOverview.Tags, item.Tags)
	require.Equal(t, sdkItemOverview.CreatedAt, item.CreatedAt)
}
