package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"reflect"
	"sync"
	"sync/atomic"
	"time"

	"github.com/searchktools/fast-server/core/rpc/codec"
	"github.com/searchktools/fast-server/core/rpc/protocol"
	"github.com/searchktools/fast-server/core/rpc/registry"
)

var (
	ErrServerClosed = errors.New("server closed")
)

// Server represents an RPC server
type Server struct {
	registry  *registry.ServiceRegistry
	listener  net.Listener
	codec     codec.Codec
	mu        sync.RWMutex
	conns     map[net.Conn]struct{}
	activeReqs atomic.Int64
	shutdown   atomic.Bool
}

// Metadata holds RPC request metadata
type Metadata struct {
	Service string
	Method  string
}

// NewServer creates a new RPC server
func NewServer(opts ...Option) *Server {
	s := &Server{
		registry: registry.NewRegistry(),
		codec:    &codec.JSONCodec{},
		conns:    make(map[net.Conn]struct{}),
	}

	for _, opt := range opts {
		opt(s)
	}

	return s
}

// Option configures a server
type Option func(*Server)

// WithCodec sets the codec
func WithCodec(c codec.Codec) Option {
	return func(s *Server) {
		s.codec = c
	}
}

// Register registers a service
func (s *Server) Register(serviceName string, service interface{}) error {
	return s.registry.Register(serviceName, service)
}

// ListenAndServe starts the RPC server
func (s *Server) ListenAndServe(addr string) error {
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = ln
	log.Printf("🚀 RPC Server listening on %s", addr)

	return s.Serve(ln)
}

// Serve accepts connections on the listener
func (s *Server) Serve(ln net.Listener) error {
	defer ln.Close()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if s.shutdown.Load() {
				return nil
			}
			log.Printf("❌ Accept error: %v", err)
			continue
		}

		s.trackConn(conn, true)
		go s.handleConn(conn)
	}
}

// trackConn tracks active connections
func (s *Server) trackConn(conn net.Conn, add bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if add {
		s.conns[conn] = struct{}{}
	} else {
		delete(s.conns, conn)
	}
}

// handleConn handles a client connection
func (s *Server) handleConn(conn net.Conn) {
	defer func() {
		conn.Close()
		s.trackConn(conn, false)
	}()

	// Set read timeout
	conn.SetReadDeadline(time.Now().Add(5 * time.Minute))

	for {
		// Read frame header
		headerBuf := make([]byte, protocol.HeaderSize)
		if _, err := io.ReadFull(conn, headerBuf); err != nil {
			if err != io.EOF {
				log.Printf("❌ Read header error: %v", err)
			}
			return
		}

		// Parse header
		frame, err := protocol.DecodeHeader(headerBuf)
		if err != nil {
			log.Printf("❌ Decode header error: %v", err)
			return
		}

		// Get frame size
		frameSize, err := protocol.GetFrameSize(headerBuf)
		if err != nil {
			log.Printf("❌ Get frame size error: %v", err)
			return
		}

		// Read full frame
		fullBuf := make([]byte, frameSize)
		copy(fullBuf, headerBuf)
		if _, err := io.ReadFull(conn, fullBuf[protocol.HeaderSize:]); err != nil {
			log.Printf("❌ Read frame error: %v", err)
			return
		}

		// Decode full frame
		frame, err = protocol.Decode(fullBuf)
		if err != nil {
			log.Printf("❌ Decode frame error: %v", err)
			return
		}

		// Handle frame based on type
		switch frame.Type {
		case protocol.TypeRequest:
			s.handleRequest(conn, frame)
		case protocol.TypePing:
			s.handlePing(conn, frame)
		default:
			log.Printf("⚠️  Unknown frame type: %d", frame.Type)
		}

		// Reset read deadline
		conn.SetReadDeadline(time.Now().Add(5 * time.Minute))
	}
}

// handleRequest handles an RPC request
func (s *Server) handleRequest(conn net.Conn, frame *protocol.Frame) {
	s.activeReqs.Add(1)
	defer s.activeReqs.Add(-1)

	// Parse metadata
	var meta Metadata
	if err := json.Unmarshal(frame.Metadata, &meta); err != nil {
		s.sendError(conn, frame.RequestID, fmt.Errorf("invalid metadata: %w", err))
		return
	}

	// Get service and method
	svc, method, err := s.registry.GetMethod(meta.Service, meta.Method)
	if err != nil {
		s.sendError(conn, frame.RequestID, err)
		return
	}

	// Decode argument
	arg := reflect.New(method.ArgType).Interface()
	if err := s.codec.Decode(frame.Payload, arg); err != nil {
		s.sendError(conn, frame.RequestID, fmt.Errorf("decode arg error: %w", err))
		return
	}

	// Call method
	ctx := context.Background()
	reply, err := s.registry.Call(ctx, svc.Name, method.Name, arg)
	if err != nil {
		s.sendError(conn, frame.RequestID, err)
		return
	}

	// Encode reply
	replyData, err := s.codec.Encode(reply)
	if err != nil {
		s.sendError(conn, frame.RequestID, fmt.Errorf("encode reply error: %w", err))
		return
	}

	// Send response
	respFrame := protocol.NewFrame(protocol.TypeResponse, frame.RequestID)
	respFrame.Payload = replyData

	if _, err := conn.Write(respFrame.Encode()); err != nil {
		log.Printf("❌ Write response error: %v", err)
	}
}

// handlePing handles a ping request
func (s *Server) handlePing(conn net.Conn, frame *protocol.Frame) {
	pongFrame := protocol.NewFrame(protocol.TypePong, frame.RequestID)
	conn.Write(pongFrame.Encode())
}

// sendError sends an error response
func (s *Server) sendError(conn net.Conn, requestID uint32, err error) {
	errFrame := protocol.NewFrame(protocol.TypeError, requestID)
	errFrame.Payload = []byte(err.Error())
	conn.Write(errFrame.Encode())
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.shutdown.Store(true)

	if s.listener != nil {
		s.listener.Close()
	}

	// Close all connections
	s.mu.Lock()
	conns := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.mu.Unlock()

	for _, conn := range conns {
		conn.Close()
	}

	// Wait for active requests to complete
	done := make(chan struct{})
	go func() {
		for s.activeReqs.Load() > 0 {
			time.Sleep(100 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Stats returns server statistics
func (s *Server) Stats() map[string]interface{} {
	s.mu.RLock()
	numConns := len(s.conns)
	s.mu.RUnlock()

	return map[string]interface{}{
		"connections":     numConns,
		"active_requests": s.activeReqs.Load(),
		"services":        len(s.registry.ListServices()),
	}
}
