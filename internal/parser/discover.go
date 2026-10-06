package parser

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/fuzzspec/fuzzspec/internal/config"
)

// CommonDiscoveryEndpoints contains standard default documentation paths across frameworks.
var CommonDiscoveryEndpoints = []string{
	"/openapi.json",                // FastAPI, Django Ninja
	"/v3/api-docs",                 // Spring Boot Springdoc
	"/v3/api-docs.yaml",            // Spring Boot YAML
	"/api-json",                    // NestJS Swagger
	"/swagger/doc.json",            // Go Gin/Echo Swag
	"/swagger.json",                // Express / Fastify Swagger
	"/docs/api.json",               // Laravel Scramble
	"/api/documentation",           // Laravel L5-Swagger
	"/swagger/v1/swagger.json",     // ASP.NET Core Swashbuckle
	"/api-docs/openapi.json",       // Rust Utoipa
	"/api-docs/v1/swagger.yaml",    // Ruby on Rails Rswag
}

// AutoDiscoverer probes a live target server to find its exposed OpenAPI specification.
type AutoDiscoverer struct {
	httpClient *http.Client
}

// NewAutoDiscoverer creates a new auto-discovery probe engine.
func NewAutoDiscoverer() *AutoDiscoverer {
	return &AutoDiscoverer{
		httpClient: &http.Client{
			Timeout: 3 * time.Second,
		},
	}
}

// Discover probes the target base URL and returns the working OpenAPI specification URL.
func (d *AutoDiscoverer) Discover(ctx context.Context, baseURL string, customLangs []config.CustomLang) (string, error) {
	baseURL = strings.TrimRight(baseURL, "/")

	var endpoints []string
	endpoints = append(endpoints, CommonDiscoveryEndpoints...)

	for _, cl := range customLangs {
		endpoints = append(endpoints, cl.DiscoveryEndpoints...)
	}

	for _, path := range endpoints {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		fullURL := baseURL + "/" + strings.TrimLeft(path, "/")
		req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Accept", "application/json, application/yaml, text/yaml, */*")

		resp, err := d.httpClient.Do(req)
		if err != nil {
			continue
		}
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			contentType := resp.Header.Get("Content-Type")
			// Verify it returns JSON or YAML rather than a generic HTML 404 page
			if strings.Contains(contentType, "json") || strings.Contains(contentType, "yaml") || strings.Contains(contentType, "text") {
				return fullURL, nil
			}
		}
	}

	return "", fmt.Errorf("no OpenAPI/Swagger documentation endpoint discovered on %s", baseURL)
}
