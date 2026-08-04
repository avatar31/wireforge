# Makefile for C roundtrip tests
#
# Targets:
#   make          — build the test binary
#   make run      — build and run all tests
#   make clean    — remove build artifacts
#   make asan     — build with AddressSanitizer + UBSan

CC      = gcc
CFLAGS  = -std=c11 -Wall -Wextra -Wpedantic -g
LDFLAGS = -lm

SRC     = messages.c roundtrip_test.c
BIN     = roundtrip_test

ASAN_CFLAGS = $(CFLAGS) -fsanitize=address,undefined -fno-omit-frame-pointer
ASAN_BIN    = roundtrip_test_asan

.PHONY: all run asan run-asan clean

all: $(BIN)

$(BIN): $(SRC) messages.h
	$(CC) $(CFLAGS) $(LDFLAGS) -o $@ $(SRC)

run: $(BIN)
	./$(BIN)

asan: $(ASAN_BIN)

$(ASAN_BIN): $(SRC) messages.h
	$(CC) $(ASAN_CFLAGS) $(LDFLAGS) -o $@ $(SRC)

run-asan: $(ASAN_BIN)
	./$(ASAN_BIN)

clean:
	rm -f $(BIN) $(ASAN_BIN)
