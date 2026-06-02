package importer

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestReadStamp_NotFound(t *testing.T) {
	_, ok := readStamp(t.TempDir())
	if ok {
		t.Error("expected ok=false when no stamp file")
	}
}

func TestWriteReadStamp_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	if err := writeStamp(dir, "TEST-UUID", "James", 42, false); err != nil {
		t.Fatalf("writeStamp: %v", err)
	}
	s, ok := readStamp(dir)
	if !ok {
		t.Fatal("readStamp: expected ok=true after write")
	}
	if s.CardUUID != "TEST-UUID" {
		t.Errorf("CardUUID = %q, want %q", s.CardUUID, "TEST-UUID")
	}
	if s.Owner != "James" {
		t.Errorf("Owner = %q, want %q", s.Owner, "James")
	}
	if s.FileCount != 42 {
		t.Errorf("FileCount = %d, want 42", s.FileCount)
	}
	if s.ImportedAt.IsZero() {
		t.Error("ImportedAt is zero")
	}
	if time.Since(s.ImportedAt) > 5*time.Second {
		t.Error("ImportedAt is too far in the past")
	}
}

func TestCountMatchingFiles(t *testing.T) {
	dir := t.TempDir()
	ext := map[string]bool{".jpg": true, ".raf": true}

	for _, name := range []string{"a.jpg", "b.jpg", "c.raf", "d.txt", ".hidden.jpg"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sub := filepath.Join(dir, "DCIM")
	os.MkdirAll(sub, 0o755)
	os.WriteFile(filepath.Join(sub, "e.jpg"), []byte("x"), 0o644)

	got := countMatchingFiles(dir, ext)
	if got != 4 { // a.jpg, b.jpg, c.raf, DCIM/e.jpg — d.txt and .hidden.jpg excluded
		t.Errorf("countMatchingFiles = %d, want 4", got)
	}
}

func TestAlreadyImported_NoStamp(t *testing.T) {
	imp := New(makeConfig(t.TempDir()), &discardNotifier{})
	if imp.AlreadyImported(t.TempDir()) {
		t.Error("AlreadyImported should be false when no stamp exists")
	}
}

func TestAlreadyImported_MatchingCount(t *testing.T) {
	cardDir := t.TempDir()
	imp := New(makeConfig(t.TempDir()), &discardNotifier{})

	makeFile(t, cardDir, "a.jpg", randomBytes(t, 256))
	makeFile(t, cardDir, "b.jpg", randomBytes(t, 256))

	if imp.AlreadyImported(cardDir) {
		t.Fatal("AlreadyImported should be false before first import")
	}

	if _, err := imp.Import(context.Background(), "James", cardDir, "UUID"); err != nil {
		t.Fatalf("Import: %v", err)
	}

	if !imp.AlreadyImported(cardDir) {
		t.Error("AlreadyImported should be true after successful import")
	}
}

func TestAlreadyImported_CountChanged(t *testing.T) {
	cardDir := t.TempDir()
	imp := New(makeConfig(t.TempDir()), &discardNotifier{})

	makeFile(t, cardDir, "a.jpg", randomBytes(t, 256))

	if _, err := imp.Import(context.Background(), "James", cardDir, "UUID"); err != nil {
		t.Fatalf("Import: %v", err)
	}

	// Add a new file after import.
	makeFile(t, cardDir, "b.jpg", randomBytes(t, 256))

	if imp.AlreadyImported(cardDir) {
		t.Error("AlreadyImported should be false when file count changed")
	}
}

func TestAlreadyImported_RatedOnlyMismatch(t *testing.T) {
	cardDir := t.TempDir()
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 256))

	// Stamp written with ratedOnly=false.
	if err := writeStamp(cardDir, "UUID", "James", 1, false); err != nil {
		t.Fatalf("writeStamp: %v", err)
	}

	// Importer configured with RatedOnly=true — stamp is stale.
	cfg := makeConfig(t.TempDir())
	cfg.RatedOnly = true
	imp := New(cfg, &discardNotifier{})

	if imp.AlreadyImported(cardDir) {
		t.Error("AlreadyImported should be false when rated_only config changed")
	}
}

func TestImport_NoStampOnFailedFiles(t *testing.T) {
	cardDir := t.TempDir()
	imp := New(makeConfig(t.TempDir()), &discardNotifier{})

	// Create a file that will fail — point at a non-existent src via a symlink.
	link := filepath.Join(cardDir, "bad.jpg")
	os.Symlink("/nonexistent/missing.jpg", link)

	imp.Import(context.Background(), "James", cardDir, "UUID")

	if _, ok := readStamp(cardDir); ok {
		t.Error("stamp should not be written when files failed")
	}
}
