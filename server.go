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

var (
	connList      = make([]Client, 0)
	ipBlacklist   []string
	portBlacklist []string
)

const (
	Reset = "\033[0m"
	Green = "\033[32m"
	Red   = "\033[31m"
)

func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatal(err)
	}
	logInfo("Listening on port :8080...")
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				logWarning("Accept error: %v", err)
				continue
			}
			host, port, err := net.SplitHostPort(conn.RemoteAddr().String())
			if !isIpBlacklisted(host) && !isPortBlacklisted(port) {
				logInfo("Client %d - %s:%s connected", len(connList)+1, host, port)
				connList = append(connList, Client{conn: conn, n: len(connList) + 1, port: port, ip: host})
				go updateClients()
				go handleConnection(connList[len(connList)-1])
			} else {
				logWarning("Client with blacklisted IP or port attempted to connect: %s:%s", host, port)
				conn.Close()
			}
		}
	}()
	for {
		var operation int
		fmt.Printf("\n------------------------------------------------------------\n")
		fmt.Printf("1-Blacklist an IP\n2-Blacklist a port\n3-Show blacklisted IPs\n4-Show blacklisted ports\n")
		fmt.Printf("------------------------------------------------------------\n")

		fmt.Scanf("%d", &operation)
		switch operation {
		case 1:
			fmt.Print("Enter IP to blacklist: ")
			var ip string
			fmt.Scanf("%s", &ip)
			if len(strings.Split(ip, ".")) == 4 {
				if !isIpBlacklisted(ip) {
					ipBlacklist = append(ipBlacklist, ip)
					logInfo("IP %s is now blacklisted!\n", ip)
				} else {
					logWarning("IP %sis already blacklisted\n", ip)
				}
			}
		case 2:
			fmt.Print("Enter port to blacklist: ")
			var port string
			fmt.Scanf("%s", &port)
			if !isIpBlacklisted(port) {
				ipBlacklist = append(ipBlacklist, port)
				logInfo("IP %s is now blacklisted!\n", port)
			} else {
				logWarning("IP %sis already blacklisted\n", port)
			}

		case 3:
			if len(ipBlacklist) == 0 {
				fmt.Println("No blacklisted IPs")
			} else {
				fmt.Println("Blacklisted IPs:")
				for _, ip := range ipBlacklist {
					fmt.Println(ip)
				}
			}

		case 4:
			if len(portBlacklist) == 0 {
				fmt.Println("No blacklisted ports")
			} else {
				fmt.Println("Blacklisted ports:")
				for _, port := range portBlacklist {
					fmt.Println(port)
				}
			}
		}
	}
}

func handleConnection(client Client) {
	defer client.conn.Close()
	for {
		buf := make([]byte, 1026)
		n, err := client.conn.Read(buf)
		if err == io.EOF {
			logWarning("Client %d - %s:%s disconnected. Connection Closed", client.n, client.ip, client.port)
			removeClient(client)
			return
		} else if err != nil {
			logWarning("Connection closed: %v", err)
			return
		}

		operation := binary.BigEndian.Uint16(buf[2:4])
		switch operation {
		case 1:
			logInfo("Broadcasting message...")
			Broadcast(buf[:n], client)
		case 2:
			clientToSendIndex := binary.BigEndian.Uint16(buf[4:6]) - 1
			if clientToSendIndex >= uint16(len(connList)) {
				logWarning("Client does not exist")
				sendResponseMsg(client, "Client does not exist, try again with an existing client.")
			} else {
				clientToSend := connList[clientToSendIndex]
				logInfo("Sending message from Client %d - '%s:%s' to %d - '%s:%s'...", client.n, client.ip, client.port, clientToSend.n, clientToSend.ip, clientToSend.port)
				writeToSpecificClient(client, buf[:n])
			}
		case 3:
			clientToBlock := binary.BigEndian.Uint16(buf[4:6]) - 1
			logInfo("Client - %d - %s:%s wants to block client - %d - %s:%s...", client.n, client.ip, client.port, clientToBlock+1, connList[clientToBlock].ip, connList[clientToBlock].port)
			blockClient(client, int(clientToBlock))
		case 4:
			if binary.BigEndian.Uint16(buf[4:6]) == 1 {
				connList[client.n-1].isBroadcastBlocked = true
				logInfo("Client - %d - %s:%s blocked recieving broadcasts", client.n, client.ip, client.port)
				sendResponseMsg(connList[client.n-1], "Broadcasts are blocked")
			} else {
				connList[client.n-1].isBroadcastBlocked = false
				logInfo("Broadcasts are unblocked")
				sendResponseMsg(connList[client.n-1], "Broadcasts are unblocked")
			}
		default:
			logWarning("Unknown operation: %d", operation)
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
				logWarning("Error broadcasting: %v", err)
				sendResponseMsg(iniClient, "Error happened while broadcasting")
			}
		}
	}
	logInfo("Message broadcasted succesfully!")
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
			logWarning("Error writing to specific client: %v", err)
			sendResponseMsg(senderClient, "An error happened while sending your request")
		} else {
			logInfo("Message sent successfully!")
			sendResponseMsg(senderClient, "Sent successfully!")
		}
	} else {
		logWarning("User is blocked")
		sendResponseMsg(senderClient, "This user has blocked you, your request cannot be sent")
	}
}

