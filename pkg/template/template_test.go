package template

import (
	"testing"

	"github.com/1Password/onepassword-operator/pkg/onepassword/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildTemplateContext(t *testing.T) {
	item := &model.Item{
		ID:      "test-item-id",
		VaultID: "test-vault-id",
		Fields: []model.ItemField{
			{
				ID:        "field-1",
				Label:     "username",
				Value:     "testuser",
				SectionID: "section-1",
				FieldType: "STRING",
			},
			{
				ID:        "field-2",
				Label:     "password",
				Value:     "testpass",
				SectionID: "section-1",
				FieldType: "CONCEALED",
			},
			{
				ID:        "field-3",
				Label:     "api_key",
				Value:     "key123",
				SectionID: "",
				FieldType: "CONCEALED",
			},
		},
		Sections: []model.ItemSection{
			{
				ID:    "section-1",
				Title: "Credentials",
			},
		},
	}

	ctx := BuildTemplateContext(item)

	// Test Fields map
	assert.Equal(t, "testuser", ctx.Fields["username"])
	assert.Equal(t, "testpass", ctx.Fields["password"])
	assert.Equal(t, "key123", ctx.Fields["api_key"])

	// Test FieldsByID map
	assert.Equal(t, "testuser", ctx.FieldsByID["field-1"])
	assert.Equal(t, "testpass", ctx.FieldsByID["field-2"])
	assert.Equal(t, "key123", ctx.FieldsByID["field-3"])

	// Test Sections map
	assert.NotNil(t, ctx.Sections["Credentials"])
	assert.Equal(t, "testuser", ctx.Sections["Credentials"]["username"])
	assert.Equal(t, "testpass", ctx.Sections["Credentials"]["password"])

	// Test default section for fields without section
	assert.NotNil(t, ctx.Sections[""])
	assert.Equal(t, "key123", ctx.Sections[""]["api_key"])
}

func TestProcessTemplate(t *testing.T) {
	ctx := &TemplateContext{
		Fields: map[string]string{
			"username": "testuser",
			"password": "testpass",
			"endpoint": "https://example.com",
		},
		Sections: map[string]map[string]string{
			"Credentials": {
				"username": "testuser",
				"password": "testpass",
			},
		},
		FieldsByID: map[string]string{
			"field-1": "testuser",
			"field-2": "testpass",
		},
	}

	tests := []struct {
		name     string
		template string
		expected string
	}{
		{
			name:     "simple field access",
			template: "username: {{ .Fields.username }}",
			expected: "username: testuser",
		},
		{
			name:     "multiple fields",
			template: "provider: AWS\nusername: {{ .Fields.username }}\npassword: {{ .Fields.password }}",
			expected: "provider: AWS\nusername: testuser\npassword: testpass",
		},
		{
			name:     "section access",
			template: `user: {{ index .Sections "Credentials" "username" }}`,
			expected: "user: testuser",
		},
		{
			name:     "field by ID",
			template: `user: {{ index .FieldsByID "field-1" }}`,
			expected: "user: testuser",
		},
		{
			name: "complex template",
			template: "endpoint: {{ .Fields.endpoint }}\ncredentials:\n" +
				"  username: {{ .Fields.username }}\n  password: {{ .Fields.password }}",
			expected: "endpoint: https://example.com\ncredentials:\n" +
				"  username: testuser\n  password: testpass",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ProcessTemplate(tt.template, ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, string(result))
		})
	}
}

func TestProcessTemplate_InvalidTemplate(t *testing.T) {
	ctx := &TemplateContext{
		Fields: map[string]string{},
	}

	// Accessing non-existent field will error in Go templates
	_, err := ProcessTemplate("{{ .InvalidField }}", ctx)
	assert.Error(t, err) // Template execution errors on missing top-level fields

	_, err = ProcessTemplate("{{ .Fields.username }", ctx)
	assert.Error(t, err) // Invalid syntax should error
}

func TestBuildTemplateContext_DuplicateLabels(t *testing.T) {
	item := &model.Item{
		Fields: []model.ItemField{
			{
				ID:    "field-1",
				Label: "password",
				Value: "first",
			},
			{
				ID:    "field-2",
				Label: "password",
				Value: "second",
			},
		},
	}

	ctx := BuildTemplateContext(item)

	// Last one should win
	assert.Equal(t, "second", ctx.Fields["password"])
	// But both should be accessible by ID
	assert.Equal(t, "first", ctx.FieldsByID["field-1"])
	assert.Equal(t, "second", ctx.FieldsByID["field-2"])
}
