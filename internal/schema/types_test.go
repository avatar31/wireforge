// Copyright (c) 2026 Sachin S. All rights reserved.
//
// Licensed under the MIT License.
// See LICENSE in the project root.

package schema

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFieldTypeSize(t *testing.T) {
	tests := []struct {
		ft       FieldType
		wantSize int
	}{
		{FieldTypeUint64, 8},
		{FieldTypeInt64, 8},
		{FieldTypeFloat64, 8},
		{FieldTypeUint32, 4},
		{FieldTypeInt32, 4},
		{FieldTypeFloat32, 4},
		{FieldTypeString, 4},
		{FieldTypeBytes, 4},
		{FieldTypeObject, 4},
		{FieldTypeArray, 4},
		{FieldTypeUint16, 2},
		{FieldTypeInt16, 2},
		{FieldTypeUint8, 1},
		{FieldTypeInt8, 1},
		{FieldTypeBool, 1},
	}
	for _, tc := range tests {
		t.Run(tc.ft.GoType(), func(t *testing.T) {
			assert.Equal(t, tc.wantSize, tc.ft.Size())
		})
	}
}

func TestFieldTypeAlignment(t *testing.T) {
	tests := []struct {
		ft        FieldType
		wantAlign int
	}{
		{FieldTypeUint64, 8},
		{FieldTypeInt64, 8},
		{FieldTypeFloat64, 8},
		{FieldTypeUint32, 4},
		{FieldTypeInt32, 4},
		{FieldTypeFloat32, 4},
		{FieldTypeString, 4},
		{FieldTypeBytes, 4},
		{FieldTypeObject, 4},
		{FieldTypeArray, 4},
		{FieldTypeUint16, 2},
		{FieldTypeInt16, 2},
		{FieldTypeUint8, 1},
		{FieldTypeInt8, 1},
		{FieldTypeBool, 1},
	}
	for _, tc := range tests {
		t.Run(tc.ft.GoType(), func(t *testing.T) {
			assert.Equal(t, tc.wantAlign, tc.ft.Alignment())
		})
	}
}

func TestFieldTypeGoType(t *testing.T) {
	tests := []struct {
		ft       FieldType
		wantType string
	}{
		{FieldTypeUint8, "uint8"},
		{FieldTypeUint16, "uint16"},
		{FieldTypeUint32, "uint32"},
		{FieldTypeUint64, "uint64"},
		{FieldTypeInt8, "int8"},
		{FieldTypeInt16, "int16"},
		{FieldTypeInt32, "int32"},
		{FieldTypeInt64, "int64"},
		{FieldTypeFloat32, "float32"},
		{FieldTypeFloat64, "float64"},
		{FieldTypeBool, "bool"},
		{FieldTypeString, "string"},
		{FieldTypeBytes, "[]byte"},
		{FieldTypeObject, "struct"},
		{FieldTypeArray, "[]any"},
	}
	for _, tc := range tests {
		t.Run(tc.wantType, func(t *testing.T) {
			assert.Equal(t, tc.wantType, tc.ft.GoType())
		})
	}
}

func TestFieldTypeCType(t *testing.T) {
	tests := []struct {
		ft       FieldType
		wantType string
	}{
		{FieldTypeUint8, "uint8_t"},
		{FieldTypeInt8, "int8_t"},
		{FieldTypeBool, "uint8_t"}, // bool maps to uint8_t in C
		{FieldTypeUint16, "uint16_t"},
		{FieldTypeInt16, "int16_t"},
		{FieldTypeUint32, "uint32_t"},
		{FieldTypeInt32, "int32_t"},
		{FieldTypeFloat32, "float"},
		{FieldTypeUint64, "uint64_t"},
		{FieldTypeInt64, "int64_t"},
		{FieldTypeFloat64, "double"},
		{FieldTypeString, "string"},
		{FieldTypeBytes, "[]byte"},
		{FieldTypeObject, "struct"},
		{FieldTypeArray, "[]any"},
	}
	for _, tc := range tests {
		t.Run(tc.wantType, func(t *testing.T) {
			assert.Equal(t, tc.wantType, tc.ft.CType())
		})
	}
}

func TestFieldTypeIsVariable(t *testing.T) {
	fixedTypes := []FieldType{
		FieldTypeUint64, FieldTypeInt64, FieldTypeFloat64,
		FieldTypeUint32, FieldTypeInt32, FieldTypeFloat32,
		FieldTypeUint16, FieldTypeInt16,
		FieldTypeUint8, FieldTypeInt8,
		FieldTypeBool,
	}
	variableTypes := []FieldType{
		FieldTypeString,
		FieldTypeBytes,
		FieldTypeObject,
		FieldTypeArray,
	}

	for _, ft := range fixedTypes {
		t.Run("fixed/"+ft.GoType(), func(t *testing.T) {
			assert.False(t, ft.IsVariable(), "expected %v to be a fixed-width type", ft.GoType())
		})
	}
	for _, ft := range variableTypes {
		t.Run("variable/"+ft.GoType(), func(t *testing.T) {
			assert.True(t, ft.IsVariable(), "expected %v to be a variable-length type", ft.GoType())
		})
	}
}

// TestFieldTypeSizeAlignmentConsistency verifies that Size and Alignment are
// always equal — a key invariant relied on by the alignment engine in the
// compiler.
func TestFieldTypeSizeAlignmentConsistency(t *testing.T) {
	for ft := range FieldType(NumOfFieldTypes) {
		t.Run(ft.GoType(), func(t *testing.T) {
			assert.Equal(t, ft.Size(), ft.Alignment(),
				"FieldType %d: Size() and Alignment() must be equal", ft)
		})
	}
}
