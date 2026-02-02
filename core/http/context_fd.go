package http

import (
	"encoding/json"
	"net"
	"syscall"
)

// FDContext is a file-descriptor based context for epoll/kqueue
type FDContext struct {
	// File descriptor for syscall.Write
	fd int

	// Request
	request *Request

	// Parameters (fixed array for performance)
	paramKeys        [4]string
	paramValues      [4]string
	paramCount       int
	paramMapOverflow map[string]string

	// Response buffer (pre-allocated)
	responseBuf []byte

	// Response state
	responseHeaders map[string]string
	statusCode      int
	aborted         bool
}

// NewFDContext creates a new FD-based context
func NewFDContext(fd int, req *Request) *FDContext {
	return &FDContext{
		fd:          fd,
		request:     req,
		responseBuf: make([]byte, 0, 4096),
		statusCode:  200,
		aborted:     false,
	}
}

// Request information methods
func (c *FDContext) Method() string {
	return c.request.Method
}

func (c *FDContext) Path() string {
	return c.request.Path
}

func (c *FDContext) Param(key string) string {
	// Check fixed array first
	for i := 0; i < c.paramCount && i < 4; i++ {
		if c.paramKeys[i] == key {
			return c.paramValues[i]
		}
	}
	// Check overflow map
	if c.paramMapOverflow != nil {
		return c.paramMapOverflow[key]
	}
	return ""
}

func (c *FDContext) Query(key string) string {
	if c.request.Query == nil {
		return ""
	}
	return c.request.Query[key]
}

func (c *FDContext) Header(key string) string {
	// Check predefined headers first
	switch key {
	case "Content-Type":
		return c.request.ContentType
	case "Content-Length":
		return c.request.ContentLength
	case "User-Agent":
		return c.request.UserAgent
	case "Accept":
		return c.request.Accept
	case "Host":
		return c.request.Host
	case "Connection":
		return c.request.Connection
	}
	// Check extra headers
	if c.request.ExtraHeaders != nil {
		return c.request.ExtraHeaders[key]
	}
	return ""
}

func (c *FDContext) Body() []byte {
	return c.request.Body
}

func (c *FDContext) SetParam(key, value string) {
	if c.paramCount < 4 {
		c.paramKeys[c.paramCount] = key
		c.paramValues[c.paramCount] = value
		c.paramCount++
	} else {
		if c.paramMapOverflow == nil {
			c.paramMapOverflow = make(map[string]string)
		}
		c.paramMapOverflow[key] = value
	}
}

// writeResponse writes the response buffer to the file descriptor
func (c *FDContext) writeResponse() error {
	// Write all data (handle partial writes)
	written := 0
	for written < len(c.responseBuf) {
		n, err := syscall.Write(c.fd, c.responseBuf[written:])
		if err != nil {
			if err == syscall.EAGAIN || err == syscall.EWOULDBLOCK {
				// Would block - in production, should return to event loop
				// For now, retry immediately (simplified)
				continue
			}
			return err
		}
		written += n
	}
	return nil
}

// String sends a plain text response
func (c *FDContext) String(code int, s string) {
	c.responseBuf = c.responseBuf[:0]

	// Status line
	c.responseBuf = append(c.responseBuf, "HTTP/1.1 "...)
	c.responseBuf = appendInt(c.responseBuf, code)
	c.responseBuf = append(c.responseBuf, ' ')
	c.responseBuf = append(c.responseBuf, statusText(code)...)
	c.responseBuf = append(c.responseBuf, "\r\n"...)

	// Headers
	c.responseBuf = append(c.responseBuf, "Content-Type: text/plain\r\n"...)
	c.responseBuf = append(c.responseBuf, "Content-Length: "...)
	c.responseBuf = appendInt(c.responseBuf, len(s))
	c.responseBuf = append(c.responseBuf, "\r\n\r\n"...)

	// Body
	c.responseBuf = append(c.responseBuf, s...)

	// Write to socket
	c.writeResponse()
}

// JSON sends a JSON response
func (c *FDContext) JSON(code int, v any) {
	data, err := json.Marshal(v)
	if err != nil {
		c.Error(500, "Failed to marshal JSON")
		return
	}

	c.responseBuf = c.responseBuf[:0]

	// Status line
	c.responseBuf = append(c.responseBuf, "HTTP/1.1 "...)
	c.responseBuf = appendInt(c.responseBuf, code)
	c.responseBuf = append(c.responseBuf, ' ')
	c.responseBuf = append(c.responseBuf, statusText(code)...)
	c.responseBuf = append(c.responseBuf, "\r\n"...)

	// Headers
	c.responseBuf = append(c.responseBuf, "Content-Type: application/json\r\n"...)
	c.responseBuf = append(c.responseBuf, "Content-Length: "...)
	c.responseBuf = appendInt(c.responseBuf, len(data))
	c.responseBuf = append(c.responseBuf, "\r\n\r\n"...)

	// Body
	c.responseBuf = append(c.responseBuf, data...)

	// Write to socket
	c.writeResponse()
}

