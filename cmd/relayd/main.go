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

	// One tunnel connection for now. Accept a single relayc and use it for the
	// lifetime of the process. Milestone 7 will make this a loop with subdomain routing
	// so multiple relayc can connect
	clientConn, clientErr := cListener.Accept()
	if clientErr != nil {
		fmt.Println("Connection error", clientErr)
		return
	}

	m := tunnel.NewMux(clientConn)

	// AcceptVisitorConnections runs in the background. writeToVisitors runs in main
	// so it block here until the tunnel breaks. When it returns, main exits and
	// deferred Close()s fire.
	go AcceptVisitorConnections(vListener, m)
	writeToVisitors(m)
}

// writeToVisitors is the single reader for relayd's side of the tunnel.
// It reads frames, looks up the visitor conn by streamId, and dispatches
// by message type. Returns when the tunnel breaks; io.EOF is expected on
// peer shutdown and not logged.
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

// AcceptVisitorConnections runs an Accept loop to wait on connections from visitors
// and spawns a new goroutine to write the visitor's bytes to the tunnel.
// Each visitor connection get a new stream Id and pump goroutine so one visitor's traffic
// doesn't block the Accept loop
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

// writeToTunnel writes bytes from a single visitor to the tunnel.
// Bytes chunks from visitor are wrapped in a Frame, tagged the streamId
// assigned to the connection. It send OPEN Frame at the start and CLOSE at the end
// If tunnel broke or visitor connection closes, it returns
func writeToTunnel(visitorConn net.Conn, streamId uint32, m *tunnel.Mux) {
	// RemoveStream first, then Close since defer run in reverse. Both fire on any exit
	// path (visitor error, tunnel error, panic) so cleanup is guaranteed.
	defer visitorConn.Close()
	defer m.RemoveStream(streamId)

	err := m.WriteFrame(streamId, tunnel.OPEN, nil)
	if err != nil {
		return // tunnel already broken, don't try
	}

	buf := make([]byte, 1024)
	for {
		n, err := visitorConn.Read(buf)

		if err != nil {
			// Visitor conn is dead. Notify the peer so it can clean up its
			// side; ignore the error since we're exiting either way.
			m.WriteFrame(streamId, tunnel.CLOSE, nil)
			return
		}

		err = m.WriteFrame(streamId, tunnel.DATA, buf[:n])
		if err != nil {
			return // tunnel broken up mid-stream, give up on the stream
		}
	}
}
