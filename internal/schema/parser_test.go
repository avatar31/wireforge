// Copyright (c) 2026 Sachin S. All rights reserved.
//
// Licensed under the MIT License.
// See LICENSE in the project root.

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
	template := `openapi: "3.1.0"
info:
  version: "1.0.0"
  title: Test API
paths: {}
components:
  schemas:
    Message:
      type: object
      x-message-id: 1
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
	assert.False(t, field.Nullable, "field should not be nullable by default")
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

				assert.Equal(t, msg.TypeID, uint16(1), "message type ID mismatch")
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
  x-message-id: 1
  properties:
    Request:
      type: object
      properties:
        key:
          type: string
`),
			expectErr:     false,
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
				assert.Equal(t, nestedField.Nested.Name, "MessageRequest", "nested field name mismatch")
				assert.Len(t, nestedField.Nested.Fields, 1, "expected exactly one field")
				validateField(t, nestedField.Nested.Properties, "key", FieldTypeString)
			},
		},
		{
			name: "Test Object type with multiple levels of nesting",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
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
			expectErr:     false,
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
				assert.Equal(t, nestedField.Nested.Name, "MessageRequest", "nested field name mismatch")
				assert.Len(t, nestedField.Nested.Fields, 2, "expected exactly one field")
				validateField(t, nestedField.Nested.Properties, "key", FieldTypeString)
				validateField(t, nestedField.Nested.Properties, "metadata", FieldTypeObject)

				// Validate second level nested object
				metadataField := nestedField.Nested.Properties["metadata"]
				assert.NotNil(t, metadataField.Nested, "expected nested schema for 'metadata'")
				assert.Equal(t, metadataField.Nested.Name, "MessageRequestMetadata", "nested field name mismatch")
				assert.Len(t, metadataField.Nested.Fields, 1, "expected exactly one field")
				validateField(t, metadataField.Nested.Properties, "timestamp", FieldTypeString)
			},
		},

		// Array type tests
		// ==================
		{
			name: "Test Array of string type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Tags:
      type: array
      items:
        type: string
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Tags", FieldTypeArray)

				arr := msg.Properties["Tags"]
				assert.True(t, arr.IsVariable, "array field must be variable-length")
				assert.NotNil(t, arr.ArrElem, "expected array element descriptor")

				assert.Equal(t, FieldTypeString, arr.ArrElem.Type, "array element type mismatch")
				assert.Nil(t, arr.ArrElem.Nested, "array element should not have nested schema for string type")
				assert.Nil(t, arr.ArrElem.ArrElem, "array element should not have nested array for string type")
				assert.True(t, arr.ArrElem.IsVariable, "array element should be variable-length for string type")
			},
		},
		{
			name: "Test Array of integer type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Ports:
      type: array
      items:
        type: integer
        format: uint16
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Ports", FieldTypeArray)

				arr := msg.Properties["Ports"]
				assert.True(t, arr.IsVariable, "array field must be variable-length")
				assert.NotNil(t, arr.ArrElem, "expected array element descriptor")

				assert.Equal(t, FieldTypeUint16, arr.ArrElem.Type, "array element type mismatch")
				assert.Nil(t, arr.ArrElem.Nested, "array element should not have nested schema for integer type")
				assert.Nil(t, arr.ArrElem.ArrElem, "array element should not have nested array for integer type")
				assert.False(t, arr.ArrElem.IsVariable, "array element should not be variable-length for integer type")
			},
		},
		{
			name: "Test Array of object type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Users:
      type: array
      items:
        type: object
        properties:
          username:
            type: string
          email:
            type: string
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Users", FieldTypeArray)

				arr := msg.Properties["Users"]
				assert.True(t, arr.IsVariable, "array field must be variable-length")
				assert.Nil(t, arr.Nested, "array field should not have nested schema for object type")
				assert.NotNil(t, arr.ArrElem, "expected array element descriptor")

				assert.Equal(t, FieldTypeObject, arr.ArrElem.Type, "array element type mismatch")
				assert.NotNil(t, arr.ArrElem.Nested, "array element should have nested schema for object type")
				assert.Nil(t, arr.ArrElem.ArrElem, "array element should not have nested array for object type")
				assert.Len(t, arr.ArrElem.Nested.Fields, 2, "expected two fields in nested object")

				validateField(t, arr.ArrElem.Nested.Properties, "username", FieldTypeString)
				validateField(t, arr.ArrElem.Nested.Properties, "email", FieldTypeString)
			},
		},
		{
			name: "Test Array of array type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Matrix:
      type: array
      items:
        type: array
        items:
          type: integer
          format: int32
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Matrix", FieldTypeArray)

				arr := msg.Properties["Matrix"]
				assert.True(t, arr.IsVariable, "outer array field must be variable-length")
				assert.NotNil(t, arr.ArrElem, "expected outer array element descriptor")

				lvl1Arr := arr.ArrElem
				assert.Equal(t, FieldTypeArray, lvl1Arr.Type, "inner array element type mismatch")
				assert.NotNil(t, lvl1Arr.ArrElem, "expected inner array element descriptor")

				lvl2Arr := lvl1Arr.ArrElem
				assert.Equal(t, FieldTypeInt32, lvl2Arr.Type, "innermost array element type mismatch")
				assert.Nil(t, lvl2Arr.Nested, "innermost array element should not have nested schema for int32 type")
				assert.Nil(t, lvl2Arr.ArrElem, "innermost array element should not have nested array for int32 type")
				assert.False(t, lvl2Arr.IsVariable, "innermost array element should not be variable-length for int32 type")
			},
		},

		// Enum type tests
		// ==================
		{
			name: "Test Enum type of string",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Status:
      type: string
      enum:
        - ACTIVE
        - INACTIVE
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Status", FieldTypeString)

				enumField := msg.Properties["Status"]
				assert.Nil(t, enumField.Nested, "enum field should not have nested schema for string type")
				assert.Nil(t, enumField.ArrElem, "enum field should not have nested array for string type")
				assert.NotNil(t, enumField.EnumValues, "expected enum values to be populated")
				assert.Equal(t, []string{"ACTIVE", "INACTIVE"}, enumField.EnumValues, "enum values mismatch")
			},
		},
		{
			name: "Test Enum type of integer",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Level:
      type: integer
      enum:
        - 1
        - 2
        - 3
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Level", FieldTypeInt32)

				enumField := msg.Properties["Level"]
				assert.Nil(t, enumField.Nested, "enum field should not have nested schema for integer type")
				assert.Nil(t, enumField.ArrElem, "enum field should not have nested array for integer type")
				assert.NotNil(t, enumField.EnumValues, "expected enum values to be populated")
				assert.Equal(t, []string{"1", "2", "3"}, enumField.EnumValues, "enum values mismatch")
			},
		},
		{
			name: "Test Array of Enum type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Protocols:
      type: array
      items:
        type: string
        enum:
          - HTTP
          - HTTPS
          - FTP
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")
				validateField(t, msg.Properties, "Protocols", FieldTypeArray)

				arr := msg.Properties["Protocols"]
				assert.True(t, arr.IsVariable, "array field must be variable-length")
				assert.NotNil(t, arr.ArrElem, "expected array element descriptor")

				enumField := arr.ArrElem
				assert.Equal(t, FieldTypeString, enumField.Type, "array element type mismatch")
				assert.Nil(t, enumField.Nested, "enum field should not have nested schema for string type")
				assert.Nil(t, enumField.ArrElem, "enum field should not have nested array for string type")
				assert.NotNil(t, enumField.EnumValues, "expected enum values to be populated")
				assert.Equal(t, []string{"HTTP", "HTTPS", "FTP"}, enumField.EnumValues, "enum values mismatch")
			},
		},

		// Nullable type tests
		// ==================
		{
			name: "Test Nullable type of string",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Description:
      type: string
      nullable: true
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")

				nullableField, exists := msg.Properties["Description"]
				assert.True(t, exists, "expected field '%s' to exist", "Description")

				assert.Equal(t, "Description", nullableField.Name, "field name mismatch")
				assert.Equal(t, FieldTypeString, nullableField.Type, "field type mismatch")

				assert.True(t, nullableField.Nullable, "field should be marked as nullable")
				assert.True(t, nullableField.IsVariable, "string field should be variable-length")
			},
		},
		{
			name: "Test Nullable type of integer",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Age:
      type: integer
      format: int32
      nullable: true
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")

				nullableField, exists := msg.Properties["Age"]
				assert.True(t, exists, "expected field '%s' to exist", "Age")

				assert.Equal(t, "Age", nullableField.Name, "field name mismatch")
				assert.Equal(t, FieldTypeInt32, nullableField.Type, "field type mismatch")

				assert.True(t, nullableField.Nullable, "field should be marked as nullable")
				assert.True(t, nullableField.IsVariable, "integer field should not be variable-length")
			},
		},
		{
			name: "Test Nullable type of number",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Price:
      type: number
      format: float
      nullable: true
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")

				nullableField, exists := msg.Properties["Price"]
				assert.True(t, exists, "expected field '%s' to exist", "Price")

				assert.Equal(t, "Price", nullableField.Name, "field name mismatch")
				assert.Equal(t, FieldTypeFloat32, nullableField.Type, "field type mismatch")

				assert.True(t, nullableField.Nullable, "field should be marked as nullable")
				assert.True(t, nullableField.IsVariable, "number field should not be variable-length")
			},
		},
		{
			name: "Test Nullable type of boolean",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    IsEnabled:
      type: boolean
      nullable: true
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")

				nullableField, exists := msg.Properties["IsEnabled"]
				assert.True(t, exists, "expected field '%s' to exist", "IsEnabled")

				assert.Equal(t, "IsEnabled", nullableField.Name, "field name mismatch")
				assert.Equal(t, FieldTypeBool, nullableField.Type, "field type mismatch")

				assert.True(t, nullableField.Nullable, "field should be marked as nullable")
				assert.True(t, nullableField.IsVariable, "boolean field should not be variable-length")
			},
		},
		{
			name: "Test Nullable type of object",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Profile:
      type: object
      nullable: true
      properties:
        username:
          type: string
        email:
          type: string
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")

				nullableField, exists := msg.Properties["Profile"]
				assert.True(t, exists, "expected field '%s' to exist", "Profile")

				assert.Equal(t, "Profile", nullableField.Name, "field name mismatch")
				assert.Equal(t, FieldTypeObject, nullableField.Type, "field type mismatch")

				assert.True(t, nullableField.Nullable, "field should be marked as nullable")
				assert.NotNil(t, nullableField.Nested, "nullable object field should have nested schema")
				assert.Len(t, nullableField.Nested.Fields, 2, "expected two fields in nested object")

				validateField(t, nullableField.Nested.Properties, "username", FieldTypeString)
				validateField(t, nullableField.Nested.Properties, "email", FieldTypeString)
			},
		},
		{
			name: "Test Nullable type of array",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Tags:
      type: array
      nullable: true
      items:
        type: string
`),
			expectErr:     false,
			expectedCount: 1,
			validateFunc: func(t *testing.T, schema *Schema) {
				msg := schema.Messages[0]
				assert.Equal(t, "Message", msg.Name, "message name mismatch")
				assert.Len(t, msg.Fields, 1, "expected exactly one field")

				nullableField, exists := msg.Properties["Tags"]
				assert.True(t, exists, "expected field '%s' to exist", "Tags")

				assert.Equal(t, "Tags", nullableField.Name, "field name mismatch")
				assert.Equal(t, FieldTypeArray, nullableField.Type, "field type mismatch")

				assert.True(t, nullableField.Nullable, "field should be marked as nullable")
				assert.NotNil(t, nullableField.ArrElem, "nullable array field should have array element descriptor")
				assert.Equal(t, FieldTypeString, nullableField.ArrElem.Type, "array element type mismatch")
				assert.Nil(t, nullableField.ArrElem.Nested, "array element should not have nested schema for string type")
				assert.Nil(t, nullableField.ArrElem.ArrElem, "array element should not have nested array for string type")
				assert.True(t, nullableField.ArrElem.IsVariable, "array element should be variable-length for string type")
			},
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
