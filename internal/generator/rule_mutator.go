package generator

import (
	"fmt"
	"math"
	"strings"

	"github.com/fuzzspec/fuzzspec/internal/parser"
	"github.com/getkin/kin-openapi/openapi3"
)

// Standard adversarial payloads dictionary
var (
	SQLInjectionPayloads = []string{
		"' OR '1'='1",
		"1; DROP TABLE users;--",
		"' UNION SELECT NULL, NULL--",
		"admin'--",
		"1' OR 1=1#",
	}

	CommandInjectionPayloads = []string{
		"; ls -la",
		"| cat /etc/passwd",
		"$(whoami)",
		"& ping -c 1 127.0.0.1 &",
	}

	SpecialCharPayloads = []string{
		"\x00",                // Null byte
		"\r\n\r\n",            // CRLF injection
		"\u202Ereversed",      // Right-to-Left Override
		"\uFEFFBOM_INJECTION", // Byte Order Mark
		"<script>alert(1)</script>",
		"../../../../etc/passwd",
	}

	InvalidFormatDates = []string{
		"2024-02-30", // Invalid leap day
		"1970-00-00",
		"invalid-date-string",
		"99999-99-99",
	}
)

// RuleMutator generates deterministic boundary value and adversarial test vectors.
type RuleMutator struct{}

// NewRuleMutator creates a new deterministic RuleMutator.
func NewRuleMutator() *RuleMutator {
	return &RuleMutator{}
}

// GenerateOperationVectors generates all rule-based test vectors for an OpenAPI operation.
func (m *RuleMutator) GenerateOperationVectors(op parser.EndpointOperation) []TestVector {
	var vectors []TestVector
	vecCounter := 1

	// 1. Generate Baseline Valid Vector
	baselineVec := m.generateBaselineVector(op, &vecCounter)
	vectors = append(vectors, baselineVec)

	// 2. Generate Parameter Mutations (Query, Path, Header)
	for _, param := range op.Parameters {
		paramVectors := m.mutateParameter(op, param, &vecCounter)
		vectors = append(vectors, paramVectors...)
	}

	// 3. Generate Request Body Mutations (JSON Schema)
	if op.RequestBody != nil {
		for mediaType, schema := range op.RequestBody.Content {
			if strings.Contains(mediaType, "json") && schema != nil {
				bodyVectors := m.mutateJSONBody(op, schema, op.RequestBody.Required, &vecCounter)
				vectors = append(vectors, bodyVectors...)
			}
		}
	}

	return vectors
}

// generateBaselineVector builds a valid happy-path test vector as reference.
func (m *RuleMutator) generateBaselineVector(op parser.EndpointOperation, counter *int) TestVector {
	id := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
	*counter++

	pathParams := make(map[string]string)
	queryParams := make(map[string]string)
	headers := make(map[string]string)

	for _, p := range op.Parameters {
		validVal := m.generateValidParamValue(p)
		switch p.In {
		case "path":
			pathParams[p.Name] = validVal
		case "query":
			if p.Required {
				queryParams[p.Name] = validVal
			}
		case "header":
			if p.Required {
				headers[p.Name] = validVal
			}
		}
	}

	var body interface{}
	if op.RequestBody != nil {
		for mediaType, schema := range op.RequestBody.Content {
			if strings.Contains(mediaType, "json") && schema != nil {
				body = m.generateValidJSONBody(schema)
				break
			}
		}
	}

	return TestVector{
		ID:                   id,
		Path:                 op.Path,
		Method:               op.Method,
		Scenario:             "Baseline Happy Path",
		Description:          "Expected valid payload satisfying all schema constraints",
		MutationType:         MutationTypeBaseline,
		TargetLocation:       "endpoint",
		PathParams:           pathParams,
		QueryParams:          queryParams,
		Headers:              headers,
		Body:                 body,
		ExpectedStatusFamily: "2xx",
		Source:               "heuristic",
	}
}

