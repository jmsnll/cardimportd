package webui_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/notify"
	"github.com/jmsnll/cardimportd/internal/webui"
)

type discardNotify struct{}

func (d *discardNotify) Notify(_ context.Context, _ notify.Event) error { return nil }

func makeRunnerCfg(importRoot string) *config.Config {
	return &config.Config{
		ImportRoot:     importRoot,
		FileExtensions: []string{".jpg"},
		Cards: map[string]config.CardEntry{
			"ACTIVE": {Owner: "James", Status: config.StatusActive},
			"PEND":   {Owner: "", Status: config.StatusPending},
		},
	}
}

func TestRunner_UnknownCard(t *testing.T) {
	cfg := makeRunnerCfg(t.TempDir())
	r := webui.NewImportRunner(func() *config.Config { return cfg }, &discardNotify{})
	if err := r.Run(context.Background(), "UNKNOWN", "/m"); err == nil {
		t.Error("expected error")
	}
}

func TestRunner_PendingCard(t *testing.T) {
	cfg := makeRunnerCfg(t.TempDir())
	r := webui.NewImportRunner(func() *config.Config { return cfg }, &discardNotify{})
	if err := r.Run(context.Background(), "PEND", "/m"); err == nil {
		t.Error("expected error")
	}
}

func TestRunner_CompletesCleanly(t *testing.T) {
	dir := t.TempDir()
	mount := filepath.Join(dir, "mount")
	dest := filepath.Join(dir, "dest")
	os.MkdirAll(mount, 0o755)
	os.MkdirAll(dest, 0o755)
	cfg := makeRunnerCfg(dest)
	r := webui.NewImportRunner(func() *config.Config { return cfg }, &discardNotify{})
	if err := r.Run(context.Background(), "ACTIVE", mount); err != nil {
		t.Fatalf("Run: %v", err)
	}
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if !r.IsRunning("ACTIVE") {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	if r.IsRunning("ACTIVE") {
		t.Error("still running after 3s")
	}
}
