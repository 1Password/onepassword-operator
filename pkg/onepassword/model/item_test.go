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
			{
				ID:    "f1",
				Label: "field1",
				Value: "value1",
				Type:  "STRING",
				Section: &connect.ItemSection{
					ID:    "sec1",
					Label: "Section One",
				},
			},
			{
				ID:    "f2",
				Label: "field2",
				Value: "value2",
				Type:  "CONCEALED",
				Section: &connect.ItemSection{
					ID:    "sec1",
					Label: "Section One",
				},
			},
			{
				ID:    "f3",
				Label: "field3",
				Value: "value3",
				Type:  "STRING",
			},
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
		require.Equal(t, field.ID, item.Fields[i].ID)
		require.Equal(t, string(field.Type), item.Fields[i].FieldType)
	}

	// Verify sections are built from field references.
	require.Len(t, item.Sections, 1)
	require.Equal(t, "sec1", item.Sections[0].ID)
	require.Equal(t, "Section One", item.Sections[0].Title)

	// Verify section IDs on fields.
	require.Equal(t, "sec1", item.Fields[0].SectionID)
	require.Equal(t, "sec1", item.Fields[1].SectionID)
	require.Equal(t, "", item.Fields[2].SectionID)

	for i, file := range connectItem.Files {
		require.Equal(t, file.ID, item.Files[i].ID)
		require.Equal(t, file.Name, item.Files[i].Name)
		require.Equal(t, file.Size, item.Files[i].Size)
	}

	require.Equal(t, connectItem.CreatedAt, item.CreatedAt)
}

func TestItem_FromSDKItem(t *testing.T) {
	sec1ID := "sec1"
	sdkItem := &sdk.Item{
		ID:      "test-item-id",
		VaultID: "test-vault-id",
		Version: 1,
		Tags:    []string{"tag1", "tag2"},
		Sections: []sdk.ItemSection{
			{ID: "sec1", Title: "Section One"},
		},
		Fields: []sdk.ItemField{
			{ID: "1", Title: "field1", Value: "value1", SectionID: &sec1ID, FieldType: sdk.ItemFieldTypeText},
			{ID: "2", Title: "field2", Value: "value2", FieldType: sdk.ItemFieldTypeConcealed},
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
		require.Equal(t, field.ID, item.Fields[i].ID)
		require.Equal(t, string(field.FieldType), item.Fields[i].FieldType)
	}

	// Verify sections are populated from SDK item.
	require.Len(t, item.Sections, 1)
	require.Equal(t, "sec1", item.Sections[0].ID)
	require.Equal(t, "Section One", item.Sections[0].Title)

	// Verify section ID on fields.
	require.Equal(t, "sec1", item.Fields[0].SectionID)
	require.Equal(t, "", item.Fields[1].SectionID)

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
