package importer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteManifest_Basic(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.txt")
	entries := []manifestEntry{{"aabb", "owner/2024/a.jpg"}, {"ccdd", "owner/2024/b.jpg"}}
	if err := writeManifest(path, entries); err != nil {
		t.Fatalf("writeManifest: %v", err)
	}
	data, _ := os.ReadFile(path)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 2 {
		t.Fatalf("want 2 lines, got %d", len(lines))
	}
	if !strings.HasPrefix(lines[0], "aabb  ") {
		t.Errorf("line[0] = %q", lines[0])
	}
}

func TestWriteManifest_EmptyNoFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m.txt")
	writeManifest(path, nil)
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("empty manifest created file")
	}
}
