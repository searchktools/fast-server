package rpc

import (
	"testing"

	"github.com/searchktools/fast-server/core/rpc/protocol"
)

func TestFrameEncodeDecode(t *testing.T) {
	frame := protocol.NewFrame(protocol.TypeRequest, 12345)
	frame.Metadata = []byte("test metadata")
	frame.Payload = []byte("test payload")

	// Encode
	encoded := frame.Encode()

	// Decode
	decoded, err := protocol.Decode(encoded)
	if err != nil {
		t.Fatalf("Decode error: %v", err)
	}

	// Verify
	if decoded.Type != protocol.TypeRequest {
		t.Errorf("Expected type %d, got %d", protocol.TypeRequest, decoded.Type)
	}
	if decoded.RequestID != 12345 {
		t.Errorf("Expected requestID 12345, got %d", decoded.RequestID)
	}
	if string(decoded.Metadata) != "test metadata" {
		t.Errorf("Expected metadata 'test metadata', got '%s'", decoded.Metadata)
	}
	if string(decoded.Payload) != "test payload" {
		t.Errorf("Expected payload 'test payload', got '%s'", decoded.Payload)
	}
}

func TestFrameFlags(t *testing.T) {
	frame := protocol.NewFrame(protocol.TypeRequest, 1)
	
	// Test flag setting
	frame.SetFlag(protocol.FlagCompressed)
	if !frame.HasFlag(protocol.FlagCompressed) {
		t.Error("Expected compressed flag to be set")
	}

	frame.SetFlag(protocol.FlagPriority)
	if !frame.HasFlag(protocol.FlagPriority) {
		t.Error("Expected priority flag to be set")
	}

	// Test flag persistence through encode/decode
	encoded := frame.Encode()
	decoded, _ := protocol.Decode(encoded)

	if !decoded.HasFlag(protocol.FlagCompressed) {
		t.Error("Compressed flag lost after encode/decode")
	}
	if !decoded.HasFlag(protocol.FlagPriority) {
		t.Error("Priority flag lost after encode/decode")
	}
}

func BenchmarkFrameEncode(b *testing.B) {
	frame := protocol.NewFrame(protocol.TypeRequest, 1)
	frame.Metadata = []byte("service:Calculator,method:Add")
	frame.Payload = make([]byte, 1024)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = frame.Encode()
	}
}

func BenchmarkFrameDecode(b *testing.B) {
	frame := protocol.NewFrame(protocol.TypeRequest, 1)
	frame.Metadata = []byte("service:Calculator,method:Add")
	frame.Payload = make([]byte, 1024)
	encoded := frame.Encode()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = protocol.Decode(encoded)
	}
}
