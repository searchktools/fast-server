package sse

import (
"strings"
"testing"
"time"
)

// TestBrokerBasic - Basic broker functionality
func TestBrokerBasic(t *testing.T) {
broker := NewBroker(100, 30*time.Second)
if broker == nil {
t.Fatal("NewBroker returned nil")
}

// Give broker time to start
time.Sleep(50 * time.Millisecond)

count := broker.ClientCount()
if count != 0 {
t.Errorf("Expected 0 clients, got %d", count)
}

// Broker runs automatically and cleans up
}

// TestClient - Test client creation
func TestClient(t *testing.T) {
client := NewClient("test-client", 10)
if client.ID != "test-client" {
t.Errorf("Expected client ID 'test-client', got '%s'", client.ID)
}
client.Close()
}

// TestFormatEvent - Test SSE event formatting
func TestFormatEvent(t *testing.T) {
event := &Event{
ID:    "123",
Event: "message",
Data:  "Hello, World!",
Retry: 5000,
}

formatted := string(FormatEvent(event))

// Check all fields are present
if !strings.Contains(formatted, "id: 123") {
t.Error("Missing id field")
}
if !strings.Contains(formatted, "event: message") {
t.Error("Missing event field")
}
if !strings.Contains(formatted, "data: Hello, World!") {
t.Error("Missing data field")
}
if !strings.Contains(formatted, "retry: 5000") {
t.Error("Missing retry field")
}
if !strings.HasSuffix(formatted, "\n\n") {
t.Error("Should end with double newline")
}
}

// TestEventBuilder - Test event builder
func TestEventBuilder(t *testing.T) {
// Simple event test
event := &Event{
ID:    "456",
Event: "update",
Data:  "test data",
Retry: 3000,
}

if event.ID != "456" {
t.Errorf("Expected ID '456', got '%s'", event.ID)
}
if event.Event != "update" {
t.Errorf("Expected event 'update', got '%s'", event.Event)
}
if event.Data != "test data" {
t.Errorf("Expected data 'test data', got '%s'", event.Data)
}
if event.Retry != 3000 {
t.Errorf("Expected retry 3000, got %d", event.Retry)
}
}
