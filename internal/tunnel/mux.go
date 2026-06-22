package tunnel

import (
	"net"
	"sync"
)

type Mux struct {
	Conn      net.Conn
	WriteMu   sync.Mutex
	Streams   map[uint32]net.Conn
	StreamsMu sync.Mutex
	Counter   uint32
}

func (m *Mux) WriteFrame(streamId uint32, msgType uint8, payload []byte) error {
	m.WriteMu.Lock()
	defer m.WriteMu.Unlock()
	
	err := WriteFrame(m.Conn, streamId, msgType, payload)
	if err != nil {
		return err
	}

	return nil
}

func (m *Mux) AddStream(conn net.Conn) uint32 {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()

	streamId := m.Counter
	m.Counter++
	m.Streams[streamId] = conn
	
	return streamId
}

func (m *Mux) GetStream(streamId uint32) net.Conn {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()

	conn := m.Streams[streamId]

	return conn
}

func (m *Mux) RemoveStream(streamId uint32) {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()

	delete(m.Streams, streamId)
}

func NewMux(conn net.Conn) *Mux {
	m := &Mux{
		Conn:    conn,
		Streams: make(map[uint32]net.Conn),
	}

	return m
}
