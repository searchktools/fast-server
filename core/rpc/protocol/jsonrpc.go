package protocol

import (
	"encoding/json"
	"errors"
)

// JSON-RPC 2.0 specification: https://www.jsonrpc.org/specification

var (
	ErrInvalidRequest = errors.New("invalid request")
	ErrMethodNotFound = errors.New("method not found")
	ErrInvalidParams  = errors.New("invalid params")
	ErrInternalError  = errors.New("internal error")
)

// JSONRPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"` // Must be "2.0"
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      interface{}     `json:"id"` // string, number, or null
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string       `json:"jsonrpc"` // Must be "2.0"
	Result  interface{}  `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
	ID      interface{}  `json:"id"`
}

// JSONRPCError represents a JSON-RPC 2.0 error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// NewJSONRPCRequest creates a new JSON-RPC request
func NewJSONRPCRequest(method string, params interface{}, id interface{}) (*JSONRPCRequest, error) {
	var paramsData json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return nil, err
		}
		paramsData = data
	}

	return &JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsData,
		ID:      id,
	}, nil
}

// NewJSONRPCResponse creates a success response
func NewJSONRPCResponse(result interface{}, id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// NewJSONRPCError creates an error response
func NewJSONRPCError(code int, message string, data interface{}, id interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &JSONRPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
		ID: id,
	}
}

// Validate validates the JSON-RPC request
func (r *JSONRPCRequest) Validate() error {
	if r.JSONRPC != "2.0" {
		return ErrInvalidRequest
	}
	if r.Method == "" {
		return ErrInvalidRequest
	}
	return nil
}
