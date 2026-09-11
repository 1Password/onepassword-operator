package template

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
)

// TemplateContext provides data for Go template processing.
type TemplateContext struct {
	// Fields is a flat map: field_label -> value
	// If duplicate labels exist across sections, the last one wins.
	Fields map[string]string
	// Sections is nested: section_title -> field_label -> value
	// Allows access to fields organized by section.
	Sections map[string]map[string]string
	// FieldsByID provides precise access: field_id -> value
	// Use this when field labels might collide across sections.
	FieldsByID map[string]string
}

// BuildTemplateContext constructs a TemplateContext from a 1Password item.
func BuildTemplateContext(item *model.Item) *TemplateContext {
	ctx := &TemplateContext{
		Fields:     make(map[string]string),
		Sections:   make(map[string]map[string]string),
		FieldsByID: make(map[string]string),
	}

	// Build section map by section ID for efficient lookup
	sectionMap := make(map[string]string) // section_id -> section_title
	for _, section := range item.Sections {
		sectionMap[section.ID] = section.Title
		if ctx.Sections[section.Title] == nil {
			ctx.Sections[section.Title] = make(map[string]string)
		}
	}

	// Process all fields
	for _, field := range item.Fields {
		// Add to flat Fields map (last one wins if duplicate labels)
		ctx.Fields[field.Label] = field.Value

		// Add to FieldsByID for precise access
		ctx.FieldsByID[field.ID] = field.Value

		// Add to Sections map if field has a section
		if field.SectionID != "" {
			sectionTitle := sectionMap[field.SectionID]
			if sectionTitle == "" {
				// Section ID exists but not in sections array, use ID as fallback
				sectionTitle = field.SectionID
			}
			if ctx.Sections[sectionTitle] == nil {
				ctx.Sections[sectionTitle] = make(map[string]string)
			}
			ctx.Sections[sectionTitle][field.Label] = field.Value
		} else {
			// Field without section - add to a default/empty section
			if ctx.Sections[""] == nil {
				ctx.Sections[""] = make(map[string]string)
			}
			ctx.Sections[""][field.Label] = field.Value
		}
	}

	return ctx
}

// ProcessTemplate processes a Go template string with the given context.
func ProcessTemplate(tmpl string, ctx *TemplateContext) ([]byte, error) {
	t, err := template.New("secret").Parse(tmpl)
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := t.Execute(&buf, ctx); err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.Bytes(), nil
}
