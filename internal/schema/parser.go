package schema

import (
	"context"
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// ParseFile reads and parses an OpenAPI YAML file, extracting message schemas.
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

	// Track an auto-incrementing TypeID since standard OpenAPI doesn't provide one
	var currentTypeID uint16 = 1

	// 2. Iterate over components.schemas
	for schemaName, schemaRef := range doc.Components.Schemas {
		schemaValue := schemaRef.Value
		if schemaValue == nil {
			continue
		}

		message := &Message{
			Name:       schemaName,
			TypeID:     currentTypeID,
			Fields:     make([]*Field, 0),
			Properties: make(map[string]*Field),
		}
		currentTypeID++

		// Loop through the fields/properties of the schema
		for propName, propRef := range schemaValue.Properties {
			propValue := propRef.Value
			if propValue == nil {
				continue
			}

			if propValue.Type == nil || len(*propValue.Type) == 0 {
				return nil, fmt.Errorf("property %q in schema %q has no type defined", propName, schemaName)
			}

			ft, err := resolveFieldType((*propValue.Type)[0], propValue.Format)
			if err != nil {
				return nil, err
			}

			field := &Field{
				Name:        propName,
				Description: propValue.Description,
				Type:        ft,
				Format:      propValue.Format,
				IsVariable:  ft.IsVariable(),
			}

			message.Fields = append(message.Fields, field)
			message.Properties[propName] = field
		}

		resultSchema.Messages = append(resultSchema.Messages, message)
	}

	return resultSchema, nil
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
		if format == "binary" || format == "byte" {
			return FieldTypeBytes, nil
		}
		return FieldTypeString, nil
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
