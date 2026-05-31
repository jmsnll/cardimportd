package config_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
)

const fixtureYAML = `watch_paths:
  - /volumeUSB1/usbshare
  - /volumeUSB2/usbshare
import_root: /volume1/photos
cards:
  1A2B-3C4D:
    owner: James
    status: active
    first_seen: 2026-01-15T08:30:00Z
  5E6F-7A8B:
    owner: Sophie
    status: pending
file_extensions:
  - .jpg
  - .raf
  - .arw
log_path: /var/log/cardimportd.log
`

func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "config-*.yaml")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	if _, err := f.WriteString(content); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func TestDefault(t *testing.T) {
	cfg := config.Default()
	if cfg == nil {
		t.Fatal("Default() returned nil")
	}
	if cfg.ImportRoot == "" {
		t.Error("Default ImportRoot is empty")
	}
	if len(cfg.WatchPaths) == 0 {
		t.Error("Default WatchPaths is empty")
	}
	if len(cfg.FileExtensions) == 0 {
		t.Error("Default FileExtensions is empty")
	}
	if cfg.Cards == nil {
		t.Error("Default Cards map is nil")
	}
}

func TestLoadSaveRoundTrip(t *testing.T) {
	src := writeTempYAML(t, fixtureYAML)
	cfg, err := config.Load(src)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.ImportRoot != "/volume1/photos" {
		t.Errorf("ImportRoot = %q, want %q", cfg.ImportRoot, "/volume1/photos")
	}
	james, ok := cfg.LookupCard("1A2B-3C4D")
	if !ok {
		t.Fatal("card 1A2B-3C4D not found after Load")
	}
	if james.Status != config.StatusActive {
		t.Errorf("Status = %q, want active", james.Status)
	}

	dst := filepath.Join(t.TempDir(), "config-out.yaml")
	if err := cfg.Save(dst); err != nil {
		t.Fatalf("Save: %v", err)
	}
	cfg2, err := config.Load(dst)
	if err != nil {
		t.Fatalf("Load after Save: %v", err)
	}
	if cfg2.ImportRoot != cfg.ImportRoot {
		t.Errorf("round-trip ImportRoot = %q, want %q", cfg2.ImportRoot, cfg.ImportRoot)
	}
}

func TestLoadDefaultExtensions(t *testing.T) {
	const noExtYAML = "watch_paths:\n  - /volumeUSB1/usbshare\nimport_root: /volume1/photos\nlog_path: /var/log/cardimportd.log\n"
	src := writeTempYAML(t, noExtYAML)
	cfg, err := config.Load(src)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(cfg.FileExtensions) == 0 {
		t.Fatal("FileExtensions empty, want defaults")
	}
}

func TestLoadUnknownField(t *testing.T) {
	const badYAML = "watch_paths:\n  - /v\nimport_root: /v\nlog_path: /v\nunexpected_key: boom\n"
	src := writeTempYAML(t, badYAML)
	if _, err := config.Load(src); err == nil {
		t.Fatal("expected error for unknown field, got nil")
	}
}

func TestRegisterPendingNew(t *testing.T) {
	cfg := &config.Config{Cards: make(map[string]config.CardEntry)}
	before := time.Now().UTC()
	added := cfg.RegisterPending("AABB-CCDD")
	after := time.Now().UTC()

	if !added {
		t.Fatal("RegisterPending: expected true for new uuid")
	}
	entry, ok := cfg.LookupCard("AABB-CCDD")
	if !ok {
		t.Fatal("card not found after RegisterPending")
	}
	if entry.Status != config.StatusPending {
		t.Errorf("Status = %q, want pending", entry.Status)
	}
	if entry.FirstSeen == nil || entry.FirstSeen.Before(before) || entry.FirstSeen.After(after) {
		t.Errorf("FirstSeen %v out of range [%v, %v]", entry.FirstSeen, before, after)
	}
}

func TestRegisterPendingIdempotent(t *testing.T) {
	cfg := &config.Config{Cards: make(map[string]config.CardEntry)}
	cfg.RegisterPending("AABB-CCDD")
	if cfg.RegisterPending("AABB-CCDD") {
		t.Fatal("RegisterPending: expected false for duplicate uuid")
	}
}

func TestRegisterPendingActiveCardUntouched(t *testing.T) {
	cfg := &config.Config{
		Cards: map[string]config.CardEntry{
			"1A2B-3C4D": {Owner: "James", Status: config.StatusActive},
		},
	}
	if cfg.RegisterPending("1A2B-3C4D") {
		t.Fatal("RegisterPending: expected false for already-registered uuid")
	}
	entry, _ := cfg.LookupCard("1A2B-3C4D")
	if entry.Status != config.StatusActive {
		t.Errorf("Status changed to %q, want active", entry.Status)
	}
}

func TestSaveNoTmpFileLeftBehind(t *testing.T) {
	src := writeTempYAML(t, fixtureYAML)
	cfg, _ := config.Load(src)
	dst := filepath.Join(t.TempDir(), "config-out.yaml")
	if err := cfg.Save(dst); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(dst + ".tmp"); !os.IsNotExist(err) {
		t.Errorf("temp file still exists after Save")
	}
}
