package tunnel

import (
	"net"
	"sync"
)

o
// Mux holds the per-tunnel state which lets many streams share a single
// connection. It owns the tunnel conn, a map of stream IDs to endpoint
// connections (visitor conns on relayd, local conns on relayc), and the
// mutex that serializes concurrent writes to the tunnel.
//
// Callers of relayd generate stream IDs via AddStream (counter-based).
// Callers of relayc receive stream IDs from the peer and register via ddStreamWithID.
type Mux struct {
	Conn      net.Conn // tunnel connection
	WriteMu   sync.Mutex // guards wrties to Conn
	Streams   map[uint32]net.Conn // streamId to endpoint Conn(Visitor Conns or Local Conns)
	StreamsMu sync.Mutex // guards write and read from Streams
	Counter   uint32 // next stream ID to assign. Used by only relayd
}


// WriteFrame method safely writes Frame to tunnelConn owned by Mux.
// concurrent write are serialized by writeMu. Without it, two goroutines 
// writing at the same time might interleave bytes and corrupt frames on the wire.
func (m *Mux) WriteFrame(streamId uint32, msgType uint8, payload []byte) error {
	m.WriteMu.Lock()
	defer m.WriteMu.Unlock()

	err := WriteFrame(m.Conn, streamId, msgType, payload)
	if err != nil {
		return err
	}

	return nil
}

// CloseAll closes every registered endpoint connection and clears the streams map. 
// Used on tunnel shutdown: closing each conn unblocks any pump goroutine 
// currently in Read, letting them exit cleanly rather than leaking.
func (m *Mux) CloseAll() {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()
	for streamId, conn := range m.Streams {
		conn.Close()
		delete(m.Streams, streamId)
	}
}

// AddStreamWithID registers conn under the given streamID. It is used by relayc,
// which receives stream IDs from relayd in OPEN frames rather than generating
// its own. relayd should use AddStream, which auto-assigns from the counter.
func (m *Mux) AddStreamWithID(streamID uint32, conn net.Conn) {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()
	m.Streams[streamID] = conn
}

// AddStream regsters conn under a new streamId gotten from mux Counter and 
// returns the Id. It is used by relayd which assigns Id to each newly accepted
// visitor connection. Relayc should use AddStreamWithID which takes stream id chosen by peer
func (m *Mux) AddStream(conn net.Conn) uint32 {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()

	streamId := m.Counter
	m.Counter++
	m.Streams[streamId] = conn

	return streamId
}

// GetStream returns connection associated with StreamId from streams
// or nil if not found
func (m *Mux) GetStream(streamId uint32) net.Conn {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()

	conn := m.Streams[streamId]

	return conn
}

// RemoveStream deletes from Streams, the connection registered under streamId
// it is called by both side when a Frame with messageType CLOSE arrives
// or by the sending pump when its endpoint conn errors out.
func (m *Mux) RemoveStream(streamId uint32) {
	m.StreamsMu.Lock()
	defer m.StreamsMu.Unlock()

	delete(m.Streams, streamId)
}


// NewMux creates a new Mux with the given tunnel connection. The internal
// streams map is initialized here. Callers should always use NewMux rather
// than constructing a Mux literal since writes to a nil map panic.
func NewMux(conn net.Conn) *Mux {
	m := &Mux{
		Conn:    conn,
		Streams: make(map[uint32]net.Conn),
	}

	return m
}
