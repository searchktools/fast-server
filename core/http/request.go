package http

import "sync"

// Request is a zero-allocation HTTP request structure
type Request struct {
	Method string
	Path   string
	Proto  string

	// Predefined common header fields (zero-allocation)
	ContentType   string
	ContentLength string
	UserAgent     string
	Accept        string
	Host          string
	Connection    string

	// Extra headers (allocated only when needed)
	ExtraHeaders map[string]string

	// Query parameters
	Query map[string]string

	// Request body
	Body []byte
}

var requestPool = sync.Pool{
	New: func() any {
		return &Request{
			Body: make([]byte, 0, 1024),
		}
	},
}

func AcquireRequest() *Request {
	return requestPool.Get().(*Request)
}

// Reset resets the request for reuse (memory not freed, just reset)
func (r *Request) Reset() {
	r.Method = ""
	r.Path = ""
	r.Proto = ""
	r.ContentType = ""
	r.ContentLength = ""
	r.UserAgent = ""
	r.Accept = ""
	r.Host = ""
	r.Connection = ""

	// Clear maps without freeing memory
	if r.ExtraHeaders != nil {
		for k := range r.ExtraHeaders {
			delete(r.ExtraHeaders, k)
		}
	}

	if r.Query != nil {
		for k := range r.Query {
			delete(r.Query, k)
		}
	}

	// Keep slice capacity, just reset length
	r.Body = r.Body[:0]
}

func ReleaseRequest(req *Request) {
	req.Reset()
	requestPool.Put(req)
}

// SetHeader sets a header (prioritizes predefined fields)
func (r *Request) SetHeader(key, value string) {
	switch key {
	case "Content-Type":
		r.ContentType = value
	case "Content-Length":
		r.ContentLength = value
	case "User-Agent":
		r.UserAgent = value
	case "Accept":
		r.Accept = value
	case "Host":
		r.Host = value
	case "Connection":
		r.Connection = value
	default:
		if r.ExtraHeaders == nil {
			r.ExtraHeaders = make(map[string]string)
		}
		r.ExtraHeaders[key] = value
	}
}
