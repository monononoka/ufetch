package main

import (
	"path/filepath"
	"testing"
)

func TestResolveDestination(t *testing.T) {
	t.Run("uses selected directory and source basename", func(t *testing.T) {
		got, err := resolveDestination("", "downloads", "nested/model.bin")
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Join("downloads", "model.bin")
		if got != want {
			t.Fatalf("resolveDestination() = %q, want %q", got, want)
		}
	})

	t.Run("keeps explicit output path", func(t *testing.T) {
		got, err := resolveDestination(filepath.Join("downloads", "model.bin"), "", "ignored.bin")
		if err != nil {
			t.Fatal(err)
		}
		want := filepath.Clean(filepath.Join("downloads", "model.bin"))
		if got != want {
			t.Fatalf("resolveDestination() = %q, want %q", got, want)
		}
	})

	t.Run("rejects output and directory together", func(t *testing.T) {
		if _, err := resolveDestination("model.bin", "downloads", "source.bin"); err == nil {
			t.Fatal("expected -o and -dir to be rejected together")
		}
	})

	t.Run("rejects an empty source name", func(t *testing.T) {
		if _, err := resolveDestination("", "downloads", ""); err == nil {
			t.Fatal("expected an empty source name to be rejected")
		}
	})
}
