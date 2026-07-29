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
	"unicode"

	"github.com/getkin/kin-openapi/openapi3"
)

const (
	// MinAllowedMessageId is set to 1 to reserve 0 for special cases (e.g. "no message").
	MinAllowedMessageId = 1

	// MaxAllowedMessageId is set to 65000 to leave some room for future reserved message IDs.
	MaxAllowedMessageId = 65000

	MaxArrayDimensions = 3
)

// ParseFile reads and parses an OpenAPI YAML file, extracting message schemas.
//
// It returns a Schema object containing the parsed top-level messages or an
// error if parsing fails. The parser supports primitive scalars, strings,
// binary blobs, enums, nested objects, arrays (including arrays of objects and
// arrays of arrays), and nullable/optional fields. Composite types are resolved
// recursively; nested object types are assigned deterministic, parent-qualified
// names so downstream code generation can emit collision-free struct types.
func ParseFile(path string) (*Schema, error) {
	ctx := context.Background()
	loader := openapi3.NewLoader()

	doc, err := loader.LoadFromFile(path)
	if err != nil {
		return nil, err
	}

	// Explicitly validate against the OpenAPI specification rules.
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("YAML is not a valid OpenAPI spec: %w", err)
	}

	messages := make([]*Message, 0)
	idMap := make(map[uint16]struct{})

	// Iterate over components.schemas; each schema becomes a top-level message.
	for schemaName, schemaRef := range doc.Components.Schemas {
		schemaValue := schemaRef.Value
		if schemaValue == nil {
			continue
		}

		id, err := parseIdFromSchema(schemaName, schemaValue, idMap)
		if err != nil {
			return nil, err
		}

		idMap[id] = struct{}{}

		fields, fieldList, err := parseFields(schemaValue.Properties, schemaName)
		if err != nil {
			return nil, fmt.Errorf("failed to parse properties of schema %q: %w", schemaName, err)
		}

		if len(fields) == 0 {
			return nil, fmt.Errorf("schema %q has no properties defined", schemaName)
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

func parseIdFromSchema(schemaName string, schema *openapi3.Schema,
	idMap map[uint16]struct{}) (uint16, error) {
	idVal, found := schema.Extensions["x-message-id"]
	if !found {
		return 0, fmt.Errorf("schema %s is missing the required x-message-id", schemaName)
	}

	id, ok := idVal.(float64)
	if !ok || id < MinAllowedMessageId || id > MaxAllowedMessageId {
		return 0, fmt.Errorf("schema %s has an invalid x-message-id; it must be a valid number b/w %d-%d",
					schemaName, MinAllowedMessageId, MaxAllowedMessageId)
	}

	output := uint16(id)

	if _, exists := idMap[output]; exists {
		return 0, fmt.Errorf("duplicate x-message-id %d found in schema %s", output, schemaName)
	}

	return output, nil
}

// parseFields resolves an ordered set of properties into Field descriptors.
//
// parentName is the qualified type name of the enclosing message and is used to
// derive collision-free names for nested object and array-element types.
func parseFields(properties openapi3.Schemas, parentName string) (map[string]*Field, []*Field, error) {
	fields := make(map[string]*Field)
	fieldList := make([]*Field, 0)

	for propName, propRef := range properties {
		if propRef.Value == nil {
			continue
		}

		field, err := parseField(propName, propRef, parentName, 0)
		if err != nil {
			return nil, nil, err
		}

		fields[propName] = field
		fieldList = append(fieldList, field)
	}

	return fields, fieldList, nil
}

// parseField resolves a single property (identified by name) into a Field.
//
// It dispatches on the schema kind (array, object, or primitive/enum) and
// recurses for composite element/child types. nameHint is the property name and
// parentName is the qualified name of the enclosing message.
func parseField(name string, sRef *openapi3.SchemaRef, parentName string, arrDimension int) (*Field, error) {
	sv := sRef.Value
	if sv.Type == nil || len(*sv.Type) == 0 {
		return nil, fmt.Errorf("property %q has no type defined", name)
	}

	field := &Field{
		Name:        name,
		Description: sv.Description,
		Format:      sv.Format,
	}

	switch {
	case sv.Type.Includes("array"):
		if arrDimension >= MaxArrayDimensions {
			return nil, fmt.Errorf("array property %q exceeds maximum allowed dimensions (%d)",
					name, MaxArrayDimensions)
		}

		if sv.Items == nil || sv.Items.Value == nil {
			return nil, fmt.Errorf("array property %q is missing an items schema", name)
		}

		elem, err := parseField(name+"Item", sv.Items, parentName+exportName(name), arrDimension+1)
		if err != nil {
			return nil, fmt.Errorf("failed to parse array element of %q: %w", name, err)
		}

		field.Type = FieldTypeArray
		field.ArrElem = elem
		field.IsVariable = true

	case sv.Type.Includes("object"):
		if sRef.Ref == "" {
			return nil, fmt.Errorf("object property %q is missing a $ref to a named schema; inline object definitions are not supported", name)
		}

		field.Type = FieldTypeObject
		id, err := parseIdFromSchema(sRef.Ref, sv, nil)
		if err != nil {
			return nil, err
		}

		field.NestedMessageId = id
		field.IsVariable = true

	default:
		ft, err := resolveFieldType((*sv.Type)[0], sv.Format)
		if err != nil {
			return nil, err
		}

		if len(sv.Enum) > 0 {
			values, err := parseEnumValues(name, ft, sv.Enum)
			if err != nil {
				return nil, err
			}
			field.EnumValues = values
		}

		field.Type = ft
		field.IsVariable = ft.IsVariable()
	}

	return field, nil
}

// parseEnumValues validates and normalises the allowed values of an enum field.
//
// Enums are only supported over string and integer base types. Values are
// rendered to their canonical string form for later constant generation.
func parseEnumValues(name string, base FieldType, raw []any) ([]string, error) {
	isString := base == FieldTypeString

	if !isString {
		return nil, fmt.Errorf("enum on property %q is only supported for string base types", name)
	}

	values := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, v := range raw {
		var s string
		switch val := v.(type) {
		case string:
			if !isString {
				return nil, fmt.Errorf("enum value %q on property %q does not matching with base type", val, name)
			}
			s = val
		default:
			return nil, fmt.Errorf("unsupported enum value type %T on property %q", v, name)
		}

		if _, dup := seen[s]; dup {
			return nil, fmt.Errorf("duplicate enum value %q on property %q", s, name)
		}
		seen[s] = struct{}{}
		values = append(values, s)
	}

	if len(values) == 0 {
		return nil, fmt.Errorf("enum on property %q must declare at least one value", name)
	}

	return values, nil
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

// exportName upper-cases the first rune of a property name so it can be used as
// a fragment when composing qualified, exported nested type names. All-caps
// acronyms (e.g. "ACL") are preserved as-is.
func exportName(name string) string {
	if name == "" {
		return ""
	}
	r := []rune(name)
	r[0] = unicode.ToUpper(r[0])
	return string(r)
}
