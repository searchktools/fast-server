package core

import "errors"

// HTTP header constants
const (
	HeaderContentType   = "Content-Type"
	HeaderContentLength = "Content-Length"
	HeaderUserAgent     = "User-Agent"
	HeaderAccept        = "Accept"
	HeaderHost          = "Host"
	HeaderConnection    = "Connection"
)

// Error definitions
var (
	ErrInvalidRequest = errors.New("invalid HTTP request")
	ErrMethodTooLong  = errors.New("method too long")
	ErrPathTooLong    = errors.New("path too long")
)
