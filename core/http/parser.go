package http

import (
	"bytes"
	"errors"
	"unsafe"
)

// unsafeString converts byte slice to string without allocation
// WARNING: The returned string shares memory with the byte slice
func unsafeString(b []byte) string {
	return *(*string)(unsafe.Pointer(&b))
}

var (
	ErrInvalidRequest = errors.New("invalid HTTP request")
)

// ParseRequest is a zero-allocation HTTP parser
func ParseRequest(data []byte) (*Request, error) {
	req := AcquireRequest()

	// Parse request line
	lineEnd := bytes.IndexByte(data, '\n')
	if lineEnd == -1 {
		ReleaseRequest(req)
		return nil, ErrInvalidRequest
	}

	line := data[:lineEnd]
	if line[len(line)-1] == '\r' {
		line = line[:len(line)-1]
	}

	// Parse METHOD PATH PROTO (zero-allocation: avoid SplitN)
	// Find first space (end of METHOD)
	sp1 := bytes.IndexByte(line, ' ')
	if sp1 == -1 {
		ReleaseRequest(req)
		return nil, ErrInvalidRequest
	}

	// Find second space (end of PATH)
	sp2 := bytes.IndexByte(line[sp1+1:], ' ')
	if sp2 == -1 {
		ReleaseRequest(req)
		return nil, ErrInvalidRequest
	}
	sp2 += sp1 + 1

	// Use unsafe string conversion to avoid allocation
	// These strings point to the original data buffer
	req.Method = unsafeString(line[:sp1])
	req.Path = unsafeString(line[sp1+1 : sp2])
	req.Proto = unsafeString(line[sp2+1:])

	// Parse query parameters
	if idx := bytes.IndexByte([]byte(req.Path), '?'); idx != -1 {
		req.Path, _ = parseQuery(req, req.Path, idx)
	}

	// Parse headers
	data = data[lineEnd+1:]
	headerEnd := bytes.Index(data, []byte("\r\n\r\n"))
	if headerEnd == -1 {
		headerEnd = bytes.Index(data, []byte("\n\n"))
		if headerEnd == -1 {
			ReleaseRequest(req)
			return nil, ErrInvalidRequest
		}
		data = data[headerEnd+2:]
	} else {
		headerData := data[:headerEnd]
		parseHeaders(req, headerData)
		data = data[headerEnd+4:]
	}

	// Parse request body
	if len(data) > 0 {
		req.Body = append(req.Body[:0], data...)
	}

	return req, nil
}

// parseHeaders parses HTTP headers
func parseHeaders(req *Request, data []byte) {
	for len(data) > 0 {
		lineEnd := bytes.IndexByte(data, '\n')
		if lineEnd == -1 {
			lineEnd = len(data)
		}

		line := data[:lineEnd]
		if len(line) > 0 && line[len(line)-1] == '\r' {
			line = line[:len(line)-1]
		}

		if len(line) == 0 {
			break
		}

		// Parse key-value pair
		colon := bytes.IndexByte(line, ':')
		if colon > 0 {
			key := string(bytes.TrimSpace(line[:colon]))
			value := string(bytes.TrimSpace(line[colon+1:]))
			req.SetHeader(key, value)
		}

		if lineEnd == len(data) {
			break
		}
		data = data[lineEnd+1:]
	}
}

// parseQuery parses query parameters
func parseQuery(req *Request, path string, idx int) (string, error) {
	queryStr := path[idx+1:]
	path = path[:idx]

	if req.Query == nil {
		req.Query = make(map[string]string)
	}

	// Simple query parameter parsing
	pairs := bytes.Split([]byte(queryStr), []byte("&"))
	for _, pair := range pairs {
		kv := bytes.SplitN(pair, []byte("="), 2)
		if len(kv) == 2 {
			req.Query[string(kv[0])] = string(kv[1])
		} else if len(kv) == 1 {
			req.Query[string(kv[0])] = ""
		}
	}

	return path, nil
}
