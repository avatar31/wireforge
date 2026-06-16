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

const (
	mySock     = "/tmp/go_listen.sock"
	targetSock = "/tmp/c_listen.sock"
)

type Message struct {
	Sender    string `json:"sender"`
	Content   string `json:"content"`
	Timestamp string `json:"timestamp"`
}

var (
	outboundConn net.Conn
	outMutex     sync.Mutex
)

// go build -o chatapp
func main() {
	fmt.Println("[System] Starting Resilient Go Peer with JSON & Heartbeats...")

	// 1. Inbound Server Listener
	go startInboundListener()

	// 2. Outbound Active Reconnection Dialer
	go startOutboundDialer()

	// 3. Heartbeat Generator (Sends a ping every 1 second)
	go startHeartbeatDaemon()

	// 4. Local User Console Interface
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
			break
		}

		sendJSONMessage("Go-Peer", input)
	}
}

func sendJSONMessage(sender, content string) {
	msg := Message{
		Sender:    sender,
		Content:   content,
		Timestamp: time.Now().Format("15:04:05"),
	}

	outMutex.Lock()
	defer outMutex.Unlock()

	if outboundConn != nil {
		// Set a brief write deadline so heartbeats don't block infinitely if the pipe freezes
		_ = outboundConn.SetWriteDeadline(time.Now().Add(2 * time.Second))
		
		// Encode automatically appends the '\n' at the end of the JSON line
		err := json.NewEncoder(outboundConn).Encode(msg)
		if err != nil {
			outboundConn.Close()
			outboundConn = nil
			fmt.Print("\r\x1b[K[System] Outbound pipe broken. Reconnecting in background...\n> ")
		}
		_ = outboundConn.SetWriteDeadline(time.Time{}) // Reset
	} else if sender != "system" {
		fmt.Println("[System] Cannot send. C listener is currently offline.")
	}
}

func startInboundListener() {
	_ = os.Remove(mySock)
	listener, err := net.Listen("unix", mySock)
	if err != nil {
		fmt.Printf("Go Bind error: %v\n", err)
		os.Exit(1)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		fmt.Print("\r\x1b[K[System] C Peer connected to our listener!\n> ")

		go func(c net.Conn) {
			defer c.Close()
			decoder := json.NewDecoder(c)
			for {
				var msg Message
				if err := decoder.Decode(&msg); err != nil {
					break // Disconnected
				}

				// Filter out silent background heartbeats from terminal rendering
				if msg.Sender == "system" && msg.Content == "ping" {
					continue 
				}

				fmt.Printf("\r\x1b[K[%s] %s: %s\n> ", msg.Timestamp, msg.Sender, msg.Content)
			}
			fmt.Print("\r\x1b[K[System] C Peer disconnected from our listener.\n> ")
		}(conn)
	}
}

func startOutboundDialer() {
	for {
		outMutex.Lock()
		needConnect := (outboundConn == nil)
		outMutex.Unlock()

		if needConnect {
			conn, err := net.Dial("unix", targetSock)
			if err == nil {
				outMutex.Lock()
				outboundConn = conn
				fmt.Print("\r\x1b[K[System] Successfully connected to C's listener!\n> ")
				outMutex.Unlock()
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func startHeartbeatDaemon() {
	for {
		time.Sleep(1 * time.Second)
		sendJSONMessage("system", "ping")
	}
}
