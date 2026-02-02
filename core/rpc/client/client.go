package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/searchktools/fast-server/core/rpc/codec"
	"github.com/searchktools/fast-server/core/rpc/protocol"
)

var (
	ErrClientClosed = errors.New("client closed")
	ErrTimeout      = errors.New("request timeout")
)

// Client represents an RPC client
type Client struct {
	conn      net.Conn
	codec     codec.Codec
	reqID     atomic.Uint32
	pending   sync.Map // requestID -> *Call
	mu        sync.Mutex
	closed    bool
	closeOnce sync.Once
}

// Call represents an active RPC call
type Call struct {
	Service string
	Method  string
	Args    interface{}
	Reply   interface{}
	Error   error
	Done    chan *Call
}

// NewClient creates a new RPC client
func NewClient(addr string, opts ...Option) (*Client, error) {
	conn, err := net.DialTimeout("tcp", addr, 5*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial error: %w", err)
	}

	client := &Client{
		conn:  conn,
		codec: &codec.JSONCodec{},
	}

	for _, opt := range opts {
		opt(client)
	}

	// Start receive loop
	go client.receive()

	return client, nil
}

// Option configures a client
type Option func(*Client)

// WithClientCodec sets the codec
func WithClientCodec(c codec.Codec) Option {
	return func(client *Client) {
		client.codec = c
	}
}

// Call makes a synchronous RPC call
func (c *Client) Call(ctx context.Context, service, method string, args, reply interface{}) error {
	call := &Call{
		Service: service,
		Method:  method,
		Args:    args,
		Reply:   reply,
		Done:    make(chan *Call, 1),
	}

	c.Go(call)

	select {
	case <-ctx.Done():
		return ctx.Err()
	case call := <-call.Done:
		return call.Error
	}
}

// Go makes an asynchronous RPC call
func (c *Client) Go(call *Call) *Call {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		call.Error = ErrClientClosed
		call.done()
		return call
	}
	c.mu.Unlock()

	// Generate request ID
	requestID := c.reqID.Add(1)

	// Register call
	c.pending.Store(requestID, call)

	// Prepare metadata
	meta := map[string]string{
		"service": call.Service,
		"method":  call.Method,
	}
	metaData, _ := json.Marshal(meta)

	// Encode arguments
	payload, err := c.codec.Encode(call.Args)
	if err != nil {
		call.Error = fmt.Errorf("encode args error: %w", err)
		c.pending.Delete(requestID)
		call.done()
		return call
	}

	// Create frame
	frame := protocol.NewFrame(protocol.TypeRequest, requestID)
	frame.Metadata = metaData
	frame.Payload = payload

	// Send frame
	if err := c.send(frame); err != nil {
		call.Error = err
		c.pending.Delete(requestID)
		call.done()
		return call
	}

	return call
}

// send sends a frame to the server
func (c *Client) send(frame *protocol.Frame) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return ErrClientClosed
	}

	data := frame.Encode()
	_, err := c.conn.Write(data)
	return err
}

// receive receives responses from the server
func (c *Client) receive() {
	for {
		// Read frame header
		headerBuf := make([]byte, protocol.HeaderSize)
		if _, err := io.ReadFull(c.conn, headerBuf); err != nil {
			if err != io.EOF {
				fmt.Printf("❌ Read header error: %v\n", err)
			}
			c.Close()
			return
		}

		// Get frame size
		frameSize, err := protocol.GetFrameSize(headerBuf)
		if err != nil {
			fmt.Printf("❌ Get frame size error: %v\n", err)
			c.Close()
			return
		}

		// Read full frame
		fullBuf := make([]byte, frameSize)
		copy(fullBuf, headerBuf)
		if _, err := io.ReadFull(c.conn, fullBuf[protocol.HeaderSize:]); err != nil {
			fmt.Printf("❌ Read frame error: %v\n", err)
			c.Close()
			return
		}

		// Decode frame
		frame, err := protocol.Decode(fullBuf)
		if err != nil {
			fmt.Printf("❌ Decode frame error: %v\n", err)
			c.Close()
			return
		}

		// Handle frame
		c.handleFrame(frame)
	}
}

// handleFrame handles a received frame
func (c *Client) handleFrame(frame *protocol.Frame) {
	requestID := frame.RequestID

	// Get pending call
	val, ok := c.pending.LoadAndDelete(requestID)
	if !ok {
		fmt.Printf("⚠️  Unexpected response for request %d\n", requestID)
		return
	}

	call := val.(*Call)

	switch frame.Type {
	case protocol.TypeResponse:
		// Decode reply
		if err := c.codec.Decode(frame.Payload, call.Reply); err != nil {
			call.Error = fmt.Errorf("decode reply error: %w", err)
		}

	case protocol.TypeError:
		call.Error = errors.New(string(frame.Payload))

	case protocol.TypePong:
		// Ping response, no action needed

	default:
		call.Error = fmt.Errorf("unexpected frame type: %d", frame.Type)
	}

	call.done()
}

// done completes a call
func (call *Call) done() {
	select {
	case call.Done <- call:
	default:
	}
}

// Ping sends a ping to the server
func (c *Client) Ping() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return ErrClientClosed
	}
	c.mu.Unlock()

	requestID := c.reqID.Add(1)
	frame := protocol.NewFrame(protocol.TypePing, requestID)

	// Register for pong
	call := &Call{Done: make(chan *Call, 1)}
	c.pending.Store(requestID, call)

	if err := c.send(frame); err != nil {
		c.pending.Delete(requestID)
		return err
	}

	// Wait for pong
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	select {
	case <-ctx.Done():
		c.pending.Delete(requestID)
		return ErrTimeout
	case <-call.Done:
		return call.Error
	}
}

// Close closes the client connection
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.mu.Lock()
		c.closed = true
		c.mu.Unlock()

		err = c.conn.Close()

		// Fail all pending calls
		c.pending.Range(func(key, value interface{}) bool {
			call := value.(*Call)
			call.Error = ErrClientClosed
			call.done()
			c.pending.Delete(key)
			return true
		})
	})
	return err
}
