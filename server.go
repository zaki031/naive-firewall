package main

import (
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
)

type Client struct {
	conn               net.Conn
	ip                 string
	port               string
	n                  int
	blockedClients     []Client
	isBroadcastBlocked bool
}

var connList = make([]Client, 0)
var maxClients = 10

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Listening on port :8080...")
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Println(err)
			continue
		}
		ip := strings.Split(conn.RemoteAddr().String(), ":")
		fmt.Printf("Client %d - %s  :%s connected\n", len(connList)+1, ip[0], ip[1])
		connList = append(connList, Client{conn: conn, n: len(connList) + 1, port: ip[1], ip: ip[0]})
		go updateClients()
		go handleConnection(connList[len(connList)-1])

	}
}

func handleConnection(client Client) {
	defer client.conn.Close()

	for {

		buf := make([]byte, 1026)
		n, err := client.conn.Read(buf)
		if err == io.EOF {
			fmt.Printf("Client %d - %s  :%s disconected. Connection Closed\n", client.n, client.ip, client.port)
			removeClient(client)
			return
		} else if err != nil {
			fmt.Println("Connection closed:", err)
			return
		}

		operation := binary.BigEndian.Uint16(buf[2:4])
		switch operation {
		case 1:
			fmt.Println("Broadcasting message...")
			Broadcast(buf[:n], client)
		case 2:
			clientToSendIndex := binary.BigEndian.Uint16(buf[4:6]) - 1
			if clientToSendIndex >= uint16(len(connList)) {
				fmt.Println("Client does not exist")
				sendResponseMsg(client, "Client does not exist, try again with an existing client.")
			} else {
				clientToSend := connList[clientToSendIndex]
				fmt.Printf("Sending message from Client %d - '%s'  to %d - '%s'...\n", client.n, client.port, clientToSend.n, clientToSend.port)
				writeToSpecificClient(client, buf[:n])
			}
		case 3:
			clientToBlock := binary.BigEndian.Uint16(buf[4:6]) - 1
			fmt.Println(clientToBlock)
			fmt.Printf("Client n %d blocking client n %d...\n", client.n, clientToBlock+1)
			blockClient(client, int(clientToBlock))
		case 4:
			hh := buf[4:6]
			if binary.BigEndian.Uint16(buf[4:6]) == 1 {
				connList[client.n-1].isBroadcastBlocked = true
				fmt.Println("Broadcasts are blocked")
				sendResponseMsg(connList[client.n-1], "Broadcasts are blocked")
			} else {
				connList[client.n-1].isBroadcastBlocked = false
				fmt.Println("Broadcasts are unblocked")
				sendResponseMsg(connList[client.n-1], "Broadcasts are unblocked")
			}
			fmt.Println(hh)
		default:
			fmt.Println("Unknown operation:", operation)
			sendResponseMsg(client, "Operation does not exist, try again.")
		}

	}
}

func Broadcast(msg []byte, iniClient Client) {
	clientCount := uint16(len(connList))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	binary.BigEndian.PutUint16(msg[6:8], uint16(iniClient.n))
	iniConn := iniClient.conn
	for _, client := range connList {
		if client.conn != iniConn && !client.isBroadcastBlocked && !isBlocked(iniClient, client) {
			_, err := client.conn.Write(msg)
			if err != nil {
				fmt.Println("Error broadcast:", err)
				sendResponseMsg(iniClient, "Error happened while broacasting")
			}
		}
	}
	sendResponseMsg(iniClient, "Broadcast sent successfully")

}

func writeToSpecificClient(senderClient Client, msg []byte) {
	clientCount := uint16(len(connList))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	user := binary.BigEndian.Uint16(msg[4:6])
	binary.BigEndian.PutUint16(msg[6:8], uint16(senderClient.n))
	if user > clientCount {
		sendResponseMsg(senderClient, "Client doesn't exist, try again.")
		return
	}

	ClientToSend := connList[user-1]

	if !isBlocked(senderClient, ClientToSend) {
		conn := ClientToSend.conn
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("Error writing to specific client:", err)
			sendResponseMsg(senderClient, "An error happened while sending your request")
		} else {
			fmt.Println("Message sent successfully !")
			sendResponseMsg(senderClient, "Sent successfully !")
		}
	} else {
		fmt.Println("user is blocked")
		sendResponseMsg(senderClient, "This user has blocked you, your request cannot be sent")
	}
}

func sendResponseMsg(client Client, msg string) {
	responseMsg := make([]byte, 1024)
	binary.BigEndian.PutUint16(responseMsg[2:4], 0)
	copy(responseMsg[8:], []byte(msg))
	_, err := client.conn.Write(responseMsg)
	if err != nil {
		fmt.Println("Error sending response to client", err)
	}
}

func blockClient(client Client, clientToBlock int) {

	if clientToBlock < len(connList) {
		connList[client.n-1].blockedClients = append(connList[client.n-1].blockedClients, connList[clientToBlock])
		sendResponseMsg(client, fmt.Sprintf("User %d blocked succesfuly!\n", clientToBlock+1))
		fmt.Printf("Client n %d blocked client n %d\n", client.n, clientToBlock+1)

	} else {
		sendResponseMsg(client, fmt.Sprintf("Client n %d was not blocked client n %d\n", client.n, clientToBlock+1))
		fmt.Printf("Client n %d was not blocked client n %d\n", client.n, clientToBlock+1)

	}

}

func isBlocked(senderClient Client, RecieverCLient Client) bool {
	for _, blockedClient := range RecieverCLient.blockedClients {
		if blockedClient.conn == senderClient.conn {
			return true
		}
	}
	return false
}

func removeClient(client Client) {
	connList = append(connList[:client.n-1], connList[client.n:]...)
	for i := client.n - 1; i < len(connList); i++ {
		connList[i].n--
	}
}

func updateClients() {
	clientList := ""
	for _, c := range connList {
		clientList += fmt.Sprintf("%s:%s;", c.ip, c.port)
	}
	spaceForCLientList := maxClients*20 + maxClients
	msg := make([]byte, spaceForCLientList)
	binary.BigEndian.PutUint16(msg[0:2], uint16(len(clientList)))
	binary.BigEndian.PutUint16(msg[2:4], 5)

	copy(msg[6:spaceForCLientList], []byte(clientList))
	for _, client := range connList {
		client.conn.Write(msg)
	}
}
