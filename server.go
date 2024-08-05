package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"io"
)

type Client struct {
	conn net.Conn
	id   string
	n int
	blockedClients []Client
	isBroadcastBlocked bool

}

var connList = make([]Client, 0)

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
		connList = append(connList, Client{conn: conn, n: len(connList) + 1})
		go handleConnection(connList[len(connList)-1])
	}
}

func handleConnection(client Client) {
	defer client.conn.Close()
	for {
		buf := make([]byte, 1024)
		_, err := client.conn.Read(buf)
		if err ==io.EOF{
			fmt.Printf("Client n %d disconected. Connection Closed\n", client.n)
			connList = append(connList[:client.n-1], connList[client.n:]...)
			return
		} else if err != nil {
			fmt.Println("Connection closed:", err)
			return
		}

		operation := binary.BigEndian.Uint16(buf[2:4])
		fmt.Println("Operation:", operation)
		switch operation {
		case 1:
			fmt.Println("Broadcasting message...")
			Broadcast(buf, client)
		case 2:
			fmt.Println("Sending message to specific client...")
			writeToSpecificClient(client, buf)
		case 3:
			clientToBlock := binary.BigEndian.Uint16(buf[4:6]) -1
			fmt.Println(clientToBlock)
			fmt.Printf("Client n %d blocking client n %d...\n", client.n, clientToBlock  +1 )
			blockClient(client, int(clientToBlock) );
		case 4:
			hh := buf[4:6];
			if binary.BigEndian.Uint16(buf[4:6]) == 1{
				connList[client.n-1].isBroadcastBlocked = true
				fmt.Println("Broadcasts are blocked")
				sendResponseMsg(connList[client.n-1],"Broadcasts are blocked");
			} else { 
				connList[client.n-1].isBroadcastBlocked = false
				fmt.Println("Broadcasts are unblocked")
			}
			fmt.Println(hh);
		default:
			fmt.Println("Unknown operation:", operation)
		}

	}
}

func Broadcast(msg []byte, iniClient Client) {
	clientCount := uint16(len(connList)) 
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	iniConn := iniClient.conn
	for _, client := range connList {
		if client.conn != iniConn &&  !client.isBroadcastBlocked   && !isBlocked(iniClient, client){
			_, err := client.conn.Write(msg)
			if err != nil {
				fmt.Println("Error broadcast:", err)
				sendResponseMsg(iniClient, "Error happened while broacasting");
			} 
		}
	}

}

func writeToSpecificClient(senderClient Client, msg []byte) {
	clientCount := uint16(len(connList))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	user := binary.BigEndian.Uint16(msg[4:6])
	ClientToSend:= connList[user-1];
	
	fmt.Printf("Client %d sending to client %d:", senderClient.n, user);
	if !isBlocked(senderClient, ClientToSend){
		conn := ClientToSend.conn
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("Error writing to specific client:", err)
			sendResponseMsg(senderClient, "An error happened while sending your request");

		} else {
			sendResponseMsg(senderClient, "Sent successfully !");
		}
	} else{
		fmt.Println("user is blocked")
		sendResponseMsg(senderClient, "This user has blocked you, your request cannot be sent");
	}
	} 
		

	



func sendResponseMsg(client Client,msg string){
	responseMsg := make([]byte, 1024);
	copy(responseMsg[6:], []byte(msg));
	_, err := client.conn.Write(responseMsg);
	if err != nil {
		fmt.Println("Error sending response to client", err);
	}
}

func blockClient(client Client,  clientToBlock int) {

	if  clientToBlock < len(connList) {
	connList[client.n-1].blockedClients = append(connList[client.n-1].blockedClients, connList[clientToBlock])
	sendResponseMsg(client, fmt.Sprintf("User %d blocked succesfuly!\n",  clientToBlock+1))
	fmt.Printf("Client n %d blocked client n %d\n", client.n, clientToBlock  +1 )

	} else {
		sendResponseMsg(client, fmt.Sprintf("Client n %d was not blocked client n %d\n", client.n, clientToBlock  +1 ))
		fmt.Printf("Client n %d was not blocked client n %d\n", client.n, clientToBlock  +1 )

	}

}


func isBlocked(senderClient Client, RecieverCLient Client ) bool{

	for _, blockedClient := range RecieverCLient.blockedClients{
		if blockedClient.conn == senderClient.conn {
			return true
		}
	}
	return false

}


