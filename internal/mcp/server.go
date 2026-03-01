package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

const mcpVersion = "0.2.0"

// MCPServer is the core MCP server coordinating events, jobs, and tools.
type MCPServer struct {
	events  EventStore
	jobs    *JobManager
	bridge  *Bridge
	config  Config

	sessionMu sync.RWMutex
	sessions  map[string]time.Time
}

// NewMCPServer creates an MCP server.
func NewMCPServer(events EventStore, jobs *JobManager, bridge *Bridge, cfg Config) *MCPServer {
	return &MCPServer{
		events:   events,
		jobs:     jobs,
		bridge:   bridge,
		config:   cfg,
		sessions: make(map[string]time.Time),
	}
}

// HandleJSONRPC processes a JSON-RPC 2.0 request and returns a response.
func (s *MCPServer) HandleJSONRPC(ctx context.Context, raw []byte) JSONRPCResponse {
	var req JSONRPCRequest
	if err := json.Unmarshal(raw, &req); err != nil {
		return rpcError(nil, ErrCodeParseError, "parse error: invalid JSON")
	}
	if req.JSONRPC != "2.0" {
		return rpcError(req.ID, ErrCodeInvalidRequest, "unsupported JSON-RPC version")
	}

	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "ping":
		return rpcSuccess(req.ID, map[string]string{"status": "ok"})
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(ctx, req)
	default:
		return rpcError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("method not found: %s", req.Method))
	}
}

func (s *MCPServer) handleInitialize(req JSONRPCRequest) JSONRPCResponse {
	return rpcSuccess(req.ID, map[string]any{
		"protocolVersion": "2025-11-25",
		"capabilities": map[string]any{
			"tools": map[string]bool{"listChanged": false},
		},
		"serverInfo": map[string]string{
			"name":    "tgup-mcp",
			"version": mcpVersion,
		},
	})
}

func (s *MCPServer) handleToolsList(req JSONRPCRequest) JSONRPCResponse {
	schema := Schema()
	tools := make([]map[string]any, 0, len(schema.Tools))
	for _, t := range schema.Tools {
		tool := map[string]any{
			"name":        t.Name,
			"description": t.Description,
		}
		if t.InputSchema != nil {
			tool["inputSchema"] = t.InputSchema
		}
		tools = append(tools, tool)
	}
	return rpcSuccess(req.ID, map[string]any{"tools": tools})
}

func (s *MCPServer) handleToolsCall(ctx context.Context, req JSONRPCRequest) JSONRPCResponse {
	// Parse tool call envelope.
	var call struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(req.Params, &call); err != nil {
		return rpcError(req.ID, ErrCodeInvalidParams, "invalid tools/call params")
	}

	switch call.Name {
	case "tgup.health":
		return s.toolHealth(req.ID)
	case "tgup.dry_run":
		return s.toolDryRun(ctx, req.ID, call.Arguments)
	case "tgup.run.start":
		return s.toolRunStart(ctx, req.ID, call.Arguments)
	case "tgup.run.sync":
		return s.toolRunSync(ctx, req.ID, call.Arguments)
	case "tgup.run.status":
		return s.toolRunStatus(ctx, req.ID, call.Arguments)
	case "tgup.run.cancel":
		return s.toolRunCancel(ctx, req.ID, call.Arguments)
	case "tgup.run.events":
		return s.toolRunEvents(ctx, req.ID, call.Arguments)
	case "tgup.schema.get":
		return s.toolSchemaGet(req.ID)
	default:
		return rpcError(req.ID, ErrCodeMethodNotFound, fmt.Sprintf("unknown tool: %s", call.Name))
	}
}

func (s *MCPServer) toolHealth(id json.RawMessage) JSONRPCResponse {
	tools := make([]string, 0, len(Schema().Tools))
	for _, t := range Schema().Tools {
		tools = append(tools, t.Name)
	}
	return rpcSuccess(id, HealthOutput{
		Status:  "ok",
		Version: mcpVersion,
		Tools:   tools,
	})
}

func (s *MCPServer) toolDryRun(ctx context.Context, id json.RawMessage, args json.RawMessage) JSONRPCResponse {
	input, err := parseParams[DryRunInput](args)
	if err != nil {
		return rpcError(id, ErrCodeInvalidParams, err.Error())
	}
	result, err := s.bridge.DryRun(ctx, input)
	if err != nil {
		return rpcError(id, ErrCodeServerError, err.Error())
	}
	return rpcSuccess(id, result)
}

