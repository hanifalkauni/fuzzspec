package parser

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
)

// EndpointOperation represents a fully parsed and normalized API operation.
type EndpointOperation struct {
	Path        string                 `json:"path"`
	Method      string                 `json:"method"`
	OperationID string                 `json:"operation_id,omitempty"`
	Summary     string                 `json:"summary,omitempty"`
	Description string                 `json:"description,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Parameters  []ParameterSchema      `json:"parameters,omitempty"`
	RequestBody *RequestBodySchema     `json:"request_body,omitempty"`
	Responses   map[string]interface{} `json:"responses,omitempty"`
}

// ParameterSchema represents a query, path, header, or cookie parameter.
type ParameterSchema struct {
	Name        string            `json:"name"`
	In          string            `json:"in"` // "query", "path", "header", "cookie"
	Required    bool              `json:"required"`
	Description string            `json:"description,omitempty"`
	Schema      *openapi3.Schema  `json:"schema,omitempty"`
}

// RequestBodySchema represents the parsed request body.
type RequestBodySchema struct {
	Required    bool                        `json:"required"`
	Description string                      `json:"description,omitempty"`
	Content     map[string]*openapi3.Schema `json:"content"` // media type -> schema (e.g. "application/json")
}

// ParsedSpec contains the parsed OpenAPI document and all extracted operations.
type ParsedSpec struct {
	Title       string              `json:"title"`
	Version     string              `json:"version"`
	DocVersion  string              `json:"doc_version"`
	Servers     []string            `json:"servers"`
	Operations  []EndpointOperation `json:"operations"`
	RawDoc      *openapi3.T         `json:"-"`
}

// Parser handles loading, validating, and extracting OpenAPI 3.0/3.1 documents.
type Parser struct {
	loader *openapi3.Loader
}

// NewParser creates a new Parser with a dereferencing loader.
func NewParser() *Parser {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true
	return &Parser{
		loader: loader,
	}
}

// Parse loads and extracts OpenAPI operations from a local file path or remote HTTP(S) URL.
func (p *Parser) Parse(ctx context.Context, source string) (*ParsedSpec, error) {
	var doc *openapi3.T
	var err error

	if isURL(source) {
		u, err := url.Parse(source)
		if err != nil {
			return nil, fmt.Errorf("invalid URL format '%s': %w", source, err)
		}
		doc, err = p.loader.LoadFromURI(u)
		if err != nil {
			return nil, fmt.Errorf("failed to load remote OpenAPI spec from %s: %w", source, err)
		}
	} else {
		if _, err := os.Stat(source); os.IsNotExist(err) {
			return nil, fmt.Errorf("spec file not found: %s", source)
		}
		doc, err = p.loader.LoadFromFile(source)
		if err != nil {
			return nil, fmt.Errorf("failed to parse local OpenAPI spec at %s: %w", source, err)
		}
	}

	// Validate OpenAPI document structure
	if err := doc.Validate(ctx); err != nil {
		return nil, fmt.Errorf("OpenAPI specification validation error: %w", err)
	}

	parsed := &ParsedSpec{
		DocVersion: doc.OpenAPI,
		RawDoc:     doc,
	}

	if doc.Info != nil {
		parsed.Title = doc.Info.Title
		parsed.Version = doc.Info.Version
	}

	for _, s := range doc.Servers {
		if s != nil && s.URL != "" {
			parsed.Servers = append(parsed.Servers, s.URL)
		}
	}

	// Extract all operations
	for path, pathItem := range doc.Paths.Map() {
		if pathItem == nil {
			continue
		}

		// Collect path-level parameters
		var pathLevelParams []ParameterSchema
		for _, paramRef := range pathItem.Parameters {
			if paramRef != nil && paramRef.Value != nil {
				param := paramRef.Value
				var schema *openapi3.Schema
				if param.Schema != nil {
					schema = param.Schema.Value
				}
				pathLevelParams = append(pathLevelParams, ParameterSchema{
					Name:        param.Name,
					In:          param.In,
					Required:    param.Required,
					Description: param.Description,
					Schema:      schema,
				})
			}
		}

		for method, op := range pathItem.Operations() {
			if op == nil {
				continue
			}

			endpointOp := EndpointOperation{
				Path:        path,
				Method:      strings.ToUpper(method),
				OperationID: op.OperationID,
				Summary:     op.Summary,
				Description: op.Description,
				Tags:        op.Tags,
				Responses:   make(map[string]interface{}),
			}

			// Add path-level parameters first
			endpointOp.Parameters = append(endpointOp.Parameters, pathLevelParams...)

			// Add operation-level parameters
			for _, paramRef := range op.Parameters {
				if paramRef != nil && paramRef.Value != nil {
					param := paramRef.Value
					var schema *openapi3.Schema
					if param.Schema != nil {
						schema = param.Schema.Value
					}
					endpointOp.Parameters = append(endpointOp.Parameters, ParameterSchema{
						Name:        param.Name,
						In:          param.In,
						Required:    param.Required,
						Description: param.Description,
						Schema:      schema,
					})
				}
			}

			// Extract request body schema
			if op.RequestBody != nil && op.RequestBody.Value != nil {
				rb := op.RequestBody.Value
				reqBodySchema := &RequestBodySchema{
					Required:    rb.Required,
					Description: rb.Description,
					Content:     make(map[string]*openapi3.Schema),
				}

				for mediaType, mediaTypeObj := range rb.Content {
					if mediaTypeObj != nil && mediaTypeObj.Schema != nil {
						reqBodySchema.Content[mediaType] = mediaTypeObj.Schema.Value
					}
				}
				endpointOp.RequestBody = reqBodySchema
			}

			// Extract responses
			if op.Responses != nil {
				for code, respRef := range op.Responses.Map() {
					if respRef != nil && respRef.Value != nil {
						endpointOp.Responses[code] = respRef.Value.Description
					}
				}
			}

			parsed.Operations = append(parsed.Operations, endpointOp)
		}
	}

	return parsed, nil
}

func isURL(str string) bool {
	return strings.HasPrefix(str, "http://") || strings.HasPrefix(str, "https://")
}
