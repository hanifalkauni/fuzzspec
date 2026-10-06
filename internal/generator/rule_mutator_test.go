package generator_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/generator"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRuleMutator_GeneratePostPetsVectors(t *testing.T) {
	fixturePath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	p := parser.NewParser()
	parsed, err := p.Parse(context.Background(), fixturePath)
	require.NoError(t, err)

	var postPets *parser.EndpointOperation
	for _, op := range parsed.Operations {
		if op.Path == "/pets" && op.Method == "POST" {
			postPets = &op
			break
		}
	}
	require.NotNil(t, postPets)

	mutator := generator.NewRuleMutator()
	vectors := mutator.GenerateOperationVectors(*postPets)

	require.NotEmpty(t, vectors)

	// Verify baseline happy-path vector
	assert.Equal(t, generator.MutationTypeBaseline, vectors[0].MutationType)
	assert.Equal(t, "2xx", vectors[0].ExpectedStatusFamily)
	assert.Equal(t, "POST", vectors[0].Method)

	// Verify we generated mutation vectors (missing required, empty body, SQLi, etc.)
	var hasMissingRequired bool
	var hasEmptyBody bool
	var hasSQLi bool
	var hasTypeMismatch bool

	for _, v := range vectors {
		assert.Equal(t, "POST", v.Method)
		assert.Equal(t, "/pets", v.Path)

		if v.MutationType == generator.MutationTypeMissingRequired {
			hasMissingRequired = true
		}
		if v.MutationType == generator.MutationTypeEmptyPayload {
			hasEmptyBody = true
		}
		if v.MutationType == generator.MutationTypeAdversarial {
			hasSQLi = true
		}
		if v.MutationType == generator.MutationTypeInvalidType {
			hasTypeMismatch = true
		}
	}

	assert.True(t, hasMissingRequired, "Must generate missing required field vectors")
	assert.True(t, hasEmptyBody, "Must generate empty payload vector")
	assert.True(t, hasSQLi, "Must generate SQL injection vectors")
	assert.True(t, hasTypeMismatch, "Must generate type mismatch vectors")
}

func TestRuleMutator_GenerateGetWithParams(t *testing.T) {
	fixturePath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	p := parser.NewParser()
	parsed, err := p.Parse(context.Background(), fixturePath)
	require.NoError(t, err)

	var getPets *parser.EndpointOperation
	for _, op := range parsed.Operations {
		if op.Path == "/pets" && op.Method == "GET" {
			getPets = &op
			break
		}
	}
	require.NotNil(t, getPets)

	mutator := generator.NewRuleMutator()
	vectors := mutator.GenerateOperationVectors(*getPets)

	require.NotEmpty(t, vectors)

	// Should contain query param boundary vectors (e.g. limit overflow, underflow, string type mismatch)
	var hasBoundary bool
	for _, v := range vectors {
		if v.MutationType == generator.MutationTypeBoundary {
			hasBoundary = true
			break
		}
	}
	assert.True(t, hasBoundary, "Must generate boundary mutations for query parameter 'limit'")
}
