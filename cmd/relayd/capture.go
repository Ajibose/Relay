package main

import (
	"database/sql"
	"sync"
	"time"
)

type Capture struct {
	TunnelID string
	Request []byte
	Response []byte
	StartTime time.Time
}

type CaptureStore struct {
	active map[uint32]*Capture
	mu sync.Mutex
	db *sql.DB
}