func (s *MCPServer) toolRunStart(ctx context.Context, id json.RawMessage, args json.RawMessage) JSONRPCResponse {
	input, err := parseParams[RunStartInput](args)
	if err != nil {
		return rpcError(id, ErrCodeInvalidParams, err.Error())
	}

	runner := func(ctx context.Context, spec RunSpec, emit func(Event)) (*RunResult, error) {
		return s.bridge.RunJob(ctx, spec, emit)
	}

	job, err := s.jobs.Start(ctx, input.RunSpec, runner)
	if err != nil {
		return rpcError(id, ErrCodeQueueFull, err.Error())
	}

	return rpcSuccess(id, RunStartOutput{
		JobID:  job.ID,
		Status: job.Status,
	})
}

func (s *MCPServer) toolRunSync(ctx context.Context, id json.RawMessage, args json.RawMessage) JSONRPCResponse {
	input, err := parseParams[RunSyncInput](args)
	if err != nil {
		return rpcError(id, ErrCodeInvalidParams, err.Error())
	}

	timeout := time.Duration(input.TimeoutSec) * time.Second
	if timeout <= 0 {
		timeout = 300 * time.Second
	}

	runner := func(ctx context.Context, spec RunSpec, emit func(Event)) (*RunResult, error) {
		return s.bridge.RunJob(ctx, spec, emit)
	}

	job, err := s.jobs.Start(ctx, input.RunSpec, runner)
	if err != nil {
		return rpcError(id, ErrCodeQueueFull, err.Error())
	}

	waitCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	job, err = s.jobs.Wait(waitCtx, job.ID)
	if err != nil {
		return rpcError(id, ErrCodeTimeout, fmt.Sprintf("timeout waiting for job: %v", err))
	}

	return rpcSuccess(id, RunSyncOutput{
		JobID:  job.ID,
		Status: job.Status,
		Result: job.Result,
		Error:  job.Error,
	})
}

func (s *MCPServer) toolRunStatus(ctx context.Context, id json.RawMessage, args json.RawMessage) JSONRPCResponse {
	input, err := parseParams[RunStatusInput](args)
	if err != nil {
		return rpcError(id, ErrCodeInvalidParams, err.Error())
	}
	if input.JobID == "" {
		return rpcError(id, ErrCodeInvalidParams, "jobId is required")
	}

	job, err := s.jobs.Get(ctx, input.JobID)
	if err != nil {
		return rpcError(id, ErrCodeJobNotFound, err.Error())
	}

	return rpcSuccess(id, RunStatusOutput{
		JobID:      job.ID,
		Status:     job.Status,
		CreatedAt:  job.CreatedAt,
		StartedAt:  job.StartedAt,
		FinishedAt: job.FinishedAt,
		Result:     job.Result,
		Error:      job.Error,
	})
}

func (s *MCPServer) toolRunCancel(ctx context.Context, id json.RawMessage, args json.RawMessage) JSONRPCResponse {
	input, err := parseParams[RunCancelInput](args)
	if err != nil {
		return rpcError(id, ErrCodeInvalidParams, err.Error())
	}
	if input.JobID == "" {
		return rpcError(id, ErrCodeInvalidParams, "jobId is required")
	}

	job, err := s.jobs.Cancel(ctx, input.JobID)
	if err != nil {
		return rpcError(id, ErrCodeJobNotFound, err.Error())
	}

	return rpcSuccess(id, RunCancelOutput{
		JobID:  job.ID,
		Status: job.Status,
	})
}

func (s *MCPServer) toolRunEvents(ctx context.Context, id json.RawMessage, args json.RawMessage) JSONRPCResponse {
	input, err := parseParams[RunEventsInput](args)
	if err != nil {
		return rpcError(id, ErrCodeInvalidParams, err.Error())
	}
	if input.JobID == "" {
		return rpcError(id, ErrCodeInvalidParams, "jobId is required")
	}

	events, hasMore, err := s.events.List(ctx, input.JobID, input.SinceSeq, input.Limit)
	if err != nil {
		return rpcError(id, ErrCodeInternal, err.Error())
	}

	envelopes := make([]EventEnvelope, len(events))
	for i, e := range events {
		envelopes[i] = EventEnvelope{
			Seq:     e.Seq,
			JobID:   e.JobID,
			Type:    e.Type,
			Payload: e.Payload,
			Ts:      e.CreatedAt,
		}
	}

	return rpcSuccess(id, RunEventsOutput{
		Events:  envelopes,
		HasMore: hasMore,
	})
}

func (s *MCPServer) toolSchemaGet(id json.RawMessage) JSONRPCResponse {
	schema := Schema()
	return rpcSuccess(id, SchemaGetOutput{
		Version: schema.Version,
		Tools:   schema.Tools,
	})
}

// TrackSession updates session activity.
func (s *MCPServer) TrackSession(sessionID string) {
	s.sessionMu.Lock()
	s.sessions[sessionID] = time.Now().UTC()
	s.sessionMu.Unlock()
}
