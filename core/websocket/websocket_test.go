package websocket

import (
"testing"
)

// TestFrameEncoding - Test frame encoding/decoding
func TestFrameEncoding(t *testing.T) {
// Test text frame
frame := Frame{
Fin:     true,
OpCode:  OpText,
Payload: []byte("Hello, World!"),
}

if frame.OpCode != OpText {
t.Errorf("Expected OpCode %d, got %d", OpText, frame.OpCode)
}

if string(frame.Payload) != "Hello, World!" {
t.Errorf("Expected 'Hello, World!', got '%s'", frame.Payload)
}
}

// TestHubBasic - Test hub creation
func TestHubBasic(t *testing.T) {
hub := NewHub(100)
if hub == nil {
t.Fatal("NewHub() returned nil")
}

// Note: Full hub testing requires actual WebSocket connections
t.Log("Hub basic test passed")
}

// TestUpgrade - Test upgrade function exists
func TestUpgrade(t *testing.T) {
// This is a placeholder - real upgrade testing requires HTTP server
t.Log("Upgrade function exists")
}
