package test

import (
	"testing"

	"github.com/eye/eye/internal/model"
	"github.com/eye/eye/internal/service"
)

func TestValidateEventAgainstSchemaCleanPayload(t *testing.T) {
	schema := &model.Schema{Definition: `{
		"type": "object",
		"properties": {"button": {"type": "string"}, "value": {"type": "number"}},
		"required": ["button"]
	}`}
	errs, err := service.ValidateEventAgainstSchema(map[string]any{"button": "signup", "value": float64(1)}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 0 {
		t.Errorf("expected no errors, got %v", errs)
	}
}

func TestValidateEventAgainstSchemaMissingRequiredField(t *testing.T) {
	schema := &model.Schema{Definition: `{
		"type": "object",
		"properties": {"button": {"type": "string"}},
		"required": ["button"]
	}`}
	errs, err := service.ValidateEventAgainstSchema(map[string]any{}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %v", errs)
	}
}

func TestValidateEventAgainstSchemaWrongType(t *testing.T) {
	schema := &model.Schema{Definition: `{
		"type": "object",
		"properties": {"value": {"type": "number"}},
		"required": ["value"]
	}`}
	errs, err := service.ValidateEventAgainstSchema(map[string]any{"value": "not-a-number"}, schema)
	if err != nil {
		t.Fatal(err)
	}
	if len(errs) != 1 {
		t.Fatalf("expected 1 type error, got %v", errs)
	}
}

func TestValidateEventAgainstSchemaInvalidDefinitionErrors(t *testing.T) {
	schema := &model.Schema{Definition: `not json`}
	if _, err := service.ValidateEventAgainstSchema(map[string]any{}, schema); err == nil {
		t.Error("expected error for invalid schema definition JSON")
	}
}
