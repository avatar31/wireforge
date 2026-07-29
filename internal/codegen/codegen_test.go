// Copyright (c) 2026 Sachin S. All rights reserved.
//
// Licensed under the MIT License.
// See LICENSE in the project root.

package codegen

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/avatar31/wireforge/internal/compiler"
	"github.com/avatar31/wireforge/internal/schema"
)

const (
	sampleYamlWithPadding = `
openapi: "3.1.0"
info:
  version: "1.0.0"
  title: Padding & Alignment Test Schema
  description: >
    This schema is intentionally crafted to exercise all distinct padding
    scenarios in the wireforge compiler and code generator. Each message
    represents a different padding outcome so that tests can assert precise
    struct-layout properties in both the generated Go and C output.
paths: {}
components:
  schemas:

    # ── Case 1: Zero padding ──────────────────────────────────────────────────
    # A single int64 field (8 bytes, align-8).  The struct size is already a
    # multiple of 8, so no trailing padding is required.
    # Expected: TotalFixedSize=8, PaddingBlocks=0
    # C struct : { int64_t value; }   — no _padN field
    NoPaddingMsg:
      type: object
      x-message-id: 1
      properties:
        value:
          description: A single 8-byte field; no padding required.
          type: integer
          format: int64

    # ── Case 2: 1-byte trailing pad ──────────────────────────────────────────
    # Three 1-byte fields (uint8 + int8 + bool = 3 bytes).  The minimum struct
    # alignment is 4, so 1 trailing byte must be added.
    # Compiler sort order by FieldType index: uint8(12), int8(13), bool(14).
    # Expected: TotalFixedSize=4, PaddingBlocks=1
    # C struct : { uint8_t alpha; int8_t beta; uint8_t gamma; uint8_t _pad0[1]; }
    SmallFieldsMsg:
      type: object
      x-message-id: 2
      properties:
        alpha:
          description: First byte field.
          type: integer
          format: uint8
        beta:
          description: Second byte field (signed).
          type: integer
          format: int8
        gamma:
          description: Boolean field (1 byte).
          type: boolean

    # ── Case 3: 7-byte trailing pad ──────────────────────────────────────────
    # int64 (8 bytes, align-8) followed by bool (1 byte, align-1).
    # After bool the offset is 9; the struct alignment is 8, so 7 trailing bytes
    # must be appended to reach offset 16.
    # Compiler sort order: int64(1) before bool(14).
    # Expected: TotalFixedSize=16, PaddingBlocks=1
    # C struct : { int64_t big; uint8_t flag; uint8_t _pad0[7]; }
    MixedSizeMsg:
      type: object
      x-message-id: 3
      properties:
        big:
          description: 8-byte integer field.
          type: integer
          format: int64
        flag:
          description: 1-byte boolean field.
          type: boolean

    # ── Case 4: Perfect alignment — zero padding ─────────────────────────────
    # int32 + float32 + string (length-prefix) are all 4-byte fields (align-4).
    # Total = 12 bytes, which is already a multiple of 4.
    # Compiler sort order: int32(4), float32(5), string(6) — all same size group.
    # Expected: TotalFixedSize=12, PaddingBlocks=0
    # C struct : { int32_t count; float rate; uint32_t name_len; char *name; }
    AlignedMsg:
      type: object
      x-message-id: 4
      properties:
        count:
          description: 4-byte integer field.
          type: integer
          format: int32
        rate:
          description: 4-byte float field.
          type: number
          format: float
        name:
          description: Variable-length string (4-byte length prefix in fixed block).
          type: string
`
	sampleYaml = `
openapi: 3.1.0
info:
  version: 1.0.0
  title: Chatapp Message Schemas
paths: {}
components:
  schemas:
    UserMessage:
      type: object
      x-message-id: 1
      properties:
        content:
          type: string
        timestamp:
          type: integer
          format: int64
        attachment:
          type: string
          format: binary
    HeartbeatMessage:
      type: object
      x-message-id: 2
      properties:
        timestamp:
          type: integer
          format: int64
`
)