// mutateParameter generates boundary, type mismatch, and injection test vectors for a parameter.
func (m *RuleMutator) mutateParameter(op parser.EndpointOperation, param parser.ParameterSchema, counter *int) []TestVector {
	var vectors []TestVector
	schema := param.Schema

	// A. Missing Required Parameter
	if param.Required && param.In != "path" {
		id := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, TestVector{
			ID:                   id,
			Path:                 op.Path,
			Method:               op.Method,
			Scenario:             fmt.Sprintf("Missing required %s parameter '%s'", param.In, param.Name),
			Description:          fmt.Sprintf("Parameter '%s' is required by contract but omitted", param.Name),
			MutationType:         MutationTypeMissingRequired,
			TargetLocation:       param.In,
			TargetField:          param.Name,
			ExpectedStatusFamily: "4xx",
			Source:               "heuristic",
		})
	}

	if schema == nil {
		return vectors
	}

	// B. Integer / Number parameter boundaries
	if schema.Type != nil && (schema.Type.Is("integer") || schema.Type.Is("number")) {
		// Minimum boundary
		if schema.Min != nil {
			id := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			val := fmt.Sprintf("%d", int64(*schema.Min)-1)
			vectors = append(vectors, m.buildParamVector(op, param, id, "Minimum boundary underflow (min - 1)", val, MutationTypeBoundary))
		}

		// Maximum boundary
		if schema.Max != nil {
			id := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			val := fmt.Sprintf("%d", int64(*schema.Max)+1)
			vectors = append(vectors, m.buildParamVector(op, param, id, "Maximum boundary overflow (max + 1)", val, MutationTypeBoundary))
		}

		// Integer Overflow
		idIntMax := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, m.buildParamVector(op, param, idIntMax, "Int64 maximum overflow", fmt.Sprintf("%d", uint64(math.MaxInt64)+100), MutationTypeBoundary))

		// Negative number
		idNeg := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, m.buildParamVector(op, param, idNeg, "Negative integer mutation", "-999999", MutationTypeBoundary))

		// Type Mismatch: String in integer parameter
		idType := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, m.buildParamVector(op, param, idType, "Type mismatch (string passed to integer parameter)", "not_a_number_string", MutationTypeInvalidType))
	}

	// C. String parameter adversarial & injection vectors
	if schema.Type != nil && schema.Type.Is("string") {
		// SQLi
		for _, sqli := range SQLInjectionPayloads[:2] {
			id := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			vectors = append(vectors, m.buildParamVector(op, param, id, fmt.Sprintf("SQL injection payload in parameter '%s'", param.Name), sqli, MutationTypeAdversarial))
		}

		// Null Byte
		idNull := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, m.buildParamVector(op, param, idNull, fmt.Sprintf("Null byte string injection in '%s'", param.Name), "prefix\x00suffix", MutationTypeAdversarial))

		// Empty string if minLength > 0
		if schema.MinLength > 0 {
			idEmpty := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			vectors = append(vectors, m.buildParamVector(op, param, idEmpty, fmt.Sprintf("Empty string violating minLength on '%s'", param.Name), "", MutationTypeBoundary))
		}
	}

	return vectors
}

func (m *RuleMutator) buildParamVector(op parser.EndpointOperation, param parser.ParameterSchema, id, scenario, val string, mutType MutationType) TestVector {
	pathParams := make(map[string]string)
	queryParams := make(map[string]string)
	headers := make(map[string]string)

	switch param.In {
	case "path":
		pathParams[param.Name] = val
	case "query":
		queryParams[param.Name] = val
	case "header":
		headers[param.Name] = val
	}

	return TestVector{
		ID:                   id,
		Path:                 op.Path,
		Method:               op.Method,
		Scenario:             scenario,
		Description:          fmt.Sprintf("Testing parameter '%s' with mutated value: %s", param.Name, val),
		MutationType:         mutType,
		TargetLocation:       param.In,
		TargetField:          param.Name,
		PathParams:           pathParams,
		QueryParams:          queryParams,
		Headers:              headers,
		ExpectedStatusFamily: "4xx",
		Source:               "heuristic",
	}
}

