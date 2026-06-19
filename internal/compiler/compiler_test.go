// Copyright (c) 2026 Sachin S. All rights reserved.
// 
// Licensed under the MIT License.
// See LICENSE in the project root.

package compiler

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/avatar31/wireforge/internal/schema"
)

var (
	uint8Field    = &schema.Field{Name: "uint8Field", Type: schema.FieldTypeUint8, Format: "uint8", IsVariable: false}
	uint16Field   = &schema.Field{Name: "uint16Field", Type: schema.FieldTypeUint16, Format: "uint16", IsVariable: false}
	uint32Field   = &schema.Field{Name: "uint32Field", Type: schema.FieldTypeUint32, Format: "uint32", IsVariable: false}
	uint64Field   = &schema.Field{Name: "uint64Field", Type: schema.FieldTypeUint64, Format: "uint64", IsVariable: false}
	int8Field     = &schema.Field{Name: "int8Field", Type: schema.FieldTypeInt8, Format: "int8", IsVariable: false}
	int16Field    = &schema.Field{Name: "int16Field", Type: schema.FieldTypeInt16, Format: "int16", IsVariable: false}
	int32Field    = &schema.Field{Name: "int32Field", Type: schema.FieldTypeInt32, Format: "int32", IsVariable: false}
	int64Field    = &schema.Field{Name: "int64Field", Type: schema.FieldTypeInt64, Format: "int64", IsVariable: false}
	float32Field  = &schema.Field{Name: "float32Field", Type: schema.FieldTypeFloat32, Format: "float32", IsVariable: false}
	float64Field  = &schema.Field{Name: "float64Field", Type: schema.FieldTypeFloat64, Format: "float64", IsVariable: false}
	stringField   = &schema.Field{Name: "stringField", Type: schema.FieldTypeString, IsVariable: true}
	boolField     = &schema.Field{Name: "boolField", Type: schema.FieldTypeBool, IsVariable: false}
	bytesField    = &schema.Field{Name: "bytesField", Type: schema.FieldTypeBytes, Format: "bytes", IsVariable: true}
	sampleMessage = &schema.Message{
		Name:       "SampleMessage",
		Fields:     []*schema.Field{stringField, boolField},
		Properties: map[string]*schema.Field{"stringField": stringField, "boolField": boolField},
	}
	objectField = &schema.Field{Name: "objectField", Type: schema.FieldTypeObject, IsVariable: true, Nested: sampleMessage}
)

func TestCompile(t *testing.T) {
	t.Run("CompileMessage", func(t *testing.T) {
		schema := &schema.Schema{
			Messages: []*schema.Message{sampleMessage},
		}

		compiledSchema := Compile(schema, "test")
		assert.NotNil(t, compiledSchema)
		assert.Equal(t, "test", compiledSchema.PackageName)
		assert.Equal(t, len(schema.Messages), len(compiledSchema.Messages))
	})
}

