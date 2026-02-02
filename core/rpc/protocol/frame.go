package protocol

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// Frame represents an RPC protocol frame
//
// Frame Format (16-byte header + variable data):
// +--------+--------+--------+--------+--------+--------+--------+--------+
// | Magic (4 bytes) | Ver | Type | Flags | Rsvd | RequestID (4 bytes) |
// +--------+--------+--------+--------+--------+--------+--------+--------+
// | MetaLen (2 bytes)        | PayloadLen (2 bytes)                      |
// +--------+--------+--------+--------+--------+--------+--------+--------+
// | Metadata (variable)      | Payload (variable)                        |
// +--------+--------+--------+--------+--------+--------+--------+--------+

const (
	// Magic number for RPC protocol: "RPC\0"
	Magic uint32 = 0x52504300

	// Protocol version
	Version byte = 0x01

	// Header size (fixed)
	HeaderSize = 16
)

// Frame types
const (
	TypeRequest      byte = 0x01 // Unary request
	TypeResponse     byte = 0x02 // Unary response
	TypeStreamOpen   byte = 0x03 // Open stream
	TypeStreamChunk  byte = 0x04 // Stream data chunk
	TypeStreamClose  byte = 0x05 // Close stream
	TypeError        byte = 0x06 // Error response
	TypePing         byte = 0x07 // Keepalive ping
	TypePong         byte = 0x08 // Keepalive pong
)

// Frame flags
const (
	FlagCompressed byte = 1 << 0 // Payload is compressed
	FlagPriority   byte = 1 << 1 // High priority message
	FlagOneWay     byte = 1 << 2 // One-way call (no response expected)
)

var (
	ErrInvalidMagic   = errors.New("invalid magic number")
	ErrInvalidVersion = errors.New("unsupported protocol version")
	ErrFrameTooLarge  = errors.New("frame too large")
)

// Frame represents a complete RPC frame
type Frame struct {
	Magic      uint32 // Magic number
	Version    byte   // Protocol version
	Type       byte   // Message type
	Flags      byte   // Flags
	Reserved   byte   // Reserved for future use
	RequestID  uint32 // Unique request identifier
	Metadata   []byte // Metadata (e.g., method name, headers)
	Payload    []byte // Payload data
}

// NewFrame creates a new frame
func NewFrame(typ byte, requestID uint32) *Frame {
	return &Frame{
		Magic:     Magic,
		Version:   Version,
		Type:      typ,
		RequestID: requestID,
	}
}

// SetFlag sets a flag bit
func (f *Frame) SetFlag(flag byte) {
	f.Flags |= flag
}

// HasFlag checks if a flag is set
func (f *Frame) HasFlag(flag byte) bool {
	return f.Flags&flag != 0
}

// Encode encodes the frame to bytes
func (f *Frame) Encode() []byte {
	metaLen := len(f.Metadata)
	payloadLen := len(f.Payload)
	totalLen := HeaderSize + metaLen + payloadLen

	buf := make([]byte, totalLen)

	// Header (16 bytes)
	binary.BigEndian.PutUint32(buf[0:4], f.Magic)
	buf[4] = f.Version
	buf[5] = f.Type
	buf[6] = f.Flags
	buf[7] = f.Reserved
	binary.BigEndian.PutUint32(buf[8:12], f.RequestID)
	binary.BigEndian.PutUint16(buf[12:14], uint16(metaLen))
	binary.BigEndian.PutUint16(buf[14:16], uint16(payloadLen))

	// Metadata
	if metaLen > 0 {
		copy(buf[HeaderSize:], f.Metadata)
	}

	// Payload
	if payloadLen > 0 {
		copy(buf[HeaderSize+metaLen:], f.Payload)
	}

	return buf
}

// DecodeHeader decodes only the frame header (16 bytes)
func DecodeHeader(buf []byte) (*Frame, error) {
	if len(buf) < HeaderSize {
		return nil, fmt.Errorf("buffer too small: need %d, got %d", HeaderSize, len(buf))
	}

	frame := &Frame{
		Magic:     binary.BigEndian.Uint32(buf[0:4]),
		Version:   buf[4],
		Type:      buf[5],
		Flags:     buf[6],
		Reserved:  buf[7],
		RequestID: binary.BigEndian.Uint32(buf[8:12]),
	}

	// Validate magic
	if frame.Magic != Magic {
		return nil, ErrInvalidMagic
	}

	// Validate version
	if frame.Version != Version {
		return nil, ErrInvalidVersion
	}

	return frame, nil
}

// Decode decodes a complete frame from bytes
func Decode(buf []byte) (*Frame, error) {
	if len(buf) < HeaderSize {
		return nil, fmt.Errorf("buffer too small: need %d, got %d", HeaderSize, len(buf))
	}

	frame, err := DecodeHeader(buf)
	if err != nil {
		return nil, err
	}

	// Parse lengths
	metaLen := int(binary.BigEndian.Uint16(buf[12:14]))
	payloadLen := int(binary.BigEndian.Uint16(buf[14:16]))

	// Validate total length
	expectedLen := HeaderSize + metaLen + payloadLen
	if len(buf) < expectedLen {
		return nil, fmt.Errorf("buffer too small: need %d, got %d", expectedLen, len(buf))
	}

	// Extract metadata
	if metaLen > 0 {
		frame.Metadata = make([]byte, metaLen)
		copy(frame.Metadata, buf[HeaderSize:HeaderSize+metaLen])
	}

	// Extract payload
	if payloadLen > 0 {
		frame.Payload = make([]byte, payloadLen)
		copy(frame.Payload, buf[HeaderSize+metaLen:HeaderSize+metaLen+payloadLen])
	}

	return frame, nil
}

// FrameSize returns the total size of a frame given metadata and payload lengths
func FrameSize(metaLen, payloadLen int) int {
	return HeaderSize + metaLen + payloadLen
}

// GetFrameSize returns the total size from a header buffer
func GetFrameSize(headerBuf []byte) (int, error) {
	if len(headerBuf) < HeaderSize {
		return 0, fmt.Errorf("buffer too small for header")
	}

	metaLen := int(binary.BigEndian.Uint16(headerBuf[12:14]))
	payloadLen := int(binary.BigEndian.Uint16(headerBuf[14:16]))

	return FrameSize(metaLen, payloadLen), nil
}
