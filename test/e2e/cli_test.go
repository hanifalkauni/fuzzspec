package e2e_test

import (
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCLI_Run_LiveServer(t *testing.T) {
	// Start a mock live server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") == "9223372036854775907" {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte("panic: runtime error: integer overflow"))
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))
	defer server.Close()

	binPath, err := filepath.Abs("../../fuzzspec.exe")
	require.NoError(t, err)

	specPath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	jsonOut := filepath.Join(tmpDir, "cli_report.json")

	// Execute binary CLI
	cmd := exec.Command(binPath, "run",
		"--spec", specPath,
		"--target", server.URL,
		"--concurrency", "5",
		"--rps", "50",
		"--safe-mode=true",
		"--output-json", jsonOut,
	)

	output, err := cmd.CombinedOutput()
	outputStr := string(output)

	// Since there is a 500 error on the integer overflow vector, the CLI must detect it and exit with code 1
	assert.Error(t, err, "CLI should fail quality gate on 500 crash")
	assert.Contains(t, outputStr, "FUZZSPEC EXECUTION RESULTS")
	assert.Contains(t, outputStr, "DETECTED ANOMALIES")
	assert.Contains(t, outputStr, "QUALITY GATE FAILED")
	assert.FileExists(t, jsonOut, "Should have exported JSON report")
}

func TestCLI_AutoDiscover_And_Replay(t *testing.T) {
	// 1. Live server that serves OpenAPI spec on /openapi.json and has a bug on invalid limit
	isBugFixed := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"openapi": "3.0.0",
				"info": {"title": "AutoDiscover App", "version": "1.0.0"},
				"paths": {
					"/items": {
						"get": {
							"summary": "List Items",
							"parameters": [
								{
									"name": "limit",
									"in": "query",
									"schema": {"type": "integer"}
								}
							],
							"responses": {
								"200": {"description": "OK"}
							}
						}
					}
				}
			}`))
			return
		}

		if r.URL.Path == "/items" {
			if r.URL.Query().Get("limit") == "9223372036854775907" {
				if !isBugFixed {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte("panic: overflow in /items"))
					return
				}
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "bad request"}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`[]`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	binPath, err := filepath.Abs("../../fuzzspec.exe")
	require.NoError(t, err)

	tmpDir := t.TempDir()
	jsonOut := filepath.Join(tmpDir, "autodiscover_report.json")

	// 2. Run with --auto-discover
	runCmd := exec.Command(binPath, "run",
		"--auto-discover",
		"--target", server.URL,
		"--output-json", jsonOut,
		"--no-ai",
	)

	runOutput, err := runCmd.CombinedOutput()
	outputStr := string(runOutput)
	assert.Error(t, err, "Should exit with code 1 due to 500 crash")
	assert.Contains(t, outputStr, "Auto-Discovered Specification URL")
	assert.Contains(t, outputStr, "QUALITY GATE FAILED")
	assert.FileExists(t, jsonOut)

	// 3. Replay against still-broken server (should fail)
	replayFailCmd := exec.Command(binPath, "replay",
		"--file", jsonOut,
		"--target", server.URL,
	)
	replayFailOut, err := replayFailCmd.CombinedOutput()
	assert.Error(t, err)
	assert.Contains(t, string(replayFailOut), "STILL FAILING")
	assert.Contains(t, string(replayFailOut), "REPLAY FAILED")

	// 4. Fix bug and replay again (should pass with zero failures)
	isBugFixed = true
	replayPassCmd := exec.Command(binPath, "replay",
		"--file", jsonOut,
		"--target", server.URL,
	)
	replayPassOut, err := replayPassCmd.CombinedOutput()
	assert.NoError(t, err)
	assert.Contains(t, string(replayPassOut), "RESOLVED")
	assert.Contains(t, string(replayPassOut), "REPLAY PASSED")
}

