// Copyright (c) 2026 Sachin S. All rights reserved.
// 
// Licensed under the MIT License.
// See LICENSE in the project root.

package codegen

import (
	"fmt"

	"github.com/avatar31/wireforge/internal/compiler"
)

type padFieldEntry struct {
	IsPadding bool                    // Discriminator flag: true if entry is a filler block
	PadName   string                  // Code-ready identifier name (e.g., "_pad0")
	PadSize   int                     // Size of the array reservation in bytes
	Field     *compiler.CompiledField // Reference to the target metadata (nil if IsPadding is true)
}

// generatePadFields interleaves explicit padding blocks into the message field stream.
// It transforms a flat slice of compiled fields into a sequence of data fields
// and filler byte arrays (_padN) based on layout offsets computed by the compiler.
//
// This translation enforces two critical system constraints:
//  1. Inter-Field Alignment: Inserts bytes before a field so its start offset
//     matches its natural CPU boundary (e.g., ensuring a uint64_t falls on an
//     8-byte divisible memory address), eliminating CPU alignment penalties.
//  2. Trailing Structural Padding: Appends trailing bytes if the final field layout
//     ends before hitting the total message structure boundary (TotalFixedSize).
//     This guarantees that if these structures are arranged sequentially in memory
//     (like an array), subsequent elements remain perfectly aligned.
//
// The resulting entries provide the code generator template with a deterministic,
// compiler-agnostic layout that ensures identical memory definitions across both
// Go and C target environments.
func generatePadFields(msg *compiler.CompiledMessage) []padFieldEntry {
	var entries []padFieldEntry
	padIdx := 0
	for _, f := range msg.Fields {
		if f.PaddingBefore > 0 {
			entries = append(entries, padFieldEntry{
				IsPadding: true,
				PadName:   fmt.Sprintf("_pad%d", padIdx),
				PadSize:   f.PaddingBefore,
			})
			padIdx++
		}
		entries = append(entries, padFieldEntry{
			IsPadding: false,
			Field:     f,
		})
	}
	totalWithoutTrailing := 0
	if len(msg.Fields) > 0 {
		last := msg.Fields[len(msg.Fields)-1]
		totalWithoutTrailing = last.Offset + last.Size
	}
	if totalWithoutTrailing < msg.TotalFixedSize {
		entries = append(entries, padFieldEntry{
			IsPadding: true,
			PadName:   fmt.Sprintf("_pad%d", padIdx),
			PadSize:   msg.TotalFixedSize - totalWithoutTrailing,
		})
	}
	return entries
}