func compileYaml(t *testing.T, yamlContent, packageName string) *compiler.CompiledSchema {
	t.Helper()

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "openapi_test.yaml")
	err := os.WriteFile(tmpFile, []byte(strings.TrimSpace(yamlContent)), 0644)
	require.NoError(t, err, "WriteFile(%s)", tmpFile)

	s, err := schema.ParseFile(tmpFile)
	require.NoError(t, err, "ParseFile(%s)", filepath.Base(tmpFile))
	cs, err := compiler.Compile(s, packageName)
	require.NoError(t, err, "Compile(%s)", filepath.Base(tmpFile))
	return cs
}

// TestPaddingAlignment_CPadFieldsPresent checks that padded C structs have
// explicit uint8_t _pad0[N] declarations.
func TestPaddingAlignment_CPadFieldsPresent(t *testing.T) {
	cs := compileYaml(t, sampleYamlWithPadding, "chatapp")
	var buf bytes.Buffer
	err := GenerateCHeader(&buf, cs)
	require.NoError(t, err, "GenerateCHeader")

	cHeaderCode := buf.String()

	// These messages must have a trailing pad.
	for _, tc := range []struct {
		msg string
		pad string
	}{
		{"SmallFieldsMsg", "uint8_t _pad0[1]"},
		{"MixedSizeMsg", "uint8_t _pad0[7]"},
	} {
		t.Run(tc.msg+"_hasPad", func(t *testing.T) {
			assert.Contains(t, cHeaderCode, tc.pad,
				"C header must declare %s for %s", tc.pad, tc.msg)
		})
	}

	// These messages must NOT have any pad field.
	for _, msgName := range []string{"NoPaddingMsg", "AlignedMsg"} {
		msgName := msgName
		t.Run(msgName+"_noPad", func(t *testing.T) {
			structMarker := fmt.Sprintf("struct %s {", snakeLower(msgName))
			start := strings.Index(cHeaderCode, structMarker)
			if start < 0 {
				t.Fatalf("struct %s not found in C header", snakeLower(msgName))
			}
			end := strings.Index(cHeaderCode[start:], "}")
			if end >= 0 {
				block := cHeaderCode[start : start+end]
				// Use "uint8_t _pad" to avoid matching struct names containing "_pad" (e.g. no_padding_msg).
				assert.NotContains(t, block, "uint8_t _pad",
					"C struct for %s must not contain any uint8_t _padN field", msgName)
			}
		})
	}
}

