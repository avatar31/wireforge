package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const socketPath = "/tmp/chat.sock"

type Message struct {
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

var (
	activeClient net.Conn
	clientMutex  sync.Mutex
)

// go build -o chatapp
func main() {
	fmt.Println("[Server] Initializing Server...")

	// Clean up any old stale socket file left on disk
	_ = os.Remove(socketPath)

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Printf("[Server] Bind error: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	go startServerHeartbeat()

	// Handle local server keyboard input to send to the client
	go handleServerConsoleInput()

	fmt.Println("[Server] Awaiting incoming C Client connection...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		clientMutex.Lock()
		activeClient = conn
		clientMutex.Unlock()

		fmt.Print("\r\x1b[K[Server] C Client connected! Bidirectional pipeline active.\n> ")

		// Read incoming JSON payloads from the client
		handleClientSession(conn)

		clientMutex.Lock()
		if activeClient == conn {
			activeClient = nil
		}
		clientMutex.Unlock()

		fmt.Print("\r\x1b[K[Server] C Client disconnected. Listening for a new connection...\n> ")
	}
}

func handleClientSession(conn net.Conn) {
	defer conn.Close()
	decoder := json.NewDecoder(conn)

	for {
		var msg Message
		if err := decoder.Decode(&msg); err != nil {
			break // Connection lost or client dropped out
		}

		// Intercept and drop background heartbeat messages quietly
		if msg.Sender == "system" && msg.Content == "ping" {
			continue
		}

		fmt.Printf("\r\x1b[K[%s] %s: %s\n> ", msg.Timestamp, msg.Sender, msg.Content)
	}
}

func transmitJSON(sender, content string) {
	clientMutex.Lock()
	defer clientMutex.Unlock()

	if activeClient != nil {
		msg := Message{
			Sender:    sender,
			Content:   content,
			Timestamp: time.Now().Format("15:04:05"),
		}

		_ = activeClient.SetWriteDeadline(time.Now().Add(2 * time.Second))
		err := json.NewEncoder(activeClient).Encode(msg) // Automatically appends '\n'
		if err != nil {
			activeClient.Close()
			activeClient = nil
			if sender != "system" {
				fmt.Print("\r\x1b[K[Server] Transmission failed. Client disconnected.\n> ")
			}
		}
		if activeClient != nil {
			_ = activeClient.SetWriteDeadline(time.Time{})
		}
	} else if sender != "system" {
		fmt.Println("[Server] Drop payload. No client currently connected.")
	}
}

func handleServerConsoleInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" {
			fmt.Println("[Server] Shutting down...")
			_ = os.Remove(socketPath)
			os.Exit(0)
		}

		transmitJSON("Server (Go)", input)
	}
}

func startServerHeartbeat() {
	for {
		time.Sleep(1 * time.Second)
		transmitJSON("system", "ping")
	}
}
