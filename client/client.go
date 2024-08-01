package main

import (
    "encoding/binary"
    "fmt"
    "net"
    "io"
)

func main() {
    conn, err := net.Dial("tcp", "localhost:8080")

    if err != nil {
        fmt.Println(err)
        return
    }

    go func() {
        for {
            
            buffer := make([]byte, 1024)
            
			_, err := conn.Read(buffer)
            if err ==io.EOF{
                fmt.Printf("Server disconected. Connection Closed\n")
                return
            }
            if err != nil {
                fmt.Println("Read error:", err)
                return
            }

            clientCount := binary.BigEndian.Uint16(buffer[:2])
            message := string(buffer[6:])
           
            fmt.Printf("\nConnected clients: %d\n", clientCount)
            fmt.Printf("Received: %s\n", message)
        }
    }()

    var message string
    var operation int;
    for {
        fmt.Println("1- Broacast message\n2-Send to specific user\n3-Block user")
        fmt.Scanf("%d",&operation);
        switch operation{
        case 1:
            for{
                fmt.Println("Enter Message : ")
                fmt.Scanf("%s", &message)
                if len(message)>1 {
                    break
                }
            }
            actionType := make([]byte,1024);
            binary.BigEndian.PutUint16(actionType[2:4],1);
            copy(actionType[6:], []byte(message))     
            _, err = conn.Write(actionType)
            if err != nil {
                fmt.Println("Error sending message:", err)
                break
            }

        case 2:
            var DestinationClient int;
            fmt.Println("Which client you wanna send a request to ? : ")
            fmt.Scanf("%d", &DestinationClient);
            fmt.Scanf("%s", &message)
            actionType := make([]byte,1024);
            binary.BigEndian.PutUint16(actionType[2:4],2);
            binary.BigEndian.PutUint16(actionType[4:6],uint16(DestinationClient));
			fmt.Println(binary.BigEndian.Uint16(actionType[4:6]))
            copy(actionType[6:], []byte(message))     
            _, err = conn.Write(actionType)
            if err != nil {
                fmt.Println("Error sending message:", err)
                break
            }

        case 3:
            var clientToBlock int;
            fmt.Println("Which Client you wanna block? : ");
            fmt.Scanf("%d", &clientToBlock)
            fmt.Printf("Blocking Client n* %d...\n", clientToBlock);
            actionType := make([]byte, 1024);
            binary.BigEndian.PutUint16(actionType[2:4], 3);
            binary.BigEndian.PutUint16(actionType[4:6], uint16(clientToBlock));
            _, err := conn.Write(actionType);
            if err != nil{
                panic(err);
            }
            fmt.Printf("Client %d blocked succefully!\n", clientToBlock);
        }   

    }
}


