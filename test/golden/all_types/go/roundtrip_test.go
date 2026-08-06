// Package messages exercises the generated messages package via roundtrip,
// wire-format, error, and benchmark tests.
package messages

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func scalarRoundtrip(t *testing.T, src WireMessage) (*OnlyScalarTypesMsg, []byte) {
	t.Helper()
	wire, err := src.Marshal()
	require.NoError(t, err, "Marshal should not fail")
	r := bytes.NewReader(wire)
	_, fixedLen, overallLen, err := ReadMessageFrame(r)
	require.NoError(t, err, "ReadMessageFrame should not fail")
	var dst OnlyScalarTypesMsg
	require.NoError(t, dst.Unmarshal(r, fixedLen, overallLen))
	return &dst, wire
}

func variableRoundtrip(t *testing.T, src WireMessage) (*OnlyVariableTypesMsg, []byte) {
	t.Helper()
	wire, err := src.Marshal()
	require.NoError(t, err)
	r := bytes.NewReader(wire)
	_, fixedLen, overallLen, err := ReadMessageFrame(r)
	require.NoError(t, err)
	var dst OnlyVariableTypesMsg
	require.NoError(t, dst.Unmarshal(r, fixedLen, overallLen))
	return &dst, wire
}

func allTypesRoundtrip(t *testing.T, src WireMessage) (*AllTypesFieldsMsg, []byte) {
	t.Helper()
	wire, err := src.Marshal()
	require.NoError(t, err)
	r := bytes.NewReader(wire)
	_, fixedLen, overallLen, err := ReadMessageFrame(r)
	require.NoError(t, err)
	var dst AllTypesFieldsMsg
	require.NoError(t, dst.Unmarshal(r, fixedLen, overallLen))
	return &dst, wire
}

func recursiveRoundtrip(t *testing.T, src WireMessage) (*RecursiveNestedMsg, []byte) {
	t.Helper()
	wire, err := src.Marshal()
	require.NoError(t, err)
	r := bytes.NewReader(wire)
	_, fixedLen, overallLen, err := ReadMessageFrame(r)
	require.NoError(t, err)
	var dst RecursiveNestedMsg
	require.NoError(t, dst.Unmarshal(r, fixedLen, overallLen))
	return &dst, wire
}

func arraysRoundtrip(t *testing.T, src WireMessage) (*AllTypesOfArraysMsg, []byte) {
	t.Helper()
	wire, err := src.Marshal()
	require.NoError(t, err)
	r := bytes.NewReader(wire)
	_, fixedLen, overallLen, err := ReadMessageFrame(r)
	require.NoError(t, err)
	var dst AllTypesOfArraysMsg
	require.NoError(t, dst.Unmarshal(r, fixedLen, overallLen))
	return &dst, wire
}

// ---------------------------------------------------------------------------
// Roundtrip — OnlyScalarTypesMsg
// ---------------------------------------------------------------------------

func TestOnlyScalarTypesMsgRoundTrip(t *testing.T) {
	t.Run("MessageTypeID()", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{}
		assert.Equal(t, uint16(1), src.MessageTypeID())
	})

	t.Run("DynamicPayloadSize()", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{
			ValInt64: math.MinInt64,
			ValInt32: math.MinInt32,
			ValInt16: math.MinInt16,
			ValInt8:  math.MinInt8,
		}
		actual := src.DynamicPayloadSize()
		assert.Equal(t, 0, actual, "DynamicPayloadSize is not matching")
	})

	t.Run("Size()", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{
			ValInt64: math.MinInt64,
			ValInt32: math.MinInt32,
			ValInt16: math.MinInt16,
			ValInt8:  math.MinInt8,
		}
		actual := src.Size()
		expected := FrameHeaderSize + OnlyScalarTypesMsgFixedSize
		assert.Equal(t, expected, actual, "Size is not matching")
	})

	t.Run("Marshal()", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{
			ValInt64: math.MinInt64,
			ValInt32: math.MinInt32,
			ValInt16: math.MinInt16,
			ValInt8:  math.MinInt8,
		}
		wire, err := src.Marshal()
		require.NoError(t, err, "Marshal should not fail")
		assert.Equal(t, FrameHeaderSize+OnlyScalarTypesMsgFixedSize, len(wire), "wire length mismatch")

		assert.Equal(t, uint16(1), binary.BigEndian.Uint16(wire[0:2]), "TypeID mismatch")
		assert.Equal(t, uint16(OnlyScalarTypesMsgFixedSize), binary.BigEndian.Uint16(wire[2:4]), "FixedPayloadLen mismatch")
		assert.Equal(t, uint32(OnlyScalarTypesMsgFixedSize), binary.BigEndian.Uint32(wire[4:8]), "OverallPayloadLen mismatch")

		assert.Len(t, wire[FrameHeaderSize:], OnlyScalarTypesMsgFixedSize, "fixed block length mismatch")
	})

	t.Run("Unmarshal()", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{
			ValUint64: 0xDEADBEEFCAFEBABE,
			ValInt64:  -9123456789012345678,
			ValDouble: math.Pi,
			ValUint32: 0xDEADBEEF,
			ValInt32:  -2147483648,
			ValFloat:  math.E,
			ValUint16: 0xBEEF,
			ValInt16:  -32768,
			ValUint8:  255,
			ValInt8:   -128,
			ValBool:   true,
		}
		dst, _ := scalarRoundtrip(t, src)
		assert.Equal(t, src.ValUint64, dst.ValUint64)
		assert.Equal(t, src.ValInt64, dst.ValInt64)
		assert.Equal(t, src.ValDouble, dst.ValDouble)
		assert.Equal(t, src.ValUint32, dst.ValUint32)
		assert.Equal(t, src.ValInt32, dst.ValInt32)
		assert.Equal(t, src.ValFloat, dst.ValFloat)
		assert.Equal(t, src.ValUint16, dst.ValUint16)
		assert.Equal(t, src.ValInt16, dst.ValInt16)
		assert.Equal(t, src.ValUint8, dst.ValUint8)
		assert.Equal(t, src.ValInt8, dst.ValInt8)
		assert.Equal(t, src.ValBool, dst.ValBool)
	})

	t.Run("ZeroValues", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{}
		dst, _ := scalarRoundtrip(t, src)
		assert.Equal(t, src, dst)
	})

	t.Run("MaxValues", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{
			ValUint64: math.MaxUint64,
			ValInt64:  math.MaxInt64,
			ValDouble: math.MaxFloat64,
			ValUint32: math.MaxUint32,
			ValInt32:  math.MaxInt32,
			ValFloat:  math.MaxFloat32,
			ValUint16: math.MaxUint16,
			ValInt16:  math.MaxInt16,
			ValUint8:  math.MaxUint8,
			ValInt8:   math.MaxInt8,
			ValBool:   true,
		}
		dst, _ := scalarRoundtrip(t, src)
		assert.Equal(t, src, dst)
	})

	t.Run("NegativeValues", func(t *testing.T) {
		src := &OnlyScalarTypesMsg{
			ValInt64:  math.MinInt64,
			ValInt32:  math.MinInt32,
			ValInt16:  math.MinInt16,
			ValInt8:   math.MinInt8,
			ValDouble: -math.MaxFloat64,
			ValFloat:  -math.MaxFloat32,
		}
		dst, _ := scalarRoundtrip(t, src)
		assert.Equal(t, src, dst)
	})
}

