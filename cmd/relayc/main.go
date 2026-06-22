package main

import (
	"fmt"
	"io"
	"net"
)

func main() {
	tunnelConn, tErr := net.Dial("tcp", ":5000")
	if tErr != nil {
		fmt.Println("Failed to create Listener", tErr)
		return
	}
	defer tunnelConn.Close()

	localConn, lErr := net.Dial("tcp", ":8080")
	if lErr != nil {
		fmt.Println("Failed to create Listener", lErr)
		return
	}
	defer localConn.Close()

	go Forward(tunnelConn, localConn)
	Forward(localConn, tunnelConn)
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
