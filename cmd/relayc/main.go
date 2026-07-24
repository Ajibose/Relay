package main

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net"

	"github.com/Ajibose/Relay/internal/tunnel"
)

func main() {
	tunnelConn, tErr := net.Dial("tcp", ":5000")
	if tErr != nil {
		fmt.Println("Failed to create connection to relay server", tErr)
		return
	}
	defer tunnelConn.Close()

	mux := tunnel.NewMux(tunnelConn)

	// One reader for the tunnel. ReadFrame reads a 9-byte header, then reads
	// the payload of the declared length, all consecutive on the connection.
	// Two goroutines calling ReadFrame would race for those bytes and each
	// parse a half-formed frame.
	err := readFromTunnel(mux)
	if err != nil {
		log.Println("Tunnel Closed:", err)
		mux.CloseAll()
	}
}

// DialLocal creates a connection to the client's local server
// Returns the connection dialed
func DialLocal() net.Conn {
	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		log.Println("Error connecting to local", err)
		return nil
	}

	return conn
}

// readFromTunnel is the single reader for relayc's side of the tunnel. It
// reads frames one at a time and dispatches by message type: OPEN dials
// localhost and spawns a pump for the new stream, DATA writes payload to
// the matching local conn, CLOSE tears down the stream. Returns when the
// tunnel breaks.
func readFromTunnel(mux *tunnel.Mux) error {
	for {
		frame, err := tunnel.ReadFrame(mux.Conn)
		if err != nil {
			log.Println("Error reading frame from tunnel: ", err)
			return err
		}

		switch frame.MsgType {
		case tunnel.OPEN:
			localConn := DialLocal()
			if localConn == nil {
				mux.WriteFrame(frame.StreamId, tunnel.CLOSE, nil)
				continue
			}
			mux.AddStreamWithID(frame.StreamId, localConn)
			go WriteLocaltoTunnel(frame.StreamId, localConn, mux)
		case tunnel.DATA:
			localConn := mux.GetStream(frame.StreamId)
			if localConn == nil {
				continue
			}
			localConn.Write(frame.Payload)
		case tunnel.CLOSE:
			localConn := mux.GetStream(frame.StreamId)
			if localConn != nil {
				localConn.Close()
			}
			mux.RemoveStream(frame.StreamId)
		}
	}
}

// WriteLocaltoTunnel is a per-stream pump from the local connection
// It reads response bytes from the local server and forward them as Data Frame
// into the tunnel. Sends CLOSE on exit. Returns when the local conn closes
// or the tunnel breaks.
func WriteLocaltoTunnel(streamId uint32, localConn net.Conn, mux *tunnel.Mux) {
	buf := make([]byte, 2048)
	defer localConn.Close()
	for {
		n, err := localConn.Read(buf)
		if err != nil {
			// io.EOF: normal end of an HTTP response body.
			// net.ErrClosed: readFromTunnel closed this conn (CLOSE frame arrived
			// while we were mid-Read). Both are expected, so don't log
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Println("Errors reading from local connection: ", err)
			}

			mux.RemoveStream(streamId)
			mux.WriteFrame(streamId, tunnel.CLOSE, nil)
			return
		}

		mux.WriteFrame(streamId, tunnel.DATA, buf[:n])
	}
}
