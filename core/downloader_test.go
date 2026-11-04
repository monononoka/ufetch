package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestDownloadResumesWhenServerSupportsRange(t *testing.T) {
	const contents = "hello, resumable world"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") != "bytes=7-" {
			t.Errorf("Range header = %q, want bytes=7-", r.Header.Get("Range"))
		}
		w.Header().Set("Content-Range", fmt.Sprintf("bytes 7-%d/%d", len(contents)-1, len(contents)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write([]byte(contents[7:]))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "result.bin")
	if err := os.WriteFile(destination+".part", []byte(contents[:7]), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination+".part.url", []byte(server.URL), 0600); err != nil {
		t.Fatal(err)
	}

	if err := NewDownloader().Download(context.Background(), server.URL, destination, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Fatalf("downloaded %q, want %q", got, contents)
	}
}

func TestDownloadRestartsWhenServerIgnoresRange(t *testing.T) {
	const contents = "full replacement content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Range") == "" {
			t.Error("expected resume request to include Range")
		}
		// Deliberately return 200: this server ignores the Range request.
		_, _ = w.Write([]byte(contents))
	}))
	defer server.Close()

	destination := filepath.Join(t.TempDir(), "result.bin")
	if err := os.WriteFile(destination+".part", []byte("stale"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(destination+".part.url", []byte(server.URL), 0600); err != nil {
		t.Fatal(err)
	}

	if err := NewDownloader().Download(context.Background(), server.URL, destination, nil); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(destination)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != contents {
		t.Fatalf("downloaded %q, want %q", got, contents)
	}
}
