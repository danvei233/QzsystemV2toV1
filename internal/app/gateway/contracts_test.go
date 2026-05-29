package gateway

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"xiaoheiproxy/internal/domain"
)

func TestEndpointContractsMatchOpenAPIExamples(t *testing.T) {
	spec := loadOpenAPISpec(t)
	for path, contract := range endpointContracts {
		item, ok := spec[path]
		if !ok {
			t.Fatalf("contract path missing from openapi: %s", path)
		}
		if !contract.HasExample {
			continue
		}
		example := item.Example
		if example == nil {
			t.Fatalf("openapi example missing for %s", path)
		}
		if got, want := intFromAny(example["code"]), contract.SuccessCode; got != want {
			t.Fatalf("%s code mismatch: got %d want %d", path, got, want)
		}
		if got, want := stringFromAny(example["msg"]), contract.SuccessMsg; got != want {
			t.Fatalf("%s msg mismatch: got %q want %q", path, got, want)
		}
		if contract.ConflictPolicy == ConflictPolicyFunction {
			continue
		}
		_, hasData := example["data"]
		if hasData != contract.HasData {
			t.Fatalf("%s data presence mismatch: example=%v contract=%v", path, hasData, contract.HasData)
		}
		if !contract.HasData {
			continue
		}
		if got, want := item.Shape, contract.Shape; got != want {
			t.Fatalf("%s data shape mismatch: got %s want %s", path, got, want)
		}
	}
}

func TestFailureEnvelopeUsesContractShape(t *testing.T) {
	cases := []struct {
		path      string
		hasData   bool
		shapeName string
	}{
		{path: "/api/v1/updateHost", hasData: false, shapeName: "none"},
		{path: "/api/v1/findport", hasData: true, shapeName: "array"},
		{path: "/api/v1/snapshot", hasData: true, shapeName: "data+extend"},
		{path: "/api/v1/vnc", hasData: true, shapeName: "url"},
	}

	for _, tc := range cases {
		resp := buildFailureResponse(tc.path, "not implemented")
		var payload map[string]any
		if err := json.Unmarshal(resp.Body, &payload); err != nil {
			t.Fatalf("%s unmarshal failure envelope: %v", tc.path, err)
		}
		if got := intFromAny(payload["code"]); got != 0 {
			t.Fatalf("%s expected code 0, got %d", tc.path, got)
		}
		_, hasData := payload["data"]
		if hasData != tc.hasData {
			t.Fatalf("%s data presence mismatch: got %v want %v", tc.path, hasData, tc.hasData)
		}
		switch tc.shapeName {
		case "array":
			if _, ok := payload["data"].([]any); !ok {
				t.Fatalf("%s expected array data, got %#v", tc.path, payload["data"])
			}
		case "data+extend":
			dataObj, ok := payload["data"].(map[string]any)
			if !ok {
				t.Fatalf("%s expected object data, got %#v", tc.path, payload["data"])
			}
			if _, ok := dataObj["data"].([]any); !ok {
				t.Fatalf("%s expected nested data array", tc.path)
			}
			if _, ok := dataObj["extend"].(map[string]any); !ok {
				t.Fatalf("%s expected nested extend object", tc.path)
			}
		case "url":
			dataObj, ok := payload["data"].(map[string]any)
			if !ok || dataObj["url"] == nil {
				t.Fatalf("%s expected url object, got %#v", tc.path, payload["data"])
			}
		}
	}
}

func TestApplyHostMetadataUsesV2Semantics(t *testing.T) {
	host := map[string]any{
		"id":            1,
		"memory":        8192,
		"hard_disks":    120,
		"bandwidth_out": 10,
		"bandwidth_in":  10,
	}
	meta := &domain.HostV2Metadata{
		HostID:         1,
		MemoryMB:       8192,
		SysDiskSizeGB:  50,
		DataDiskSizeGB: 100,
		NetOutMbps:     20,
		NetInMbps:      100,
	}
	got := applyHostMetadata(host, meta)
	if intFromAny(got["memory"]) != 8 {
		t.Fatalf("expected memory 8, got %#v", got["memory"])
	}
	if intFromAny(got["os_size"]) != 50 {
		t.Fatalf("expected os_size 50, got %#v", got["os_size"])
	}
	if intFromAny(got["hard_disks"]) != 100 {
		t.Fatalf("expected hard_disks 100, got %#v", got["hard_disks"])
	}
	if intFromAny(got["bandwidth_out"]) != 20 || intFromAny(got["bandwidth_in"]) != 100 {
		t.Fatalf("expected bandwidth override, got out=%#v in=%#v", got["bandwidth_out"], got["bandwidth_in"])
	}
}

type openAPISpec struct {
	Paths map[string]openAPIPath `json:"paths"`
}

type openAPIPath struct {
	Post *openAPIOperation `json:"post"`
	Get  *openAPIOperation `json:"get"`
}

type openAPIOperation struct {
	Responses map[string]openAPIResponse `json:"responses"`
}

type openAPIResponse struct {
	Content map[string]openAPIContent `json:"content"`
}

type openAPIContent struct {
	Example map[string]any `json:"example"`
	Schema  openAPISchema  `json:"schema"`
}

type openAPISchema struct {
	Example    map[string]any               `json:"example"`
	Properties map[string]openAPIDataSchema `json:"properties"`
}

type openAPIDataSchema struct {
	Type       string                       `json:"type"`
	Properties map[string]openAPIDataSchema `json:"properties"`
}

type openAPIContractCase struct {
	Example map[string]any
	Shape   DataShape
}

func loadOpenAPISpec(t *testing.T) map[string]openAPIContractCase {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	specPath := filepath.Join(filepath.Dir(file), "..", "..", "..", "默认模块.openapi.json")
	raw, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("read openapi: %v", err)
	}
	var spec openAPISpec
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("unmarshal openapi: %v", err)
	}
	out := map[string]openAPIContractCase{}
	for path, item := range spec.Paths {
		op := item.Post
		if op == nil {
			op = item.Get
		}
		if op == nil {
			continue
		}
		content, ok := op.Responses["200"].Content["application/json"]
		if !ok {
			continue
		}
		example := content.Example
		if example == nil {
			example = content.Schema.Example
		}
		shape := DataShapeNone
		if dataSchema, ok := content.Schema.Properties["data"]; ok {
			switch dataSchema.Type {
			case "array":
				shape = DataShapeArray
			case "object":
				if _, ok := dataSchema.Properties["url"]; ok {
					shape = DataShapeURL
				} else if _, ok := dataSchema.Properties["data"]; ok {
					shape = DataShapeDataExtend
				} else {
					shape = DataShapeObject
				}
			}
		}
		out[path] = openAPIContractCase{
			Example: example,
			Shape:   shape,
		}
	}
	return out
}

func intFromAny(value any) int {
	switch t := value.(type) {
	case int:
		return t
	case float64:
		return int(t)
	default:
		return 0
	}
}

func stringFromAny(value any) string {
	if value == nil {
		return ""
	}
	if text, ok := value.(string); ok {
		return text
	}
	return ""
}
