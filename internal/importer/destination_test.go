package importer

import (
	"path/filepath"
	"testing"
	"text/template"
	"time"
)

func TestResolveDestDir_Default(t *testing.T) {
	dt := time.Date(2026, 5, 28, 0, 0, 0, 0, time.UTC)
	tmpl, err := template.New("dest").Parse(defaultDestTemplate)
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	got, err := resolveDestDir("/photos", tmpl, "James", "UUID", "", dt)
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
	tmpl, err := template.New("dest").Parse("{{ .Owner }}/{{ .Year }}-{{ .Month }}")
	if err != nil {
		t.Fatalf("parse template: %v", err)
	}
	got, err := resolveDestDir("/root", tmpl, "Alice", "", "", dt)
	if err != nil {
		t.Fatalf("resolveDestDir: %v", err)
	}
	want := filepath.Join("/root", "Alice", "2026-05")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestResolveDestDir_Invalid(t *testing.T) {
	if _, err := template.New("dest").Parse("{{ .Bad"); err == nil {
		t.Error("expected error for invalid template")
	}
}