// ---------------------------------------------------------------------------
// Roundtrip — OnlyVariableTypesMsg
// ---------------------------------------------------------------------------

func TestOnlyVariableTypesMsgRoundTrip(t *testing.T) {
	msg := &OnlyVariableTypesMsg{
		Name:      "hello world",
		Data:      []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Nested:    &OnlyScalarTypesMsg{ValUint64: 42},
		Tags:      []string{"alpha", "beta", "gamma", "xyz"},
		Matrix:    [][]int32{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
		ByteArray: [][]byte{{0x01, 0x02}, {0x03, 0x04, 0x05}},
	}

	msgDynSize := func(msg *OnlyVariableTypesMsg) int {
		expected := len(msg.Name)
		expected += len(msg.Data)
		expected += msg.Nested.Size()

		// ByteArray
		byteArrayLen := TypeMarkerSize       // value: TagArray (2 bytes)
		byteArrayLen += TypeMarkerSize       // value: TagBytes (2 bytes)
		byteArrayLen += ArrayCountPrefixSize // value: 2 (2 bytes) i.e. overall elements count
		byteArrayLen += ArrayCountPrefixSize // value: 2 (2 bytes) i.e. 1D array elements count
		for _, b := range msg.ByteArray {
			byteArrayLen += StrOrByteLenPrefixSize // value: 2 & 3 (4 bytes) i.e. length of each byte array
			byteArrayLen += len(b)                 // value: {0x01, 0x02} (2 bytes) and {0x03, 0x04, 0x05} (3 bytes)
		}
		expected += byteArrayLen

		// Matrix
		matrixLen := TypeMarkerSize       // value: TagArray (2 bytes)
		matrixLen += TypeMarkerSize       // value: TagInt32 (2 bytes)
		matrixLen += ArrayCountPrefixSize // value: 9 (2 bytes) i.e. overall elements count
		matrixLen += ArrayCountPrefixSize // value: 3 (2 bytes) i.e. 2D array elements count
		for _, row := range msg.Matrix {
			matrixLen += ArrayCountPrefixSize // value: 3 (2 bytes) i.e. 1D array elements count
			matrixLen += len(row) * 4         // value: 1, 2, 3, 4, 5, 6, 7, 8, 9 (4 bytes each)
		}
		expected += matrixLen

		// Tags
		tagsLen := TypeMarkerSize       // value: TagArray (2 bytes)
		tagsLen += TypeMarkerSize       // value: TagString (2 bytes)
		tagsLen += ArrayCountPrefixSize // value: 4 (2 bytes) i.e. overall elements count
		tagsLen += ArrayCountPrefixSize // value: 4 (2 bytes) i.e. 1D array elements count
		for _, tag := range msg.Tags {
			tagsLen += StrOrByteLenPrefixSize // value: 5, 4, 5, 3 (4 bytes) i.e. length of each string
			tagsLen += len(tag)               // value: "alpha" (5 bytes), "beta" (4 bytes), "gamma" (5 bytes), "xyz" (3 bytes)
		}
		expected += tagsLen

		return expected // 183
	}

	t.Run("MessageTypeID()", func(t *testing.T) {
		assert.Equal(t, uint16(2), msg.MessageTypeID())
	})

	t.Run("DynamicPayloadSize()", func(t *testing.T) {
		expected := msgDynSize(msg)
		actual := msg.DynamicPayloadSize()
		assert.Equal(t, expected, actual, "DynamicPayloadSize is not matching")
	})

	t.Run("Size()", func(t *testing.T) {
		expected := FrameHeaderSize + OnlyVariableTypesMsgFixedSize + msgDynSize(msg)
		actual := msg.Size()
		assert.Equal(t, expected, actual, "Size is not matching")
	})

	t.Run("Marshal()", func(t *testing.T) {
		wire, err := msg.Marshal()
		require.NoError(t, err, "Marshal should not fail")

		dynSize := msgDynSize(msg)
		assert.Equal(t, FrameHeaderSize+OnlyVariableTypesMsgFixedSize+dynSize, len(wire), "wire length mismatch")

		assert.Equal(t, uint16(2), binary.BigEndian.Uint16(wire[0:2]), "TypeID mismatch")
		assert.Equal(t, uint16(OnlyVariableTypesMsgFixedSize), binary.BigEndian.Uint16(wire[2:4]), "FixedPayloadLen mismatch")
		assert.Equal(t, uint32(OnlyVariableTypesMsgFixedSize+dynSize), binary.BigEndian.Uint32(wire[4:8]), "OverallPayloadLen mismatch")

		assert.Len(t, wire[FrameHeaderSize:], OnlyVariableTypesMsgFixedSize+dynSize, "fixed block length mismatch")
	})

	t.Run("Unmarshal()", func(t *testing.T) {
		dst, _ := variableRoundtrip(t, msg)
		assert.Equal(t, msg.Name, dst.Name)
		assert.Equal(t, msg.Data, dst.Data)
		assert.Equal(t, msg.Nested, dst.Nested)
		assert.Equal(t, msg.Tags, dst.Tags)
		assert.Equal(t, msg.Matrix, dst.Matrix)
		assert.Equal(t, msg.ByteArray, dst.ByteArray)
	})

	t.Run("EmptyValues", func(t *testing.T) {
		src := &OnlyVariableTypesMsg{
			Name:      "",
			Data:      []byte{},
			Tags:      []string{},
			Matrix:    [][]int32{},
			ByteArray: [][]byte{},
		}
		dst, _ := variableRoundtrip(t, src)
		assert.Empty(t, dst.Name)
		assert.Nil(t, dst.Nested)
		assert.Equal(t, len(src.Data), len(dst.Data))
		assert.Equal(t, len(src.Tags), len(dst.Tags))
		assert.Equal(t, len(src.Matrix), len(dst.Matrix))
		assert.Equal(t, len(src.ByteArray), len(dst.ByteArray))
	})

	t.Run("UnassignedValues", func(t *testing.T) {
		src := &OnlyVariableTypesMsg{}
		dst, _ := variableRoundtrip(t, src)
		assert.Empty(t, dst.Name)
		assert.Nil(t, dst.Nested)
		assert.Equal(t, len(src.Data), len(dst.Data))
		assert.Equal(t, len(src.Tags), len(dst.Tags))
		assert.Equal(t, len(src.Matrix), len(dst.Matrix))
		assert.Equal(t, len(src.ByteArray), len(dst.ByteArray))
	})

	t.Run("UnicodeString", func(t *testing.T) {
		src := &OnlyVariableTypesMsg{Name: "日本語テスト 🎯"}
		dst, _ := variableRoundtrip(t, src)
		assert.Equal(t, src.Name, dst.Name)
	})

	t.Run("LargeString", func(t *testing.T) {
		src := &OnlyVariableTypesMsg{Name: strings.Repeat("x", 1<<20)} // 1 MiB
		dst, _ := variableRoundtrip(t, src)
		assert.Equal(t, len(src.Name), len(dst.Name))
		assert.Equal(t, src.Name, dst.Name)
	})

	t.Run("LargeByteArray", func(t *testing.T) {
		src := &OnlyVariableTypesMsg{Data: bytes.Repeat([]byte{0xAB}, 1<<20)} // 1 MiB
		dst, _ := variableRoundtrip(t, src)
		assert.Equal(t, len(src.Data), len(dst.Data))
		assert.Equal(t, src.Data, dst.Data)
	})

	t.Run("BinaryData", func(t *testing.T) {
		file, err := os.Open("../../../schemas/all_types.yaml")
		require.NoError(t, err, "failed to open file for binary data test")
		defer file.Close()

		data, err := io.ReadAll(file)
		require.NoError(t, err, "failed to read file for binary data test")

		src := &OnlyVariableTypesMsg{Data: data}
		dst, _ := variableRoundtrip(t, src)
		assert.Equal(t, src.Data, dst.Data)
	})
}

// ---------------------------------------------------------------------------
// Roundtrip — AllTypesFieldsMsg
// ---------------------------------------------------------------------------

func TestAllTypesFieldsMsgRoundTrip(t *testing.T) {
	msg := &AllTypesFieldsMsg{
		ValUint64: 0x0102030405060708,
		ValInt64:  -1234567890123456789,
		ValDouble: math.Pi,
		ValUint32: 0xABCDEF01,
		ValInt32:  -1000000,
		ValFloat:  1.234,
		Name:      "hello world",
		Data:      []byte{0xDE, 0xAD, 0xBE, 0xEF},
		Nested:    &OnlyScalarTypesMsg{ValUint64: 42},
		Tags:      []string{"alpha", "beta", "gamma", "xyz"},
		Matrix:    [][]int32{{1, 2, 3}, {4, 5, 6}, {7, 8, 9}},
		ByteArray: [][]byte{{0x01, 0x02}, {0x03, 0x04, 0x05}},
		ValUint16: 0x1234,
		ValInt16:  -100,
		ValUint8:  200,
		ValInt8:   -50,
		ValBool:   true,
	}

	msgDynSize := func(msg *AllTypesFieldsMsg) int {
		expected := len(msg.Name)
		expected += len(msg.Data)
		expected += msg.Nested.Size()

		// ByteArray
		byteArrayLen := TypeMarkerSize       // value: TagArray (2 bytes)
		byteArrayLen += TypeMarkerSize       // value: TagBytes (2 bytes)
		byteArrayLen += ArrayCountPrefixSize // value: 1 (2 bytes)
		byteArrayLen += ArrayCountPrefixSize // value: 1 (2 bytes)
		for _, b := range msg.ByteArray {
			byteArrayLen += StrOrByteLenPrefixSize // value: 2 (4 bytes)
			byteArrayLen += len(b)                 // value: {0xAA, 0xBB} (2 bytes)
		}
		expected += byteArrayLen

		// Matrix
		matrixLen := TypeMarkerSize       // value: TagArray (2 bytes)
		matrixLen += TypeMarkerSize       // value: TagInt32 (2 bytes)
		matrixLen += ArrayCountPrefixSize // value: 4 (2 bytes) i.e. overall elements count
		matrixLen += ArrayCountPrefixSize // value: 3 (2 bytes) i.e. 2D array elements count
		for _, row := range msg.Matrix {
			matrixLen += ArrayCountPrefixSize // value: 2 (2 bytes) i.e. 1D array elements count
			matrixLen += len(row) * 4         // value: 10, 20, 30, 40, 50, 60 (4 bytes each)
		}
		expected += matrixLen

		// Tags
		tagsLen := TypeMarkerSize       // value: TagArray (2 bytes)
		tagsLen += TypeMarkerSize       // value: TagString (2 bytes)
		tagsLen += ArrayCountPrefixSize // value: 2 (2 bytes) i.e. overall elements count
		tagsLen += ArrayCountPrefixSize // value: 2 (2 bytes) i.e. 1D array elements count
		for _, tag := range msg.Tags {
			tagsLen += StrOrByteLenPrefixSize // value: 3 (4 bytes) i.e. length of "one" and "two"
			tagsLen += len(tag)               // value: "one" (3 bytes) and "two" (3 bytes)
		}
		expected += tagsLen

		return expected // 183
	}

	t.Run("MessageTypeID()", func(t *testing.T) {
		assert.Equal(t, uint16(3), msg.MessageTypeID())
	})

	t.Run("DynamicPayloadSize()", func(t *testing.T) {
		expected := msgDynSize(msg)
		actual := msg.DynamicPayloadSize()
		assert.Equal(t, expected, actual, "DynamicPayloadSize is not matching")
	})

	t.Run("Size()", func(t *testing.T) {
		expected := FrameHeaderSize + AllTypesFieldsMsgFixedSize + msgDynSize(msg)
		actual := msg.Size()
		assert.Equal(t, expected, actual, "Size is not matching")
	})

	t.Run("Marshal()", func(t *testing.T) {
		wire, err := msg.Marshal()
		require.NoError(t, err, "Marshal should not fail")

		dynSize := msgDynSize(msg)
		assert.Equal(t, FrameHeaderSize+AllTypesFieldsMsgFixedSize+dynSize, len(wire), "wire length mismatch")

		assert.Equal(t, uint16(3), binary.BigEndian.Uint16(wire[0:2]), "TypeID mismatch")
		assert.Equal(t, uint16(AllTypesFieldsMsgFixedSize), binary.BigEndian.Uint16(wire[2:4]), "FixedPayloadLen mismatch")
		assert.Equal(t, uint32(AllTypesFieldsMsgFixedSize+dynSize), binary.BigEndian.Uint32(wire[4:8]), "OverallPayloadLen mismatch")

		assert.Len(t, wire[FrameHeaderSize:], AllTypesFieldsMsgFixedSize+dynSize, "fixed block length mismatch")
	})

	t.Run("Unmarshal()", func(t *testing.T) {
		fmt.Printf("Overall size of message: actaul:%d, manual:%d\n", msg.DynamicPayloadSize(), msgDynSize(msg))
		dst, _ := allTypesRoundtrip(t, msg)
		assert.Equal(t, msg.ValUint64, dst.ValUint64)
		assert.Equal(t, msg.ValInt64, dst.ValInt64)
		assert.Equal(t, msg.ValDouble, dst.ValDouble)
		assert.Equal(t, msg.ValUint32, dst.ValUint32)
		assert.Equal(t, msg.ValInt32, dst.ValInt32)
		assert.Equal(t, msg.ValFloat, dst.ValFloat)
		assert.Equal(t, msg.Name, dst.Name)
		assert.Equal(t, msg.Data, dst.Data)
		require.NotNil(t, dst.Nested)
		assert.Equal(t, msg.Nested.ValUint64, dst.Nested.ValUint64)
		assert.Equal(t, msg.Tags, dst.Tags)
		assert.Equal(t, msg.Matrix, dst.Matrix)
		assert.Equal(t, msg.ByteArray, dst.ByteArray)
		assert.Equal(t, msg.ValUint16, dst.ValUint16)
		assert.Equal(t, msg.ValInt16, dst.ValInt16)
		assert.Equal(t, msg.ValUint8, dst.ValUint8)
		assert.Equal(t, msg.ValInt8, dst.ValInt8)
		assert.Equal(t, msg.ValBool, dst.ValBool)
	})

	t.Run("OnlyScalars", func(t *testing.T) {
		src := &AllTypesFieldsMsg{
			ValUint64: 1,
			ValInt64:  -1,
			ValDouble: 1.0,
			ValBool:   true,
		}

		assert.Equal(t, 0, src.DynamicPayloadSize())

		dst, _ := allTypesRoundtrip(t, src)
		assert.Equal(t, src.ValUint64, dst.ValUint64)
		assert.Equal(t, src.ValBool, dst.ValBool)
		assert.Nil(t, dst.Nested)
		assert.Nil(t, dst.Tags)
	})
}

// ---------------------------------------------------------------------------
// Roundtrip — RecursiveNestedMsg
// ---------------------------------------------------------------------------

func TestRecursiveNestedMsgRoundTrip(t *testing.T) {
	msg := &RecursiveNestedMsg{
		Name: "level1",
		Nested: &RecursiveNestedMsg{
			Name: "level2",
			Nested: &RecursiveNestedMsg{
				Name: "level3",
			},
		},
	}

	msgDynSize := func(msg *RecursiveNestedMsg) int {
		expected := len(msg.Name)
		if msg.Nested != nil {
			expected += msg.Nested.Size()
		}

		return expected // 50
	}

	t.Run("MessageTypeID()", func(t *testing.T) {
		src := &RecursiveNestedMsg{Name: "leaf"}
		assert.Equal(t, uint16(4), src.MessageTypeID())
	})

	t.Run("DynamicPayloadSize()", func(t *testing.T) {
		expected := msgDynSize(msg)
		actual := msg.DynamicPayloadSize()
		assert.Equal(t, expected, actual, "DynamicPayloadSize is not matching")
	})

	t.Run("Size()", func(t *testing.T) {
		expected := FrameHeaderSize + RecursiveNestedMsgFixedSize + msgDynSize(msg)
		actual := msg.Size()
		assert.Equal(t, expected, actual, "Size is not matching")
	})

	t.Run("Marshal()", func(t *testing.T) {
		wire, err := msg.Marshal()
		require.NoError(t, err, "Marshal should not fail")

		dynSize := msgDynSize(msg)
		assert.Equal(t, FrameHeaderSize+RecursiveNestedMsgFixedSize+dynSize, len(wire), "wire length mismatch")

		assert.Equal(t, uint16(4), binary.BigEndian.Uint16(wire[0:2]), "TypeID mismatch")
		assert.Equal(t, uint16(RecursiveNestedMsgFixedSize), binary.BigEndian.Uint16(wire[2:4]), "FixedPayloadLen mismatch")
		assert.Equal(t, uint32(RecursiveNestedMsgFixedSize+dynSize), binary.BigEndian.Uint32(wire[4:8]), "OverallPayloadLen mismatch")

		assert.Len(t, wire[FrameHeaderSize:], RecursiveNestedMsgFixedSize+dynSize, "fixed block length mismatch")
	})

	t.Run("Unmarshal()", func(t *testing.T) {
		dst, _ := recursiveRoundtrip(t, msg)

		assert.Equal(t, "level1", dst.Name)
		require.NotNil(t, dst.Nested)
		assert.Equal(t, "level2", dst.Nested.Name)
		require.NotNil(t, dst.Nested.Nested)
		assert.Equal(t, "level3", dst.Nested.Nested.Name)
		assert.Nil(t, dst.Nested.Nested.Nested)
	})

	t.Run("SingleLevel", func(t *testing.T) {
		src := &RecursiveNestedMsg{Name: "leaf"}
		dst, _ := recursiveRoundtrip(t, src)
		assert.Equal(t, src.Name, dst.Name)
		assert.Nil(t, dst.Nested)
	})
}

// ---------------------------------------------------------------------------
// Roundtrip — AllTypesOfArraysMsg
// ---------------------------------------------------------------------------

func TestAllTypesOfArraysMsgRoundTrip(t *testing.T) {
	msg := &AllTypesOfArraysMsg{
		// All 1D arrays with 3 elements each
		Arr1DBool:   []bool{true, false, true},
		Arr1DUint8:  []uint8{0, 128, 255},
		Arr1DInt8:   []int8{-128, 0, 127},
		Arr1DUint16: []uint16{0, 1000, 65535},
		Arr1DInt16:  []int16{-32768, 0, 32767},
		Arr1DUint32: []uint32{0, 1, 4294967295},
		Arr1DInt32:  []int32{-2147483648, 0, 2147483647},
		Arr1DFloat:  []float32{-1.5, 0.0, 1.5},
		Arr1DUint64: []uint64{0, 1, math.MaxUint64},
		Arr1DInt64:  []int64{math.MinInt64, 0, math.MaxInt64},
		Arr1DDouble: []float64{-math.Pi, 0.0, math.Pi},
		Arr1DString: []string{"hello", "world", "日本語"},
		Arr1DBytes:  [][]byte{{0x01, 0x02}, {0xFF}, {0xFF}},
		Arr1DNested: []*OnlyVariableTypesMsg{{Name: "something"}, {Name: "something"}, {Name: "something"}},
		Arr1DObject: []*OnlyScalarTypesMsg{{ValUint64: 100}, {ValUint64: 100}, {ValUint64: 100}},

		// All 2D arrays with 2x2 elements each
		Arr2DBool:   [][]bool{{true, false}, {false, true}},
		Arr2DUint8:  [][]uint8{{0, 128}, {255, 64}},
		Arr2DInt8:   [][]int8{{-128, 0}, {127, -64}},
		Arr2DUint16: [][]uint16{{0, 1000}, {65535, 500}},
		Arr2DInt16:  [][]int16{{-32768, 0}, {32767, -500}},
		Arr2DUint32: [][]uint32{{0, 1}, {4294967295, 100}},
		Arr2DInt32:  [][]int32{{-2147483648, 0}, {2147483647, -100}},
		Arr2DUint64: [][]uint64{{0, 1}, {math.MaxUint64, 1000}},
		Arr2DInt64:  [][]int64{{math.MinInt64, 0}, {math.MaxInt64, -1000}},
		Arr2DFloat:  [][]float32{{-1.5, 0.0}, {1.5, -2.5}},
		Arr2DDouble: [][]float64{{-math.Pi, 0.0}, {math.Pi, -math.E}},
		Arr2DString: [][]string{{"a", "b"}, {"c", "d"}},
		Arr2DBytes:  [][][]byte{{{0x01}, {0x02}}, {{0x03}, {0x04, 0x05}}},
		Arr2DNested: [][]*OnlyVariableTypesMsg{{{Name: "nested1"}, {Name: "nested2"}}, {{Name: "nested3"}, {Name: "nested4"}}},
		Arr2DObject: [][]*OnlyScalarTypesMsg{{{ValUint64: 1}, {ValUint64: 2}}, {{ValUint64: 3}, {ValUint64: 4}}},

		// All 3D arrays with 2x2x2 elements each
		Arr3DBool:   [][][]bool{{{true, false}, {false, true}}, {{true, true}, {false, false}}},
		Arr3DUint8:  [][][]uint8{{{0, 128}, {255, 64}}, {{1, 2}, {3, 4}}},
		Arr3DInt8:   [][][]int8{{{-128, 0}, {127, -64}}, {{-1, -2}, {-3, -4}}},
		Arr3DUint16: [][][]uint16{{{0, 1000}, {65535, 500}}, {{1, 2}, {3, 4}}},
		Arr3DInt16:  [][][]int16{{{-32768, 0}, {32767, -500}}, {{-1, -2}, {-3, -4}}},
		Arr3DUint32: [][][]uint32{{{0, 1}, {4294967295, 100}}, {{1, 2}, {3, 4}}},
		Arr3DInt32:  [][][]int32{{{-2147483648, 0}, {2147483647, -100}}, {{-1, -2}, {-3, -4}}},
		Arr3DUint64: [][][]uint64{{{0, 1}, {math.MaxUint64, 1000}}, {{1, 2}, {3, 4}}},
		Arr3DInt64:  [][][]int64{{{math.MinInt64, 0}, {math.MaxInt64, -1000}}, {{-1, -2}, {-3, -4}}},
		Arr3DFloat:  [][][]float32{{{-1.5, 0.0}, {1.5, -2.5}}, {{-3.5, 4.5}, {5.5, -6.5}}},
		Arr3DDouble: [][][]float64{{{-math.Pi, 0.0}, {math.Pi, -math.E}}, {{-1.0, 2.0}, {3.0, -4.0}}},
		Arr3DString: [][][]string{{{"a", "b"}, {"c", "d"}}, {{"e", "f"}, {"g", "h"}}},
		Arr3DBytes:  [][][][]byte{{{{0x01}, {0x02}}, {{0x03}, {0x04, 0x05}}}, {{{0x06}, {0x07}}, {{0x07, 0x08}, {0x09}}}},
		Arr3DNested: [][][]*OnlyVariableTypesMsg{{
				{{Name: "nested1"}, {Name: "nested2"}}, 
				{{Name: "nested3"}, {Name: "nested4"}},
			},{
				{{Name: "nested5"}, {Name: "nested6"}}, 
				{{Name: "nested7"}, {Name: "nested8"}},
			}},
		Arr3DObject: [][][]*OnlyScalarTypesMsg{{
			{{ValUint64: 1}, {ValUint64: 2}},
			{{ValUint64: 3}, {ValUint64: 4}},
			}, {
				{{ValUint64: 5}, {ValUint64: 6}},
				{{ValUint64: 7}, {ValUint64: 8}},
			},
		},
	}

	msgDynSize := func(msg *AllTypesOfArraysMsg) int {
		expected := 0

		// Calculate dynamic size for 1D arrays
		// Arr1DBool:   []bool{true, false, true},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DBool)*1
		// Arr1DUint8:  []uint8{0, 128, 255},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DUint8)*1
		// Arr1DInt8:   []int8{-128, 0, 127},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DInt8)*1
		// Arr1DUint16: []uint16{0, 1000, 65535},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DUint16)*2
		// Arr1DInt16:  []int16{-32768, 0, 32767},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DInt16)*2
		// Arr1DUint32: []uint32{0, 1, 4294967295},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DUint32)*4
		// Arr1DInt32:  []int32{-2147483648, 0, 2147483647},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DInt32)*4
		// Arr1DUint64: []uint64{0, 1, math.MaxUint64},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DUint64)*8
		// Arr1DInt64:  []int64{math.MinInt64, 0, math.MaxInt64},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DInt64)*8
		// Arr1DFloat:  []float32{-1.5, 0.0, 1.5},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DFloat)*4
		// Arr1DDouble: []float64{-math.Pi, 0.0, math.Pi},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2) + len(msg.Arr1DDouble)*8

		// Arr1DString: []string{"hello", "world", "日本語"},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, s := range msg.Arr1DString {
			expected += StrOrByteLenPrefixSize + len(s)
		}
		// Arr1DBytes:  [][]byte{{0x01, 0x02}, {0xFF}, {0xFF}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, b := range msg.Arr1DBytes {
			expected += StrOrByteLenPrefixSize + len(b)
		}
		// Arr1DNested: []*OnlyVariableTypesMsg{{Name: "something"}, {Name: "something"}, {Name: "something"}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, n := range msg.Arr1DNested {
			expected += ObjectSetUnsetPrefixSize + n.Size()
		}
		// Arr1DObject: []*OnlyScalarTypesMsg{{ValUint64: 100}, {ValUint64: 100}, {ValUint64: 100}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, o := range msg.Arr1DObject {
			expected += ObjectSetUnsetPrefixSize + o.Size()
		}

		// Arr2DBool:   [][]bool{{true, false}, {false, true}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DBool {
			expected += ArrayCountPrefixSize + len(arr2D)*1
		}
		// Arr2DUint8:  [][]uint8{{0, 128}, {255, 64}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DUint8 {
			expected += ArrayCountPrefixSize + len(arr2D)*1
		}
		// Arr2DInt8:   [][]int8{{-128, 0}, {127, -64}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DInt8 {
			expected += ArrayCountPrefixSize + len(arr2D)*1
		}
		// Arr2DUint16: [][]uint16{{0, 1000}, {65535, 500}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DUint16 {
			expected += ArrayCountPrefixSize + len(arr2D)*2
		}
		// Arr2DInt16:  [][]int16{{-32768, 0}, {32767, -500}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DInt16 {
			expected += ArrayCountPrefixSize + len(arr2D)*2
		}
		// Arr2DUint32: [][]uint32{{0, 1}, {4294967295, 100}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DUint32 {
			expected += ArrayCountPrefixSize + len(arr2D)*4
		}
		// Arr2DInt32:  [][]int32{{-2147483648, 0}, {2147483647, -100}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DInt32 {
			expected += ArrayCountPrefixSize + len(arr2D)*4
		}
		// Arr2DUint64: [][]uint64{{0, 1}, {math.MaxUint64, 1000}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DUint64 {
			expected += ArrayCountPrefixSize + len(arr2D)*8
		}
		// Arr2DInt64:  [][]int64{{math.MinInt64, 0}, {math.MaxInt64, -1000}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DInt64 {
			expected += ArrayCountPrefixSize + len(arr2D)*8
		}
		// Arr2DFloat:  [][]float32{{-1.5, 0.0}, {1.5, -2.5}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DFloat {
			expected += ArrayCountPrefixSize + len(arr2D)*4
		}
		// Arr2DDouble: [][]float64{{-math.Pi, 0.0}, {math.Pi, -math.E}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DDouble {
			expected += ArrayCountPrefixSize + len(arr2D)*8
		}

		// Arr2DString: [][]string{{"a", "b"}, {"c", "d"}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DString {
			expected += ArrayCountPrefixSize
			for _, s := range arr2D {
				expected += StrOrByteLenPrefixSize + len(s)
			}
		}
		// Arr2DBytes:  [][][]byte{{{0x01}, {0x02}}, {{0x03}, {0x04, 0x05}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DBytes {
			expected += ArrayCountPrefixSize
			for _, b := range arr2D {
				expected += StrOrByteLenPrefixSize + len(b)
			}
		}
		// Arr2DNested: [][]*OnlyVariableTypesMsg{{{Name: "nested1"}, {Name: "nested2"}}, {{Name: "nested3"}, {Name: "nested4"}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DNested {
			expected += ArrayCountPrefixSize
			for _, n := range arr2D {
				expected += ObjectSetUnsetPrefixSize + n.Size()
			}
		}
		// Arr2DObject: [][]*OnlyScalarTypesMsg{{{ValUint64: 1}, {ValUint64: 2}}, {{ValUint64: 3}, {ValUint64: 4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr2D := range msg.Arr2DObject {
			expected += ArrayCountPrefixSize
			for _, o := range arr2D {
				expected += ObjectSetUnsetPrefixSize + o.Size()
			}
		}

		// Arr3DBool:   [][][]bool{{{true, false}, {false, true}}, {{true, true}, {false, false}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DBool {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*1
			}
		}
		// Arr3DUint8:  [][][]uint8{{{0, 128}, {255, 64}}, {{1, 2}, {3, 4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DUint8 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*1
			}
		}
		// Arr3DInt8:   [][][]int8{{{-128, 0}, {127, -64}}, {{-1, -2}, {-3, -4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DInt8 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*1
			}
		}
		// Arr3DUint16: [][][]uint16{{{0, 1000}, {65535, 500}}, {{1, 2}, {3, 4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DUint16 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*2
			}
		}
		// Arr3DInt16:  [][][]int16{{{-32768, 0}, {32767, -500}}, {{-1, -2}, {-3, -4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DInt16 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*2
			}
		}
		// Arr3DUint32: [][][]uint32{{{0, 1}, {4294967295, 100}}, {{1, 2}, {3, 4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DUint32 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*4
			}
		}
		// Arr3DInt32:  [][][]int32{{{-2147483648, 0}, {2147483647, -100}}, {{-1, -2}, {-3, -4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DInt32 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*4
			}
		}
		// Arr3DUint64: [][][]uint64{{{0, 1}, {math.MaxUint64, 1000}}, {{1, 2}, {3, 4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DUint64 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*8
			}
		}
		// Arr3DInt64:  [][][]int64{{{math.MinInt64, 0}, {math.MaxInt64, -1000}}, {{-1, -2}, {-3, -4}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DInt64 {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*8
			}
		}
		// Arr3DFloat:  [][][]float32{{{-1.5, 0.0}, {1.5, -2.5}}, {{-3.5, 4.5}, {5.5, -6.5}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DFloat {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*4
			}
		}
		// Arr3DDouble: [][][]float64{{{-math.Pi, 0.0}, {math.Pi, -math.E}}, {{-1.0, 2.0}, {3.0, -4.0}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DDouble {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize + len(arr2D)*8
			}
		}
		// Arr3DString: [][][]string{{{"a", "b"}, {"c", "d"}}, {{"e", "f"}, {"g", "h"}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DString {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize
				for _, s := range arr2D {
					expected += StrOrByteLenPrefixSize + len(s)
				}
			}
		}
		// Arr3DBytes:  [][][][]byte{{{{0x01}, {0x02}}, {{0x03}, {0x04, 0x05}}}, {{{0x06}, {0x07}}, {{0x07, 0x08}, {0x09}}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DBytes {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize
				for _, b := range arr2D {
					expected += StrOrByteLenPrefixSize + len(b)
				}
			}
		}
		// Arr3DNested: [][][]*OnlyVariableTypesMsg{{{{Name: "nested1"}, {Name: "nested2"}}, {{Name: "nested3"}, {Name: "nested4"}}},{{{Name: "nested5"}, {Name: "nested6"}}, {{Name: "nested7"}, {Name: "nested8"}}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DNested {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize
				for _, n := range arr2D {
					expected += ObjectSetUnsetPrefixSize + n.Size()
				}
			}
		}
		// Arr3DObject: [][][]*OnlyScalarTypesMsg{{{{ValUint64: 1}, {ValUint64: 2}},{{ValUint64: 3}, {ValUint64: 4}}}, {{{ValUint64: 5}, {ValUint64: 6}}{{ValUint64: 7}, {ValUint64: 8}}}},
		expected += (TypeMarkerSize * 2) + (ArrayCountPrefixSize * 2)
		for _, arr3D := range msg.Arr3DObject {
			expected += ArrayCountPrefixSize
			for _, arr2D := range arr3D {
				expected += ArrayCountPrefixSize
				for _, o := range arr2D {
					expected += ObjectSetUnsetPrefixSize + o.Size()
				}
			}
		}

		fmt.Printf("DynamicPayloadSize: %d\n", expected)
		return expected	// 2059
	}

	t.Run("MessageTypeID()", func(t *testing.T) {
		assert.Equal(t, uint16(5), msg.MessageTypeID())
	})

	t.Run("DynamicPayloadSize()", func(t *testing.T) {
		expected := msgDynSize(msg)
		actual := msg.DynamicPayloadSize()
		assert.Equal(t, expected, actual, "DynamicPayloadSize is not matching")
	})

	t.Run("Size()", func(t *testing.T) {
		expected := FrameHeaderSize + AllTypesOfArraysMsgFixedSize + msgDynSize(msg)
		actual := msg.Size()
		assert.Equal(t, expected, actual, "Size is not matching")
	})

	t.Run("Marshal()", func(t *testing.T) {
		wire, err := msg.Marshal()
		require.NoError(t, err, "Marshal should not fail")

		dynSize := msgDynSize(msg)
		assert.Equal(t, FrameHeaderSize+AllTypesOfArraysMsgFixedSize+dynSize, len(wire), "wire length mismatch")

		assert.Equal(t, uint16(5), binary.BigEndian.Uint16(wire[0:2]), "TypeID mismatch")
		assert.Equal(t, uint16(AllTypesOfArraysMsgFixedSize), binary.BigEndian.Uint16(wire[2:4]), "FixedPayloadLen mismatch")
		assert.Equal(t, uint32(AllTypesOfArraysMsgFixedSize+dynSize), binary.BigEndian.Uint32(wire[4:8]), "OverallPayloadLen mismatch")

		assert.Len(t, wire[FrameHeaderSize:], AllTypesOfArraysMsgFixedSize+dynSize, "fixed block length mismatch")
	})

	t.Run("Unmarshal()", func(t *testing.T) {
		dst, _ := arraysRoundtrip(t, msg)
		assert.Equal(t, msg.Arr1DBool, dst.Arr1DBool)
		assert.Equal(t, msg.Arr1DUint8, dst.Arr1DUint8)
		assert.Equal(t, msg.Arr1DInt8, dst.Arr1DInt8)
		assert.Equal(t, msg.Arr1DUint16, dst.Arr1DUint16)
		assert.Equal(t, msg.Arr1DInt16, dst.Arr1DInt16)
		assert.Equal(t, msg.Arr1DUint32, dst.Arr1DUint32)
		assert.Equal(t, msg.Arr1DInt32, dst.Arr1DInt32)
		assert.Equal(t, msg.Arr1DUint64, dst.Arr1DUint64)
		assert.Equal(t, msg.Arr1DInt64, dst.Arr1DInt64)
		assert.Equal(t, msg.Arr1DFloat, dst.Arr1DFloat)
		assert.Equal(t, msg.Arr1DDouble, dst.Arr1DDouble)
		assert.Equal(t, msg.Arr1DString, dst.Arr1DString)
		assert.Equal(t, msg.Arr1DBytes, dst.Arr1DBytes)
		assert.Equal(t, msg.Arr1DNested, dst.Arr1DNested)
		assert.Equal(t, msg.Arr1DObject, dst.Arr1DObject)

		assert.Equal(t, msg.Arr2DBool, dst.Arr2DBool)
		assert.Equal(t, msg.Arr2DUint8, dst.Arr2DUint8)
		assert.Equal(t, msg.Arr2DInt8, dst.Arr2DInt8)
		assert.Equal(t, msg.Arr2DUint16, dst.Arr2DUint16)
		assert.Equal(t, msg.Arr2DInt16, dst.Arr2DInt16)
		assert.Equal(t, msg.Arr2DUint32, dst.Arr2DUint32)
		assert.Equal(t, msg.Arr2DInt32, dst.Arr2DInt32)
		assert.Equal(t, msg.Arr2DUint64, dst.Arr2DUint64)
		assert.Equal(t, msg.Arr2DInt64, dst.Arr2DInt64)
		assert.Equal(t, msg.Arr2DFloat, dst.Arr2DFloat)
		assert.Equal(t, msg.Arr2DDouble, dst.Arr2DDouble)
		assert.Equal(t, msg.Arr2DString, dst.Arr2DString)
		assert.Equal(t, msg.Arr2DBytes, dst.Arr2DBytes)
		assert.Equal(t, msg.Arr2DNested, dst.Arr2DNested)
		assert.Equal(t, msg.Arr2DObject, dst.Arr2DObject)

		assert.Equal(t, msg.Arr3DBool, dst.Arr3DBool)
		assert.Equal(t, msg.Arr3DUint8, dst.Arr3DUint8)
		assert.Equal(t, msg.Arr3DInt8, dst.Arr3DInt8)
		assert.Equal(t, msg.Arr3DUint16, dst.Arr3DUint16)
		assert.Equal(t, msg.Arr3DInt16, dst.Arr3DInt16)
		assert.Equal(t, msg.Arr3DUint32, dst.Arr3DUint32)
		assert.Equal(t, msg.Arr3DInt32, dst.Arr3DInt32)
		assert.Equal(t, msg.Arr3DUint64, dst.Arr3DUint64)
		assert.Equal(t, msg.Arr3DInt64, dst.Arr3DInt64)
		assert.Equal(t, msg.Arr3DFloat, dst.Arr3DFloat)
		assert.Equal(t, msg.Arr3DDouble, dst.Arr3DDouble)
		assert.Equal(t, msg.Arr3DString, dst.Arr3DString)
		assert.Equal(t, msg.Arr3DBytes, dst.Arr3DBytes)
		assert.Equal(t, msg.Arr3DNested, dst.Arr3DNested)
		assert.Equal(t, msg.Arr3DObject, dst.Arr3DObject)
	})
}

