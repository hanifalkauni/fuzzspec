package generator_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVectorCache_Operations(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fuzzspec-cache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cache := generator.NewVectorCache(tempDir, true)

	op := parser.EndpointOperation{
		Method: "GET",
		Path:   "/users/{id}",
		Parameters: []parser.ParameterSchema{
			{Name: "id", In: "path", Required: true},
		},
	}

	key := cache.ComputeKey(op)
	assert.NotEmpty(t, key)

	// Verify miss on empty cache
	vectors, found := cache.Get(key)
	assert.False(t, found)
	assert.Nil(t, vectors)

	// Store test vectors
	sampleVectors := []generator.TestVector{
		{
			ID:           "vec_test_1",
			Path:         "/users/{id}",
			Method:       "GET",
			Scenario:     "Negative User ID",
			MutationType: generator.MutationTypeAISemantic,
			PathParams:   map[string]string{"id": "-999"},
		},
	}

	err = cache.Set(key, sampleVectors)
	require.NoError(t, err)

	// Verify file was written
	cacheFile := filepath.Join(tempDir, key+".json")
	assert.FileExists(t, cacheFile)

	// Verify hit on populated cache
	cached, found := cache.Get(key)
	assert.True(t, found)
	require.Len(t, cached, 1)
	assert.Equal(t, "vec_test_1", cached[0].ID)
	assert.Equal(t, "-999", cached[0].PathParams["id"])
}

func TestVectorCache_Disabled(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "fuzzspec-cache-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	cache := generator.NewVectorCache(tempDir, false)

	op := parser.EndpointOperation{
		Method: "GET",
		Path:   "/health",
	}

	key := cache.ComputeKey(op)
	err = cache.Set(key, []generator.TestVector{{ID: "vec_disabled"}})
	require.NoError(t, err)

	_, found := cache.Get(key)
	assert.False(t, found)
}
