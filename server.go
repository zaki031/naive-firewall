package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
)

type client struct {
	conn net.Conn
	id   string
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
		go handleConnection(conn)
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()
	for {
		buf := make([]byte, 1024)
		_, err := conn.Read(buf)
		if err != nil {
			fmt.Println("Connection closed:", err)
			return
		}

		operation := binary.BigEndian.Uint16(buf[2:4])
		fmt.Println("Operation:", operation)
		switch operation {
		case 1:
			fmt.Println("Broadcasting message...")
			writeToConnection(connList, buf, conn)
		case 2:
			fmt.Println("Sending message to specific client...")
			writeToSpecificOperation(buf)
		default:
			fmt.Println("Unknown operation:", operation)
		}
	}
}

func writeToConnection(conns []client, msg []byte, iniConn net.Conn) {
	clientCount := uint16(len(conns))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	fmt.Printf("%s", msg[4:])
	for i := 0; i < len(conns); i++ {
		if conns[i].conn != iniConn {
			_, err := conns[i].conn.Write(msg)
			if err != nil {
				fmt.Println("Error writing to client:", err)
			}
		}
	}
}

func writeToSpecificOperation(msg []byte) {
	clientCount := uint16(len(connList))
	binary.BigEndian.PutUint16(msg[:2], clientCount)
	user := binary.BigEndian.Uint16(msg[4:6])
	fmt.Println("Sending to user:", binary.BigEndian.Uint16(msg[4:6]))
		conn := connList[user-1].conn
		_, err := conn.Write(msg)
		if err != nil {
			fmt.Println("Error writing to specific client:", err)
		}
	} 