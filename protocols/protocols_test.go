package protocols

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"runtime"
	"strings"
	"testing"
)

func TestGitHubResolveLatestReleaseAssets(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/example/tool/releases/latest" {
			t.Errorf("request path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"tag_name": "v1.2.3",
			"assets": []map[string]any{
				{"name": "tool-" + runtime.GOOS + "-" + runtime.GOARCH + ".zip", "browser_download_url": "https://downloads.example/tool.zip", "size": 1234},
				{"name": "checksums.txt", "browser_download_url": "https://downloads.example/checksums.txt", "size": 100},
			},
		})
	}))
	defer server.Close()

	got, err := (GitHub{APIBase: server.URL, Client: server.Client()}).Resolve(context.Background(), "example/tool")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Filename != "tool-"+runtime.GOOS+"-"+runtime.GOARCH+".zip" {
		t.Fatalf("resolved assets = %#v", got)
	}
	if got[0].Size != 1234 {
		t.Errorf("asset size = %d, want 1234", got[0].Size)
	}
}

func TestGitHubSearchResolvesTopRepository(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/repositories":
			_ = json.NewEncoder(w).Encode(map[string]any{"items": []map[string]string{{"full_name": "example/searched"}}})
		case "/repos/example/searched/releases/latest":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"tag_name": "v1",
				"assets":   []map[string]any{{"name": "portable.zip", "browser_download_url": "https://downloads.example/portable.zip", "size": 50}},
			})
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	got, err := (GitHub{APIBase: server.URL, Client: server.Client()}).Resolve(context.Background(), "searched tool")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Description != "portable.zip · example/searched v1" {
		t.Fatalf("resolved result = %#v", got)
	}
}

func TestHuggingFaceSearchResolvesTopModel(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/models":
			if r.URL.Query().Get("search") != "demo model" {
				t.Errorf("search query = %q", r.URL.Query().Get("search"))
			}
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": "example/demo-model", "downloads": 100}})
		case "/api/models/example/demo-model":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"id":       "example/demo-model",
				"sha":      "revision1",
				"siblings": []map[string]any{{"rfilename": "model.gguf", "size": 1000}},
			})
		default:
			t.Errorf("unexpected request path %q", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	provider := HuggingFace{APIBase: server.URL + "/api", WebBase: server.URL, Client: server.Client()}
	got, err := provider.Resolve(context.Background(), "demo model")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || !strings.Contains(got[0].URL, "/example/demo-model/resolve/revision1/model.gguf") {
		t.Fatalf("resolved result = %#v", got)
	}
}

func TestHuggingFaceResolveModelFiles(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/models/example/model" || r.URL.Query().Get("blobs") != "true" {
			t.Errorf("request = %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":  "example/model",
			"sha": "commit123",
			"siblings": []map[string]any{
				{"rfilename": "config.json", "size": 50},
				{"rfilename": "weights/model.safetensors", "size": 4096},
				{"rfilename": "weights/model.safetensors.index.json", "size": 100},
			},
		})
	}))
	defer server.Close()

	provider := HuggingFace{APIBase: server.URL + "/api", WebBase: server.URL, Client: server.Client()}
	got, err := provider.Resolve(context.Background(), "example/model")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Filename != "model.safetensors" {
		t.Fatalf("resolved files = %#v", got)
	}
	parsed, err := url.Parse(got[0].URL)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Path != "/example/model/resolve/commit123/weights/model.safetensors" || parsed.Query().Get("download") != "true" {
		t.Errorf("download URL = %s", got[0].URL)
	}

	explicit, err := provider.Resolve(context.Background(), "example/model/config.json")
	if err != nil {
		t.Fatal(err)
	}
	if len(explicit) != 1 || !strings.HasSuffix(explicit[0].URL, "/config.json?download=true") {
		t.Fatalf("explicit file result = %#v", explicit)
	}
}

func TestCompatibleAssetFiltersPlatformAndChecksums(t *testing.T) {
	if !compatibleAsset("tool-linux-x86_64.tar.gz", "linux", "amd64") {
		t.Error("expected Linux amd64 asset to be compatible")
	}
	if compatibleAsset("tool-windows-arm64.zip", "linux", "amd64") {
		t.Error("Windows arm64 asset must not be selected on Linux amd64")
	}
	if compatibleAsset("checksums.txt", "linux", "amd64") {
		t.Error("checksum file must not be selected as an application")
	}
}
