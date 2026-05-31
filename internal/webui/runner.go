package webui

import (
	"context"
	"fmt"
	"log/slog"
	"sync"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/importer"
	"github.com/jmsnll/cardimportd/internal/notify"
)

type ImportRunner struct {
	getCfg   func() *config.Config
	notifier notify.Notifier
	mu       sync.Mutex
	running  map[string]bool
}

func NewImportRunner(getCfg func() *config.Config, notifier notify.Notifier) *ImportRunner {
	return &ImportRunner{getCfg: getCfg, notifier: notifier, running: make(map[string]bool)}
}

func (r *ImportRunner) Run(ctx context.Context, uuid, mountPath string) error {
	cfg := r.getCfg()
	entry, ok := cfg.LookupCard(uuid)
	if !ok {
		return fmt.Errorf("card %q not found", uuid)
	}
	if entry.Status != config.StatusActive {
		return fmt.Errorf("card %q is not active", uuid)
	}

	r.mu.Lock()
	if r.running[uuid] {
		r.mu.Unlock()
		return fmt.Errorf("import already running for card %q", uuid)
	}
	r.running[uuid] = true
	r.mu.Unlock()

	go func() {
		defer func() { r.mu.Lock(); delete(r.running, uuid); r.mu.Unlock() }()
		imp := importer.New(cfg, r.notifier)
		if _, err := imp.Import(ctx, entry.Owner, mountPath); err != nil {
			slog.Warn("manual import failed", "uuid", uuid, "error", err)
		}
	}()
	return nil
}

func (r *ImportRunner) IsRunning(uuid string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.running[uuid]
}
