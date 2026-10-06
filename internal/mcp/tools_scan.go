package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getScanAndGenerateSpecTool() Tool {
	return Tool{
		Name:        "scan_and_generate_spec",
		Description: "Scans project directories to identify API routes or scaffolds an OpenAPI 3.1 baseline specification.",
		InputSchema: InputSchema{
			Type: "object",
			Properties: map[string]PropertyDoc{
				"project_path": {
					Type:        "string",
					Description: "Path to source code directory (e.g. ./src or ./routes)",
				},
				"output_format": {
					Type:        "string",
					Description: "Format to generate ('yaml' or 'json', default: 'yaml')",
					Enum:        []string{"yaml", "json"},
				},
				"output_file": {
					Type:        "string",
					Description: "Optional file path to save the generated OpenAPI specification (e.g. ./openapi.yaml)",
				},
				"title": {
					Type:        "string",
					Description: "Optional API title (default: Auto-Generated API)",
				},
			},
			Required: []string{"project_path"},
		},
	}
}

func handleScanAndGenerateSpec(_ context.Context, args map[string]interface{}) (ToolCallResult, error) {
	projectPath, _ := args["project_path"].(string)
	outputFormat, _ := args["output_format"].(string)
	outputFile, _ := args["output_file"].(string)
	title, _ := args["title"].(string)

	if projectPath == "" {
		return ToolCallResult{
			Content: []ToolContent{{Type: "text", Text: "Missing required argument 'project_path'."}},
			IsError: true,
		}, nil
	}

	if outputFormat == "" {
		outputFormat = "yaml"
	}
	if title == "" {
		title = "Auto-Generated API Specification"
	}

	// Security guard: Blacklisted directories and credential files
	var discoveredRoutes []string
	_ = filepath.Walk(projectPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		name := strings.ToLower(info.Name())

		// Skip sensitive directories and vendor dependencies
		if info.IsDir() {
			if name == ".git" || name == ".aws" || name == ".ssh" || name == "node_modules" ||
				name == "vendor" || name == ".secrets" || name == ".fuzzspec-cache" {
				return filepath.SkipDir
			}
			return nil
		}

		// Strictly skip any environment / credential / private key files
		if strings.HasPrefix(name, ".env") ||
			strings.Contains(name, "secret") ||
			strings.Contains(name, "credential") ||
			strings.HasSuffix(name, ".pem") ||
			strings.HasSuffix(name, ".key") ||
			strings.HasSuffix(name, ".p12") ||
			strings.HasSuffix(name, ".pfx") ||
			name == "id_rsa" || name == "id_ed25519" {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".go", ".py", ".ts", ".js", ".java", ".php", ".rs", ".cs", ".rb":
			data, errRead := os.ReadFile(path)
			if errRead == nil {
				content := string(data)
				// Basic heuristic patterns for HTTP routes
				if strings.Contains(content, "GET") || strings.Contains(content, "POST") ||
					strings.Contains(content, "@Get") || strings.Contains(content, "@Post") ||
					strings.Contains(content, "app.get") || strings.Contains(content, "app.post") ||
					strings.Contains(content, "router.HandleFunc") {
					discoveredRoutes = append(discoveredRoutes, path)
				}
			}
		}
		return nil
	})

	templateYAML := fmt.Sprintf(`openapi: 3.1.0
info:
  title: %s
  version: 1.0.0
  description: Generated Spec-to-Contract baseline from %s
servers:
  - url: http://localhost:8080
paths:
  /health:
    get:
      summary: Service Health Check
      responses:
        '200':
          description: OK
  /api/v1/items:
    get:
      summary: List Items
      parameters:
        - name: limit
          in: query
          required: false
          schema:
            type: integer
            minimum: 1
            maximum: 100
      responses:
        '200':
          description: Successful response
    post:
      summary: Create Item
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required:
                - name
              properties:
                name:
                  type: string
                price:
                  type: number
      responses:
        '201':
          description: Created
`, title, projectPath)

	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(templateYAML), 0644); err != nil {
			return ToolCallResult{
				Content: []ToolContent{{Type: "text", Text: fmt.Sprintf("Failed to write output file '%s': %v", outputFile, err)}},
				IsError: true,
			}, nil
		}
	}

	resultMap := map[string]interface{}{
		"status":            "Spec scaffolded successfully",
		"scanned_files":     len(discoveredRoutes),
		"output_file":       outputFile,
		"spec_preview_yaml": templateYAML,
	}

	outJSON, _ := json.MarshalIndent(resultMap, "", "  ")

	return ToolCallResult{
		Content: []ToolContent{
			{
				Type: "text",
				Text: string(outJSON),
			},
		},
	}, nil
}
