package main

import (
	"encoding/binary"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

type Client struct {
	conn           net.Conn
	id             string
	number         int
	blockedClients []Client
	broadcast      bool
}

var (
	connList = make([]Client, 0)
	mu       sync.Mutex
)

func main() {
	var listen_addr = flag.String("addr", "127.0.0.1:9090", "tcp addresse that our tcp server listen to")
	flag.Parse()
	ln, err := net.Listen("tcp", *listen_addr)
	if err != nil {
		log.Fatalf("Failed to listen on port %v", err)
	}
	defer ln.Close()

	fmt.Println("Listening...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		mu.Lock()
		client := Client{conn: conn, number: len(connList) + 1}
		connList = append(connList, client)
		mu.Unlock()

		go handleConnection(client)
	}
}

func handleConnection(client Client) {
	defer client.conn.Close()

	for {
		buf := make([]byte, 1024)
		n, err := client.conn.Read(buf)
		if err == io.EOF {
			fmt.Printf(
				"Client #%d disconnected. Connection Closed\n",
				client.number,
			)
			removeClient(client.number)
			return
		} else if err != nil {
			fmt.Printf("Error reading from client #%d: %v\n", client.number, err)
			return
		}

		operation := binary.BigEndian.Uint16(buf[2:4])
		fmt.Printf("Operation from client #%d: %d\n", client.number, operation)

		switch operation {
		case 1:
			fmt.Println("Broadcasting message...")
			broadcastMessage(buf[:n], client.conn)
		case 2:
			fmt.Println("Sending message to specific client...")
			sendToSpecificClient(client, buf[:n])
		case 3:
			clientToBlock := int(binary.BigEndian.Uint16(buf[4:6])) - 1
			fmt.Printf(
				"Client #%d blocking client #%d...\n",
				client.number,
				clientToBlock+1,
			)
			blockClient(client, clientToBlock)
		default:
			fmt.Printf(
				"Unknown operation from client #%d: %d\n",
				client.number,
				operation,
			)
		}
	}
}

func broadcastMessage(msg []byte, senderConn net.Conn) {
	mu.Lock()
	defer mu.Unlock()

	clientCount := uint16(len(connList))
	binary.BigEndian.PutUint16(msg[:2], clientCount)

	for _, client := range connList {
		if client.conn != senderConn && !isBlocked(client, senderConn) {
			if _, err := client.conn.Write(msg); err != nil {
				fmt.Printf(
					"Error broadcasting to client #%d: %v\n",
					client.number,
					err,
				)
			}
		}
	}
}

func sendToSpecificClient(sender Client, msg []byte) {
	user := int(binary.BigEndian.Uint16(msg[4:6])) - 1

	mu.Lock()
	defer mu.Unlock()

	if user < 0 || user >= len(connList) {
		sendErrorMsg(sender, "Invalid client number")
		return
	}

	recipient := connList[user]
	if isBlocked(recipient, sender.conn) {
		sendErrorMsg(
			sender,
			"This user has blocked you, your request cannot be sent",
		)
		return
	}

	if _, err := recipient.conn.Write(msg); err != nil {
		fmt.Printf(
			"Error sending to specific client #%d: %v\n",
			recipient.number,
			err,
		)
	}
}

func blockClient(client Client, clientToBlock int) {
	mu.Lock()
	defer mu.Unlock()

	if clientToBlock < 0 || clientToBlock >= len(connList) {
		sendErrorMsg(
			client,
			fmt.Sprintf("User %d does not exist", clientToBlock+1),
		)
		return
	}

	connList[client.number-1].blockedClients = append(
		connList[client.number-1].blockedClients,
		connList[clientToBlock],
	)
	sendSuccessMsg(
		client,
		fmt.Sprintf("User %d blocked successfully", clientToBlock+1),
	)
}

func isBlocked(client Client, senderConn net.Conn) bool {
	for _, blockedClient := range client.blockedClients {
		if senderConn == blockedClient.conn {
			return true
		}
	}
	return false
}

func removeClient(clientNumber int) {
	mu.Lock()
	defer mu.Unlock()

	for i, client := range connList {
		if client.number == clientNumber {
			connList = append(connList[:i], connList[i+1:]...)
			return
		}
	}
}

func sendErrorMsg(client Client, msg string) {
	response := make([]byte, 1024)
	copy(response[4:], msg)
	client.conn.Write(response)
}

func sendSuccessMsg(client Client, msg string) {
	response := make([]byte, 1024)
	copy(response[4:], msg)
	client.conn.Write(response)
}
