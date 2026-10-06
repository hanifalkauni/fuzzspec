package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// Server represents the FuzzSpec MCP JSON-RPC Server.
type Server struct {
	tools map[string]Tool
}

// NewServer initializes a new MCP server with all registered FuzzSpec tools.
func NewServer() *Server {
	s := &Server{
		tools: make(map[string]Tool),
	}

	s.registerTool(getInspectSpecTool())
	s.registerTool(getFuzzEndpointTool())
	s.registerTool(getReplayAnomalyTool())
	s.registerTool(getScanAndGenerateSpecTool())

	return s
}

func (s *Server) registerTool(t Tool) {
	s.tools[t.Name] = t
}

// ServeStdio starts the stdio JSON-RPC listening loop.
func (s *Server) ServeStdio(ctx context.Context) error {
	reader := bufio.NewReader(os.Stdin)
	writer := bufio.NewWriter(os.Stdout)

	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}

		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}

		if len(line) == 0 || (len(line) == 1 && line[0] == '\n') {
			continue
		}

		respBytes, err := s.HandleMessage(ctx, line)
		if err != nil {
			continue
		}

		if len(respBytes) > 0 {
			_, _ = writer.Write(respBytes)
			_ = writer.WriteByte('\n')
			_ = writer.Flush()
		}
	}
}

// HandleMessage processes a single JSON-RPC request and returns the response.
func (s *Server) HandleMessage(ctx context.Context, reqBytes []byte) ([]byte, error) {
	var req JSONRPCRequest
	if err := json.Unmarshal(reqBytes, &req); err != nil {
		return json.Marshal(JSONRPCResponse{
			JSONRPC: "2.0",
			Error: &JSONRPCError{
				Code:    -32700,
				Message: "Parse error",
			},
		})
	}

	switch req.Method {
	case "initialize":
		res := InitializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities: ServerCapabilities{
				Tools: map[string]interface{}{},
			},
			ServerInfo: ServerInfo{
				Name:    "fuzzspec-mcp",
				Version: "1.0.0",
			},
		}
		return json.Marshal(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  res,
		})

	case "notifications/initialized":
		// Notification ACK (no response needed)
		return nil, nil

	case "ping":
		return json.Marshal(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  map[string]interface{}{},
		})

	case "tools/list":
		var toolList []Tool
		for _, t := range s.tools {
			toolList = append(toolList, t)
		}
		return json.Marshal(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result: ToolsListResult{
				Tools: toolList,
			},
		})

	case "tools/call":
		var params ToolCallParams
		if err := json.Unmarshal(req.Params, &params); err != nil {
			return json.Marshal(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32602,
					Message: fmt.Sprintf("Invalid params: %v", err),
				},
			})
		}

		toolResult, err := s.dispatchToolCall(ctx, params.Name, params.Arguments)
		if err != nil {
			return json.Marshal(JSONRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error: &JSONRPCError{
					Code:    -32603,
					Message: err.Error(),
				},
			})
		}

		return json.Marshal(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  toolResult,
		})

	default:
		// If it's a notification without ID, ignore
		if req.ID == nil {
			return nil, nil
		}
		return json.Marshal(JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &JSONRPCError{
				Code:    -32601,
				Message: fmt.Sprintf("Method not found: %s", req.Method),
			},
		})
	}
}

func (s *Server) dispatchToolCall(ctx context.Context, name string, args map[string]interface{}) (ToolCallResult, error) {
	switch name {
	case "inspect_spec":
		return handleInspectSpec(ctx, args)
	case "fuzz_endpoint":
		return handleFuzzEndpoint(ctx, args)
	case "replay_anomaly":
		return handleReplayAnomaly(ctx, args)
	case "scan_and_generate_spec":
		return handleScanAndGenerateSpec(ctx, args)
	default:
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Unknown tool '%s'", name)}},
			IsError: true,
		}, nil
	}
}
