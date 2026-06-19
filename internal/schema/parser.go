// Copyright (c) 2026 Sachin S. All rights reserved.
// 
// Licensed under the MIT License.
// See LICENSE in the project root.

package schema

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseFile reads and parses an OpenAPI YAML file, extracting message schemas.
// It returns a Schema object containing the parsed messages or an error if parsing fails.
// This function doesn't support array types and will return an error if any schema 
// contains an array property.
func ParseFile(path string) (*Schema, error) {
	ctx := context.Background()
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load/parse YAML file: %w", err)
	}

	// Explicitly validate against the OpenAPI 3.0 specification rules
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("YAML is not a valid OpenAPI spec: %w", err)
	}

	messages := make([]*Message, 0)

	// 2. Iterate over components.schemas
	for schemaName, schemaRef := range doc.Components.Schemas {
		schemaValue := schemaRef.Value
		if schemaValue == nil {
			continue
		}

		id, err := parseIdFromSchema(schemaName, schemaValue)
		if err != nil {
			return nil, err
		}

		fields, fieldList, err := parseFields(schemaValue.Properties)
		if err != nil {
			return nil, fmt.Errorf("failed to parse properties of schema %q: %w", schemaName, err)
		}

		message := &Message{
			Name:       schemaName,
			TypeID:     id,
			Fields:     fieldList,
			Properties: fields,
		}

		messages = append(messages, message)
	}

	sort.Slice(messages, func(i, j int) bool {
		return messages[i].TypeID < messages[j].TypeID
	})

	return &Schema{Messages: messages}, nil
}

func parseIdFromSchema(schemaName string, schema *openapi3.Schema) (uint16, error) {
	idVal, found := schema.Extensions["x-message-id"]
	if !found {
		return 0, fmt.Errorf("schema %s is missing the required x-message-id", schemaName)
	}

	id, ok := idVal.(float64)
	if !ok || id < 0 || id > 65535 {
		return 0, fmt.Errorf("schema %s has an invalid x-message-id; it must be a valid number b/w 0-65535", schemaName)
	}

	return uint16(id), nil
}

func parseFields(properties openapi3.Schemas) (map[string]*Field, []*Field, error) {
	fields := make(map[string]*Field)
	fieldList := make([]*Field, 0)

	for propName, propRef := range properties {
		propValue := propRef.Value
		if propValue == nil {
			continue
		}

		if propValue.Type == nil || len(*propValue.Type) == 0 {
			return nil, nil, fmt.Errorf("property %q has no type defined", propName)
		}

		if propValue.Type.Includes("array") {
			return nil, nil, fmt.Errorf("property %q has unsupported type %q", propName, *propValue.Type)
		}

		if propValue.Type.Includes("object") {
			nestedFields, nestedFieldList, err := parseFields(propValue.Properties)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to parse nested properties of %q: %w", propName, err)
			}

			field := &Field{
				Name:        propName,
				Description: propValue.Description,
				Type:        FieldTypeObject,
				Format:      propValue.Format,
				IsVariable:  false, // Objects are not variable, but they can contain variable fields
				Nested: &Message{
					Name:       propName,
					Fields:     nestedFieldList,
					Properties: nestedFields,
				},
			}

			fields[propName] = field
			fieldList = append(fieldList, field)
			continue
		}

		ft, err := resolveFieldType((*propValue.Type)[0], propValue.Format)
		if err != nil {
			return nil, nil, err
		}

		field := &Field{
			Name:        propName,
			Description: propValue.Description,
			Type:        ft,
			Format:      propValue.Format,
			IsVariable:  ft.IsVariable(),
		}

		fields[propName] = field
		fieldList = append(fieldList, field)
	}

	return fields, fieldList, nil
}

func resolveFieldType(typeName, format string) (FieldType, error) {
	switch typeName {
	case "integer":
		return resolveIntegerFormat(format)
	case "number":
		return resolveNumberFormat(format)
	case "boolean":
		return FieldTypeBool, nil
	case "string":
		return resolveStringFormat(format)
	case "":
		return 0, fmt.Errorf("field type is empty — every field must have a concrete type")
	default:
		return 0, fmt.Errorf("unsupported type %q — generic/untyped fields are rejected for safety", typeName)
	}
}

func resolveIntegerFormat(format string) (FieldType, error) {
	switch strings.ToLower(format) {
	case "int8":
		return FieldTypeInt8, nil
	case "uint8":
		return FieldTypeUint8, nil
	case "int16":
		return FieldTypeInt16, nil
	case "uint16":
		return FieldTypeUint16, nil
	case "int32", "":
		return FieldTypeInt32, nil
	case "uint32":
		return FieldTypeUint32, nil
	case "int64":
		return FieldTypeInt64, nil
	case "uint64":
		return FieldTypeUint64, nil
	default:
		return 0, fmt.Errorf("unsupported integer format %q", format)
	}
}

func resolveNumberFormat(format string) (FieldType, error) {
	switch strings.ToLower(format) {
	case "float", "":
		return FieldTypeFloat32, nil
	case "double":
		return FieldTypeFloat64, nil
	default:
		return 0, fmt.Errorf("unsupported number format %q", format)
	}
}

func resolveStringFormat(format string) (FieldType, error) {
	switch strings.ToLower(format) {
	case "byte", "binary":
		return FieldTypeBytes, nil
	default:
		return FieldTypeString, nil
	}
}
