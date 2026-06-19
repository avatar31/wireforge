// Copyright (c) 2026 Sachin S. All rights reserved.
// 
// Licensed under the MIT License.
// See LICENSE in the project root.

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

	"github.com/avatar31/wireforge/examples/peer-to-peer/go-peer/messages"
)

const (
	mySock     = "/tmp/go_listen.sock"
	targetSock = "/tmp/c_listen.sock"
)

type MessageType int

const (
	MsgTypeUserText   MessageType = 1
	MsgTypeHeartbeat  MessageType = 2
	MsgTypeUserJoined MessageType = 3
	MsgTypeUserLeft   MessageType = 4
)

type Peer struct {
	outboundConn net.Conn
	outMutex     sync.Mutex
	name         string
	joined       bool
}

var (
	peer   = Peer{outboundConn: nil, joined: false}
	myname = ""
)

func sendMessage(mstType MessageType, buf []byte) {
	peer.outMutex.Lock()
	defer peer.outMutex.Unlock()

	if peer.outboundConn != nil {
		if mstType == MsgTypeUserJoined || peer.joined {
			_ = peer.outboundConn.SetWriteDeadline(time.Now().Add(2 * time.Second))

			_, err := peer.outboundConn.Write(buf)
			if err != nil {
				peer.outboundConn.Close()
				peer.outboundConn = nil
				peer.joined = false
				fmt.Print("User Disconnected\n")
			}
			if peer.outboundConn != nil {
				_ = peer.outboundConn.SetWriteDeadline(time.Time{})
			}
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
		case MsgTypeUserJoined:
			msg := &messages.UserJoinedMessage{}
			if err := msg.Unmarshal(conn, fixedLen); err != nil {
				fmt.Printf("[Server] Failed to unmarshal user joined body: %v\n", err)
				return
			}
			peer.outMutex.Lock()
			peer.joined = true
			peer.name = msg.Username
			peer.outMutex.Unlock()
			fmt.Printf("\nUser %s joined chat...\n", msg.Username)

		case MsgTypeUserLeft:
			msg := &messages.UserLeftMessage{}
			if err := msg.Unmarshal(conn, fixedLen); err != nil {
				fmt.Printf("[Server] Failed to unmarshal user left body: %v\n", err)
				return
			}
			peer.outMutex.Lock()
			peer.joined = false
			peer.name = ""
			peer.outMutex.Unlock()
			fmt.Printf("\nUser %s left chat...\n", msg.Username)

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
			fmt.Printf("\r\x1b[K[%s] %s> %s\n> ", t, peer.name, msg.Content)

		default:
			fmt.Printf("[Server] Unknown type frame encountered (%d). Disconnecting client for safety.\n", msgType)
			return
		}
	}
}

func startInboundListener() {
	_ = os.Remove(mySock)
	listener, err := net.Listen("unix", mySock)
	if err != nil {
		fmt.Println("Failed to open channel")
		os.Exit(1)
	}
	defer listener.Close()

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}

		go func(c net.Conn) {
			handleClientSession(c)
		}(conn)
	}
}

func startOutboundDialer() {
	for {
		peer.outMutex.Lock()
		needConnect := (peer.outboundConn == nil)
		peer.outMutex.Unlock()

		if needConnect {
			conn, err := net.Dial("unix", targetSock)
			if err == nil {
				peer.outMutex.Lock()
				peer.outboundConn = conn
				peer.outMutex.Unlock()

				msg := messages.UserJoinedMessage{Timestamp: time.Now().Unix(), Username: myname}
				buf, err := msg.Marshal()
				if err != nil {
					fmt.Print("Failed to ping user\n")
					peer.outMutex.Lock()
					peer.outboundConn.Close()
					peer.outboundConn = nil
					peer.joined = false
					peer.outMutex.Unlock()
					continue
				}
				sendMessage(MsgTypeUserJoined, buf)
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func startServerHeartbeat() {
	for {
		peer.outMutex.Lock()
		isJoined := peer.joined
		peer.outMutex.Unlock()

		if isJoined {
			msg := messages.HeartbeatMessage{Timestamp: time.Now().Unix()}
			buf, err := msg.Marshal()
			if err != nil {
				fmt.Printf("[Server] Failed to build heartbeat message: %v\n", err)
				continue
			}
			sendMessage(MsgTypeHeartbeat, buf)
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

// go build -o chatapp
func main() {
	clearScreen()
	fmt.Println("=========== Welcome ===========")
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("Enter your name: ")
		input, err := reader.ReadString('\n')
		if err != nil {
			os.Exit(1)
		}
		input = strings.TrimSpace(input)
		if input != "" {
			myname = input
			break
		}
		fmt.Println("Please enter your name to enter the chat.")
	}

	go startInboundListener()

	go startOutboundDialer()

	go startServerHeartbeat()

	for {
		fmt.Printf("%s> ", myname)
		input, err := reader.ReadString('\n')
		if err != nil {
			break
		}
		input = strings.TrimSpace(input)
		if input == "" {
			continue
		}
		if strings.ToLower(input) == "exit" {
			msg := messages.UserLeftMessage{Timestamp: time.Now().Unix(), Username: myname}
			buf, err := msg.Marshal()
			if err != nil {
				break
			}
			sendMessage(MsgTypeUserLeft, buf)
			break
		}

		peer.outMutex.Lock()
		joined := peer.joined
		peer.outMutex.Unlock()

		if !joined {
			fmt.Println("Wait for user to join before sending messages.")
			continue
		}

		buf, err := buildUserMessage(input)
		if err != nil {
			fmt.Printf("Failed to construct message. Please try again\n")
			continue
		}
		sendMessage(MsgTypeUserText, buf)
	}
}

func clearScreen() {
	var cmd *exec.Cmd
	cmd = exec.Command("clear")
	cmd.Stdout = os.Stdout
	_ = cmd.Run()
}
