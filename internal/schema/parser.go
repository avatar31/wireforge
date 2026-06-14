package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseFile reads and parses an OpenAPI YAML file, extracting message schemas.
// Unsupported Types:
// - array
// - null
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

	resultSchema := &Schema{
		Messages: make([]*Message, 0),
	}

	// 2. Iterate over components.schemas
	for schemaName, schemaRef := range doc.Components.Schemas {
		schemaValue := schemaRef.Value
		if schemaValue == nil {
			continue
		}

		fields, fieldList, err := parseFields(schemaValue.Properties)
		if err != nil {
			return nil, fmt.Errorf("failed to parse properties of schema %q: %w", schemaName, err)
		}

		message := &Message{
			Name:       schemaName,
			Fields:     fieldList,
			Properties: fields,
		}

		resultSchema.Messages = append(resultSchema.Messages, message)
	}

	return resultSchema, nil
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
