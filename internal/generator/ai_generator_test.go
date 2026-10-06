package generator_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAIGenerator_GracefulFallbackNoKey(t *testing.T) {
	// Ensure no API keys set
	os.Unsetenv("GEMINI_API_KEY")
	os.Unsetenv("OPENAI_API_KEY")
	os.Unsetenv("ANTHROPIC_API_KEY")

	gen := generator.NewAIGenerator("gemini", "gemini-1.5-flash", "", false)

	op := parser.EndpointOperation{
		Method: "GET",
		Path:   "/items",
	}

	vectors, err := gen.GenerateAIVectors(context.Background(), op)
	require.NoError(t, err)
	assert.Nil(t, vectors)
}

func TestAIGenerator_OpenAICompatibleMock(t *testing.T) {
	// Mock OpenAI API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer mock-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{
			"choices": [
				{
					"message": {
						"content": "[{\"scenario\": \"SQL Injection in Search\", \"description\": \"Checks for crash on raw SQL syntax\", \"mutation_type\": \"AI_SEMANTIC_EDGE\", \"target_location\": \"query\", \"target_field\": \"q\", \"query_params\": {\"q\": \"' OR 1=1--\"}}]"
					}
				}
			]
		}`))
	}))
	defer server.Close()

	os.Setenv("OPENAI_BASE_URL", server.URL)
	os.Setenv("OPENAI_API_KEY", "mock-key")
	defer func() {
		os.Unsetenv("OPENAI_BASE_URL")
		os.Unsetenv("OPENAI_API_KEY")
	}()

	gen := generator.NewAIGenerator("openai", "gpt-4o-mini", "", false)

	op := parser.EndpointOperation{
		Method: "GET",
		Path:   "/search",
	}

	vectors, err := gen.GenerateAIVectors(context.Background(), op)
	require.NoError(t, err)
	require.Len(t, vectors, 1)

	vec := vectors[0]
	assert.Equal(t, "GET", vec.Method)
	assert.Equal(t, "/search", vec.Path)
	assert.Equal(t, "SQL Injection in Search", vec.Scenario)
	assert.Equal(t, "' OR 1=1--", vec.QueryParams["q"])
	assert.Equal(t, "ai", vec.Source)
}
