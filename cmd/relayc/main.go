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

	err := readFromTunnel(mux)
	if err != nil {
		log.Println("Tunnel Closed:", err)
		mux.CloseAll()
	}
}

func DialLocal() net.Conn {
	conn, err := net.Dial("tcp", "localhost:8080")

	if err != nil {
		log.Println("Error connecting to local", err)
		return nil
	}

	return conn
}

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

func WriteLocaltoTunnel(streamId uint32, localConn net.Conn, mux *tunnel.Mux) {
	buf := make([]byte, 2048)
	defer localConn.Close()
	for {
		n, err := localConn.Read(buf)
		if err != nil {
			// EOF is normal after an http response, should not be logged
			if !errors.Is(err, io.EOF) && !errors.Is(err, net.ErrClosed) {
				log.Println("Errors reading from local connection: ", err)
				break
			}

			mux.RemoveStream(streamId)
			mux.WriteFrame(streamId, tunnel.CLOSE, nil)
			return
		}

		mux.WriteFrame(streamId, tunnel.DATA, buf[:n])
	}
}
