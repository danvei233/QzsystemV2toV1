package gateway

import (
	"net/http"
	"net/url"
	"testing"
)

func TestMapV2ToV1ConvertsMemoryMBToGBForOpenHost(t *testing.T) {
	got := MapV2ToV1("/api/v1/openHost", map[string]any{
		"line_id": "7",
		"os_name": "ubuntu24.04",
		"cpu":     "4",
		"memory":  "8192",
	})

	if !got.Supported {
		t.Fatalf("expected supported mapping: %s", got.Reason)
	}
	if got.Request.Method != http.MethodPost || got.Request.Path != "create_host" {
		t.Fatalf("unexpected request target: %#v", got.Request)
	}
	if got.Request.Query.Get("memory") != "8" {
		t.Fatalf("expected v1 memory=8 GB, got query %s", got.Request.Query.Encode())
	}
}

func TestMapV2ToV1ConvertsApproximateMemoryMBToGBForUpdateHost(t *testing.T) {
	got := MapV2ToV1("/api/v1/updateHost", map[string]any{
		"hostid": "4370",
		"memory": "8000",
	})

	if !got.Supported {
		t.Fatalf("expected supported mapping: %s", got.Reason)
	}
	values, err := url.ParseQuery(string(got.Request.Body))
	if err != nil {
		t.Fatalf("parse mapped body: %v", err)
	}
	if values.Get("memory") != "" {
		t.Fatalf("expected updateHost to use query params, got body %s", values.Encode())
	}
	if got.Request.Query.Get("memory") != "8" {
		t.Fatalf("expected v1 memory=8 GB, got query %s", got.Request.Query.Encode())
	}
}