func TestReadMessageFrame(t *testing.T) {
	src := &AllTypesFieldsMsg{ValUint64: 99, Name: "test"}
	wire, err := src.Marshal()
	require.NoError(t, err)
	r := bytes.NewReader(wire)
	typeID, fixedLen, overallLen, err := ReadMessageFrame(r)
	require.NoError(t, err)
	assert.Equal(t, uint16(3), typeID)
	assert.Equal(t, uint16(AllTypesFieldsMsgFixedSize), fixedLen)
	assert.EqualValues(t, int(fixedLen)+len("test"), overallLen)
}

// ---------------------------------------------------------------------------
// Unmarshal Error Tests
// ---------------------------------------------------------------------------

func TestUnmarshalErrorCases(t *testing.T) {
	t.Run("Wire data less then FrameHeader", func(t *testing.T) {
		r := bytes.NewReader([]byte{})
		var msg AllTypesFieldsMsg
		_, fixedLen, overallLen, err := ReadMessageFrame(r)
		assert.Error(t, err, "should fail with empty wire data")
		assert.Equal(t, uint16(0), fixedLen)
		assert.Equal(t, uint32(0), overallLen)
		err = msg.Unmarshal(r, fixedLen, overallLen)
		assert.Error(t, err, "should fail to unmarshal with empty wire data")
	})

	t.Run("Wire data less than fixed block", func(t *testing.T) {
		msg := &AllTypesFieldsMsg{ValUint64: 99, Name: "test"}
		wire, err := msg.Marshal()
		require.NoError(t, err)

		truncated := wire[:FrameHeaderSize+(AllTypesFieldsMsgFixedSize/2)]
		r := bytes.NewReader(truncated)
		_, fixedLen, overallLen, err := ReadMessageFrame(r)
		require.NoError(t, err)

		var dst AllTypesFieldsMsg
		err = dst.Unmarshal(r, fixedLen, overallLen)
		assert.Error(t, err, "should fail to unmarshal with truncated wire data")
	})

	t.Run("Wire data less than dynamic block", func(t *testing.T) {
		msg := &AllTypesFieldsMsg{ValUint64: 99, Name: "test"}
		wire, err := msg.Marshal()
		require.NoError(t, err)

		truncated := wire[:len(wire)-1]
		r := bytes.NewReader(truncated)
		_, fixedLen, overallLen, err := ReadMessageFrame(r)
		require.NoError(t, err)

		var dst AllTypesFieldsMsg
		err = dst.Unmarshal(r, fixedLen, overallLen)
		assert.Error(t, err, "should fail to unmarshal with truncated wire data")
	})

	t.Run("Wire data with invalid length prefix", func(t *testing.T) {
		msg := &AllTypesFieldsMsg{ValUint64: 99, Name: "test"}
		wire, err := msg.Marshal()
		require.NoError(t, err)

		// Corrupt the fixed length prefix to be larger than actual data
		binary.BigEndian.PutUint16(wire[2:4], uint16(len(wire)+10))
		r := bytes.NewReader(wire)
		_, fixedLen, overallLen, err := ReadMessageFrame(r)
		require.NoError(t, err)

		var dst AllTypesFieldsMsg
		err = dst.Unmarshal(r, fixedLen, overallLen)
		assert.Error(t, err, "should fail to unmarshal with invalid length prefix")
	})

	t.Run("Wire data larger then MaxAllowedPacket", func(t *testing.T) {
		msg := &AllTypesFieldsMsg{ValUint64: 99, Name: "test"}
		wire, err := msg.Marshal()
		require.NoError(t, err)

		// Corrupt the overall length prefix to be larger than MaxAllowedPacket
		binary.BigEndian.PutUint32(wire[4:8], MaxAllowedPacket+1)
		r := bytes.NewReader(wire)
		_, fixedLen, overallLen, err := ReadMessageFrame(r)
		require.NoError(t, err)

		var dst AllTypesFieldsMsg
		err = dst.Unmarshal(r, fixedLen, overallLen)
		assert.Error(t, err, "should fail to unmarshal with oversized packet")
	})

	t.Run("Wire data oversized variable field", func(t *testing.T) {
		msg := &OnlyVariableTypesMsg{Name: "test"}
		wire, err := msg.Marshal()
		require.NoError(t, err)
		binary.BigEndian.PutUint32(wire[8:12], MaxAllowedPacket+1) // Corrupt Name length

		r := bytes.NewReader(wire)
		_, fixedLen, overallLen, err := ReadMessageFrame(r)
		require.NoError(t, err)

		var dst OnlyVariableTypesMsg
		err = dst.Unmarshal(r, fixedLen, overallLen)
		assert.Error(t, err, "should fail to unmarshal with oversized variable field")
	})

	t.Run("Empty Reader", func(t *testing.T) {
		r := bytes.NewReader(nil)
		_, _, _, err := ReadMessageFrame(r)
		assert.ErrorIs(t, err, io.EOF)

		var msg OnlyScalarTypesMsg
		err = msg.Unmarshal(r, uint16(OnlyScalarTypesMsgFixedSize), uint32(OnlyScalarTypesMsgFixedSize))
		assert.Error(t, err, "should fail to unmarshal with empty reader")
	})

	t.Run("Forward compatibility with smaller fixedPayloadSize", func(t *testing.T) {
		extra := 8
		extendedFixed := make([]byte, OnlyScalarTypesMsgFixedSize+extra)
		r := bytes.NewReader(extendedFixed)
		var msg OnlyScalarTypesMsg
		err := msg.Unmarshal(r, uint16(OnlyScalarTypesMsgFixedSize+extra), uint32(OnlyScalarTypesMsgFixedSize+extra))
		assert.NoError(t, err, "forward-compatible larger fixedPayloadSize should succeed")
	})
}

