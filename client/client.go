package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"time"
)

type Client struct {
	ip   string
	port string
}

var (
	isBlockedBroadcast bool
	conn               net.Conn
	clients            []Client
)

func main() {
	err := connect()
	if err != nil {
		println("Error coonecting to server: ", err)
	}

	go Read()

	var message string
	var operation int
	fmt.Printf("1- Broacast message\n2-Send to specific user\n3-Block user\n4-Block/Unblock Broadcast\n")
	fmt.Println("------------------------------------------------------------")

	for {

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
				fmt.Println("Error sending message:", err)
				break
			}

		case 2:
			var DestinationClient int
			fmt.Print("Which client you wanna send a request to ? : ")
			fmt.Scanf("%d", &DestinationClient)
			fmt.Print("Enter message : ")
			fmt.Scanf("%s", &message)
			actionType := make([]byte, 1024)
			binary.BigEndian.PutUint16(actionType[2:4], 2)
			binary.BigEndian.PutUint16(actionType[4:6], uint16(DestinationClient))
			copy(actionType[8:], []byte(message))
			_, err = conn.Write(actionType)
			if err != nil {
				fmt.Println("Error sending message:", err)
				break
			}

		case 3:
			var clientToBlock int
			fmt.Printf("Which Client you wanna block? : ")
			fmt.Scanf("%d", &clientToBlock)
			fmt.Printf("Blocking Client n* %d...\n", clientToBlock)
			actionType := make([]byte, 1024)
			binary.BigEndian.PutUint16(actionType[2:4], 3)
			binary.BigEndian.PutUint16(actionType[4:6], uint16(clientToBlock))
			_, err := conn.Write(actionType)
			if err != nil {
				panic(err)
			}
			fmt.Printf("Client - %d - %s:%s blocked succefully!\n", clientToBlock, clients[clientToBlock-1].ip, clients[clientToBlock-1].port)
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
				fmt.Println("Erorr writing message", err)
			}
		}

	}
}

func connect() error {
	var err error
	conn, err = net.Dial("tcp", "192.168.100.57:8080")
	if err != nil {
		return err
	}
	fmt.Println("Connected to server again !")
	return nil
}

func reconnect() {
	fmt.Println("Reconnecting to server...")
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for i := 0; i < 160; i += 20 {
		<-ticker.C
		err := connect()
		if err == nil {
			return
		}
	}
}
func Read() {
	for {

		buffer := make([]byte, 1024)

		_, err := conn.Read(buffer)
		if err == io.EOF {
			fmt.Printf("Server disconnected. Connection Closed\n")
			reconnect()
			Read()
		}
		if err != nil {
			fmt.Println("Read error:", err)
			return
		}
		if binary.BigEndian.Uint16(buffer[2:4]) == 5 {
			clients = clients[:0]
			clientsString := strings.Split(string(buffer[6:216]), ";")

			for _, client := range clientsString {

				clientInfo := strings.Split(client, ":")
				if len(clientInfo) == 2 {
					clients = append(clients, Client{ip: clientInfo[0], port: clientInfo[1]})
				}
			}

			if len(clients) > 1 {
				fmt.Printf("Clients : \n")
				for i, client := range clients {
					fmt.Printf("%d - \t %s:%s\n", i+1, client.ip, client.port)
				}
				fmt.Println("------------------------------------------------------------")

			}
		} else {
			message := string(buffer[8:])
			if binary.BigEndian.Uint16(buffer[2:4]) == 0 {
				fmt.Printf("- system - : %s\n", message)

			} else {
				fromIndex := binary.BigEndian.Uint16(buffer[6:8]) - 1
				from := clients[fromIndex]
				fmt.Printf("- %d - %s:%s: %s\n", fromIndex+1, from.ip, from.port, message)

			}
		}

	}
}
