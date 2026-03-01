package mcp

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 error codes.
const (
	ErrCodeParseError     = -32700
	ErrCodeInvalidRequest = -32600
	ErrCodeMethodNotFound = -32601
	ErrCodeInvalidParams  = -32602
	ErrCodeInternal       = -32603
	ErrCodeServerError    = -32000
	ErrCodeJobNotFound    = -32001
	ErrCodeQueueFull      = -32002
	ErrCodeTimeout        = -32004
)

// JSONRPCRequest is a JSON-RPC 2.0 request.
type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
	ID      json.RawMessage `json:"id"`
}

// JSONRPCResponse is a JSON-RPC 2.0 response.
type JSONRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	Result  any             `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      json.RawMessage `json:"id"`
}

// JSONRPCError is a JSON-RPC 2.0 error object.
type JSONRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func rpcSuccess(id json.RawMessage, result any) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

func rpcError(id json.RawMessage, code int, message string) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Error:   &JSONRPCError{Code: code, Message: message},
		ID:      id,
	}
}

// parseParams decodes JSON-RPC params into target struct.
func parseParams[T any](params json.RawMessage) (T, error) {
	var v T
	if len(params) == 0 || string(params) == "null" {
		return v, nil
	}
	if err := json.Unmarshal(params, &v); err != nil {
		return v, fmt.Errorf("invalid params: %w", err)
	}
	return v, nil
}
