package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/avatar31/wireforge/examples/client-server/go-server/messages"
)

type Client struct {
	Name   string
	Joined bool
}

type MessageType int

const (
	MsgTypeUserText  MessageType = 1
	MsgTypeHeartbeat MessageType = 2
)

const socketPath = "/tmp/chat.sock"

var (
	sockConn      net.Conn
	sockConnMutex sync.Mutex
	client        = Client{Joined: false}
)

func sendMessage(buf []byte) {
	sockConnMutex.Lock()
	defer sockConnMutex.Unlock()

	if sockConn != nil {
		_ = sockConn.SetWriteDeadline(time.Now().Add(2 * time.Second))

		_, err := sockConn.Write(buf)
		if err != nil {
			sockConn.Close()
			sockConn = nil
			client.Joined = false
			fmt.Print("\r\x1b[K[Server] Transmission failed. Client disconnected.\n")
		}
		if sockConn != nil {
			_ = sockConn.SetWriteDeadline(time.Time{})
		}
	}
}

func handleClientSession(conn net.Conn) {
	defer conn.Close()

	for {
		msgType, fixedLen, err := messages.ReadMessageFrame(conn)
		if err != nil {
			break // Connection naturally closed or aborted
		}

		switch MessageType(msgType) {
		case MsgTypeHeartbeat:
			msg := &messages.HeartbeatMessage{}
			if err := msg.Unmarshal(conn, fixedLen); err != nil {
				fmt.Printf("[Server] Malformed heartbeat payload: %v\n", err)
				return
			}
			// Heartbeat processed cleanly. Loop repeats.

		case MsgTypeUserText:
			msg := &messages.UserMessage{}
			if err := msg.Unmarshal(conn, fixedLen); err != nil {
				fmt.Printf("[Server] Failed to unmarshal user message body: %v\n", err)
				return
			}

			t := time.Unix(msg.Timestamp, 0).Format("15:04:05")
			fmt.Printf("\r\x1b[K[%s] User> %s\n> ", t, msg.Content)

		default:
			fmt.Printf("[Server] Unknown type frame encountered (%d). Disconnecting client for safety.\n", msgType)
			return
		}
	}
}

func handleServerConsoleInput() {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("> ")
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" {
			fmt.Println("Leaving chat...")
			_ = os.Remove(socketPath)
			os.Exit(0)
		}

		if !client.Joined {
			fmt.Println("Wait for user to join before sending messages.")
			continue
		}

		buf, err := buildUserMessage(input)
		if err != nil {
			fmt.Printf("Failed to construct message. Please try again\n")
			continue
		}
		sendMessage(buf)
	}
}

func startServerHeartbeat() {
	for {
		sockConnMutex.Lock()
		isJoined := client.Joined
		sockConnMutex.Unlock()

		if isJoined {
			msg := messages.HeartbeatMessage{Timestamp: time.Now().Unix()}
			buf, err := msg.Marshal()
			if err != nil {
				fmt.Printf("[Server] Failed to build heartbeat message: %v\n", err)
				continue
			}
			sendMessage(buf)
		}
		time.Sleep(1 * time.Second)
	}
}

func buildUserMessage(message string) ([]byte, error) {
	msg := messages.UserMessage{
		Content:   message,
		Timestamp: time.Now().Unix(),
	}
	return msg.Marshal()
}

func main() {
	clearScreen()
	fmt.Println("=========== Welcome ===========")
	_ = os.Remove(socketPath)
	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		fmt.Println("Failed to open channel")
		os.Exit(1)
	}
	defer listener.Close()

	go startServerHeartbeat()
	go handleServerConsoleInput()

	fmt.Println("Type 'exit' to close the application.")
	fmt.Println("Waiting for user to connect...")

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		sockConnMutex.Lock()
		sockConn = conn
		client.Joined = true // FIX: Turn state flag true upon client handshake
		sockConnMutex.Unlock()

		fmt.Println("\nUser joined chat...")
		fmt.Print("> ")

		handleClientSession(conn)

		sockConnMutex.Lock()
		if sockConn == conn {
			sockConn = nil
			client.Joined = false
		}
		sockConnMutex.Unlock()

		fmt.Println("User left the chat...")
	}
}

func clearScreen() {
	var cmd *exec.Cmd
	cmd = exec.Command("clear")
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
