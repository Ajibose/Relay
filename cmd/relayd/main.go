package main

import (
	"errors"
	"fmt"
	"io"
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
			if !errors.Is(err, io.EOF) {
				fmt.Println("Error Reading Frame", err)
			}
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
	defer m.RemoveStream(streamId)

	err := m.WriteFrame(streamId, tunnel.OPEN, nil)
	if err != nil {
		return  // tunnel already broken, don't try
	}
	

	buf := make([]byte, 1024)
	for {
		n, err := visitorConn.Read(buf)
		if err != nil {
			m.WriteFrame(streamId, tunnel.CLOSE, nil)
			return
		}

		err = m.WriteFrame(streamId, tunnel.DATA, buf[:n])
		if err != nil {
			return // tunnel broken up mid-stream, give up on the stream
		}
	}
}
