package importer

import (
	"path/filepath"
	"testing"
	"time"
)

func TestResolveDestDir_Default(t *testing.T) {
	dt := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	got, err := resolveDestDir("/photos", "", "James", "UUID", "", dt)
	if err != nil {
		t.Fatalf("resolveDestDir: %v", err)
	}
	want := filepath.Join("/photos", "James's Library", "2026", "05", "28")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveDestDir_Custom(t *testing.T) {
	dt := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	got, err := resolveDestDir("/root", "{{ .Owner }}/{{ .Year }}-{{ .Month }}", "Alice", "", "", dt)
	if err != nil {
		t.Fatalf("resolveDestDir: %v", err)
	}
	want := filepath.Join("/root", "Alice", "2026-05")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveDestDir_Invalid(t *testing.T) {
	if _, err := resolveDestDir("/r", "{{ .Bad", "J", "", "", time.Now()); err == nil {
		t.Error("expected error for invalid template")
	}
}
