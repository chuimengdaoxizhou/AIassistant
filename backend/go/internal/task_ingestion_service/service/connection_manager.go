package service

import (
	"github.com/gorilla/websocket"
	"sync"
)

// ConnectionManager manages WebSocket connections.
type ConnectionManager struct {
	connections map[string]*websocket.Conn
	mu          sync.RWMutex
}

// NewConnectionManager creates a new ConnectionManager.
func NewConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		connections: make(map[string]*websocket.Conn),
	}
}

// Add registers a new connection for a user.
func (m *ConnectionManager) Add(userID string, conn *websocket.Conn) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.connections[userID] = conn
}

// Remove removes a connection for a user.
func (m *ConnectionManager) Remove(userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if conn, ok := m.connections[userID]; ok {
		conn.Close()
		delete(m.connections, userID)
	}
}

// Get retrieves a connection for a user.
func (m *ConnectionManager) Get(userID string) (*websocket.Conn, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	conn, ok := m.connections[userID]
	return conn, ok
}

// SendMessage sends a message to a specific user.
func (m *ConnectionManager) SendMessage(userID string, message []byte) bool {
	conn, ok := m.Get(userID)
	if !ok {
		return false
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	
	err := conn.WriteMessage(websocket.TextMessage, message)
	return err == nil
}
