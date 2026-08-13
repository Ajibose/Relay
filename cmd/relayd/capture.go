package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"sync"
	"time"
)

// createCaptureTable is the schema for the capture table. It uses IF NOT
// EXISTS so it's safe to run on every relayd startup - the first run
// creates the table, every run after is a no-op. Schema changes made after
// a table already exists on disk aren't picked up by this; the db file has
// to be deleted and let relayd recreate it.
var createCaptureTable = `CREATE TABLE IF NOT EXISTS capture (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	tunnel_id VARCHAR(50),
	ts_in TEXT,
	method VARCHAR(10),
	path VARCHAR(25),
	request_header TEXT,
	response_header TEXT,
	request_body TEXT,
	status INTEGER,
	response_body TEXT,
	duration INTEGER,
	request_body_size INTEGER,
	outcome TEXT
)`

// Capture holds the raw bytes and lifecycle state for one stream's
// request/response exchange. One per stream, keyed by streamId in
// CaptureStore.active. Request/Response are buffered raw and parsed once,
// at Flush - not incrementally, since two different goroutines write to them.
type Capture struct {
	TunnelID  string    // which tunnel this stream belongs to. Hardcoded until M7 gives it real meaning
	Request   []byte    // raw bytes read from the visitor, accumulated by appendRequest
	Response  []byte    // raw bytes written to the visitor, accumulated by appendResponse
	Outcome   string    // set by Flush, describes why/how the stream ended
	StartTime time.Time // when Start was called, i.e. when the stream began

	// True once a write of response bytes to the visitor has succeeded.
	// Set after Write returns, not inside appendResponse.
	ResponseStarted bool

	// True once relayc's CLOSE frame confirms the response finished. No
	// ordering guarantee vs. the visitor disconnecting - see parked_decisions.md.
	ResponseComplete bool
}

// CaptureStore owns every in-flight Capture for the life of the process. It
// receives its db connection rather than opening one itself, so main.go
// keeps ownership thoughout the connection's lifetime
type CaptureStore struct {
	active map[uint32]*Capture
	mu     sync.Mutex
	db     *sql.DB
}

// appendRequest appends a chunk of raw request bytes to Request. Called only
// from writeToTunnel's per-visitor goroutine, so no locking needed - a given
// Capture's Request field only ever has one goroutine writing to it.
func (c *Capture) appendRequest(chunk []byte) {
	c.Request = append(c.Request, chunk...)
}

// appendResponse appends a chunk of raw response bytes to Response. Called
// only from writeToVisitors, the single shared demux goroutine. No locking
// needed here either, though note this is a different goroutine than the
// one calling appendRequest on the same Capture - each just only ever
// touches its own field.
func (c *Capture) appendResponse(chunk []byte) {
	c.Response = append(c.Response, chunk...)
}

// Get returns the Capture for streamId, or ok=false if no stream is active.
// (*Capture, bool) instead of a bare pointer forces callers to check ok,
// so a late signal after flush can't panic on a nil Capture.
func (st *CaptureStore) Get(id uint32) (*Capture, bool) {
	st.mu.Lock()
	defer st.mu.Unlock()

	c, ok := st.active[id]

	return c, ok
}

// Start registers a new Capture and returns it. Called before the OPEN
// frame is sent, so the entry exists before a response could come back.
func (st *CaptureStore) Start(tunnelID string, streamID uint32) *Capture {
	st.mu.Lock()
	defer st.mu.Unlock()

	newCapture := Capture{
		TunnelID:  tunnelID,
		Request:   []byte{},
		Response:  []byte{},
		Outcome:   "",
		StartTime: time.Now(),
	}

	st.active[streamID] = &newCapture

	return &newCapture
}

// Flush parses capture's bytes, writes one row, and removes streamId from
// active. Takes *Capture directly since the caller already holds it from
// Start. request_body/response_body fall back to raw bytes if parsing
// fails, so a malformed message still leaves something in the row.
// INSERT failures are logged but streamId is removed regardless - no retyr
func (st *CaptureStore) Flush(capture *Capture, streamId uint32, outcome string) {
	st.mu.Lock()
	defer st.mu.Unlock()

	var method, path, request_body string
	var request_header = make(map[string]string)

	requestStruct, err := parseRequest(capture.Request)
	request_body = string(capture.Request)

	if err == 0 {
		request_body = string(requestStruct.body)
		method = requestStruct.method
		path = requestStruct.requestTarget
		request_header = requestStruct.headers
	}

	var status int
	var response_body string
	var response_header = make(map[string]string)

	if capture.ResponseStarted {
		response_body = string(capture.Response)
		responseStruct, respErr := parseResponse(capture.Response)
		if respErr == 0 {
			status = responseStruct.statusCode
			response_header = responseStruct.headers
			response_body = string(responseStruct.body)
		}
	}

	elapsed := time.Now().Sub(capture.StartTime)
	duration := int(elapsed / time.Millisecond)

	requestHeaderJson, _ := json.Marshal(request_header)
	responseHeaderJson, _ := json.Marshal(response_header)
	formatedStartTime := capture.StartTime.Format(time.RFC3339)

	var insertErr error
	_, insertErr = st.db.Exec(
		`INSERT INTO capture 
			(
				tunnel_id, ts_in, method, path, request_header, response_header,
				request_body, status, response_body, duration, request_body_size, outcome
			)
			VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		capture.TunnelID, formatedStartTime, method, path, requestHeaderJson, responseHeaderJson,
		request_body, status, response_body, duration, len(request_body), outcome,
	)

	if insertErr != nil {
		log.Printf("Flush insert failed for stream %d: %v", streamId, insertErr)
	}

	delete(st.active, streamId)
}
