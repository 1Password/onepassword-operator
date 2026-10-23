package model

import (
	"time"

	connect "github.com/1Password/connect-sdk-go/onepassword"
	sdk "github.com/1password/onepassword-sdk-go"
)

// Item represents 1Password item.
type Item struct {
	ID        string
	VaultID   string
	Version   int
	Tags      []string
	URLs      []ItemURL
	Sections  []ItemSection
	Fields    []ItemField
	Files     []File
	CreatedAt time.Time
}

// ItemURL represents a URL associated with a 1Password item.
type ItemURL struct {
	URL     string
	Label   string
	Primary bool
}

// ItemSection represents a section within a 1Password item.
type ItemSection struct {
	ID    string
	Title string
}

// FromConnectItem populates the Item from a Connect item.
func (i *Item) FromConnectItem(item *connect.Item) {
	i.ID = item.ID
	i.VaultID = item.Vault.ID
	i.Version = item.Version

	i.Tags = append(i.Tags, item.Tags...)

	for _, url := range item.URLs {
		i.URLs = append(i.URLs, ItemURL{
			URL:     url.URL,
			Label:   url.Label,
			Primary: url.Primary,
		})
	}

	// Build sections from field references. The Connect SDK stores section
	// info on each field rather than as a top-level list.
	sectionSeen := make(map[string]bool)
	for _, field := range item.Fields {
		sectionID := ""
		if field.Section != nil {
			sectionID = field.Section.ID
			if !sectionSeen[sectionID] {
				sectionSeen[sectionID] = true
				title := field.Section.Label
				i.Sections = append(i.Sections, ItemSection{
					ID:    sectionID,
					Title: title,
				})
			}
		}

		i.Fields = append(i.Fields, ItemField{
			ID:        field.ID,
			Label:     field.Label,
			Value:     field.Value,
			SectionID: sectionID,
			FieldType: string(field.Type),
		})
	}

	for _, file := range item.Files {
		i.Files = append(i.Files, File{
			ID:   file.ID,
			Name: file.Name,
			Size: file.Size,
		})
	}

	i.CreatedAt = item.CreatedAt
}

// FromSDKItem populates the Item from an SDK item.
func (i *Item) FromSDKItem(item *sdk.Item) {
	i.ID = item.ID
	i.VaultID = item.VaultID
	i.Version = int(item.Version)

	i.Tags = make([]string, len(item.Tags))
	copy(i.Tags, item.Tags)

	for idx, url := range item.Websites {
		i.URLs = append(i.URLs, ItemURL{
			URL:     url.URL,
			Label:   url.Label,
			Primary: idx == 0,
		})
	}

	// Populate sections from the SDK item.
	for _, section := range item.Sections {
		i.Sections = append(i.Sections, ItemSection{
			ID:    section.ID,
			Title: section.Title,
		})
	}

	for _, field := range item.Fields {
		sectionID := ""
		if field.SectionID != nil {
			sectionID = *field.SectionID
		}
		i.Fields = append(i.Fields, ItemField{
			ID:        field.ID,
			Label:     field.Title,
			Value:     field.Value,
			SectionID: sectionID,
			FieldType: string(field.FieldType),
		})
	}

	for _, file := range item.Files {
		i.Files = append(i.Files, File{
			ID:   file.Attributes.ID,
			Name: file.Attributes.Name,
			Size: int(file.Attributes.Size),
		})
	}

	// Items of 'Document' category keeps file information in the Document field.
	if item.Category == sdk.ItemCategoryDocument {
		i.Files = append(i.Files, File{
			ID:   item.Document.ID,
			Name: item.Document.Name,
			Size: int(item.Document.Size),
		})
	}

	i.CreatedAt = item.CreatedAt
}

// FromSDKItemOverview populates the Item from an SDK item overview.
func (i *Item) FromSDKItemOverview(item *sdk.ItemOverview) {
	i.ID = item.ID
	i.VaultID = item.VaultID

	i.Tags = make([]string, len(item.Tags))
	copy(i.Tags, item.Tags)

	i.CreatedAt = item.CreatedAt
}
