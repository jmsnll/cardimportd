package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/history"
	"github.com/jmsnll/cardimportd/internal/importer"
	"github.com/jmsnll/cardimportd/internal/notify"
	"github.com/jmsnll/cardimportd/internal/watcher"
	"github.com/jmsnll/cardimportd/internal/webui"
)

func main() {
	cfgPath      := flag.String("config", "/usr/local/etc/cardimportd/config.yaml", "path to config.yaml")
	webuiPort    := flag.Int("webui-port", 8085, "web management UI port (0 to disable)")
	procMounts   := flag.String("proc-mounts", "/proc/mounts", "mounts file to poll")
	pollInterval := flag.Duration("poll-interval", 2*time.Second, "watcher poll interval")
	flag.Parse()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	initialCfg, err := config.Load(*cfgPath)
	if err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			slog.Error("startup: failed to load config", "path", *cfgPath, "error", err)
			os.Exit(1)
		}
		initialCfg = config.Default()
		if mkErr := os.MkdirAll(filepath.Dir(*cfgPath), 0o755); mkErr == nil {
			if saveErr := initialCfg.Save(*cfgPath); saveErr == nil {
				slog.Info("startup: no config found, wrote defaults — edit to configure", "path", *cfgPath)
			} else {
				slog.Warn("startup: no config found, could not write defaults", "path", *cfgPath, "error", saveErr)
			}
		}
	}

	histPath := filepath.Join(filepath.Dir(*cfgPath), "import-history.jsonl")
	hist := history.New(histPath)

	logNotifier := notify.NewLogNotifier(logger)
	notifiers := []notify.Notifier{logNotifier}
	if p := initialCfg.Notifications.Pushover; p != nil && p.AppToken != "" && p.UserKey != "" {
		n := notify.NewFilteredNotifier(notify.NewPushoverNotifier(p.AppToken, p.UserKey), p.Events)
		notifiers = append(notifiers, n)
		slog.Info("pushover notifications enabled")
	}
	if nt := initialCfg.Notifications.Ntfy; nt != nil && nt.URL != "" {
		n := notify.NewFilteredNotifier(notify.NewNtfyNotifier(nt.URL, nt.Token), nt.Events)
		notifiers = append(notifiers, n)
		slog.Info("ntfy notifications enabled", "url", nt.URL)
	}
	if wh := initialCfg.Notifications.Webhook; wh != nil && wh.URL != "" {
		n := notify.NewFilteredNotifier(notify.NewWebhookNotifier(wh.URL, wh.Secret), wh.Events)
		notifiers = append(notifiers, n)
		slog.Info("webhook notifications enabled", "url", wh.URL)
	}
	notifier := notify.NewMultiNotifier(notifiers...)

	// mu guards cfg and imp; both are replaced atomically on config changes.
	var mu sync.RWMutex
	cfg := initialCfg
	imp := importer.New(cfg, notifier)

	getCfg := func() *config.Config {
		mu.RLock()
		defer mu.RUnlock()
		// Return a copy so callers can mutate the Cards map (e.g. RegisterPending)
		// without touching shared state. setCfg is then used to promote changes back.
		c := *cfg
		c.Cards = make(map[string]config.CardEntry, len(cfg.Cards))
		for k, v := range cfg.Cards {
			c.Cards[k] = v
		}
		c.WatchPaths = append([]string(nil), cfg.WatchPaths...)
		c.FileExtensions = append([]string(nil), cfg.FileExtensions...)
		return &c
	}
	setCfg := func(c *config.Config) {
		mu.Lock()
		defer mu.Unlock()
		cfg = c
		imp = importer.New(c, notifier)
	}
	getImp := func() *importer.Importer {
		mu.RLock()
		defer mu.RUnlock()
		return imp
	}

	w := watcher.New(*procMounts, *pollInterval, cfg.WatchPaths...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if *webuiPort > 0 {
		acc := webui.ConfigAccessor{
			Get:  getCfg,
			Set:  setCfg,
			Path: *cfgPath,
		}
		go func() {
			srv := webui.New(acc, hist, *webuiPort)
			if err := srv.Start(ctx); err != nil {
				slog.Error("webui: stopped", "error", err)
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			slog.Error("watcher: stopped unexpectedly", "error", err)
		}
	}()

	slog.Info("cardimportd started", "config", *cfgPath, "webui_port", *webuiPort)

	for {
		select {
		case sig := <-sigCh:
			switch sig {
			case syscall.SIGHUP:
				newCfg, err := config.Load(*cfgPath)
				if err != nil {
					slog.Error("sighup: failed to reload config", "error", err)
					continue
				}
				setCfg(newCfg)
				slog.Info("sighup: config reloaded")
			default:
				slog.Info("shutdown signal received", "signal", sig)
				cancel()
				return
			}

		case evt := <-w.Events():
			if evt.Action != watcher.Mounted {
				continue
			}
			go handleMount(ctx, getCfg, getImp, notifier, setCfg, hist, evt, *cfgPath)
		}
	}
}

func handleMount(
	ctx context.Context,
	getCfg func() *config.Config,
	getImp func() *importer.Importer,
	n notify.Notifier,
	setCfg func(*config.Config),
	hist *history.Log,
	evt watcher.MountEvent,
	cfgPath string,
) {
	cfg := getCfg()
	imp := getImp()
	slog.Info("mount detected", "mount_point", evt.MountPoint, "device", evt.Device)

	uuid, err := cardUUID(evt.Device)
	if err != nil {
		slog.Warn("could not determine card UUID", "device", evt.Device, "error", err)
		return
	}

	slog.Info("card identified", "uuid", uuid, "mount_point", evt.MountPoint)

	entry, ok := cfg.LookupCard(uuid)
	if !ok || entry.Status == config.StatusPending {
		if cfg.RegisterPending(uuid) {
			if err := cfg.Save(cfgPath); err != nil {
				slog.Error("failed to save pending card", "uuid", uuid, "error", err)
			} else {
				setCfg(cfg)
				slog.Warn("new card registered as pending — edit config to activate",
					"uuid", uuid, "config", cfgPath)
			}
			n.Notify(ctx, notify.Event{
				Kind:      notify.KindNewCardPending,
				CardUUID:  uuid,
				MountPath: evt.MountPoint,
				Time:      time.Now(),
				Detail:    "edit config to activate",
			})
		}
		return
	}

	n.Notify(ctx, notify.Event{
		Kind:      notify.KindImportStarted,
		CardUUID:  uuid,
		Owner:     entry.Owner,
		CardLabel: entry.Label,
		MountPath: evt.MountPoint,
		Time:      time.Now(),
	})

	start := time.Now()
	res, err := imp.Import(ctx, entry.Owner, evt.MountPoint, uuid)
	elapsed := time.Since(start)

	if err != nil {
		n.Notify(ctx, notify.Event{
			Kind:      notify.KindImportFailed,
			CardUUID:  uuid,
			Owner:     entry.Owner,
			CardLabel: entry.Label,
			MountPath: evt.MountPoint,
			Time:      time.Now(),
			Detail:    err.Error(),
		})
		return
	}

	n.Notify(ctx, notify.Event{
		Kind:      notify.KindImportCompleted,
		CardUUID:  uuid,
		Owner:     entry.Owner,
		CardLabel: entry.Label,
		MountPath: evt.MountPoint,
		Time:      time.Now(),
		Stats: &notify.ImportStats{
			Total:        res.Total,
			Imported:     res.Imported,
			Skipped:      res.Skipped,
			Failed:       res.Failed,
			MirrorFailed: res.MirrorFailed,
			BytesCopied:  res.BytesCopied,
			Duration:     elapsed,
		},
	})

	if err := hist.Append(history.Entry{
		UUID:        uuid,
		Owner:       entry.Owner,
		MountPath:   evt.MountPoint,
		StartedAt:   start,
		CompletedAt: time.Now(),
		Total:       res.Total,
		Imported:    res.Imported,
		Skipped:     res.Skipped,
		Failed:      res.Failed,
		BytesCopied: res.BytesCopied,
	}); err != nil {
		slog.Warn("history: append failed", "error", err)
	}

	slog.Info("import complete",
		"owner", entry.Owner,
		"total", res.Total,
		"imported", res.Imported,
		"skipped", res.Skipped,
		"failed", res.Failed,
		"mirror_failed", res.MirrorFailed,
		"bytes", res.BytesCopied,
		"duration", fmt.Sprintf("%.1fs", elapsed.Seconds()),
	)
}

func cardUUID(device string) (string, error) {
	out, err := exec.Command("blkid", "-s", "UUID", "-o", "value", device).Output()
	if err != nil {
		return "", fmt.Errorf("blkid %s: %w", device, err)
	}
	uuid := strings.TrimSpace(string(out))
	if uuid == "" {
		return "", fmt.Errorf("blkid returned empty UUID for %s", device)
	}
	return uuid, nil
}