// ---------------------------------------------------------------------------
// Benchmarks
// ---------------------------------------------------------------------------

func BenchmarkMarshal_OnlyScalarTypesMsg(b *testing.B) {
	src := &OnlyScalarTypesMsg{
		ValUint64: 0xDEADBEEFCAFEBABE,
		ValInt64:  -1234567890,
		ValDouble: math.Pi,
		ValUint32: 0xABCD,
		ValInt32:  -42,
		ValFloat:  1.5,
		ValUint16: 1000,
		ValInt16:  -500,
		ValUint8:  200,
		ValInt8:   -100,
		ValBool:   true,
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = src.Marshal()
	}
}

func BenchmarkUnmarshal_OnlyScalarTypesMsg(b *testing.B) {
	src := &OnlyScalarTypesMsg{ValUint64: 12345, ValInt64: -67890, ValDouble: math.E}
	wire, _ := src.Marshal()
	fixedLen := binary.BigEndian.Uint16(wire[2:4])
	overallLen := binary.BigEndian.Uint32(wire[4:8])
	payload := wire[FrameHeaderSize:]
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(payload)
		var dst OnlyScalarTypesMsg
		_ = dst.Unmarshal(r, fixedLen, overallLen)
	}
}

func BenchmarkMarshal_OnlyVariableTypesMsg(b *testing.B) {
	src := &OnlyVariableTypesMsg{
		Name:   "benchmark string payload",
		Data:   bytes.Repeat([]byte{0xAB}, 256),
		Tags:   []string{"tag1", "tag2", "tag3", "tag4"},
		Matrix: [][]int32{{1, 2, 3}, {4, 5, 6}},
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = src.Marshal()
	}
}

func BenchmarkRoundtrip_AllTypesFieldsMsg(b *testing.B) {
	src := &AllTypesFieldsMsg{
		ValUint64: 0xDEADBEEF,
		ValInt64:  -999,
		ValDouble: math.Pi,
		ValFloat:  1.5,
		Name:      "benchmark",
		Data:      bytes.Repeat([]byte{1}, 64),
		Tags:      []string{"x", "y"},
		Matrix:    [][]int32{{1, 2}, {3, 4}},
		ValBool:   true,
	}
	wire, _ := src.Marshal()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := bytes.NewReader(wire)
		_, fixedLen, overallLen, _ := ReadMessageFrame(r)
		var dst AllTypesFieldsMsg
		_ = dst.Unmarshal(r, fixedLen, overallLen)
	}
}

func BenchmarkMarshal_AllTypesOfArraysMsg(b *testing.B) {
	src := &AllTypesOfArraysMsg{
		Arr1DInt32:  make([]int32, 100),
		Arr1DString: make([]string, 10),
		Arr2DFloat:  [][]float32{{1, 2, 3}, {4, 5, 6}},
	}
	for i := range src.Arr1DString {
		src.Arr1DString[i] = "item"
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = src.Marshal()
	}
}
