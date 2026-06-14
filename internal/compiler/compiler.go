package compiler

import (
	"fmt"
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
func Compile(s *schema.Schema, packageName string) (*CompiledSchema, error) {
	cs := &CompiledSchema{
		PackageName: packageName,
		Messages:    make([]*CompiledMessage, 0, len(s.Messages)),
	}

	for _, msg := range s.Messages {
		cm, err := compileMessage(msg)
		if err != nil {
			return nil, fmt.Errorf("compiling message %q: %w", msg.Name, err)
		}
		cs.Messages = append(cs.Messages, cm)
	}

	return cs, nil
}

func compileMessage(msg *schema.Message) (*CompiledMessage, error) {
	cm := &CompiledMessage{Name: msg.Name}

	offset := 0
	maxAlign := 1

	for _, field := range msg.Fields {
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

	return cm, nil
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
