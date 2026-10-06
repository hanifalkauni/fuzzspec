package parser_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParser_ParseLocalSpec(t *testing.T) {
	fixturePath, err := filepath.Abs("../../test/fixtures/petstore.yaml")
	require.NoError(t, err)

	p := parser.NewParser()
	parsed, err := p.Parse(context.Background(), fixturePath)

	require.NoError(t, err)
	require.NotNil(t, parsed)

	assert.Equal(t, "Sample Petstore API", parsed.Title)
	assert.Equal(t, "1.0.0", parsed.Version)
	assert.Equal(t, "3.0.3", parsed.DocVersion)
	assert.Len(t, parsed.Servers, 1)
	assert.Equal(t, "http://localhost:8080/v1", parsed.Servers[0])

	// Should extract 4 operations: GET /pets, POST /pets, GET /pets/{petId}, DELETE /pets/{petId}
	assert.Len(t, parsed.Operations, 4)

	// Check GET /pets operation
	var getPets *parser.EndpointOperation
	for _, op := range parsed.Operations {
		if op.Path == "/pets" && op.Method == "GET" {
			getPets = &op
			break
		}
	}
	require.NotNil(t, getPets, "GET /pets should be present")
	assert.Equal(t, "listPets", getPets.OperationID)
	assert.Len(t, getPets.Parameters, 2)
	assert.Equal(t, "limit", getPets.Parameters[0].Name)
	assert.Equal(t, "query", getPets.Parameters[0].In)
	assert.False(t, getPets.Parameters[0].Required)

	// Check POST /pets operation
	var postPets *parser.EndpointOperation
	for _, op := range parsed.Operations {
		if op.Path == "/pets" && op.Method == "POST" {
			postPets = &op
			break
		}
	}
	require.NotNil(t, postPets, "POST /pets should be present")
	require.NotNil(t, postPets.RequestBody)
	assert.True(t, postPets.RequestBody.Required)
	assert.Contains(t, postPets.RequestBody.Content, "application/json")

	// Check GET /pets/{petId} inherits path parameter
	var getPetById *parser.EndpointOperation
	for _, op := range parsed.Operations {
		if op.Path == "/pets/{petId}" && op.Method == "GET" {
			getPetById = &op
			break
		}
	}
	require.NotNil(t, getPetById, "GET /pets/{petId} should be present")
	assert.Len(t, getPetById.Parameters, 1)
	assert.Equal(t, "petId", getPetById.Parameters[0].Name)
	assert.Equal(t, "path", getPetById.Parameters[0].In)
	assert.True(t, getPetById.Parameters[0].Required)
}

func TestParser_FileNotFound(t *testing.T) {
	p := parser.NewParser()
	_, err := p.Parse(context.Background(), "non_existent_file.yaml")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "spec file not found")
}
