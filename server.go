package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"io"
)

type client struct {
	conn net.Conn
	id   string
	n int
	blockedClients []client
	brodcast bool

}

var connList = make([]client, 0)

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
		connList = append(connList, client{conn: conn, n: len(connList) + 1})
		go handleConnection(connList[len(connList)-1])
	}
}

func handleConnection(Client client) {
	defer Client.conn.Close()
	for {
		buf := make([]byte, 1024)
		_, err := Client.conn.Read(buf)
		if err ==io.EOF{
			fmt.Printf("Client n %d disconected. Connection Closed\n", Client.n)
			connList = append(connList[:Client.n-1], connList[Client.n:]...)
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
			Broadcast(connList, buf, Client.conn)
		case 2:
			fmt.Println("Sending message to specific client...")
			writeToSpecificClient(Client, buf)
		case 3:
			clientToBlock := binary.BigEndian.Uint16(buf[4:6]) -1
			fmt.Println(clientToBlock)
			fmt.Printf("Client n %d blocking client n %d...\n", Client.n, clientToBlock  +1 )
			blockClient(Client, int(clientToBlock) );
		default:
			fmt.Println("Unknown operation:", operation)
		}
	}
}

func Broadcast(conns []client, msg []byte, iniConn net.Conn) {
	clientCount := uint16(len(conns))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	
	fmt.Printf("%s", msg[4:])
	for _, client := range conns {
		var isBlocked bool

		for _ ,blockedClient := range client.blockedClients{

			if iniConn == blockedClient.conn {
				isBlocked = true;
			}
		}
		if client.conn != iniConn  && !isBlocked{
			_, err := client.conn.Write(msg)
			if err != nil {
				fmt.Println("Error writing to client:", err)
			}
		}
	}

}

func writeToSpecificClient(sender client, msg []byte) {
	clientCount := uint16(len(connList))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	user := binary.BigEndian.Uint16(msg[4:6])
	ClientToSend:= connList[user-1];
	var isBlocked bool;
	for _, blockedClient := range ClientToSend.blockedClients{
		if sender.conn == blockedClient.conn{
			isBlocked = true;
			break;
		}
	}


	fmt.Println("Sending to user:", binary.BigEndian.Uint16(msg[4:6]))
	if !isBlocked {
		
	
		conn := ClientToSend.conn
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("Error writing to specific client:", err)
		}
	} else{
		fmt.Println("user is blocked")
		blockedMsg := make([]byte, 1024)
		copy(blockedMsg[4:],"This user has blocked you, your request cannot be sent")
		sender.conn.Write(blockedMsg)
	}
	} 

func blockClient(Client client,  clientToBlock int) {
	returnMsg := make([]byte, 1024);

	if  clientToBlock < len(connList) {
	connList[Client.n-1].blockedClients = append(connList[Client.n-1].blockedClients, connList[clientToBlock])
	copy(returnMsg[6:], []byte(fmt.Sprintf("User %d blocked succesfuly!\n",  clientToBlock+1)))
	fmt.Printf("Client n %d blocked client n %d\n", Client.n, clientToBlock  +1 )

	} else {
		copy(returnMsg[6:], []byte(fmt.Sprintf("User %d was not blocked succesfuly!\n",  clientToBlock+1)))
		fmt.Printf("Client n %d was not blocked client n %d\n", Client.n, clientToBlock  +1 )

	}
	Client.conn.Write(returnMsg);

	}