// mutateJSONBody generates boundary, type mismatch, and adversarial body payloads.
func (m *RuleMutator) mutateJSONBody(op parser.EndpointOperation, schema *openapi3.Schema, isRequired bool, counter *int) []TestVector {
	var vectors []TestVector

	// 1. Empty Object Payload `{}`
	idEmpty := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
	*counter++
	vectors = append(vectors, TestVector{
		ID:                   idEmpty,
		Path:                 op.Path,
		Method:               op.Method,
		Scenario:             "Empty JSON Object Payload",
		Description:          "Sending empty JSON object {} to evaluate missing field handling",
		MutationType:         MutationTypeEmptyPayload,
		TargetLocation:       "body",
		Body:                 map[string]interface{}{},
		ExpectedStatusFamily: "4xx",
		Source:               "heuristic",
	})

	// 2. Non-Object Root Type (e.g. sending Array `[]` or Primitive string)
	idArrayRoot := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
	*counter++
	vectors = append(vectors, TestVector{
		ID:                   idArrayRoot,
		Path:                 op.Path,
		Method:               op.Method,
		Scenario:             "Type Mismatch: Array passed as Root Object",
		Description:          "Sending JSON array [1, 2, 3] when object schema is expected",
		MutationType:         MutationTypeInvalidType,
		TargetLocation:       "body",
		Body:                 []interface{}{1, 2, 3},
		ExpectedStatusFamily: "4xx",
		Source:               "heuristic",
	})

	// 3. Mutate required properties
	baselineMap := m.generateValidJSONBody(schema).(map[string]interface{})

	for _, reqKey := range schema.Required {
		// A. Omit required field
		omittedMap := copyMap(baselineMap)
		delete(omittedMap, reqKey)

		idOmit := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, TestVector{
			ID:                   idOmit,
			Path:                 op.Path,
			Method:               op.Method,
			Scenario:             fmt.Sprintf("Omit required body field '%s'", reqKey),
			Description:          fmt.Sprintf("Contract requires '%s' in body, sending payload without it", reqKey),
			MutationType:         MutationTypeMissingRequired,
			TargetLocation:       "body",
			TargetField:          reqKey,
			Body:                 omittedMap,
			ExpectedStatusFamily: "4xx",
			Source:               "heuristic",
		})

		// B. Null value for required field
		nullMap := copyMap(baselineMap)
		nullMap[reqKey] = nil

		idNull := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
		*counter++
		vectors = append(vectors, TestVector{
			ID:                   idNull,
			Path:                 op.Path,
			Method:               op.Method,
			Scenario:             fmt.Sprintf("Null value for required body field '%s'", reqKey),
			Description:          fmt.Sprintf("Sending explicit null for non-nullable required key '%s'", reqKey),
			MutationType:         MutationTypeInvalidType,
			TargetLocation:       "body",
			TargetField:          reqKey,
			Body:                 nullMap,
			ExpectedStatusFamily: "4xx",
			Source:               "heuristic",
		})
	}

	// 4. Mutate individual property boundaries & injection payloads
	for propName, propRef := range schema.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}
		propSchema := propRef.Value

		// String Property Mutations
		if propSchema.Type != nil && propSchema.Type.Is("string") {
			// SQLi injection in field
			for _, sqli := range SQLInjectionPayloads[:2] {
				mutMap := copyMap(baselineMap)
				mutMap[propName] = sqli

				id := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
				*counter++
				vectors = append(vectors, TestVector{
					ID:                   id,
					Path:                 op.Path,
					Method:               op.Method,
					Scenario:             fmt.Sprintf("SQL injection in field '%s'", propName),
					Description:          fmt.Sprintf("Sending SQL payload '%s' in field '%s'", sqli, propName),
					MutationType:         MutationTypeAdversarial,
					TargetLocation:       "body",
					TargetField:          propName,
					Body:                 mutMap,
					ExpectedStatusFamily: "4xx",
					Source:               "heuristic",
				})
			}

			// Buffer length overflow: 10KB string
			overflowMap := copyMap(baselineMap)
			overflowMap[propName] = strings.Repeat("A", 10000)
			idOverflow := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			vectors = append(vectors, TestVector{
				ID:                   idOverflow,
				Path:                 op.Path,
				Method:               op.Method,
				Scenario:             fmt.Sprintf("Long buffer string (10KB) in field '%s'", propName),
				Description:          fmt.Sprintf("Sending 10,000 character string in '%s' to test buffer boundary", propName),
				MutationType:         MutationTypeBoundary,
				TargetLocation:       "body",
				TargetField:          propName,
				Body:                 overflowMap,
				ExpectedStatusFamily: "4xx",
				Source:               "heuristic",
			})
		}

		// Integer Property Mutations
		if propSchema.Type != nil && (propSchema.Type.Is("integer") || propSchema.Type.Is("number")) {
			// Negative number
			negMap := copyMap(baselineMap)
			negMap[propName] = -99999

			idNeg := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			vectors = append(vectors, TestVector{
				ID:                   idNeg,
				Path:                 op.Path,
				Method:               op.Method,
				Scenario:             fmt.Sprintf("Negative number in field '%s'", propName),
				Description:          fmt.Sprintf("Sending negative number -99999 to field '%s'", propName),
				MutationType:         MutationTypeBoundary,
				TargetLocation:       "body",
				TargetField:          propName,
				Body:                 negMap,
				ExpectedStatusFamily: "4xx",
				Source:               "heuristic",
			})

			// Type mismatch: String in integer field
			typeMap := copyMap(baselineMap)
			typeMap[propName] = "invalid_string_instead_of_number"

			idType := fmt.Sprintf("vec_%s_%s_%03d", strings.ToLower(op.Method), sanitizePath(op.Path), *counter)
			*counter++
			vectors = append(vectors, TestVector{
				ID:                   idType,
				Path:                 op.Path,
				Method:               op.Method,
				Scenario:             fmt.Sprintf("Type mismatch: string in integer field '%s'", propName),
				Description:          fmt.Sprintf("Sending string value to numeric field '%s'", propName),
				MutationType:         MutationTypeInvalidType,
				TargetLocation:       "body",
				TargetField:          propName,
				Body:                 typeMap,
				ExpectedStatusFamily: "4xx",
				Source:               "heuristic",
			})
		}
	}

	return vectors
}

