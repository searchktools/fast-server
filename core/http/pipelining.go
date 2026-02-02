package http

import (
	"bufio"
	"io"
	"net"
)

// PipelineHandler handles HTTP/1.1 pipelining
type PipelineHandler struct {
	conn    net.Conn
	reader  *bufio.Reader
	maxSize int // Max pipeline queue size
}

// NewPipelineHandler creates a new pipeline handler
func NewPipelineHandler(conn net.Conn, maxSize int) *PipelineHandler {
	if maxSize == 0 {
		maxSize = 16 // Default: handle up to 16 pipelined requests
	}

	return &PipelineHandler{
		conn:    conn,
		reader:  bufio.NewReaderSize(conn, 8192),
		maxSize: maxSize,
	}
}

// ReadRequests reads multiple pipelined requests
func (ph *PipelineHandler) ReadRequests() ([]*Request, error) {
	requests := make([]*Request, 0, ph.maxSize)
	buf := make([]byte, 4096)

	for i := 0; i < ph.maxSize; i++ {
		// Try to read next request
		n, err := ph.reader.Read(buf)
		if err != nil {
			if err == io.EOF && len(requests) > 0 {
				// End of current batch
				break
			}
			return requests, err
		}

		if n == 0 {
			break
		}

		// Parse request
		req, err := ParseRequest(buf[:n])
		if err != nil {
			return requests, err
		}

		requests = append(requests, req)

		// If no more data immediately available, break
		if ph.reader.Buffered() == 0 {
			break
		}
	}

	return requests, nil
}

// WriteResponses writes multiple responses in batch
func (ph *PipelineHandler) WriteResponses(responses [][]byte) error {
	// Combine all responses into one write for efficiency
	totalSize := 0
	for _, resp := range responses {
		totalSize += len(resp)
	}

	combined := make([]byte, 0, totalSize)
	for _, resp := range responses {
		combined = append(combined, resp...)
	}

	_, err := ph.conn.Write(combined)
	return err
}

// PipelineConfig configures pipelining behavior
type PipelineConfig struct {
	Enabled     bool
	MaxPipeline int  // Max requests in pipeline
	MaxBatch    int  // Max batch size for writing
	KeepAlive   bool // Enable keep-alive
	IdleTimeout int  // Seconds before closing idle connection
}

// DefaultPipelineConfig returns default pipelining configuration
func DefaultPipelineConfig() PipelineConfig {
	return PipelineConfig{
		Enabled:     true,
		MaxPipeline: 16,
		MaxBatch:    16,
		KeepAlive:   true,
		IdleTimeout: 60,
	}
}

// HandlePipelinedConnection handles a connection with pipelining support
func HandlePipelinedConnection(conn net.Conn, config PipelineConfig, handler func(*Request) []byte) error {
	defer conn.Close()

	ph := NewPipelineHandler(conn, config.MaxPipeline)

	for {
		// Read batch of requests
		requests, err := ph.ReadRequests()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(requests) == 0 {
			return nil
		}

		// Process all requests and collect responses
		responses := make([][]byte, 0, len(requests))
		for _, req := range requests {
			resp := handler(req)
			responses = append(responses, resp)
			ReleaseRequest(req)
		}

		// Write all responses in batch
		if err := ph.WriteResponses(responses); err != nil {
			return err
		}

		// Check if connection should close
		if !config.KeepAlive {
			return nil
		}

		// Check if last request wanted to close
		if len(requests) > 0 && requests[len(requests)-1].Connection == "close" {
			return nil
		}
	}
}
