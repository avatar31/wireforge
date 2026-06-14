package schema

// Schema represents a parsed collection of message types from an OpenAPI spec.
type Schema struct {
	Messages []*Message
}

// Message represents a single struct/message type.
type Message struct {
	Name       string
	Fields     []*Field
	Properties map[string]*Field
}

// Field represents a single field within a message.
type Field struct {
	Name        string
	Description string
	Type        FieldType
	Format      string
	Nested      *Message
	IsVariable  bool
}

// FieldType enumerates the supported concrete field types.
type FieldType int

const (
	FieldTypeUint8 FieldType = iota
	FieldTypeUint16
	FieldTypeUint32
	FieldTypeUint64
	FieldTypeInt8
	FieldTypeInt16
	FieldTypeInt32
	FieldTypeInt64
	FieldTypeFloat32
	FieldTypeFloat64
	FieldTypeBool
	FieldTypeString
	FieldTypeBytes
	FieldTypeObject
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
	case FieldTypeString, FieldTypeBytes:
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
	case FieldTypeString, FieldTypeBytes:
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
	case FieldTypeString, FieldTypeBytes:
		return ""
	default:
		return "uint8_t"
	}
}

// IsVariable returns true if the field type has variable length.
func (ft FieldType) IsVariable() bool {
	return ft == FieldTypeString || ft == FieldTypeBytes
}
