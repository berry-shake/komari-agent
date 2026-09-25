package server

import (
	"bytes"
	"log"
)

const maxTaskOutput = 64 << 10 // Per stream; worst-case JSON escaping still fits the report limit.
type limitedOutput struct {
	data      bytes.Buffer
	truncated bool
}

func (b *limitedOutput) Write(p []byte) (int, error) {
	n := len(p)
	remaining := maxTaskOutput - b.data.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.data.Write(p)
	return n, nil
}
func (b *limitedOutput) String() string {
	if b.truncated {
		return b.data.String() + "\n[output truncated]"
	}
	return b.data.String()
}
func (b *limitedOutput) Len() int { return b.data.Len() }

var commandSlots = make(chan struct{}, 4)
var terminalSlots = make(chan struct{}, 4)
var pingSlots = make(chan struct{}, 64)

func launchBounded(slots chan struct{}, task func()) bool {
	select {
	case slots <- struct{}{}:
		go func() { defer func() { <-slots }(); task() }()
		return true
	default:
		log.Println("Task concurrency limit reached")
		return false
	}
}
