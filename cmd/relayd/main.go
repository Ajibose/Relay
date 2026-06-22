package main

import (
	"fmt"
	"net"

	"github.com/Ajibose/Relay/internal/tunnel"
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

	m := tunnel.NewMux(clientConn)

	go AcceptVisitorConnections(vListener, m)
	writeToVisitors(m)
}

func writeToVisitors(m *tunnel.Mux) error {
	for {
		f, err := tunnel.ReadFrame(m.Conn)
		if err != nil {
			return err
		}

		visitorConn := m.GetStream(f.StreamId)
		if visitorConn == nil {
			continue
		}

		switch f.MsgType {
		case tunnel.OPEN:
			continue
		case tunnel.CLOSE:
			visitorConn.Close()
			m.RemoveStream(f.StreamId)
		default:
			visitorConn.Write(f.Payload)
		}
	}
}

func AcceptVisitorConnections(vListener net.Listener, m *tunnel.Mux) {
	for {
		visitorConn, visitorErr := vListener.Accept()
		if visitorErr != nil {
			fmt.Println("Connection error", visitorErr)
			return
		}

		streamId := m.AddStream(visitorConn)

		go writeToTunnel(visitorConn, streamId, m)
	}
}

func writeToTunnel(visitorConn net.Conn, streamId uint32, m *tunnel.Mux) {
	defer visitorConn.Close()
	m.WriteFrame(streamId, tunnel.OPEN, nil)
	buf := make([]byte, 1024)
	for {
		n, err := visitorConn.Read(buf)
		if err != nil {
			m.RemoveStream(streamId)
			m.WriteFrame(streamId, tunnel.CLOSE, nil)
			return
		}

		m.WriteFrame(streamId, tunnel.DATA, buf[:n])
	}
}
