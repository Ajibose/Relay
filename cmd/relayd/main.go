package main

import (
	"fmt"
	"io"
	"net"
)

func main() {
	cListener, cErr := net.Listen("tcp", ":5000")
	if cErr != nil {
		fmt.Println("Failed to create Listener", cErr)
		return
	}
	fmt.Println("Client Server listening on: ", 5000)
	defer cListener.Close()

	vListener, vErr := net.Listen("tcp", ":5001")
	if vErr != nil {
		fmt.Println("Failed to create Listener", vErr)
		return
	}
	fmt.Println("Visitor Server listening on: ", 5001)
	defer vListener.Close()

	clientConn, clientErr := cListener.Accept()
	if clientErr != nil {
		fmt.Println("Connection error", clientErr)
		return
	}

	visitorConn, visitorErr := vListener.Accept()
	if visitorErr != nil {
		fmt.Println("Connection error", visitorErr)
		return
	}

	fmt.Println("Client connection: ", clientConn)
	fmt.Println("Visitor connection: ", visitorConn)

	go Forward(visitorConn, clientConn)
	Forward(clientConn, visitorConn)
}

func Forward(conn1 net.Conn, conn2 net.Conn) {

	buffer := make([]byte, 2048)
	for {
		n, err := conn1.Read(buffer)
		if err != nil {
			if err != io.EOF {
				fmt.Println("Error Reading From Connection: ", conn1)
			}
			return
		}

		_, err = conn2.Write(buffer[:n])
		if err != nil {
			fmt.Println("Error Writing to Connection: ", conn2)
			return
		}
	}
}
