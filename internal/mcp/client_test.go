package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestToolCatalogAndCall(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/tools/schema", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"tools":[{"server":"filesystem","mcp_tool":"read_file","type":"function","function":{"name":"read_file","description":"Read a file"}}]}`))
	})
	mux.HandleFunc("/v1/servers/filesystem/tools/read_file", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"content":"hello"}`))
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()

	c := NewClient(srv.URL, "")
	cat, err := c.ToolCatalog(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(cat, "filesystem.read_file") {
		t.Fatalf("catalog=%q", cat)
	}
	out, err := c.CallTool(context.Background(), "filesystem", "read_file", json.RawMessage(`{"path":"/x"}`))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "hello") {
		t.Fatalf("out=%q", out)
	}
}
