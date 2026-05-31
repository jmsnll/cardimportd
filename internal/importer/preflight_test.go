package importer

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirSize_Empty(t *testing.T) {
	size, err := dirSize(t.TempDir())
	if err != nil { t.Fatalf("dirSize: %v", err) }
	if size != 0 { t.Errorf("size = %d, want 0", size) }
}

func TestDirSize_WithFiles(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "a.bin"), make([]byte, 1024), 0o644)
	os.WriteFile(filepath.Join(dir, "b.bin"), make([]byte, 512), 0o644)
	size, err := dirSize(dir)
	if err != nil { t.Fatalf("dirSize: %v", err) }
	if size < 1536 { t.Errorf("size = %d, want >= 1536", size) }
}

func TestPreflight_Sufficient(t *testing.T) {
	dir := t.TempDir()
	mount := filepath.Join(dir, "mount"); dest := filepath.Join(dir, "dest")
	os.MkdirAll(mount, 0o755); os.MkdirAll(dest, 0o755)
	os.WriteFile(filepath.Join(mount, "tiny.jpg"), make([]byte, 100), 0o644)
	if err := preflight(mount, dest, 0); err != nil {
		t.Errorf("should not fail for tiny file: %v", err)
	}
}

func TestPreflight_ImpossibleFloor(t *testing.T) {
	dir := t.TempDir()
	mount := filepath.Join(dir, "mount"); dest := filepath.Join(dir, "dest")
	os.MkdirAll(mount, 0o755); os.MkdirAll(dest, 0o755)
	os.WriteFile(filepath.Join(mount, "tiny.jpg"), make([]byte, 100), 0o644)
	if err := preflight(mount, dest, 1_000_000); err == nil {
		t.Error("expected error for 1_000_000 GB floor")
	}
}
