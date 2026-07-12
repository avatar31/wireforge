// Copyright (c) 2026 Sachin S. All rights reserved.
//
// Licensed under the MIT License.
// See LICENSE in the project root.

package schema

// Schema represents a parsed collection of message types from an OpenAPI spec.
type Schema struct {
	Messages []*Message
}

// Message represents a single schema in the OpenAPI spec and
// its corresponding message/struct type in the generated code.
type Message struct {
	Name       string
	TypeID     uint16
	Fields     []*Field
	Properties map[string]*Field
}

// Field represents a single field within a message.
//
// The Type discriminates how the field is encoded on the wire. Composite types
// carry additional descriptors:
//
//   - FieldTypeObject: Nested points at the embedded message definition.
//   - FieldTypeArray:  Elem describes the element encoding (which may itself be
//     an object, array, enum, or primitive).
//   - Enum fields:     EnumValues lists the allowed constant values; the base
//     Type remains the underlying primitive (string or an integer type).
//
// TODO: Revisit here
// Nullable marks the field as optional. A nullable field is always length
// prefixed on the wire and uses a sentinel prefix (0xFFFFFFFF) to encode the
// null/absent state, independent of the underlying type.
type Field struct {
	Name            string
	Description     string
	Type            FieldType
	Format          string
	NestedMessageId uint16 // object element id (FieldTypeObject)
	ArrElem         *Field // array element descriptor (FieldTypeArray)
	EnumValues      []string // allowed values for enum fields (rendered as constants)
	IsVariable      bool
}

// FieldType enumerates the supported concrete field types.
type FieldType int

const (
	FieldTypeUint64 FieldType = iota
	FieldTypeInt64
	FieldTypeFloat64
	FieldTypeUint32
	FieldTypeInt32
	FieldTypeFloat32
	FieldTypeString // uint32 length prefix in fixed header
	FieldTypeBytes  // uint32 length prefix in fixed header
	FieldTypeObject // uint32 length prefix in fixed header
	FieldTypeArray  // uint32 length prefix in fixed header
	FieldTypeUint16
	FieldTypeInt16
	FieldTypeUint8
	FieldTypeInt8
	FieldTypeBool
)

const (
	// Supported number of field types
	NumOfFieldTypes = 15
)

// Size returns the fixed byte size of the field type.
func (ft FieldType) Size() int {
	switch ft {
	case FieldTypeUint8, FieldTypeInt8, FieldTypeBool:
		return 1
	case FieldTypeUint16, FieldTypeInt16:
		return 2
	case FieldTypeUint32, FieldTypeInt32, FieldTypeFloat32:
		return 4
	case FieldTypeUint64, FieldTypeInt64, FieldTypeFloat64:
		return 8
	case FieldTypeString, FieldTypeBytes, FieldTypeObject, FieldTypeArray:
		return 4 // uint32 length prefix in fixed header
	default:
		return 0
	}
}

// Alignment returns the natural alignment requirement for this type.
func (ft FieldType) Alignment() int {
	switch ft {
	case FieldTypeUint8, FieldTypeInt8, FieldTypeBool:
		return 1
	case FieldTypeUint16, FieldTypeInt16:
		return 2
	case FieldTypeUint32, FieldTypeInt32, FieldTypeFloat32:
		return 4
	case FieldTypeUint64, FieldTypeInt64, FieldTypeFloat64:
		return 8
	case FieldTypeString, FieldTypeBytes, FieldTypeObject, FieldTypeArray:
		return 4
	default:
		return 1
	}
}

// GoType returns the Go type string for this field type.
func (ft FieldType) GoType() string {
	switch ft {
	case FieldTypeUint8:
		return "uint8"
	case FieldTypeUint16:
		return "uint16"
	case FieldTypeUint32:
		return "uint32"
	case FieldTypeUint64:
		return "uint64"
	case FieldTypeInt8:
		return "int8"
	case FieldTypeInt16:
		return "int16"
	case FieldTypeInt32:
		return "int32"
	case FieldTypeInt64:
		return "int64"
	case FieldTypeFloat32:
		return "float32"
	case FieldTypeFloat64:
		return "float64"
	case FieldTypeBool:
		return "bool"
	case FieldTypeString:
		return "string"
	case FieldTypeBytes:
		return "[]byte"
	case FieldTypeObject:
		return "struct"
	case FieldTypeArray:
		return "[]any"
	default:
		return "uint8"
	}
}

// CType returns the C type string for this field type.
func (ft FieldType) CType() string {
	switch ft {
	case FieldTypeUint8:
		return "uint8_t"
	case FieldTypeUint16:
		return "uint16_t"
	case FieldTypeUint32:
		return "uint32_t"
	case FieldTypeUint64:
		return "uint64_t"
	case FieldTypeInt8:
		return "int8_t"
	case FieldTypeInt16:
		return "int16_t"
	case FieldTypeInt32:
		return "int32_t"
	case FieldTypeInt64:
		return "int64_t"
	case FieldTypeFloat32:
		return "float"
	case FieldTypeFloat64:
		return "double"
	case FieldTypeBool:
		return "uint8_t"
	case FieldTypeString:
		return "char*"
	case FieldTypeBytes:
		return "uint8_t*"
	default:
		return "uint8_t"
	}
}

// IsVariable reports whether the type is encoded with a length prefix in the
// fixed header and a payload in the dynamic section. Objects and arrays are
// always variable-length; scalars are fixed-width.
func (ft FieldType) IsVariable() bool {
	return ft == FieldTypeString ||
		ft == FieldTypeBytes ||
		ft == FieldTypeObject ||
		ft == FieldTypeArray
}
