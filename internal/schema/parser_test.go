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
      $ref: '#/components/schemas/Request'
Request:
  type: object
  x-message-id: 2
  properties:
    key:
      type: string
`),
			expectErr:     false,
			expectedCount: 2,
			validateFunc: func(t *testing.T, schema *Schema) {
				s1 := schema.Messages[0]
				assert.Equal(t, "Message", s1.Name, "schema name mismatch")
				assert.Len(t, s1.Fields, 1, "expected exactly one field")
				assert.Len(t, s1.Properties, 1, "expected exactly one property")
				validateField(t, s1.Properties, "Request", FieldTypeObject)

				nestedField := s1.Properties["Request"]
				assert.NotNil(t, nestedField, "expected nested field for 'Request'")
				assert.True(t, nestedField.IsVariable, "expected nested field should be variable-length")
				assert.Equal(t, nestedField.NestedMessageId, uint16(2), "nested message ID mismatch")

				s2 := schema.Messages[1]
				assert.Equal(t, "Request", s2.Name, "schema name mismatch")
				assert.Len(t, s2.Fields, 1, "expected exactly one field")
				assert.Len(t, s2.Properties, 1, "expected exactly one property")
				validateField(t, s2.Properties, "key", FieldTypeString)
			},
		},
		{
			name: "Test Inline Object type",
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
			expectErr: true,
		},
		{
			name: "Test Object type with multiple levels of nesting",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Request:
      $ref: '#/components/schemas/Request'
Request:
  type: object
  x-message-id: 2
  properties:
    key:
      type: string
    metadata:
      $ref: '#/components/schemas/Metadata'
Metadata:
  type: object
  x-message-id: 3
  properties:
    timestamp:
      type: string
      format: date-time
`),
			expectErr:     false,
			expectedCount: 3,
			validateFunc: func(t *testing.T, schema *Schema) {
				s1 := schema.Messages[0]
				assert.Equal(t, "Message", s1.Name, "schema name mismatch")
				assert.Len(t, s1.Fields, 1, "expected exactly one field")
				assert.Len(t, s1.Properties, 1, "expected exactly one property")
				validateField(t, s1.Properties, "Request", FieldTypeObject)

				s1NestedField := s1.Properties["Request"]
				assert.NotNil(t, s1NestedField, "expected nested field for 'Request'")
				assert.True(t, s1NestedField.IsVariable, "expected nested field should be variable-length")
				assert.Equal(t, s1NestedField.NestedMessageId, uint16(2), "nested message ID mismatch")

				s2 := schema.Messages[1]
				assert.Equal(t, "Request", s2.Name, "schema name mismatch")
				assert.Len(t, s2.Fields, 2, "expected exactly 2 field")
				assert.Len(t, s2.Properties, 2, "expected exactly 2 property")
				validateField(t, s2.Properties, "key", FieldTypeString)
				validateField(t, s2.Properties, "metadata", FieldTypeObject)

				s2NestedField := s2.Properties["metadata"]
				assert.NotNil(t, s2NestedField, "expected nested field for 'metadata'")
				assert.True(t, s2NestedField.IsVariable, "expected nested field should be variable-length")
				assert.Equal(t, s2NestedField.NestedMessageId, uint16(3), "nested message ID mismatch")

				s3 := schema.Messages[2]
				assert.Equal(t, "Metadata", s3.Name, "schema name mismatch")
				assert.Len(t, s3.Fields, 1, "expected exactly one field")
				assert.Len(t, s3.Properties, 1, "expected exactly one property")
				validateField(t, s3.Properties, "timestamp", FieldTypeString)
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
        $ref: '#/components/schemas/Profile'
Profile:
  type: object
  x-message-id: 2
  properties:
    username:
      type: string
    email:
      type: string
`),
			expectErr:     false,
			expectedCount: 2,
			validateFunc: func(t *testing.T, schema *Schema) {
				s1 := schema.Messages[0]
				assert.Equal(t, "Message", s1.Name, "message name mismatch")
				assert.Len(t, s1.Fields, 1, "expected exactly one field")
				validateField(t, s1.Properties, "Users", FieldTypeArray)

				usersArr := s1.Properties["Users"]
				assert.True(t, usersArr.IsVariable, "array field must be variable-length")
				assert.NotNil(t, usersArr.ArrElem, "expected array element descriptor")

				assert.Equal(t, FieldTypeObject, usersArr.ArrElem.Type, "array element type mismatch")
				assert.Nil(t, usersArr.ArrElem.ArrElem, "array element should not have nested array for object type")
				assert.True(t, usersArr.ArrElem.IsVariable, "array element should be variable-length for object type")
				assert.Equal(t, uint16(2), usersArr.ArrElem.NestedMessageId, "nested message ID mismatch for array element")

				s2 := schema.Messages[1]
				assert.Equal(t, "Profile", s2.Name, "schema name mismatch")
				assert.Len(t, s2.Fields, 2, "expected exactly two fields")
				assert.Len(t, s2.Properties, 2, "expected exactly two properties")
				validateField(t, s2.Properties, "username", FieldTypeString)
				validateField(t, s2.Properties, "email", FieldTypeString)
			},
		},
		{
			name: "Test 2D array type",
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
				assert.Nil(t, lvl2Arr.ArrElem, "innermost array element should not have nested array for int32 type")
				assert.False(t, lvl2Arr.IsVariable, "innermost array element should not be variable-length for int32 type")
			},
		},
		{
			name: "Test 3D array type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    Cube:
      type: array
      items:
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
				validateField(t, msg.Properties, "Cube", FieldTypeArray)

				lvl1Arr := msg.Properties["Cube"]
				assert.True(t, lvl1Arr.IsVariable, "outer array field must be variable-length")
				assert.NotNil(t, lvl1Arr.ArrElem, "expected outer array element descriptor")

				lvl2Arr := lvl1Arr.ArrElem
				assert.Equal(t, FieldTypeArray, lvl2Arr.Type, "second-level array element type mismatch")
				assert.NotNil(t, lvl2Arr.ArrElem, "expected second-level array element descriptor")

				lvl3Arr := lvl2Arr.ArrElem
				assert.Equal(t, FieldTypeArray, lvl3Arr.Type, "third-level array element type mismatch")
				assert.NotNil(t, lvl3Arr.ArrElem, "expected third-level array element descriptor")

				lvl4Elem := lvl3Arr.ArrElem
				assert.Equal(t, FieldTypeInt32, lvl4Elem.Type, "innermost array element type mismatch")
				assert.Nil(t, lvl4Elem.ArrElem, "innermost array element should not have nested array for int32 type")
				assert.False(t, lvl4Elem.IsVariable, "innermost array element should not be variable-length for int32 type")
			},
		},
		{
			name: "Test 4D array type",
			yamlContent: wrapInOpenAPIBoilerplate(`
Message:
  type: object
  x-message-id: 1
  properties:
    HyperCube:
      type: array
      items:
        type: array
        items:
          type: array
          items:
            type: array
            items:
              type: integer
              format: int32
`),
			expectErr: true,
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
			expectErr: true,
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
				assert.Nil(t, enumField.ArrElem, "enum field should not have nested array for string type")
				assert.NotNil(t, enumField.EnumValues, "expected enum values to be populated")
				assert.Equal(t, []string{"HTTP", "HTTPS", "FTP"}, enumField.EnumValues, "enum values mismatch")
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a unique temporary directory for this specific subtest
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "openapi_test.yaml")

			// Write fake file payload
			err := os.WriteFile(tmpFile, []byte(strings.TrimSpace(tc.yamlContent)), 0644)
			if err != nil {
				t.Fatalf("failed to create temporary test file: %v", err)
			}

			// Execute target code
			result, err := ParseFile(tmpFile)

			// Assert error expectations
			if tc.expectErr {
				assert.NotNil(t, err)
				return
			}

			assert.Nil(t, err)
			assert.Len(t, result.Messages, tc.expectedCount, "expected message count mismatch")

			// Execute functional assertions if it passed
			if tc.validateFunc != nil {
				tc.validateFunc(t, result)
			}
		})
	}
}
