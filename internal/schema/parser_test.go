package schema

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// helper to wrap raw component schemas into a valid OpenAPI document
func wrapInOpenAPIBoilerplate(schemaYAML string) string {
	const boilerplate = `openapi: "3.0.0"
info:
  version: "1.0.0"
  title: Test API
paths: {}
components:
  schemas:
`
	// Indent the user's schema snippet cleanly under components.schemas
	// We split by newline and add spaces so you don't have to worry about manual indentation strings
	var indentedLines []string
	for _, line := range strings.Split(strings.TrimSpace(schemaYAML), "\n") {
		if line != "" {
			indentedLines = append(indentedLines, "    "+line)
		} else {
			indentedLines = append(indentedLines, "")
		}
	}

	return boilerplate + strings.Join(indentedLines, "\n")
}

// generateSingleFieldSpec creates a fully valid OpenAPI YAML string
// containing one schema with a single property.
func generateSingleFieldSpec(fieldName, fieldType, format string) string {
	var formatLine string
	if format != "" {
		formatLine = fmt.Sprintf("\n          format: %s", format)
	}

	// We use standard strings.ReplaceAll to avoid layout/tab issues entirely
	template := `openapi: "3.0.0"
info:
  version: "1.0.0"
  title: Test API
paths: {}
components:
  schemas:
    Message:
      type: object
      properties:
        [FIELD_NAME]:
          description: Auto-generated test field
          type: [FIELD_TYPE][FORMAT_LINE]`

	replacer := strings.NewReplacer(
		"[FIELD_NAME]", fieldName,
		"[FIELD_TYPE]", fieldType,
		"[FORMAT_LINE]", formatLine,
	)

	return replacer.Replace(template)
}

func validateField(t *testing.T, properties map[string]*Field, expectedName string,
	expectedType FieldType) {
	field, exists := properties[expectedName]
	assert.True(t, exists, "expected field '%s' to exist", expectedName)

	assert.Equal(t, expectedName, field.Name, "field name mismatch")
	assert.Equal(t, expectedType, field.Type, "field type mismatch")
}