func TestGenerateCHeader(t *testing.T) {
	t.Run("Test C Header Generation", func(t *testing.T) {
		cs := compileYaml(t, sampleYaml, "chatapp")

		var buf bytes.Buffer
		err := GenerateCHeader(&buf, cs)
		require.NoError(t, err, "GenerateCHeader")

		cHeaderCode := buf.String()

		// Constant definitions
		assert.Contains(t, cHeaderCode, "#define USER_MESSAGE_TYPE_ID 1", "C header must contain type ID define for UserMessage")
		assert.Contains(t, cHeaderCode, "#define HEARTBEAT_MESSAGE_TYPE_ID 2", "C header must contain type ID define for HeartbeatMessage")
		assert.Contains(t, cHeaderCode, "#define USER_MESSAGE_FIXED_SIZE 16", "C header must contain fixed size define for UserMessage")
		assert.Contains(t, cHeaderCode, "#define HEARTBEAT_MESSAGE_FIXED_SIZE 8", "C header must contain fixed size define for HeartbeatMessage")

		// Message structs and typedefs
		assert.Contains(t, cHeaderCode, "struct user_message {", "C header must contain struct for UserMessage")
		assert.Contains(t, cHeaderCode, "struct heartbeat_message {", "C header must contain struct for HeartbeatMessage")
		assert.Contains(t, cHeaderCode, "typedef struct user_message user_message_t;", "C header must contain typedef for UserMessage")
		assert.Contains(t, cHeaderCode, "typedef struct heartbeat_message heartbeat_message_t;", "C header must contain typedef for HeartbeatMessage")

		// _Static_assert for struct sizes
		assert.Contains(t, cHeaderCode, "_Static_assert(sizeof(user_message_t) >= 16,", "C header must contain _Static_assert for UserMessage size")
		assert.Contains(t, cHeaderCode, "_Static_assert(sizeof(heartbeat_message_t) >= 8,", "C header must contain _Static_assert for HeartbeatMessage size")

		// _set_ functions
		assert.Contains(t, cHeaderCode, "void user_message_t_set_content(", "C header must contain set function for UserMessage content")
		assert.Contains(t, cHeaderCode, "void user_message_t_set_timestamp(", "C header must contain set function for UserMessage timestamp")
		assert.Contains(t, cHeaderCode, "void user_message_t_set_attachment(", "C header must contain set function for UserMessage attachment")
		assert.Contains(t, cHeaderCode, "void heartbeat_message_t_set_timestamp(", "C header must contain set function for HeartbeatMessage timestamp")

		// _dynamic_payload_size functions
		assert.Contains(t, cHeaderCode, "size_t user_message_t_dynamic_payload_size(", "C header must contain dynamic payload size function for UserMessage")
		assert.Contains(t, cHeaderCode, "size_t heartbeat_message_t_dynamic_payload_size(", "C header must contain dynamic payload size function for HeartbeatMessage")

		// _size functions
		assert.Contains(t, cHeaderCode, "size_t user_message_t_size(", "C header must contain size function for UserMessage")
		assert.Contains(t, cHeaderCode, "size_t heartbeat_message_t_size(", "C header must contain size function for HeartbeatMessage")

		// _marshal functions
		assert.Contains(t, cHeaderCode, "int user_message_t_marshal(", "C header must contain marshal function for UserMessage")
		assert.Contains(t, cHeaderCode, "int heartbeat_message_t_marshal(", "C header must contain marshal function for HeartbeatMessage")

		// _unmarshal functions
		assert.Contains(t, cHeaderCode, "int user_message_t_unmarshal(", "C header must contain unmarshal function for UserMessage")
		assert.Contains(t, cHeaderCode, "int heartbeat_message_t_unmarshal(", "C header must contain unmarshal function for HeartbeatMessage")

		// _free functions
		assert.Contains(t, cHeaderCode, "void user_message_t_free(", "C header must contain free function for UserMessage")
		assert.Contains(t, cHeaderCode, "void heartbeat_message_t_free(", "C header must contain free function for HeartbeatMessage")
	})
}

func TestGenerateC(t *testing.T) {
	t.Run("Test C Generation", func(t *testing.T) {
		cs := compileYaml(t, sampleYaml, "chatapp")

		var buf bytes.Buffer
		err := GenerateC(&buf, cs)
		require.NoError(t, err, "GenerateC")

		cCode := buf.String()

		// _set_ functions
		assert.Contains(t, cCode, "void user_message_t_set_content(", "C header must contain set function for UserMessage content")
		assert.Contains(t, cCode, "void user_message_t_set_timestamp(", "C header must contain set function for UserMessage timestamp")
		assert.Contains(t, cCode, "void user_message_t_set_attachment(", "C header must contain set function for UserMessage attachment")
		assert.Contains(t, cCode, "void heartbeat_message_t_set_timestamp(", "C header must contain set function for HeartbeatMessage timestamp")

		// _dynamic_payload_size functions
		assert.Contains(t, cCode, "size_t user_message_t_dynamic_payload_size(", "C header must contain dynamic payload size function for UserMessage")
		assert.Contains(t, cCode, "size_t heartbeat_message_t_dynamic_payload_size(", "C header must contain dynamic payload size function for HeartbeatMessage")

		// _size functions
		assert.Contains(t, cCode, "size_t user_message_t_size(", "C header must contain size function for UserMessage")
		assert.Contains(t, cCode, "size_t heartbeat_message_t_size(", "C header must contain size function for HeartbeatMessage")

		// _marshal functions
		assert.Contains(t, cCode, "int user_message_t_marshal(", "C header must contain marshal function for UserMessage")
		assert.Contains(t, cCode, "int heartbeat_message_t_marshal(", "C header must contain marshal function for HeartbeatMessage")

		// _unmarshal functions
		assert.Contains(t, cCode, "int user_message_t_unmarshal(", "C header must contain unmarshal function for UserMessage")
		assert.Contains(t, cCode, "int heartbeat_message_t_unmarshal(", "C header must contain unmarshal function for HeartbeatMessage")

		// _free functions
		assert.Contains(t, cCode, "void user_message_t_free(", "C header must contain free function for UserMessage")
		assert.Contains(t, cCode, "void heartbeat_message_t_free(", "C header must contain free function for HeartbeatMessage")
	})
}

