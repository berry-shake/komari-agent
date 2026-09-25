package ws

import (
	"encoding/json"
	"github.com/komari-monitor/komari-agent/utils"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type SafeConn struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func NewSafeConn(conn *websocket.Conn) *SafeConn {
	return &SafeConn{
		conn: conn,
		mu:   sync.Mutex{},
	}
}

func (sc *SafeConn) WriteMessage(messageType int, data []byte) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteMessage(messageType, data)
}

func (sc *SafeConn) WriteJSON(v interface{}) error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.WriteJSON(v)
}

func (sc *SafeConn) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn.Close()
}
func (sc *SafeConn) ReadMessage() (int, []byte, error) {
	kind, reader, err := sc.conn.NextReader()
	if err != nil {
		return kind, nil, err
	}
	data, err := utils.ReadBounded(reader, utils.MaxMessageBytes)
	if err != nil {
		_ = sc.conn.Close()
	}
	return kind, data, err
}
func (sc *SafeConn) ReadJSON(v interface{}) error {
	_, data, err := sc.ReadMessage()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
func (sc *SafeConn) SetReadDeadline(t time.Time) error {
	// sc.mu.Lock()
	// defer sc.mu.Unlock()
	return sc.conn.SetReadDeadline(t)
}
func (sc *SafeConn) GetConn() *websocket.Conn {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	return sc.conn
}