// Bytes sends a raw bytes response
func (c *FDContext) Bytes(code int, data []byte) {
	c.responseBuf = c.responseBuf[:0]

	// Status line
	c.responseBuf = append(c.responseBuf, "HTTP/1.1 "...)
	c.responseBuf = appendInt(c.responseBuf, code)
	c.responseBuf = append(c.responseBuf, ' ')
	c.responseBuf = append(c.responseBuf, statusText(code)...)
	c.responseBuf = append(c.responseBuf, "\r\n"...)

	// Headers
	c.responseBuf = append(c.responseBuf, "Content-Type: application/octet-stream\r\n"...)
	c.responseBuf = append(c.responseBuf, "Content-Length: "...)
	c.responseBuf = appendInt(c.responseBuf, len(data))
	c.responseBuf = append(c.responseBuf, "\r\n\r\n"...)

	// Body
	c.responseBuf = append(c.responseBuf, data...)

	// Write to socket
	c.writeResponse()
}

// Data sends a response with custom content type
func (c *FDContext) Data(code int, contentType string, data []byte) {
	c.responseBuf = c.responseBuf[:0]

	// Status line
	c.responseBuf = append(c.responseBuf, "HTTP/1.1 "...)
	c.responseBuf = appendInt(c.responseBuf, code)
	c.responseBuf = append(c.responseBuf, ' ')
	c.responseBuf = append(c.responseBuf, statusText(code)...)
	c.responseBuf = append(c.responseBuf, "\r\n"...)

	// Headers
	c.responseBuf = append(c.responseBuf, "Content-Type: "...)
	c.responseBuf = append(c.responseBuf, contentType...)
	c.responseBuf = append(c.responseBuf, "\r\n"...)
	c.responseBuf = append(c.responseBuf, "Content-Length: "...)
	c.responseBuf = appendInt(c.responseBuf, len(data))
	c.responseBuf = append(c.responseBuf, "\r\n\r\n"...)

	// Body
	c.responseBuf = append(c.responseBuf, data...)

	// Write to socket
	c.writeResponse()
}

// Error sends an error response
func (c *FDContext) Error(code int, message string) {
	c.JSON(code, map[string]any{
		"code":    code,
		"message": message,
	})
}

// Success sends a success response
func (c *FDContext) Success(data any) {
	c.JSON(200, map[string]any{
		"code":    0,
		"data":    data,
		"message": "success",
	})
}

// ServeFile serves a file using sendfile (zero-copy)
func (c *FDContext) ServeFile(filePath string) error {
	// TODO: Implement sendfile integration
	return nil
}

// Bind binds request body to a struct (not implemented for FD context)
func (c *FDContext) Bind(v any) error {
	return json.Unmarshal(c.request.Body, v)
}

// Conn returns nil (FD context doesn't use net.Conn)
func (c *FDContext) Conn() net.Conn {
	return nil
}

// GetHeader returns a request header value
func (c *FDContext) GetHeader(key string) string {
	return c.Header(key)
}

// SetHeader sets a response header
func (c *FDContext) SetHeader(key, value string) {
	// Headers are set inline in response buffer
	// For now, store in a map (can be optimized later)
	if c.responseHeaders == nil {
		c.responseHeaders = make(map[string]string, 8)
	}
	c.responseHeaders[key] = value
}

// Status sets the response status code
func (c *FDContext) Status(code int) {
	c.statusCode = code
}

// IsAborted returns whether the request has been aborted
func (c *FDContext) IsAborted() bool {
	return c.aborted
}

// Abort aborts the request processing
func (c *FDContext) Abort() {
	c.aborted = true
}

// Reset resets the context for reuse (memory not freed, just reset)
func (c *FDContext) Reset(fd int, req *Request) {
	c.fd = fd
	c.request = req

	// Reset params without freeing memory
	c.paramCount = 0
	// Clear overflow map if exists
	if c.paramMapOverflow != nil {
		for k := range c.paramMapOverflow {
			delete(c.paramMapOverflow, k)
		}
	}

	// Clear response headers
	if c.responseHeaders != nil {
		for k := range c.responseHeaders {
			delete(c.responseHeaders, k)
		}
	}

	// Keep slice capacity, just reset length
	c.responseBuf = c.responseBuf[:0]
	c.statusCode = 200
	c.aborted = false
}
