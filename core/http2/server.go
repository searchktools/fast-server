package http2

import (
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Server provides HTTP/2 support with multiplexing and HPACK compression
type Server struct {
	addr    string
	handler http.Handler
	server  *http.Server
	h2      *http2.Server

	// TLS configuration for ALPN negotiation
	tlsConfig *tls.Config

	// Statistics
	stats struct {
		activeStreams    sync.Map // connection -> stream count
		totalConnections uint64
		totalStreams     uint64
	}

	mu     sync.RWMutex
	closed bool
}

// Config contains HTTP/2 server configuration
type Config struct {
	Addr                 string
	Handler              http.Handler
	TLSConfig            *tls.Config
	MaxConcurrentStreams uint32
	MaxReadFrameSize     uint32
	IdleTimeout          time.Duration
}

// NewServer creates a new HTTP/2 server
func NewServer(cfg Config) *Server {
	if cfg.MaxConcurrentStreams == 0 {
		cfg.MaxConcurrentStreams = 250
	}
	if cfg.MaxReadFrameSize == 0 {
		cfg.MaxReadFrameSize = 1 << 20 // 1MB
	}
	if cfg.IdleTimeout == 0 {
		cfg.IdleTimeout = 120 * time.Second
	}

	s := &Server{
		addr:    cfg.Addr,
		handler: cfg.Handler,
	}

	// Configure HTTP/2 server
	s.h2 = &http2.Server{
		MaxConcurrentStreams: cfg.MaxConcurrentStreams,
		MaxReadFrameSize:     cfg.MaxReadFrameSize,
		IdleTimeout:          cfg.IdleTimeout,
	}

	// Create HTTP server
	s.server = &http.Server{
		Addr:    cfg.Addr,
		Handler: cfg.Handler,
	}

	// Configure TLS with ALPN for HTTP/2
	if cfg.TLSConfig != nil {
		s.tlsConfig = cfg.TLSConfig.Clone()
		s.tlsConfig.NextProtos = []string{"h2", "http/1.1"}
		s.server.TLSConfig = s.tlsConfig
	} else {
		// h2c (HTTP/2 cleartext)
		s.server.Handler = h2c.NewHandler(s.server.Handler, s.h2)
	}

	return s
}

// ListenAndServe starts the HTTP/2 server
func (s *Server) ListenAndServe() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return fmt.Errorf("server is closed")
	}

	log.Printf("🚀 HTTP/2 Server starting on %s", s.addr)
	if s.tlsConfig != nil {
		log.Printf("   Protocol: h2 (TLS with ALPN)")
		return s.server.ListenAndServeTLS("", "")
	}

	log.Printf("   Protocol: h2c (cleartext)")
	return s.server.ListenAndServe()
}

// Close gracefully shuts down the server
func (s *Server) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}
	s.closed = true

	return s.server.Close()
}
