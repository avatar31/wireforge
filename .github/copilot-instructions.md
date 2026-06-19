# Copilot Instructions

> **See also:** [`AGENTS.md`](../AGENTS.md) for operational guidelines.

---

## Project Overview

**wireforge** (`github.com/avatar31/wireforge`) is a schema-driven code generator that takes an OpenAPI YAML specification and produces fully self-contained Go and C files implementing a length-prefixed binary wire protocol. It handles alignment, padding, Big-Endian encoding, short-read safety, and memory lifecycle management automatically.

- **Language:** Go 1.26+
- **License:** MIT
- **Author:** Sachin S
- **Module path:** `github.com/avatar31/wireforge`

---

## Key Dependencies

| Package | Purpose |
|---|---|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/getkin/kin-openapi` | OpenAPI YAML parsing |
| `text/template` (stdlib) | Code generation templates |

---

## Architecture

```
OpenAPI YAML
      |
      v
  schema.ParseFile()        <- internal/schema/parser.go
      |                        Validates types, rejects generics
      v
  compiler.Compile()        <- internal/compiler/compiler.go
      |                        Computes offsets, padding, alignment
      v
  codegen.GenerateGo()      <- internal/codegen/go_code_template.go
  codegen.GenerateCHeader() <- internal/codegen/c_header_template.go
  codegen.GenerateC()       <- internal/codegen/c_code_template.go
      |
      v
  output/
    go/
      messages.go             Go structs + Marshal/Unmarshal
    c/
      messages.h              C typedefs + function prototypes
      messages.c              C implementation (stream I/O, free)
```

### Component Responsibilities

| File | Responsibility |
|---|---|
| `main.go` | CLI entry point (cobra); wires parser -> compiler -> codegen pipeline |
| `internal/schema/types.go` | `FieldType` enum with Size/Alignment/GoType/CType methods |
| `internal/schema/parser.go` | OpenAPI YAML parser; type validation firewall |
| `internal/compiler/compiler.go` | Alignment engine: offset/padding/struct-size calculation |
| `internal/codegen/go_code_template.go` | Go template: structs, Marshal, Unmarshal, init() validation etc. |
| `internal/codegen/c_header_template.go` | C header template: typedefs, _Static_assert, prototypes etc. |
| `internal/codegen/c_code_template.go` | C template: marshal, unmarshal, free etc. |

---

## Wire Protocol Layout

Every message on the wire follows this frame structure:

```
+--------+------------------+--------------------+------------------+
| Offset | Field            | Type               | Description      |
+--------+------------------+--------------------+------------------+
| 0      | Message Type ID  | uint16 Big-Endian  | Codec selector   |
| 2      | Fixed Header Len | uint16 Big-Endian  | Fixed block size |
| 4      | Fixed Header     | raw bytes (padded) | Primitives+lens  |
| 4+N    | Dynamic Payload  | raw bytes          | Strings/blobs    |
+--------+------------------+--------------------+------------------+
```

- Fixed header contains all fixed-width fields + uint32 length prefixes for variable fields
- Dynamic payload is concatenated variable-length data in field order
- All multi-byte integers are Big-Endian (network byte order)
- Padding is inserted between fields for natural alignment

---

## Supported Type Mappings

| OpenAPI Type | Format | Go Type | C Type | Size | Alignment |
|---|---|---|---|---|---|
| `integer` | `int8` | `int8` | `int8_t` | 1 | 1 |
| `integer` | `uint8` | `uint8` | `uint8_t` | 1 | 1 |
| `integer` | `int16` | `int16` | `int16_t` | 2 | 2 |
| `integer` | `uint16` | `uint16` | `uint16_t` | 2 | 2 |
| `integer` | `int32` / (empty) | `int32` | `int32_t` | 4 | 4 |
| `integer` | `uint32` | `uint32` | `uint32_t` | 4 | 4 |
| `integer` | `int64` | `int64` | `int64_t` | 8 | 8 |
| `integer` | `uint64` | `uint64` | `uint64_t` | 8 | 8 |
| `number` | `float` / (empty) | `float32` | `float` | 4 | 4 |
| `number` | `double` | `float64` | `double` | 8 | 8 |
| `boolean` | — | `bool` | `uint8_t` | 1 | 1 |
| `string` | — | `string` | `uint32_t len + char*` | 4 (prefix) | 4 |
| `string` | `binary` / `byte` | `[]byte` | `uint32_t len + uint8_t*` | 4 (prefix) | 4 |

---

## Safety Features (Edge Cases Handled)

1. **Alignment/Padding:** Compiler inserts explicit `_padN` fields; no `__attribute__((packed))`
2. **Endianness:** All wire data is Big-Endian via explicit byte manipulation
3. **Malloc Bombs:** `MAX_ALLOWED_PACKET` (16 MB) validated before every allocation
4. **Memory Leaks:** Every message type has a generated `*_free()` function
5. **Generic/Untyped Fields:** Rejected at parse time with a clear error message

---

## CLI Usage

```bash
./wireforge -i <schema.yaml> -o <output_dir> [-p <package_name>]
```

| Flag | Short | Default | Description |
|---|---|---|---|
| `--in` | `-i` | (required) | Input OpenAPI YAML schema file |
| `--out` | `-o` | `./out` | Output directory for generated files |
| `--package` | `-p` | `messages` | Go package name for generated code |

---

## Build & Test Commands

```bash
# Build the binary
go build -o wireforge .

# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Vet
go vet ./...
```

---

## Project Structure

```
wireforge/
+-- main.go                             # CLI entry point
+-- go.mod / go.sum
+-- resoruces/
|   +-- fsal.yaml                       # Example OpenAPI schema
+-- internal/
    +-- schema/
    |   +-- types.go                    # FieldType enum, size/alignment/type methods
    |   +-- parser.go                   # OpenAPI YAML parser, type validation
    +-- compiler/
    |   +-- compiler.go                 # Alignment engine, offset/padding calculation
    +-- codegen/
        +-- go_code_template.go         # Go code generation template
        +-- c_header_template.go        # C header generation template
        +-- c_code_template.go          # C implementation generation template
+-- examples/
    +-- client-server/                  # Example of single socket client-server chatapp using generated code
    +-- peer-to-peer/                   # Example of dual socket peer-to-peer chatapp using generated code
```

---

## Key Design Decisions

1. **Deterministic field order:** Messages are sorted alphabetically by property name for reproducible output across runs. Fields are ordered in the generated struct by their size first then name. This ensures that the largest fields are aligned first, minimizing padding and maximizing performance on all architectures.

2. **No `__attribute__((packed))`:** Explicit padding fields instead, so the struct compiles identically on any C compiler without vendor extensions.

3. **`_Static_assert` in C header:** Catches layout mismatches at compile time rather than causing silent data corruption at runtime.

4. **`init()` validation in Go:** Uses `unsafe.Sizeof` to verify the Go compiler agrees with our computed field sizes at program startup.

5. **Forward compatibility:** Unmarshal accepts a `fixedHeaderLen` from the wire, allowing newer senders to add trailing fields that older receivers skip gracefully.

---

## Template Scoping Convention

Inside Go templates, `$msg` is used to capture the current message in `{{range .Messages}}` loops. This avoids the common pitfall where `$` (root context) is used inside nested ranges, which would reference `CompiledSchema` instead of the current `CompiledMessage`.

---

## Active TODOs

**P0:**
- No support for nested object or array types (only primitives + strings/blobs)

**P1:**
- Add `--format` flag for gofmt/clang-format post-processing

**P2:**
- Protocol version negotiation in frame header
- Message registry with dispatch table generation
- Support backwards/forward compatibility