func TestParseAndMapOpenAPIWithIntegerType(t *testing.T) {
	// Define tests table
	tests := []struct {
		name          string
		yamlContent   string
		expectErr     bool
		errContains   string
		expectedCount int
		validateFunc  func(t *testing.T, schema *Schema) // Custom assertions for successful runs
	}{
		// Integer type tests
		// ==================
		{
			name:          "Test integer type",
			yamlContent:   generateSingleFieldSpec("id", "integer", ""),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeInt32)
			},
		},
		{
			name:          "Test integer type with int16 format",
			yamlContent:   generateSingleFieldSpec("id", "integer", "int16"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeInt16)
			},
		},
		{
			name:          "Test integer type with int32 format",
			yamlContent:   generateSingleFieldSpec("id", "integer", "int32"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeInt32)
			},
		},
		{
			name:          "Test integer type with int64 format",
			yamlContent:   generateSingleFieldSpec("id", "integer", "int64"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeInt64)
			},
		},
		{
			name:          "Test integer type with uint16 format",
			yamlContent:   generateSingleFieldSpec("id", "integer", "uint16"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeUint16)
			},
		},
		{
			name:          "Test integer type with uint32 format",
			yamlContent:   generateSingleFieldSpec("id", "integer", "uint32"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeUint32)
			},
		},
		{
			name:          "Test integer type with uint64 format",
			yamlContent:   generateSingleFieldSpec("id", "integer", "uint64"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "id", FieldTypeUint64)
			},
		},
		{
			name:        "Test integer type with invalid format",
			yamlContent: generateSingleFieldSpec("id", "integer", "int128"),
			expectErr:   true,
		},

		// Number type tests
		// ==================
		{
			name:          "Test number type",
			yamlContent:   generateSingleFieldSpec("value", "number", ""),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "value", FieldTypeFloat32)
			},
		},
		{
			name:          "Test number type with float format",
			yamlContent:   generateSingleFieldSpec("value", "number", "float"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "value", FieldTypeFloat32)
			},
		},
		{
			name:          "Test number type with double format",
			yamlContent:   generateSingleFieldSpec("value", "number", "double"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "value", FieldTypeFloat64)
			},
		},

		// Boolean type tests
		// ==================
		{
			name:          "Test boolean type",
			yamlContent:   generateSingleFieldSpec("isActive", "boolean", ""),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "isActive", FieldTypeBool)
			},
		},

		// String type tests
		// ==================
		{
			name:          "Test String type",
			yamlContent:   generateSingleFieldSpec("name", "string", ""),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "name", FieldTypeString)
			},
		},
		{
			name:          "Test String type with email format",
			yamlContent:   generateSingleFieldSpec("email", "string", "email"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "email", FieldTypeString)
			},
		},
		{
			name:          "Test String type with byte format",
			yamlContent:   generateSingleFieldSpec("encodedMsg", "string", "byte"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "encodedMsg", FieldTypeBytes)
			},
		},
		{
			name:          "Test String type with binary format",
			yamlContent:   generateSingleFieldSpec("fileContent", "string", "binary"),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")

				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "fileContent", FieldTypeBytes)
			},
		},

		// Object type tests
		// ==================
		{
			name: "Test Object type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  properties:
    Request:
      type: object
      properties:
        key:
          type: string
`),
			expectErr: false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "Request", FieldTypeObject)

				// Validate nested object
				nestedField := msg.Properties["Request"]
				assert.NotNil(t, nestedField.Nested, "expected nested schema for 'Request'")
				assert.Equal(t, nestedField.Nested.Name, "Request", "nested field name mismatch")
				assert.Len(t, nestedField.Nested.Fields, 1, "expected exactly one field")
				validateField(t, nestedField.Nested.Properties, "key", FieldTypeString)
			},
		},
		{
			name: "Test Object type with multiple levels of nesting",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  properties:
    Request:
      type: object
      properties:
        key:
          type: string
        metadata:
          type: object
          properties:
            timestamp:
              type: string
              format: date-time
`),
			expectErr: false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				assert.Len(t, msg.Properties, 1, "expected exactly one property")
				validateField(t, msg.Properties, "Request", FieldTypeObject)

				// Validate nested object
				nestedField := msg.Properties["Request"]
				assert.NotNil(t, nestedField.Nested, "expected nested schema for 'Request'")
				assert.Equal(t, nestedField.Nested.Name, "Request", "nested field name mismatch")
				assert.Len(t, nestedField.Nested.Fields, 2, "expected exactly one field")
				validateField(t, nestedField.Nested.Properties, "key", FieldTypeString)
				validateField(t, nestedField.Nested.Properties, "metadata", FieldTypeObject)

				// Validate second level nested object
				metadataField := nestedField.Nested.Properties["metadata"]
				assert.NotNil(t, metadataField.Nested, "expected nested schema for 'metadata'")
				assert.Equal(t, metadataField.Nested.Name, "metadata", "nested field name mismatch")
				assert.Len(t, metadataField.Nested.Fields, 1, "expected exactly one field")
				validateField(t, metadataField.Nested.Properties, "timestamp", FieldTypeString)
			},
		},

		// Array type tests
		// ==================
		{
			name: "Test Array type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  properties:
    Tags:
      type: array
      items:
        type: string
`),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a unique temporary directory for this specific subtest
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "openapi_test.yaml")

			// Write fake file payload
			err := os.WriteFile(tmpFile, []byte(strings.TrimSpace(tt.yamlContent)), 0644)
			if err != nil {
				t.Fatalf("failed to create temporary test file: %v", err)
			}

			// Execute target code
			result, err := ParseFile(tmpFile)

			// Assert error expectations
			if tt.expectErr {
				assert.NotNil(t, err)
				return
			}

			assert.Nil(t, err)
			assert.Len(t, result.Messages, 1, "expected message count mismatch")

			// Execute functional assertions if it passed
			if tt.validateFunc != nil {
				tt.validateFunc(t, result)
			}
		})
	}
}