// Helpers for generating baseline valid values
func (m *RuleMutator) generateValidParamValue(param parser.ParameterSchema) string {
	if param.Schema == nil || param.Schema.Type == nil {
		return "sample_value"
	}

	if param.Schema.Type.Is("integer") {
		if param.Schema.Min != nil {
			return fmt.Sprintf("%d", int64(*param.Schema.Min))
		}
		return "1"
	}

	if len(param.Schema.Enum) > 0 {
		return fmt.Sprintf("%v", param.Schema.Enum[0])
	}

	return "valid_param"
}

func (m *RuleMutator) generateValidJSONBody(schema *openapi3.Schema) interface{} {
	if schema == nil {
		return map[string]interface{}{}
	}

	result := make(map[string]interface{})

	for propName, propRef := range schema.Properties {
		if propRef == nil || propRef.Value == nil {
			continue
		}
		prop := propRef.Value

		if prop.Type == nil {
			result[propName] = "sample"
			continue
		}

		if prop.Type.Is("string") {
			if len(prop.Enum) > 0 {
				result[propName] = prop.Enum[0]
			} else {
				result[propName] = fmt.Sprintf("valid_%s", propName)
			}
		} else if prop.Type.Is("integer") {
			if prop.Min != nil {
				result[propName] = int64(*prop.Min)
			} else {
				result[propName] = 10
			}
		} else if prop.Type.Is("number") {
			result[propName] = 19.99
		} else if prop.Type.Is("boolean") {
			result[propName] = true
		} else if prop.Type.Is("array") {
			result[propName] = []interface{}{}
		} else if prop.Type.Is("object") {
			result[propName] = map[string]interface{}{}
		}
	}

	return result
}

func copyMap(original map[string]interface{}) map[string]interface{} {
	cp := make(map[string]interface{})
	for k, v := range original {
		cp[k] = v
	}
	return cp
}

func sanitizePath(p string) string {
	p = strings.ReplaceAll(p, "/", "_")
	p = strings.ReplaceAll(p, "{", "")
	p = strings.ReplaceAll(p, "}", "")
	return strings.Trim(p, "_")
}
