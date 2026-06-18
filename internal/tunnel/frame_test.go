package tunnel_test

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/Ajibose/Relay/internal/tunnel"
)

func TestFrameFunctions(t *testing.T) {
	cases := []struct {
		streamId uint32
		msgType  uint8
		payload  []byte
	}{
		{3, tunnel.OPEN, []byte("")},
		{3, tunnel.DATA, []byte("Hello")},
		{3, tunnel.CLOSE, []byte("")},
	}

	for _, c := range cases {
		t.Run(fmt.Sprintf("%v: %v - %s", c.streamId, c.msgType, c.payload), func(t *testing.T) {
			var wr bytes.Buffer
			writeGot := tunnel.WriteFrame(&wr, c.streamId, c.msgType, c.payload)
			if writeGot != nil {
				t.Fatalf("WriteFrame(%v, %v, %v, %v) returns an error", wr, c.streamId, c.msgType, c.payload)
			}

			readGot, err := tunnel.ReadFrame(&wr)

			if err != nil {
				t.Fatalf("ReadFrame(%v) returns an error", wr)
			}

			if readGot.StreamId != c.streamId {
				t.Fatalf("ReadFrame(%v) failed, Expected: %v, Got: %v", wr, c.streamId, readGot.StreamId)
			}

			if readGot.MsgType != c.msgType {
				t.Fatalf("ReadFrame(%v) failed, Expected: %v, Got: %v", wr, c.msgType, readGot.MsgType)
			}

			if !(bytes.Equal(readGot.Payload, c.payload)) {
				t.Fatalf("ReadFrame(%v) failed, Expected Payload Length: %v, Got: %v",
					wr, len(c.payload), readGot.Payload)
			}
		})
	}
}
