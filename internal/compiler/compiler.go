package compiler

import (
	"sort"
	"strings"
	"unicode"

	"github.com/avatar31/wireforge/internal/schema"
)

// CompiledSchema holds the fully resolved layout information for all messages.
type CompiledSchema struct {
	PackageName string
	Messages    []*CompiledMessage
}

// CompiledMessage holds the computed memory layout for a single message type.
type CompiledMessage struct {
	Name            string
	Fields          []*CompiledField
	FixedFields     []*CompiledField
	VariableFields  []*CompiledField
	TotalFixedSize  int
	StructAlignment int
	PaddingBlocks   int
}

// CompiledField holds the computed layout for a single field.
type CompiledField struct {
	Name          string
	GoName        string
	CName         string
	Description   string
	Type          schema.FieldType
	Offset        int
	Size          int
	Alignment     int
	IsVariable    bool
	PaddingBefore int
}

// Compile takes a parsed schema and computes memory layouts with alignment.
func Compile(s *schema.Schema, packageName string) *CompiledSchema {
	cs := &CompiledSchema{
		PackageName: packageName,
		Messages:    make([]*CompiledMessage, 0, len(s.Messages)),
	}

	for _, msg := range s.Messages {
		cs.Messages = append(cs.Messages, CompileMessage(msg))
	}

	return cs
}

// CompileMessage computes the memory layout for a single message, including offsets, padding, and alignment.
// It automatically optimizes the field order by sorting them by alignment requirement in descending order
// to minimize internal padding gaps before computing the final offsets.
// Then returns a CompiledMessage with all fields properly aligned and sized.
//
// For Example:
//
//	Input Message Fields (from schema):
//	- key: string       (Size: 4, Align: 4) -> uint32 length prefix
//	- count: int32      (Size: 4, Align: 4)
//	- offset: uint64    (Size: 8, Align: 8)
//	- isEOF: bool       (Size: 1, Align: 1)
//
//	Optimized Field Layout (Sorted by Alignment Descending):
//	Field Name | Offset | Size | Alignment | Padding Before
//	--------------------------------------------------------
//	offset     | 0      | 8    | 8         | 0
//	key        | 8      | 4    | 4         | 0
//	count      | 12     | 4    | 4         | 0
//	isEOF      | 16     | 1    | 1         | 0
//	--------------------------------------------------------
//	Total Fixed Size: 24 bytes (17 bytes data + 7 bytes trailing padding)
//	Struct Alignment: 8 bytes
func CompileMessage(msg *schema.Message) *CompiledMessage {
	cm := &CompiledMessage{Name: msg.Name}

	optimizedFields := make([]*schema.Field, len(msg.Fields))
	copy(optimizedFields, msg.Fields)

	// Sort fields by Alignment descending.
	// If alignments are equal, use Size descending as a stable tie-breaker.
	sort.SliceStable(optimizedFields, func(i, j int) bool {
		alignI := optimizedFields[i].Type.Alignment()
		alignJ := optimizedFields[j].Type.Alignment()
		if alignI == alignJ {
			return optimizedFields[i].Type.Size() > optimizedFields[j].Type.Size()
		}
		return alignI > alignJ
	})

	offset := 0
	maxAlign := 4 // Minimum message size is 4 bytes

	// Process the optimally ordered fields
	for _, field := range optimizedFields {
		cf := &CompiledField{
			Name:        field.Name,
			GoName:      toGoName(field.Name),
			CName:       toSnakeCase(field.Name),
			Description: field.Description,
			Type:        field.Type,
			Size:        field.Type.Size(),
			Alignment:   field.Type.Alignment(),
			IsVariable:  field.IsVariable,
		}

		align := cf.Alignment
		if align > maxAlign {
			maxAlign = align
		}

		// Calculate padding needed before this field to satisfy alignment
		padding := (align - (offset % align)) % align
		cf.PaddingBefore = padding
		cf.Offset = offset + padding
		offset = cf.Offset + cf.Size

		if padding > 0 {
			cm.PaddingBlocks++
		}

		cm.Fields = append(cm.Fields, cf)
		if cf.IsVariable {
			cm.VariableFields = append(cm.VariableFields, cf)
		}
		cm.FixedFields = append(cm.FixedFields, cf)
	}

	trailingPad := (maxAlign - (offset % maxAlign)) % maxAlign
	if trailingPad > 0 {
		cm.PaddingBlocks++
	}
	offset += trailingPad

	cm.TotalFixedSize = offset
	cm.StructAlignment = maxAlign

	return cm
}

func toGoName(name string) string {
	parts := splitName(name)
	var b strings.Builder
	for _, p := range parts {
		if len(p) == 0 {
			continue
		}
		upper := strings.ToUpper(p)
		switch upper {
		case "ID", "URL", "URI", "HTTP", "HTTPS", "API", "IP", "TCP", "UDP", "DNS":
			b.WriteString(upper)
		default:
			b.WriteRune(unicode.ToUpper(rune(p[0])))
			b.WriteString(p[1:])
		}
	}
	return b.String()
}

func toSnakeCase(name string) string {
	if strings.Contains(name, "_") && name == strings.ToLower(name) {
		return name
	}
	var b strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) {
			if i > 0 {
				b.WriteByte('_')
			}
			b.WriteRune(unicode.ToLower(r))
		} else {
			b.WriteRune(r)
		}
	}
	return b.String()
}

func splitName(name string) []string {
	if strings.Contains(name, "_") {
		return strings.Split(name, "_")
	}
	var parts []string
	var current strings.Builder
	for i, r := range name {
		if unicode.IsUpper(r) && i > 0 {
			parts = append(parts, current.String())
			current.Reset()
			current.WriteRune(unicode.ToLower(r))
		} else {
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}
