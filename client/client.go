package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"time"
)

type Client struct {
	ip   string
	port string
	n    int
}
type Server struct {
	ip   string
	port int
}

var (
	isBlockedBroadcast bool
	conn               net.Conn
	clients            []Client
	blockedClients     []Client
)

const (
	Reset = "\033[0m"
	Green = "\033[32m"
	Red   = "\033[31m"
)

func main() {
	server := Server{port: 8080}
	fmt.Print("Enter the server's ip adress : ")
	fmt.Scanf("%s", server.ip)
	err := connect(&server)
	if err != nil {
		logWarning("Error connecting to server: %v", err)
	}

	go Read(&server)

	var message string
	fmt.Printf("------------------------------------------------------------")
	fmt.Printf("\n1- Broadcast message\n2- Send to specific user\n3- Block user\n4- Block/Unblock Broadcast\n5- Show blocked clients\n")
	fmt.Printf("------------------------------------------------------------")
	var operation int

	for {
		fmt.Print("Enter operation : ")
		fmt.Scanf("%d", &operation)
		switch operation {
		case 1:
			for {
				fmt.Printf("Enter Message : ")
				fmt.Scanf("%s", &message)
				if len(message) > 1 {
					break
				}
			}
			actionType := make([]byte, 1024)
			binary.BigEndian.PutUint16(actionType[2:4], 1)
			copy(actionType[8:], []byte(message))
			_, err = conn.Write(actionType)
			if err != nil {
				logWarning("Error sending message: %v", err)
				break
			}

		case 2:
			var DestinationClient int
			fmt.Print("Which client you want to send a request to? : ")
			fmt.Scanf("%d", &DestinationClient)
			fmt.Print("Enter message : ")
			fmt.Scanf("%s", &message)
			actionType := make([]byte, 1024)
			binary.BigEndian.PutUint16(actionType[2:4], 2)
			binary.BigEndian.PutUint16(actionType[4:6], uint16(DestinationClient))
			copy(actionType[8:], []byte(message))
			_, err = conn.Write(actionType)
			if err != nil {
				logWarning("Error sending message: %v", err)
				break
			}

		case 3:
			var clientToBlock int
			fmt.Print("Which client do you want to block? : ")
			fmt.Scanf("%d", &clientToBlock)
			err := blockClient(clientToBlock)
			if err != nil {
				logWarning("%v", err)
			}

		case 4:
			msg := make([]byte, 1024)
			binary.BigEndian.PutUint16(msg[2:4], 4)
			if isBlockedBroadcast {
				binary.BigEndian.PutUint16(msg[4:6], 0)
				isBlockedBroadcast = false
			} else {
				binary.BigEndian.PutUint16(msg[4:6], 1)
				isBlockedBroadcast = true
			}
			_, err := conn.Write(msg)
			if err != nil {
				logWarning("Error writing message: %v", err)
			}

		case 5:
			logInfo("Blocked Clients: ")
			if len(blockedClients) == 0 {
				logInfo("You didn't block any client yet")
			} else {
				for _, blockedClient := range blockedClients {
					fmt.Printf("- %d - %s:%s", blockedClient.n, blockedClient.ip, blockedClient.port)
				}
				fmt.Println("")
			}
		}
	}
}

func connect(server *Server) error {
	var err error
	conn, err = net.Dial("tcp", fmt.Sprintf("%s:%d", server.ip, server.port))
	if err != nil {
		return err
	}
	logInfo("Connected to server!")
	return nil
}

func reconnect(server *Server) {
	logInfo("Reconnecting to server...")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for i := 0; i < 120; i += 20 {
		<-ticker.C
		err := connect(server)
		if err == nil {
			return
		}
	}
}

func Read(server *Server) {
	for {
		buffer := make([]byte, 1024)

		_, err := conn.Read(buffer)
		if err == io.EOF {
			logWarning("Server disconnected. Connection Closed")
			reconnect(server)
			Read(server)
		}
		if err != nil {
			logWarning("Read error: %v", err)
			return
		}

		if binary.BigEndian.Uint16(buffer[2:4]) == 5 {
			clients = clients[:0]
			clientsString := strings.Split(string(buffer[6:]), ";")

			for _, client := range clientsString {
				clientInfo := strings.Split(client, ":")
				if len(clientInfo) >= 2 {
					clients = append(clients, Client{ip: clientInfo[0], port: clientInfo[1], n: len(clients)})
				}
			}

			if len(clients) > 1 {
				fmt.Printf("\nClients:\n")
				for i, client := range clients {
					fmt.Printf("%d - \t %s:%s\n", i+1, client.ip, client.port)
				}
				fmt.Println("------------------------------------------------------------")
			}
		} else {
			message := string(buffer[8:])
			if binary.BigEndian.Uint16(buffer[2:4]) == 0 {
				logInfo("- system - : %s", message)
			} else {
				fromIndex := binary.BigEndian.Uint16(buffer[6:8]) - 1
				from := clients[fromIndex]
				logInfo("- %d - %s:%s: %s", fromIndex+1, from.ip, from.port, message)
			}
		}
	}
}

func blockClient(clientToBlock int) error {
	actionType := make([]byte, 1024)
	binary.BigEndian.PutUint16(actionType[2:4], 3)
	binary.BigEndian.PutUint16(actionType[4:6], uint16(clientToBlock))
	blockedClients = append(blockedClients, clients[clientToBlock-1])
	_, err := conn.Write(actionType)
	if err != nil {
		return err
	}
	return nil
}

func logInfo(message string, args ...interface{}) {
	log.Printf(Green+"INFO: "+Reset+message+"\n", args...)
}

func logWarning(message string, args ...interface{}) {
	log.Printf(Red+"WARNING: "+Reset+message+"\n", args...)
}
