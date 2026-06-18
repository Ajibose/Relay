package tunnel

import (
	"encoding/binary"
	"fmt"
	"io"
)

type Frame struct {
	StreamId uint32
	MsgType  uint8
	Payload  []byte
}

const (
	OPEN = iota
	DATA
	CLOSE
)

func WriteFrame(wr io.Writer, streamId uint32, msgType uint8, payload []byte) error {
	header := make([]byte, 9)

	binary.BigEndian.PutUint32(header[0:4], streamId)
	header[4] = msgType
	size := len(payload)
	binary.BigEndian.PutUint32(header[5:9], uint32(size))

	_, err := wr.Write(header)
	if err != nil {
		fmt.Println("Error writing header", err)
		return err
	}

	_, err = wr.Write(payload)
	if err != nil {
		fmt.Println("Error writing payload", err)
		return err
	}

	return nil
}

func ReadFrame(rd io.Reader) (Frame, error) {
	var f Frame

	header := make([]byte, 9)

	_, err := io.ReadFull(rd, header)
	if err != nil {
		fmt.Println("Error reading header", err)
		return f, err
	}

	f.StreamId = binary.BigEndian.Uint32(header[0:4])
	f.MsgType = uint8(header[4])
	payloadSize := binary.BigEndian.Uint32(header[5:])

	payload := make([]byte, payloadSize)

	_, err = io.ReadFull(rd, payload)
	if err != nil {
		fmt.Println("Error reading payload", err)
		return f, err
	}

	f.Payload = payload
	return f, nil
}
