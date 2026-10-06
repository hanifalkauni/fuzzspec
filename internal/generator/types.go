package generator

// MutationType identifies the category of test vector mutation.
type MutationType string

const (
	MutationTypeBaseline        MutationType = "BASELINE_VALID"
	MutationTypeBoundary        MutationType = "BOUNDARY_VALUE"
	MutationTypeAdversarial     MutationType = "ADVERSARIAL_INJECTION"
	MutationTypeInvalidType     MutationType = "INVALID_DATA_TYPE"
	MutationTypeMissingRequired MutationType = "MISSING_REQUIRED"
	MutationTypeEmptyPayload    MutationType = "EMPTY_PAYLOAD"
	MutationTypeMalformedFormat MutationType = "MALFORMED_FORMAT"
	MutationTypeAISemantic      MutationType = "AI_SEMANTIC_EDGE"
)

// TestVector represents a concrete HTTP test case generated for an API endpoint.
type TestVector struct {
	ID                   string            `json:"id"`
	Path                 string            `json:"path"`
	Method               string            `json:"method"`
	Scenario             string            `json:"scenario"`
	Description          string            `json:"description"`
	MutationType         MutationType      `json:"mutation_type"`
	TargetLocation       string            `json:"target_location"` // "body", "query", "path", "header"
	TargetField          string            `json:"target_field,omitempty"`
	PathParams           map[string]string `json:"path_params,omitempty"`
	QueryParams          map[string]string `json:"query_params,omitempty"`
	Headers              map[string]string `json:"headers,omitempty"`
	Body                 interface{}       `json:"body,omitempty"`
	ExpectedStatusFamily string            `json:"expected_status_family"` // "2xx", "4xx"
	Source               string            `json:"source"`                 // "heuristic" or "ai"
}

// Generator is the interface for generating test vectors.
type Generator interface {
	Generate(operation interface{}) ([]TestVector, error)
}
