package generator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/fuzzspec/fuzzspec/internal/parser"
)

// VectorCache manages caching of AI-generated test vectors to prevent redundant API token costs.
type VectorCache struct {
	cacheDir string
	enabled  bool
}

// NewVectorCache creates a new VectorCache instance.
func NewVectorCache(cacheDir string, enabled bool) *VectorCache {
	if cacheDir == "" {
		cacheDir = ".fuzzspec-cache"
	}
	return &VectorCache{
		cacheDir: cacheDir,
		enabled:  enabled,
	}
}

// ComputeKey calculates a deterministic SHA-256 hash for an OpenAPI endpoint operation.
func (c *VectorCache) ComputeKey(op parser.EndpointOperation) string {
	data, _ := json.Marshal(struct {
		Path        string             `json:"path"`
		Method      string             `json:"method"`
		Parameters  []parser.ParameterSchema `json:"parameters"`
		RequestBody *parser.RequestBodySchema `json:"request_body"`
	}{
		Path:        op.Path,
		Method:      op.Method,
		Parameters:  op.Parameters,
		RequestBody: op.RequestBody,
	})

	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// Get retrieves cached test vectors if available.
func (c *VectorCache) Get(key string) ([]TestVector, bool) {
	if !c.enabled {
		return nil, false
	}

	filePath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.json", key))
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, false
	}

	var vectors []TestVector
	if err := json.Unmarshal(data, &vectors); err != nil {
		return nil, false
	}

	return vectors, true
}

// Set stores generated test vectors to disk cache.
func (c *VectorCache) Set(key string, vectors []TestVector) error {
	if !c.enabled || len(vectors) == 0 {
		return nil
	}

	if err := os.MkdirAll(c.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	filePath := filepath.Join(c.cacheDir, fmt.Sprintf("%s.json", key))
	data, err := json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize cached vectors: %w", err)
	}

	return os.WriteFile(filePath, data, 0644)
}