func TestGenerateGo(t *testing.T) {
	t.Run("Test Go Generation", func(t *testing.T) {
		cs := compileYaml(t, sampleYaml, "chatapp")

		var buf bytes.Buffer
		err := GenerateGo(&buf, cs)
		require.NoError(t, err, "GenerateGo")

		goCode := buf.String()

		// Constants
		assert.Contains(t, goCode, "const UserMessageFixedSize = 16", "Go output must contain fixed size constant for UserMessage")
		assert.Contains(t, goCode, "const HeartbeatMessageFixedSize = 8", "Go output must contain fixed size constant for HeartbeatMessage")

		// Structs
		assert.Contains(t, goCode, "type UserMessage struct {", "Go output must contain struct for UserMessage")
		assert.Contains(t, goCode, "type HeartbeatMessage struct {", "Go output must contain struct for HeartbeatMessage")

		// DynamicPayloadSize()
		assert.Contains(t, goCode, "func (u *UserMessage) DynamicPayloadSize() int {", "Go output must contain DynamicPayloadSize method for UserMessage")
		assert.Contains(t, goCode, "func (h *HeartbeatMessage) DynamicPayloadSize() int {", "Go output must contain DynamicPayloadSize method for HeartbeatMessage")

		// Size()
		assert.Contains(t, goCode, "func (u *UserMessage) Size() int {", "Go output must contain Size method for UserMessage")
		assert.Contains(t, goCode, "func (h *HeartbeatMessage) Size() int {", "Go output must contain Size method for HeartbeatMessage")

		// MessageTypeID()
		assert.Contains(t, goCode, "func (u *UserMessage) MessageTypeID() uint16 {", "Go output must contain MessageTypeID method for UserMessage")
		assert.Contains(t, goCode, "func (h *HeartbeatMessage) MessageTypeID() uint16 {", "Go output must contain MessageTypeID method for HeartbeatMessage")

		// Marshal()
		assert.Contains(t, goCode, "func (u *UserMessage) Marshal() ([]byte, error) {", "Go output must contain Marshal method for UserMessage")
		assert.Contains(t, goCode, "func (h *HeartbeatMessage) Marshal() ([]byte, error) {", "Go output must contain Marshal method for HeartbeatMessage")

		// Unmarshal()
		assert.Contains(t, goCode, "func (u *UserMessage) Unmarshal(reader io.Reader,\n\tfixedPayloadSize uint16, _ uint32) error {", "Go output must contain Unmarshal method for UserMessage")
		assert.Contains(t, goCode, "func (h *HeartbeatMessage) Unmarshal(reader io.Reader,\n\tfixedPayloadSize uint16, _ uint32) error {", "Go output must contain Unmarshal method for HeartbeatMessage")

		assert.NotContains(t, goCode, "_pad", "Go structs must not contain explicit _padN fields; Go handles alignment natively")
	})
}
