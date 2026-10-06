package parser_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuzzspec/fuzzspec/internal/config"
	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAutoDiscoverer_DiscoverSuccess(t *testing.T) {
	// Mock server mimicking FastAPI returning OpenAPI spec on /openapi.json
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/openapi.json" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"openapi": "3.0.0", "info": {"title": "FastAPI App", "version": "1.0.0"}}`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	discoverer := parser.NewAutoDiscoverer()
	discoveredURL, err := discoverer.Discover(context.Background(), server.URL, nil)

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/openapi.json", discoveredURL)
}

func TestAutoDiscoverer_DiscoverCustomEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/custom/api-spec.yaml" {
			w.Header().Set("Content-Type", "application/yaml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`openapi: 3.0.0`))
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	customLangs := []config.CustomLang{
		{
			Name:               "custom-framework",
			DiscoveryEndpoints: []string{"/custom/api-spec.yaml"},
		},
	}

	discoverer := parser.NewAutoDiscoverer()
	discoveredURL, err := discoverer.Discover(context.Background(), server.URL, customLangs)

	require.NoError(t, err)
	assert.Equal(t, server.URL+"/custom/api-spec.yaml", discoveredURL)
}

func TestAutoDiscoverer_DiscoverNotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer server.Close()

	discoverer := parser.NewAutoDiscoverer()
	discoveredURL, err := discoverer.Discover(context.Background(), server.URL, nil)

	require.Error(t, err)
	assert.Empty(t, discoveredURL)
	assert.Contains(t, err.Error(), "no OpenAPI/Swagger documentation endpoint discovered")
}
