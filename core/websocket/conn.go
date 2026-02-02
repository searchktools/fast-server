package websocket

import (
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
)

// OpCode represents WebSocket operation codes
type OpCode byte

const (
	OpContinuation OpCode = 0x0
	OpText         OpCode = 0x1
	OpBinary       OpCode = 0x2
	OpClose        OpCode = 0x8
	OpPing         OpCode = 0x9
	OpPong         OpCode = 0xA
)

// Frame represents a WebSocket frame
type Frame struct {
	Fin     bool
	OpCode  OpCode
	Masked  bool
	Payload []byte
}

// Message represents a complete WebSocket message
type Message struct {
	OpCode  OpCode
	Payload []byte
}

// Conn represents a WebSocket connection
type Conn struct {
	conn    net.Conn
	reader  *bufio.Reader
	writer  *bufio.Writer
	writeMu sync.Mutex

	maxMessageSize int64

	closed    bool
	closeMu   sync.Mutex
	closeOnce sync.Once
}

// NewConn creates a new WebSocket connection
func NewConn(conn net.Conn) *Conn {
	return &Conn{
		conn:           conn,
		reader:         bufio.NewReader(conn),
		writer:         bufio.NewWriter(conn),
		maxMessageSize: 1024 * 1024,
	}
}

func (c *Conn) SetMaxMessageSize(size int64) {
	c.maxMessageSize = size
}

func (c *Conn) ReadMessage() (*Message, error) {
	if c.IsClosed() {
		return nil, io.EOF
	}

	var message Message
	var fragments [][]byte

	for {
		frame, err := c.readFrame()
		if err != nil {
			return nil, err
		}

		switch frame.OpCode {
		case OpText, OpBinary:
			message.OpCode = frame.OpCode
			if frame.Fin {
				message.Payload = frame.Payload
				return &message, nil
			}
			fragments = append(fragments, frame.Payload)

		case OpContinuation:
			fragments = append(fragments, frame.Payload)
			if frame.Fin {
				totalLen := 0
				for _, frag := range fragments {
					totalLen += len(frag)
				}
				message.Payload = make([]byte, 0, totalLen)
				for _, frag := range fragments {
					message.Payload = append(message.Payload, frag...)
				}
				return &message, nil
			}

		case OpPing:
			if err := c.WriteFrame(&Frame{
				Fin:     true,
				OpCode:  OpPong,
				Payload: frame.Payload,
			}); err != nil {
				return nil, err
			}

		case OpPong:
			continue

		case OpClose:
			c.Close()
			return nil, io.EOF

		default:
			return nil, fmt.Errorf("unknown opcode: %d", frame.OpCode)
		}
	}
}

func (c *Conn) readFrame() (*Frame, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(c.reader, header); err != nil {
		return nil, err
	}

	frame := &Frame{
		Fin:    (header[0] & 0x80) != 0,
		OpCode: OpCode(header[0] & 0x0F),
		Masked: (header[1] & 0x80) != 0,
	}

	payloadLen := int64(header[1] & 0x7F)
	if payloadLen == 126 {
		extLen := make([]byte, 2)
		if _, err := io.ReadFull(c.reader, extLen); err != nil {
			return nil, err
		}
		payloadLen = int64(binary.BigEndian.Uint16(extLen))
	} else if payloadLen == 127 {
		extLen := make([]byte, 8)
		if _, err := io.ReadFull(c.reader, extLen); err != nil {
			return nil, err
		}
		payloadLen = int64(binary.BigEndian.Uint64(extLen))
	}

	if payloadLen > c.maxMessageSize {
		return nil, fmt.Errorf("message too large: %d > %d", payloadLen, c.maxMessageSize)
	}

	var maskingKey []byte
	if frame.Masked {
		maskingKey = make([]byte, 4)
		if _, err := io.ReadFull(c.reader, maskingKey); err != nil {
			return nil, err
		}
	}

	if payloadLen > 0 {
		frame.Payload = make([]byte, payloadLen)
		if _, err := io.ReadFull(c.reader, frame.Payload); err != nil {
			return nil, err
		}

		if frame.Masked {
			for i := int64(0); i < payloadLen; i++ {
				frame.Payload[i] ^= maskingKey[i%4]
			}
		}
	}

	return frame, nil
}

func (c *Conn) WriteMessage(opcode OpCode, payload []byte) error {
	return c.WriteFrame(&Frame{
		Fin:     true,
		OpCode:  opcode,
		Payload: payload,
	})
}

func (c *Conn) WriteText(text string) error {
	return c.WriteMessage(OpText, []byte(text))
}

func (c *Conn) WriteBinary(data []byte) error {
	return c.WriteMessage(OpBinary, data)
}

func (c *Conn) WriteFrame(frame *Frame) error {
	if c.IsClosed() {
		return io.EOF
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	firstByte := byte(frame.OpCode)
	if frame.Fin {
		firstByte |= 0x80
	}

	if err := c.writer.WriteByte(firstByte); err != nil {
		return err
	}

	payloadLen := len(frame.Payload)

	if payloadLen < 126 {
		c.writer.WriteByte(byte(payloadLen))
	} else if payloadLen < 65536 {
		c.writer.WriteByte(126)
		lengthBytes := make([]byte, 2)
		binary.BigEndian.PutUint16(lengthBytes, uint16(payloadLen))
		c.writer.Write(lengthBytes)
	} else {
		c.writer.WriteByte(127)
		lengthBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(lengthBytes, uint64(payloadLen))
		c.writer.Write(lengthBytes)
	}

	if payloadLen > 0 {
		if _, err := c.writer.Write(frame.Payload); err != nil {
			return err
		}
	}

	return c.writer.Flush()
}

func (c *Conn) Ping() error {
	return c.WriteFrame(&Frame{
		Fin:    true,
		OpCode: OpPing,
	})
}

func (c *Conn) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closeMu.Lock()
		c.closed = true
		c.closeMu.Unlock()

		closeFrame := &Frame{
			Fin:    true,
			OpCode: OpClose,
		}
		c.WriteFrame(closeFrame)

		err = c.conn.Close()
	})
	return err
}

func (c *Conn) IsClosed() bool {
	c.closeMu.Lock()
	defer c.closeMu.Unlock()
	return c.closed
}

func Upgrade(conn net.Conn, reader *bufio.Reader) (*Conn, error) {
	request := make(map[string]string)

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		line = strings.TrimSpace(line)
		if line == "" {
			break
		}

		parts := strings.SplitN(line, ":", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			request[strings.ToLower(key)] = value
		}
	}

	if request["upgrade"] != "websocket" {
		return nil, fmt.Errorf("not a websocket upgrade request")
	}

	if request["connection"] != "Upgrade" {
		return nil, fmt.Errorf("invalid connection header")
	}

	key := request["sec-websocket-key"]
	if key == "" {
		return nil, fmt.Errorf("missing sec-websocket-key")
	}

	acceptKey := computeAcceptKey(key)

	response := fmt.Sprintf(
		"HTTP/1.1 101 Switching Protocols\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n"+
			"\r\n",
		acceptKey,
	)

	if _, err := conn.Write([]byte(response)); err != nil {
		return nil, err
	}

	wsConn := &Conn{
		conn:           conn,
		reader:         reader,
		writer:         bufio.NewWriter(conn),
		maxMessageSize: 1024 * 1024,
	}

	return wsConn, nil
}

func computeAcceptKey(key string) string {
	const magicGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"
	h := sha1.New()
	h.Write([]byte(key + magicGUID))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}
