package model

// ItemField Representation of a single field on an Item
type ItemField struct {
	ID        string
	Label     string
	Value     string
	SectionID string
	FieldType string
}