func TestCompileMessage(t *testing.T) {
	tests := []struct {
		name         string
		message      *schema.Message
		expectedCMsg *CompiledMessage
	}{
		{
			name:    "Simple message with string and bool fields",
			message: sampleMessage,
			expectedCMsg: &CompiledMessage{
				Name: "SampleMessage",
				Fields: []*CompiledField{
					{Name: stringField.Name, GoName: "StringField", CName: "string_field", Type: stringField.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
					{Name: boolField.Name, GoName: "BoolField", CName: "bool_field", Type: boolField.Type, Offset: 4, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: stringField.Name, GoName: "StringField", CName: "string_field", Type: stringField.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
					{Name: boolField.Name, GoName: "BoolField", CName: "bool_field", Type: boolField.Type, Offset: 4, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
				},
				VariableFields: []*CompiledField{
					{Name: stringField.Name, GoName: "StringField", CName: "string_field", Type: stringField.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
				},
				TotalFixedSize:  8, // 4 bytes for string length + 1 byte for bool + 3 bytes padding
				StructAlignment: 4,
				PaddingBlocks:   1, // 3 bytes of padding after boolField
			},
		},
		{
			name: "Message with all 1 byte fields",
			message: &schema.Message{
				Name:       "AllOneByteFields",
				Fields:     []*schema.Field{uint8Field, int8Field, boolField},
				Properties: map[string]*schema.Field{"uint8Field": uint8Field, "int8Field": int8Field, "boolField": boolField},
			},
			expectedCMsg: &CompiledMessage{
				Name: "AllOneByteFields",
				Fields: []*CompiledField{
					{Name: uint8Field.Name, GoName: "Uint8Field", CName: "uint8_field", Type: uint8Field.Type, Offset: 0, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
					{Name: int8Field.Name, GoName: "Int8Field", CName: "int8_field", Type: int8Field.Type, Offset: 1, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
					{Name: boolField.Name, GoName: "BoolField", CName: "bool_field", Type: boolField.Type, Offset: 2, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: uint8Field.Name, GoName: "Uint8Field", CName: "uint8_field", Type: uint8Field.Type, Offset: 0, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
					{Name: int8Field.Name, GoName: "Int8Field", CName: "int8_field", Type: int8Field.Type, Offset: 1, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
					{Name: boolField.Name, GoName: "BoolField", CName: "bool_field", Type: boolField.Type, Offset: 2, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
				},
				VariableFields:  []*CompiledField{},
				TotalFixedSize:  4, // 1 byte for uint8 + 1 byte for int8 + 1 byte for bool + 1 byte padding
				StructAlignment: 4,
				PaddingBlocks:   1, // 1 byte of padding after boolField to align to 4 bytes
			},
		},
		{
			name: "Message with all 2 byte fields",
			message: &schema.Message{
				Name:       "AllTwoByteFields",
				Fields:     []*schema.Field{uint16Field, int16Field},
				Properties: map[string]*schema.Field{"uint16Field": uint16Field, "int16Field": int16Field},
			},
			expectedCMsg: &CompiledMessage{
				Name: "AllTwoByteFields",
				Fields: []*CompiledField{
					{Name: uint16Field.Name, GoName: "Uint16Field", CName: "uint16_field", Type: uint16Field.Type, Offset: 0, Size: 2, Alignment: 2, IsVariable: false, PaddingBefore: 0},
					{Name: int16Field.Name, GoName: "Int16Field", CName: "int16_field", Type: int16Field.Type, Offset: 2, Size: 2, Alignment: 2, IsVariable: false, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: uint16Field.Name, GoName: "Uint16Field", CName: "uint16_field", Type: uint16Field.Type, Offset: 0, Size: 2, Alignment: 2, IsVariable: false, PaddingBefore: 0},
					{Name: int16Field.Name, GoName: "Int16Field", CName: "int16_field", Type: int16Field.Type, Offset: 2, Size: 2, Alignment: 2, IsVariable: false, PaddingBefore: 0},
				},
				VariableFields:  []*CompiledField{},
				TotalFixedSize:  4, // 2 bytes for uint16 + 2 bytes for int16
				StructAlignment: 4,
				PaddingBlocks:   0, // no padding needed
			},
		},
		{
			name: "Message with all 4 byte fields",
			message: &schema.Message{
				Name:       "AllFourByteFields",
				Fields:     []*schema.Field{uint32Field, int32Field, float32Field},
				Properties: map[string]*schema.Field{"uint32Field": uint32Field, "int32Field": int32Field, "float32Field": float32Field},
			},
			expectedCMsg: &CompiledMessage{
				Name: "AllFourByteFields",
				Fields: []*CompiledField{
					{Name: uint32Field.Name, GoName: "Uint32Field", CName: "uint32_field", Type: uint32Field.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
					{Name: int32Field.Name, GoName: "Int32Field", CName: "int32_field", Type: int32Field.Type, Offset: 4, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
					{Name: float32Field.Name, GoName: "Float32Field", CName: "float32_field", Type: float32Field.Type, Offset: 8, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: uint32Field.Name, GoName: "Uint32Field", CName: "uint32_field", Type: uint32Field.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
					{Name: int32Field.Name, GoName: "Int32Field", CName: "int32_field", Type: int32Field.Type, Offset: 4, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
					{Name: float32Field.Name, GoName: "Float32Field", CName: "float32_field", Type: float32Field.Type, Offset: 8, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
				},
				VariableFields:  []*CompiledField{},
				TotalFixedSize:  12, // 4 bytes for uint32 + 4 bytes for int32 + 4 bytes for float32
				StructAlignment: 4,
				PaddingBlocks:   0, // no padding needed
			},
		},
		{
			name: "Message with all 8 byte fields",
			message: &schema.Message{
				Name:       "AllEightByteFields",
				Fields:     []*schema.Field{uint64Field, int64Field, float64Field},
				Properties: map[string]*schema.Field{"uint64Field": uint64Field, "int64Field": int64Field, "float64Field": float64Field},
			},
			expectedCMsg: &CompiledMessage{
				Name: "AllEightByteFields",
				Fields: []*CompiledField{
					{Name: uint64Field.Name, GoName: "Uint64Field", CName: "uint64_field", Type: uint64Field.Type, Offset: 0, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
					{Name: int64Field.Name, GoName: "Int64Field", CName: "int64_field", Type: int64Field.Type, Offset: 8, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
					{Name: float64Field.Name, GoName: "Float64Field", CName: "float64_field", Type: float64Field.Type, Offset: 16, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: uint64Field.Name, GoName: "Uint64Field", CName: "uint64_field", Type: uint64Field.Type, Offset: 0, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
					{Name: int64Field.Name, GoName: "Int64Field", CName: "int64_field", Type: int64Field.Type, Offset: 8, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
					{Name: float64Field.Name, GoName: "Float64Field", CName: "float64_field", Type: float64Field.Type, Offset: 16, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
				},
				VariableFields:  []*CompiledField{},
				TotalFixedSize:  24, // 8 bytes for uint64 + 8 bytes for int64 + 8 bytes for float64
				StructAlignment: 8,
				PaddingBlocks:   0, // no padding needed
			},
		},
		{
			name: "Message with all integer field",
			message: &schema.Message{
				Name:       "AllIntegerFields",
				Fields:     []*schema.Field{int8Field, int16Field, int32Field, int64Field},
				Properties: map[string]*schema.Field{"int8Field": int8Field, "int16Field": int16Field, "int32Field": int32Field, "int64Field": int64Field},
			},
			expectedCMsg: &CompiledMessage{
				Name: "AllIntegerFields",
				Fields: []*CompiledField{
					{Name: int64Field.Name, GoName: "Int64Field", CName: "int64_field", Type: int64Field.Type, Offset: 0, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
					{Name: int32Field.Name, GoName: "Int32Field", CName: "int32_field", Type: int32Field.Type, Offset: 8, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
					{Name: int16Field.Name, GoName: "Int16Field", CName: "int16_field", Type: int16Field.Type, Offset: 12, Size: 2, Alignment: 2, IsVariable: false, PaddingBefore: 0},
					{Name: int8Field.Name, GoName: "Int8Field", CName: "int8_field", Type: int8Field.Type, Offset: 14, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: int64Field.Name, GoName: "Int64Field", CName: "int64_field", Type: int64Field.Type, Offset: 0, Size: 8, Alignment: 8, IsVariable: false, PaddingBefore: 0},
					{Name: int32Field.Name, GoName: "Int32Field", CName: "int32_field", Type: int32Field.Type, Offset: 8, Size: 4, Alignment: 4, IsVariable: false, PaddingBefore: 0},
					{Name: int16Field.Name, GoName: "Int16Field", CName: "int16_field", Type: int16Field.Type, Offset: 12, Size: 2, Alignment: 2, IsVariable: false, PaddingBefore: 0},
					{Name: int8Field.Name, GoName: "Int8Field", CName: "int8_field", Type: int8Field.Type, Offset: 14, Size: 1, Alignment: 1, IsVariable: false, PaddingBefore: 0},
				},
				VariableFields:  []*CompiledField{},
				TotalFixedSize:  16, // 1 byte for int8 + 1 byte padding + 2 bytes for int16 + 4 bytes for int32 + 8 bytes for int64
				StructAlignment: 8,
				PaddingBlocks:   1, // 1 byte of padding after int8Field to align int16Field
			},
		},
		{
			name: "Message with only variable fields",
			message: &schema.Message{
				Name:       "OnlyVariableFields",
				Fields:     []*schema.Field{stringField, bytesField},
				Properties: map[string]*schema.Field{"stringField": stringField, "bytesField": bytesField},
			},
			expectedCMsg: &CompiledMessage{
				Name: "OnlyVariableFields",
				Fields: []*CompiledField{
					{Name: stringField.Name, GoName: "StringField", CName: "string_field", Type: stringField.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
					{Name: bytesField.Name, GoName: "BytesField", CName: "bytes_field", Type: bytesField.Type, Offset: 4, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
				},
				FixedFields: []*CompiledField{
					{Name: stringField.Name, GoName: "StringField", CName: "string_field", Type: stringField.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
					{Name: bytesField.Name, GoName: "BytesField", CName: "bytes_field", Type: bytesField.Type, Offset: 4, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
				},
				VariableFields: []*CompiledField{
					{Name: stringField.Name, GoName: "StringField", CName: "string_field", Type: stringField.Type, Offset: 0, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
					{Name: bytesField.Name, GoName: "BytesField", CName: "bytes_field", Type: bytesField.Type, Offset: 4, Size: 4, Alignment: 4, IsVariable: true, PaddingBefore: 0},
				},
				TotalFixedSize:  8, // 4 bytes for string length + 4 bytes for bytes length
				StructAlignment: 4,
				PaddingBlocks:   0, // no padding needed
			},
		},
		{
			name: "Message with nested object field",
			message: &schema.Message{
				Name:       "MessageWithObjectField",
				Fields:     []*schema.Field{objectField},
				Properties: map[string]*schema.Field{"objectField": objectField},
			},
			expectedCMsg: &CompiledMessage{
				Name:            "MessageWithObjectField",
				Fields:          []*CompiledField{},
				FixedFields:     []*CompiledField{},
				VariableFields:  []*CompiledField{},
				TotalFixedSize:  0,
				StructAlignment: 1,
				PaddingBlocks:   0,
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.name == "Message with nested object field" {
				t.Skip("Skipping test for nested object field as it is not implemented yet.")
			}

			actualCMsg := CompileMessage(tc.message)
			assert.NotNil(t, actualCMsg)

			assert.Equal(t, tc.expectedCMsg.Name, actualCMsg.Name)
			assert.Equal(t, len(tc.expectedCMsg.Fields), len(actualCMsg.Fields))

			for i, expectedField := range tc.expectedCMsg.Fields {
				actualField := actualCMsg.Fields[i]
				assert.Equal(t, expectedField.Name, actualField.Name)
				assert.Equal(t, expectedField.GoName, actualField.GoName)
				assert.Equal(t, expectedField.CName, actualField.CName)
				assert.Equal(t, expectedField.Type, actualField.Type)
				assert.Equal(t, expectedField.Offset, actualField.Offset)
				assert.Equal(t, expectedField.Size, actualField.Size)
				assert.Equal(t, expectedField.Alignment, actualField.Alignment)
				assert.Equal(t, expectedField.IsVariable, actualField.IsVariable)
				assert.Equal(t, expectedField.PaddingBefore, actualField.PaddingBefore)
			}

			assert.Equal(t, len(tc.expectedCMsg.FixedFields), len(actualCMsg.FixedFields))
			for i := range tc.expectedCMsg.FixedFields {
				assert.Equal(t, tc.expectedCMsg.FixedFields[i].Name, actualCMsg.FixedFields[i].Name)
			}

			assert.Equal(t, len(tc.expectedCMsg.VariableFields), len(actualCMsg.VariableFields))
			for i := range tc.expectedCMsg.VariableFields {
				assert.Equal(t, tc.expectedCMsg.VariableFields[i].Name, actualCMsg.VariableFields[i].Name)
			}

			assert.Equal(t, tc.expectedCMsg.TotalFixedSize, actualCMsg.TotalFixedSize)
			assert.Equal(t, tc.expectedCMsg.StructAlignment, actualCMsg.StructAlignment)
			assert.Equal(t, tc.expectedCMsg.PaddingBlocks, actualCMsg.PaddingBlocks)
		})
	}
}
