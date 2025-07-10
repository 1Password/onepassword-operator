package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/1Password/connect-sdk-go/onepassword"
	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	sdk "github.com/1password/onepassword-sdk-go"
)

func CreateConnectItem() *onepassword.Item {
	return &onepassword.Item{
		ID:      "test-id",
		Vault:   onepassword.ItemVault{ID: "test-vault-id"},
		Version: 1,
		Tags:    []string{"tag1", "tag2"},
		Fields: []*onepassword.ItemField{
			{Label: "label1", Value: "value1"},
			{Label: "label2", Value: "value2"},
		},
		Files: []*onepassword.File{
			{ID: "file-id-1", Name: "file1.txt", Size: 1234},
			{ID: "file-id-2", Name: "file2.txt", Size: 1234},
		},
	}
}

func CreateSDKItem() *sdk.Item {
	return &sdk.Item{
		ID:      "test-id",
		VaultID: "test-vault-id",
		Version: 1,
		Tags:    []string{"tag1", "tag2"},
		Fields: []sdk.ItemField{
			{Title: "label1", Value: "value1"},
			{Title: "label2", Value: "value2"},
		},
		Files: []sdk.ItemFile{
			{Attributes: sdk.FileAttributes{ID: "file-id-1", Name: "file1.txt", Size: 1234}},
			{Attributes: sdk.FileAttributes{ID: "file-id-2", Name: "file2.txt", Size: 1234}},
		},
		CreatedAt: time.Now(),
	}
}

func CreateSDKItemOverview() *sdk.ItemOverview {
	return &sdk.ItemOverview{
		ID:        "test-id",
		Title:     "item-title",
		VaultID:   "test-vault-id",
		Tags:      []string{"tag1", "tag2"},
		CreatedAt: time.Now(),
	}
}

func CheckConnectItemMapping(t *testing.T, expected *onepassword.Item, actual *model.Item) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Vault.ID, actual.VaultID)
	require.Equal(t, expected.Version, actual.Version)
	require.ElementsMatch(t, expected.Tags, actual.Tags)

	for i, field := range expected.Fields {
		require.Equal(t, field.Label, actual.Fields[i].Label)
		require.Equal(t, field.Value, actual.Fields[i].Value)
	}

	for i, file := range expected.Files {
		require.Equal(t, file.ID, actual.Files[i].ID)
		require.Equal(t, file.Name, actual.Files[i].Name)
		require.Equal(t, file.Size, actual.Files[i].Size)
	}

	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
}

func CheckSDKItemMapping(t *testing.T, expected *sdk.Item, actual *model.Item) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.VaultID, actual.VaultID)
	require.Equal(t, int(expected.Version), actual.Version)
	require.ElementsMatch(t, expected.Tags, actual.Tags)

	for i, field := range expected.Fields {
		require.Equal(t, field.Title, actual.Fields[i].Label)
		require.Equal(t, field.Value, actual.Fields[i].Value)
	}

	for i, file := range expected.Files {
		require.Equal(t, file.Attributes.ID, actual.Files[i].ID)
		require.Equal(t, file.Attributes.Name, actual.Files[i].Name)
		require.Equal(t, int(file.Attributes.Size), actual.Files[i].Size)
	}

	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
}

func CheckSDKItemOverviewMapping(t *testing.T, expected *sdk.ItemOverview, actual *model.Item) {
	t.Helper()

	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.VaultID, actual.VaultID)
	require.ElementsMatch(t, expected.Tags, actual.Tags)
	require.Equal(t, expected.CreatedAt, actual.CreatedAt)
}
