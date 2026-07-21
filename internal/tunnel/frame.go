// Package tunnel implements the wire protocol used between relayd and relayc.
//
// All multi-byte integers use big-endian (network byte order), matching the
// convention of most binary wire protocols including HTTP/2
package tunnel

import (
	"encoding/binary"
	"io"
)

// Frame is a unit of communication on the tunnel. Each Frame instance 
// carries a stream identifier so many streams can share one tunnel(for multiplexing),
// a message type, and an optional payload.
type Frame struct {
	StreamId uint32
	MsgType  uint8
	Payload  []byte
}

// Message Types identifying the role of the frame
// OPEN - annpunces a new stream(when a visitor connects)
// DATA - indicates that the frame is carrying a chunk of data of the stream
// CLOSE - indicates the end of the stream (from either side of the exchange)
const (
	OPEN = iota
	DATA
	CLOSE
)

// WriteFrame encodes a frame and writes it to wr. The wire format is a 9-byte
// header (streamId, msgType, payload length) followed by the payload itself.
// Write is not synchronized. Callers writing concurrently to the same
// io.Writer must serialize their calls (see Mux.WriteFrame).
func WriteFrame(wr io.Writer, streamId uint32, msgType uint8, payload []byte) error {
	// header = 4 bytes streamId + 1 byte msgType + 4 bytes payload length
	header := make([]byte, 9)

	binary.BigEndian.PutUint32(header[0:4], streamId)
	header[4] = msgType
	binary.BigEndian.PutUint32(header[5:9], uint32(len(payload)))

	_, err := wr.Write(header)
	if err != nil {
		return err
	}

	_, err = wr.Write(payload)
	if err != nil {
		return err
	}

	return nil
}

// ReadFrame reads a frame from rd and decodes it. It blocks until a complete frame
// arrives - 9-byte header, then the payload of the length as declared in the
// header. Only one goroutine should call ReadFrame on a given rd; concurrent
// callers would race for bytes and they would parse a garbled frame.
func ReadFrame(rd io.Reader) (Frame, error) {
	var f Frame

	header := make([]byte, 9)

	// io.ReadFull loops on Read until the buffer is filled or an error occurs.
	// TCP can return partial reads, so a plain Read might give fewer than 9
	// bytes and leave the frame's header incomplete.
	_, err := io.ReadFull(rd, header)
	if err != nil {
		return f, err
	}

	f.StreamId = binary.BigEndian.Uint32(header[0:4])
	f.MsgType = uint8(header[4])
	payloadSize := binary.BigEndian.Uint32(header[5:])

	payload := make([]byte, payloadSize)

	_, err = io.ReadFull(rd, payload)
	if err != nil {
		return f, err
	}

	f.Payload = payload
	return f, nil
}
