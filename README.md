# wireforge

**Forge automated, alignment-safe, length-prefixed binary protocols for C and Go from a single schema.**

`wireforge` is a code generator that reads an OpenAPI YAML schema and produces production-ready Go and C code for a custom binary wire protocol. The generated code handles alignment, padding, Big-Endian encoding, short-read safety, and memory lifecycle — you define messages once and get correct serialization for both languages.

[![](https://img.shields.io/github/v/tag/avatar31/wireforge?color=blue\&labelColor=black\&logo=github\&style=flat-square)](https://github.com/avatar31/wireforge/releases)
[![](https://img.shields.io/github/issues/avatar31/wireforge?labelColor=black\&style=flat-square\&color=blue)](https://github.com/avatar31/wireforge/issues)
[![](https://img.shields.io/badge/license-MIT-blue?labelColor=black\&style=flat-square)](LICENSE)


## Why wireforge?

When building networked systems — game servers, storage engines, distributed databases, IoT gateways — you inevitably need a binary protocol. The alternatives are:

| Approach | Downside |
|---|---|
| Hand-rolled `struct` + `send()` | Endianness bugs, alignment traps, no safety checks, duplicated code across C/Go |
| Protobuf / FlatBuffers | Runtime dependencies, schema compilation toolchains, varint overhead, not POSIX-socket-friendly |
| JSON/MessagePack | Parsing overhead, no fixed offsets, garbage pressure in hot paths |

wireforge sits in the sweet spot: **zero runtime dependencies**, **deterministic wire layout**, **compile-time safety checks**, and **generated code you can read and audit**.


## Table of Contents
- [Features](#features)
- [Installation](#installation)
- [CLI Reference](#cli-reference)
- [Wire Protocol Format](#wire-protocol-format)
- [Alignment & Padding](#alignment--padding)
- [Schema Writing Guide](#schema-writing-guide)
  - [Rules](#rules)
  - [Example](#example)
  - [Type Mappings](#type-mappings)
  - [C-to-Go Interop](#c-to-go-interop)
- [Generated Code Details](#generated-code-details)
  - [Go Output](#go-output-messagesgo)
  - [C Output](#c-output-messagesh--messagesc)
- [Example Usage](#example-usage)
  - [Define your schema](#1-define-your-schema)
  - [Generate code](#2-generate-code)
  - [Use in Go](#3-use-in-go)
  - [Use in C](#4-use-in-c)


## Features

- **Single schema, two languages** — One OpenAPI YAML produces both Go and C code
- **Natural alignment** — Explicit padding fields; no `__attribute__((packed))` or compiler-specific tricks
- **Big-Endian wire format** — Network byte order everywhere; works across architectures
- **Memory lifecycle** — Every C message type gets a `*_free()` function; no leaks
- **Compile-time verification** — C: `_Static_assert` on struct sizes; Go: `init()` with `unsafe.Sizeof`
- **Deterministic output** — Messages are sorted alphabetically by property name for reproducible output across runs. Fields are ordered in the generated struct by their size first then name. This ensures that the largest fields are aligned first, minimizing padding and maximizing performance on all architectures.


## Installation

```bash
go install github.com/avatar31/wireforge@latest
```

Or build from source:

```bash
git clone https://github.com/avatar31/wireforge.git
cd wireforge
go build -o wireforge .
```


## CLI Reference

```
wireforge -i <schema.yaml> -o <output_dir> [-p <package_name>]
```

| Flag | Short | Default | Description |
|---|---|---|---|
| `--in` | `-i` | *(required)* | Input OpenAPI YAML schema file |
| `--out` | `-o` | `./out` | Output directory for generated files |
| `--package` | `-p` | `messages` | Go package name for generated code |


## Wire Protocol Format

Every wireforge message uses this frame layout:

```
 Byte 0    1    2    3    4          4+N        4+N+M
      +----+----+----+-----+----------+-----------+
      | Type ID |  Hdr Len |  Fixed   | Dynamic   |
      | (u16 BE)| (u16 BE) | mHeader  | Payload   |
      +---------+----------+----------+-----------+
```

| Section | Size | Contents |
|---|---|---|
| **Type ID** | 2 bytes | Message type selector (Big-Endian uint16) |
| **Fixed Header Length** | 2 bytes | Size of fixed block (Big-Endian uint16) |
| **Fixed Header** | N bytes | All fixed-width fields + uint32 length prefixes for variable fields, naturally aligned with padding |
| **Dynamic Payload** | M bytes | Concatenated variable-length data (strings, byte arrays) in field order |


## Alignment & Padding

`wireforge` computes natural alignment for each field and inserts explicit padding bytes where needed. It has following precedence order for field layout:
1. **8 byte fields** - `uint64`, `int64`, `double`
2. **4 byte fields** - `uint32`, `int32`, `float`and `string`, `bytes` (uint32 length prefix)
3. **2 byte fields** - `uint16`, `int16`
4. **1 byte fields** - `uint8`, `int8`, `bool`

**For example:**

```
Schema:
  uuid: string
  flag: uint8
  name: string
  count: uint16
  attachment: binary
  timestamp: int64
```

| Field | Offset | Size | Padding |
|---|---|---|---|
| timestamp | 0 | 8 | 0 |
| name | 8 | 4 | 0 |
| uuid | 12 | 4 | 0 |
| attachment | 16 | 4 | 0 |
| count | 20 | 2 | 0 |
| flag | 22 | 1 | 1 (to align next field) |


This schema uses 24 bytes (23 bytes for actual data and 1 byte for padding alignment) and guarantees:
- **No unaligned memory access** on any CPU architecture (ARM, MIPS, x86)
- **Identical layout** between Go and C without compiler-specific attributes
- **Deterministic padding** — the wire bytes are always the same regardless of host compiler


## Schema Writing Guide

`wireforge` parses standard OpenAPI 3.1.0 `components/schemas` sections. Each schema becomes a message type.

> [!NOTE]
>
> `wireforge` does not support validation of field values like `minimum`, `maximum`, `pattern`, or `enum`. It only generates serialization code. You must implement any validation logic in your application. 

### Rules
- Each schema must have a unique `x-message-id` integer (0-65535) to identify the message type on the wire.
- Supported field types: `integer` (with `format`), `number` (with `format`), `boolean`, `string` (with optional `format: binary`).
- No nested objects or arrays allowed; all fields must be primitive types. For variable-length data, use `string` with `format: binary` for byte arrays or `string` for text.
- Field names must be unique within a schema. The generated struct fields will be ordered by size (largest first) to minimize padding, then alphabetically by name for deterministic output.


### Example

```yaml
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
```

### Type Mappings

| OpenAPI Type | Format | Go Type | C Type | Wire Size | Alignment |
|---|---|---|---|---|---|
| `integer` | `int8` | `int8` | `int8_t` | 1 | 1 |
| `integer` | `uint8` | `uint8` | `uint8_t` | 1 | 1 |
| `integer` | `int16` | `int16` | `int16_t` | 2 | 2 |
| `integer` | `uint16` | `uint16` | `uint16_t` | 2 | 2 |
| `integer` | `int32` | `int32` | `int32_t` | 4 | 4 |
| `integer` | `uint32` | `uint32` | `uint32_t` | 4 | 4 |
| `integer` | `int64` | `int64` | `int64_t` | 8 | 8 |
| `integer` | `uint64` | `uint64` | `uint64_t` | 8 | 8 |
| `number` | `float` | `float32` | `float` | 4 | 4 |
| `number` | `double` | `float64` | `double` | 8 | 8 |
| `boolean` | — | `bool` | `uint8_t` | 1 | 1 |
| `string` | — | `string` | `uint32_t` len + `char*` | 4 (prefix) | 4 |
| `string` | `binary` | `[]byte` | `uint32_t` len + `uint8_t*` | 4 (prefix) | 4 |

### C-to-Go Interop

Since both languages produce **identical wire bytes**, you can freely mix:
- C client → C server
- Go client → Go server
- C client → Go server
- Go client → C server

No additional serialization layer or adapter code needed.


## Generated Code Details

### Go Output (`messages.go`)

For each message type, wireforge generates:

| Generated | Purpose |
|---|---|
| `type Xxx struct { ... }` | Data container with exported fields |
| `const XxxFixedSize` | Fixed header byte count (compile-time constant) |
| `func (*Xxx) MessageTypeID() uint16` | Wire type identifier |
| `func (*Xxx) Marshal() ([]byte, error)` | Serialize entire frame and returns buffer |
| `func (*Xxx) Unmarshal(io.Reader, uint16) error` | Deserialize from stream after frame header |
| `func ReadMessageFrame(io.Reader) (typeID, hdrLen uint16, err error)` | Read just the 4-byte frame header for dispatch |

**Safety guarantees in generated Go:**
- `init()` panics at startup if `unsafe.Sizeof` disagrees with computed layout
- `MaxAllowedPacket` check before every allocation

### C Output (`messages.h` + `messages.c`)

For each message type, wireforge generates:

| Generated | Purpose |
|---|---|
| `typedef struct { ... } xxx_xx_t` | Packed struct with explicit `_padN` fields |
| `_Static_assert(sizeof(...))` | Compile-time layout verification |
| `xxx_xx_set_yyy(msg, new_value)` | Setter functions for all the fields in struct |
| `calculate_xxx_xx_dynamic_payload_size(hdr_len)` | Calculate the overall size of all dynamic fields |
| `xxx_xx_marshal(msg, buf)` | Serialize to buffer; returns total bytes or -1 |
| `xxx_xx_unmarshal(buf, len, hdr_len, out)` | Deserialize from buffer; mallocs dynamic fields |
| `xxx_xx_free(msg)` | Free all heap-allocated dynamic fields |

**Safety guarantees in generated C:**
- `MAX_ALLOWED_PACKET` validated before `malloc()`
- `_Static_assert` catches struct layout bugs at compile time
- Every `malloc()` failure path calls `xxx_free()` to clean partial state
- Strings are NULL-terminated with `+1` allocation for safety


## Example Usage

### 1. Define your schema

Create any yaml file with OpenAPI 3.1 `components/schemas`. Each schema becomes a message type. Each schema must have a unique `x-message-id` integer (0-65535) to identify the message type on the wire.

```yaml
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
```

### 2. Generate code

```bash
wireforge -i examples/payload.yaml -o examples/out/ -p messages
```

Output:

```
Parsing file: examples/payload.yaml
  generated: examples/out/go/messages.go
  generated: examples/out/c/messages.h
  generated: examples/out/c/messages.c

wireforge: successfully generated 4 message type(s)
```

### 3. Use in Go

```go
func handleClientSession(conn net.Conn) {
	defer conn.Close()

	for {
		msgType, fixedLen, err := messages.ReadMessageFrame(conn)
		if err != nil {
			break
		}

		switch MessageType(msgType) {
		case MsgTypeHeartbeat:
			msg := &messages.HeartbeatMessage{}
			if err := msg.Unmarshal(conn, fixedLen); err != nil {
				fmt.Printf("[Server] Malformed heartbeat payload: %v\n", err)
				return
			}

		case MsgTypeUserText:
			msg := &messages.UserMessage{}
			if err := msg.Unmarshal(conn, fixedLen); err != nil {
				fmt.Printf("[Server] Failed to unmarshal user message body: %v\n", err)
				return
			}
			
			t := time.Unix(msg.Timestamp, 0).Format("15:04:05")
			fmt.Printf("\r\x1b[K[%s] %s> %s\n> ", t, peer.name, msg.Content)

		default:
			fmt.Printf("[Server] Unknown type frame encountered (%d). Disconnecting client for safety.\n", msgType)
			return
		}
	}
}
```

### 4. Use in C

```c
    while (1) {
        int client_sock = accept(in_sock, NULL, NULL);
        if (client_sock < 0) continue;

        fflush(stdout);

        // Configure a subtle read timeout so if Go freezes/crashes, read() unblocks
        struct timeval tv;
        tv.tv_sec = 3; 
        tv.tv_usec = 0;
        setsockopt(client_sock, SOL_SOCKET, SO_RCVTIMEO, (const char*)&tv, sizeof(tv));

        uint8_t frame[WIRE_FRAME_HEADER_SIZE];
        while (1) {
            if (read_all(client_sock, frame, WIRE_FRAME_HEADER_SIZE) != 0) break;

            uint16_t type_id = get_message_type(frame);
            uint16_t fixed_len = get_message_fixed_length(frame);

            uint8_t* fixed_buf = malloc(fixed_len);
            if (!fixed_buf) break;

            if (read_all(client_sock, fixed_buf, fixed_len) != 0) {
                free(fixed_buf);
                break;
            }

            switch (type_id) {
                case MESSAGE_TYPE_USER_MESSAGE: {
                    size_t dyn_total = calculate_user_message_dynamic_payload_size(fixed_buf);

                    size_t full_payload_len = fixed_len + dyn_total;
                    uint8_t* full_payload = malloc(full_payload_len);
                    if (!full_payload) {
                        free(fixed_buf);
                        break;
                    }

                    memcpy(full_payload, fixed_buf, fixed_len);
                    free(fixed_buf);

                    if (dyn_total > 0) {
                        if (read_all(client_sock, full_payload + fixed_len, dyn_total) != 0) {
                            free(full_payload);
                            break;
                        }
                    }

                    user_message_t msg = {0};
                    if (user_message_unmarshal(full_payload, full_payload_len, fixed_len, &msg) == 0) {
                        printf("\r\33[2K[%s] %s\n> ", peer_name, msg.content ? msg.content : "");
                        fflush(stdout);
                        user_message_free(&msg);
                    }
                    free(full_payload);
                    break;
                }
                ...
                default: {
                    free(fixed_buf);
                    break;
                }
            }
        }
        close(client_sock);
        printf("User %s left the chat...\n", peer_name);
        joined = 0;
        peer_name[0] = '\0';
        fflush(stdout);
    }
```

The complete example server and client code is available in the [examples/](examples/) directory.