func sendResponseMsg(client Client, msg string) {
	responseMsg := make([]byte, 1024)
	binary.BigEndian.PutUint16(responseMsg[2:4], 0)
	copy(responseMsg[8:], []byte(msg))
	_, err := client.conn.Write(responseMsg)
	if err != nil {
		logWarning("Error sending response to client: %v", err)
	}
}

func blockClient(client Client, clientToBlock int) {
	if clientToBlock < len(connList) {
		connList[client.n-1].blockedClients = append(connList[client.n-1].blockedClients, connList[clientToBlock])
		sendResponseMsg(client, fmt.Sprintf("Client - %d - %s:%s was successfully blocked!", clientToBlock+1, connList[clientToBlock].ip, connList[clientToBlock].port))
		logInfo("Client - %d - %s:%s blocked client %d %s:%s successfully", client.n, client.ip, client.port, clientToBlock+1, connList[clientToBlock].ip, connList[clientToBlock].port)
	} else {
		sendResponseMsg(client, fmt.Sprintf("Client - %d - %s:%s couldn't be blocked", clientToBlock+1, connList[clientToBlock].ip, connList[clientToBlock].port))
		logInfo("Client - %d - %s:%s coudln't block client - %d - %s:%s successfully", client.n, client.ip, client.port, clientToBlock+1, connList[clientToBlock].ip, connList[clientToBlock].port)

	}
}

func isBlocked(senderClient Client, receiverClient Client) bool {
	for _, blockedClient := range receiverClient.blockedClients {
		if blockedClient.conn == senderClient.conn {
			return true
		}
	}
	return false
}

func removeClient(client Client) {
	connList = append(connList[:client.n-1], connList[client.n:]...)
	if len(connList) > 0 {
		for i := client.n - 1; i < len(connList); i++ {
			connList[i].n--
		}
	}
}

func updateClients() {
	clientList := ""
	for _, c := range connList {
		clientList += fmt.Sprintf("%s:%s;", c.ip, c.port)
	}
	msg := make([]byte, 1024)
	binary.BigEndian.PutUint16(msg[0:2], uint16(len(clientList)))
	binary.BigEndian.PutUint16(msg[2:4], 5)
	copy(msg[6:], []byte(clientList))
	for _, client := range connList {
		client.conn.Write(msg)
	}
}

func isIpBlacklisted(ipToCheck string) bool {
	for _, ip := range ipBlacklist {
		if ipToCheck == ip {
			return true
		}
	}
	return false
}
func isPortBlacklisted(portToCheck string) bool {
	for _, port := range portBlacklist {
		if portToCheck == port {
			return true
		}
	}
	return false
}
func logInfo(message string, args ...interface{}) {
	log.Printf(Green+"INFO: "+Reset+message+"\n", args...)
}

func logWarning(message string, args ...interface{}) {
	log.Printf(Red+"WARNING: "+Reset+message+"\n", args...)
}
