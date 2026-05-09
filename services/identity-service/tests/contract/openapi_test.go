package contract

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestOpenAPIDefinition(t *testing.T) {
	path := "../../api/openapi/openapi.yaml"
	
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read openapi.yaml at %s: %v", path, err)
	}

	var doc interface{}
	err = yaml.Unmarshal(data, &doc)
	if err != nil {
		t.Fatalf("openapi.yaml is not valid YAML: %v", err)
	}

	// Basic check for required fields
	m := doc.(map[string]interface{})
	if _, ok := m["openapi"]; !ok {
		t.Error("openapi field missing")
	}
	if _, ok := m["info"]; !ok {
		t.Error("info field missing")
	}
	if _, ok := m["paths"]; !ok {
		t.Error("paths field missing")
	}
}
