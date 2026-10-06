package mcp_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMCPServer_InitializeAndToolsList(t *testing.T) {
	srv := mcp.NewServer()
	ctx := context.Background()

	// 1. Test initialize
	initReq := `{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {}}`
	respBytes, err := srv.HandleMessage(ctx, []byte(initReq))
	require.NoError(t, err)

	var initResp mcp.JSONRPCResponse
	err = json.Unmarshal(respBytes, &initResp)
	require.NoError(t, err)
	assert.Equal(t, 1.0, initResp.ID)
	assert.Nil(t, initResp.Error)

	// 2. Test ping
	pingReq := `{"jsonrpc": "2.0", "id": 2, "method": "ping"}`
	respBytes, err = srv.HandleMessage(ctx, []byte(pingReq))
	require.NoError(t, err)

	var pingResp mcp.JSONRPCResponse
	err = json.Unmarshal(respBytes, &pingResp)
	require.NoError(t, err)
	assert.Equal(t, 2.0, pingResp.ID)

	// 3. Test tools/list
	toolsListReq := `{"jsonrpc": "2.0", "id": 3, "method": "tools/list"}`
	respBytes, err = srv.HandleMessage(ctx, []byte(toolsListReq))
	require.NoError(t, err)

	var toolsResp struct {
		JSONRPC string              `json:"jsonrpc"`
		ID      float64             `json:"id"`
		Result  mcp.ToolsListResult `json:"result"`
	}
	err = json.Unmarshal(respBytes, &toolsResp)
	require.NoError(t, err)
	assert.Len(t, toolsResp.Result.Tools, 4)

	toolNames := make(map[string]bool)
	for _, tool := range toolsResp.Result.Tools {
		toolNames[tool.Name] = true
	}
	assert.True(t, toolNames["inspect_spec"])
	assert.True(t, toolNames["fuzz_endpoint"])
	assert.True(t, toolNames["replay_anomaly"])
	assert.True(t, toolNames["scan_and_generate_spec"])
}

func TestMCPServer_InspectSpec(t *testing.T) {
	srv := mcp.NewServer()
	ctx := context.Background()

	specPath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	reqObj := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      10,
		Method:  "tools/call",
	}
	paramsBytes, _ := json.Marshal(mcp.ToolCallParams{
		Name: "inspect_spec",
		Arguments: map[string]interface{}{
			"spec_path": specPath,
		},
	})
	reqObj.Params = paramsBytes
	reqBytes, _ := json.Marshal(reqObj)

	respBytes, err := srv.HandleMessage(ctx, reqBytes)
	require.NoError(t, err)

	var resp struct {
		Result mcp.ToolCallResult `json:"result"`
	}
	err = json.Unmarshal(respBytes, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Result.IsError)
	require.Len(t, resp.Result.Content, 1)
	assert.Contains(t, resp.Result.Content[0].Text, "Petstore")
	assert.Contains(t, resp.Result.Content[0].Text, "/pets")
}

func TestMCPServer_FuzzEndpoint_And_Replay(t *testing.T) {
	// Mock live server
	isFixed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") == "9223372036854775907" {
			if !isFixed {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte("panic: integer overflow"))
				return
			}
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error": "invalid limit"}`))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[]`))
	}))
	defer server.Close()

	srv := mcp.NewServer()
	ctx := context.Background()

	specPath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	// 1. Call fuzz_endpoint
	fuzzReq := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      20,
		Method:  "tools/call",
	}
	paramsBytes, _ := json.Marshal(mcp.ToolCallParams{
		Name: "fuzz_endpoint",
		Arguments: map[string]interface{}{
			"spec_path":  specPath,
			"target_url": server.URL,
			"path":       "/pets",
			"method":     "GET",
		},
	})
	fuzzReq.Params = paramsBytes
	reqBytes, _ := json.Marshal(fuzzReq)

	respBytes, err := srv.HandleMessage(ctx, reqBytes)
	require.NoError(t, err)

	var fuzzResp struct {
		Result mcp.ToolCallResult `json:"result"`
	}
	err = json.Unmarshal(respBytes, &fuzzResp)
	require.NoError(t, err)
	assert.False(t, fuzzResp.Result.IsError)
	assert.Contains(t, fuzzResp.Result.Content[0].Text, "anomalies")
	assert.Contains(t, fuzzResp.Result.Content[0].Text, "curl")

	// 2. Call replay_anomaly with single vector (bug still present)
	replayReq := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      21,
		Method:  "tools/call",
	}
	replayParamsBytes, _ := json.Marshal(mcp.ToolCallParams{
		Name: "replay_anomaly",
		Arguments: map[string]interface{}{
			"target_url": server.URL,
			"vector": map[string]interface{}{
				"id":                     "vec_overflow",
				"path":                   "/pets",
				"method":                 "GET",
				"query_params":           map[string]string{"limit": "9223372036854775907"},
				"expected_status_family": "4xx",
			},
		},
	})
	replayReq.Params = replayParamsBytes
	reqBytes, _ = json.Marshal(replayReq)

	respBytes, err = srv.HandleMessage(ctx, reqBytes)
	require.NoError(t, err)

	var replayResp struct {
		Result mcp.ToolCallResult `json:"result"`
	}
	err = json.Unmarshal(respBytes, &replayResp)
	require.NoError(t, err)
	assert.Contains(t, replayResp.Result.Content[0].Text, "STILL_FAILING")

	// 3. Fix bug and replay
	isFixed = true
	respBytes, err = srv.HandleMessage(ctx, reqBytes)
	require.NoError(t, err)

	err = json.Unmarshal(respBytes, &replayResp)
	require.NoError(t, err)
	assert.Contains(t, replayResp.Result.Content[0].Text, "RESOLVED")
	assert.Contains(t, replayResp.Result.Content[0].Text, `"all_resolved": true`)
}

func TestMCPServer_ScanAndGenerateSpec(t *testing.T) {
	srv := mcp.NewServer()
	ctx := context.Background()

	reqObj := mcp.JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      30,
		Method:  "tools/call",
	}
	paramsBytes, _ := json.Marshal(mcp.ToolCallParams{
		Name: "scan_and_generate_spec",
		Arguments: map[string]interface{}{
			"project_path": "./cmd",
		},
	})
	reqObj.Params = paramsBytes
	reqBytes, _ := json.Marshal(reqObj)

	respBytes, err := srv.HandleMessage(ctx, reqBytes)
	require.NoError(t, err)

	var resp struct {
		Result mcp.ToolCallResult `json:"result"`
	}
	err = json.Unmarshal(respBytes, &resp)
	require.NoError(t, err)
	assert.False(t, resp.Result.IsError)
	assert.Contains(t, resp.Result.Content[0].Text, "openapi: 3.1.0")
}
