package generator_test

import (
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/getkin/kin-openapi/openapi3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildFuzzingPrompt(t *testing.T) {
	minVal := float64(1)
	maxVal := float64(100)

	op := parser.EndpointOperation{
		Method:  "POST",
		Path:    "/orders",
		Summary: "Create a new purchase order",
		Parameters: []parser.ParameterSchema{
			{
				Name:     "X-Tenant-ID",
				In:       "header",
				Required: true,
				Schema: &openapi3.Schema{
					Type: &openapi3.Types{"string"},
				},
			},
			{
				Name:     "limit",
				In:       "query",
				Required: false,
				Schema: &openapi3.Schema{
					Type: &openapi3.Types{"integer"},
					Min:  &minVal,
					Max:  &maxVal,
				},
			},
		},
		RequestBody: &parser.RequestBodySchema{
			Required: true,
			Content: map[string]*openapi3.Schema{
				"application/json": {
					Type: &openapi3.Types{"object"},
					Properties: openapi3.Schemas{
						"item_id": &openapi3.SchemaRef{
							Value: &openapi3.Schema{Type: &openapi3.Types{"string"}},
						},
					},
				},
			},
		},
	}

	prompt, err := generator.BuildFuzzingPrompt(op)
	require.NoError(t, err)
	assert.NotEmpty(t, prompt)
	assert.Contains(t, prompt, "POST")
	assert.Contains(t, prompt, "/orders")
	assert.Contains(t, prompt, "X-Tenant-ID")
	assert.Contains(t, prompt, "limit")
	assert.Contains(t, prompt, "AI_SEMANTIC_EDGE")
}
