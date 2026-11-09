package service

import (
	"encoding/json"
	"fmt"

	"github.com/arashrasoulzadeh/eye/internal/model"
)

// jsonSchema is the minimal subset of JSON Schema this project's registered
// Schema.Definition documents actually use (see the admin UI's default
// template: {"type": "object", "properties": {...}, "required": [...]}).
// This is deliberately not a full JSON Schema implementation — just enough
// to catch the common evolution mistakes (missing required field, wrong
// primitive type) that "migration tooling" needs to flag.
type jsonSchema struct {
	Type       string                `json:"type"`
	Properties map[string]jsonSchema `json:"properties"`
	Required   []string              `json:"required"`
}

// ValidateEventAgainstSchema checks a decoded event payload (e.g. the
// "payload" field of an ingest event) against a registered schema version's
// Definition, returning a list of human-readable validation errors (empty
// means valid).
func ValidateEventAgainstSchema(payload map[string]any, schema *model.Schema) ([]string, error) {
	var def jsonSchema
	if err := json.Unmarshal([]byte(schema.Definition), &def); err != nil {
		return nil, fmt.Errorf("schema definition is not valid JSON: %w", err)
	}
	return validateAgainst(payload, def, ""), nil
}

func validateAgainst(value any, schema jsonSchema, path string) []string {
	var errs []string
	if schema.Type == "object" || (schema.Type == "" && schema.Properties != nil) {
		obj, ok := value.(map[string]any)
		if !ok {
			return []string{fieldPath(path) + "expected object"}
		}
		for _, req := range schema.Required {
			if _, ok := obj[req]; !ok {
				errs = append(errs, fmt.Sprintf("%smissing required field %q", fieldPath(path), req))
			}
		}
		for key, propSchema := range schema.Properties {
			if v, ok := obj[key]; ok {
				errs = append(errs, validateAgainst(v, propSchema, joinPath(path, key))...)
			}
		}
		return errs
	}
	if schema.Type == "" {
		return nil
	}
	if !matchesPrimitiveType(value, schema.Type) {
		errs = append(errs, fmt.Sprintf("%sexpected type %q, got %s", fieldPath(path), schema.Type, goType(value)))
	}
	return errs
}

func matchesPrimitiveType(value any, t string) bool {
	switch t {
	case "string":
		_, ok := value.(string)
		return ok
	case "number", "integer":
		_, ok := value.(float64)
		return ok
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "array":
		_, ok := value.([]any)
		return ok
	case "object":
		_, ok := value.(map[string]any)
		return ok
	default:
		// Unknown/unsupported type keyword: don't fail validation over it.
		return true
	}
}

func goType(value any) string {
	switch value.(type) {
	case string:
		return "string"
	case float64:
		return "number"
	case bool:
		return "boolean"
	case []any:
		return "array"
	case map[string]any:
		return "object"
	case nil:
		return "null"
	default:
		return "unknown"
	}
}

func fieldPath(path string) string {
	if path == "" {
		return ""
	}
	return path + ": "
}

func joinPath(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}
