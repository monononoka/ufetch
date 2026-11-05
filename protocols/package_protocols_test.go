package protocols

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNPMResolvesScopedPackageVersion(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/@scope/tool/1.2.3" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		if r.URL.EscapedPath() == r.URL.Path {
			t.Errorf("scoped package slash should be escaped in URL path: %q", r.URL.EscapedPath())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"name":    "@scope/tool",
			"version": "1.2.3",
			"dist":    map[string]string{"tarball": "https://registry.npmjs.org/@scope/tool/-/tool-1.2.3.tgz"},
		})
	}))
	defer server.Close()

	got, err := (NPM{Registry: server.URL, Client: server.Client()}).Resolve(context.Background(), "@scope/tool@1.2.3")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Filename != "tool-1.2.3.tgz" || got[0].Description != "npm @scope/tool@1.2.3" {
		t.Fatalf("resolved package = %#v", got)
	}
}

func TestParseNPMPackageSpec(t *testing.T) {
	tests := []struct {
		input       string
		wantName    string
		wantVersion string
		wantError   bool
	}{
		{input: "lodash", wantName: "lodash", wantVersion: "latest"},
		{input: "lodash@4.17.21", wantName: "lodash", wantVersion: "4.17.21"},
		{input: "@scope/tool", wantName: "@scope/tool", wantVersion: "latest"},
		{input: "@scope/tool@2.0.0", wantName: "@scope/tool", wantVersion: "2.0.0"},
		{input: "@broken", wantError: true},
		{input: "scope/name", wantError: true},
		{input: "@scope/name/sub", wantError: true},
		{input: "", wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			name, version, err := parsePackageSpec(tt.input)
			if (err != nil) != tt.wantError {
				t.Fatalf("parsePackageSpec() error = %v, wantError %t", err, tt.wantError)
			}
			if err == nil && (name != tt.wantName || version != tt.wantVersion) {
				t.Fatalf("parsePackageSpec() = %q, %q; want %q, %q", name, version, tt.wantName, tt.wantVersion)
			}
		})
	}
}

func TestPyPIResolvesVersionFilesAndSkipsYanked(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/pypi/my-project/2.4.0/json" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"info": map[string]string{"name": "My_Project", "version": "2.4.0"},
			"urls": []map[string]any{
				{"filename": "my_project-2.4.0.tar.gz", "url": "https://files.example/my_project-2.4.0.tar.gz", "packagetype": "sdist", "size": 300},
				{"filename": "my_project-2.4.0-py3-none-any.whl", "url": "https://files.example/my_project-2.4.0-py3-none-any.whl", "packagetype": "bdist_wheel", "size": 100},
				{"filename": "old.whl", "url": "https://files.example/old.whl", "packagetype": "bdist_wheel", "size": 10, "yanked": true},
				{"filename": "insecure.whl", "url": "http://files.example/insecure.whl", "packagetype": "bdist_wheel", "size": 20},
			},
		})
	}))
	defer server.Close()

	got, err := (PyPI{APIBase: server.URL + "/pypi", Client: server.Client()}).Resolve(context.Background(), "My_Project@2.4.0")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("resolved files = %#v, want 2 non-yanked HTTPS distributions", got)
	}
	if got[0].Filename != "my_project-2.4.0-py3-none-any.whl" || got[1].Filename != "my_project-2.4.0.tar.gz" {
		t.Fatalf("unexpected file order: %#v", got)
	}
}
