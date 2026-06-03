package main

import (
	"context"
	"crypto/rand"
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
	historyPath  := flag.String("history-log", "/usr/local/etc/cardimportd/history.jsonl", "path to import history log")
	webuiPort    := flag.Int("webui-port", 8085, "web management UI port (0 to disable)")
	procMounts   := flag.String("proc-mounts", "/proc/mounts", "mounts file to poll")
	pollInterval := flag.Duration("poll-interval", 2*time.Second, "watcher poll interval")
	flag.Parse()
	checkBlkid()

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
	if b := initialCfg.Notifications.Beep; b != nil && b.Enabled {
		device := b.Device
		if device == "" {
			device = notify.DefaultBeepDevice
		}
		notifiers = append(notifiers, notify.NewBeepNotifier(device))
		slog.Info("beep notifications enabled", "device", device)
	}
	notifier := notify.NewMultiNotifier(notifiers...)

	// mu guards cfg; replaced atomically on config changes.
	var mu sync.RWMutex
	cfg := initialCfg

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
	}

	histLog := history.New(*historyPath)

	bus := webui.NewEventBus(32)

	// activeMounts tracks cards currently mounted, keyed by UUID. It is updated
	// by handleMount after UUID resolution and cleaned up on Unmounted events.
	// setCfg consults it to trigger imports when a pending card is activated.
	var activeMounts sync.Map // map[string]watcher.MountEvent
	var importMu sync.Mutex

	w := watcher.New(*procMounts, *pollInterval, cfg.WatchPaths...)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Wrap setCfg so that activating a pending card while it is still mounted
	// immediately starts the import without requiring a restart or re-insertion.
	origSetCfg := setCfg
	setCfg = func(newCfg *config.Config) {
		old := getCfg()
		origSetCfg(newCfg)
		for uuid, newEntry := range newCfg.Cards {
			if newEntry.Status != config.StatusActive {
				continue
			}
			if oldEntry, existed := old.Cards[uuid]; existed && oldEntry.Status != config.StatusPending {
				continue
			}
			if v, ok := activeMounts.Load(uuid); ok {
				slog.Info("card activated while mounted, triggering import", slog.String("uuid", uuid))
				go handleMount(ctx, getCfg, notifier, setCfg, bus, v.(watcher.MountEvent), *cfgPath, &activeMounts, histLog, &importMu, false)
			}
		}
	}

	triggerImport := func(uuid string) error {
		v, ok := activeMounts.Load(uuid)
		if !ok {
			return fmt.Errorf("card not mounted")
		}
		c := getCfg()
		entry, exists := c.Cards[uuid]
		if !exists || entry.Status != config.StatusActive {
			return fmt.Errorf("card not active")
		}
		if !importMu.TryLock() {
			return fmt.Errorf("import already in progress")
		}
		go func() {
			defer importMu.Unlock()
			handleMount(ctx, getCfg, notifier, setCfg, bus, v.(watcher.MountEvent), *cfgPath, &activeMounts, histLog, nil, true)
		}()
		return nil
	}

	if *webuiPort > 0 {
		acc := webui.ConfigAccessor{
			Get:  getCfg,
			Set:  setCfg,
			Path: *cfgPath,
		}
		getMounts := func() []webui.MountedVolume {
			var out []webui.MountedVolume
			activeMounts.Range(func(k, v any) bool {
				evt := v.(watcher.MountEvent)
				out = append(out, webui.MountedVolume{
					UUID:       k.(string),
					MountPoint: evt.MountPoint,
					Device:     evt.Device,
					FSType:     evt.FSType,
				})
				return true
			})
			return out
		}
		go func() {
			srv := webui.New(acc, bus, histLog, getMounts, triggerImport, *webuiPort)
			if err := srv.Start(ctx); err != nil {
				slog.Error("webui: stopped", "error", err)
			}
		}()
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	go func() {
		if err := w.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
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
			switch evt.Action {
			case watcher.Unmounted:
				activeMounts.Range(func(k, v any) bool {
					if v.(watcher.MountEvent).Device == evt.Device {
						activeMounts.Delete(k)
						bus.Publish(webui.ProgressEvent{
							Kind:       webui.ProgressKindCardRemoved,
							MountPoint: evt.MountPoint,
						})
						return false
					}
					return true
				})
			case watcher.Mounted:
				go handleMount(ctx, getCfg, notifier, setCfg, bus, evt, *cfgPath, &activeMounts, histLog, &importMu, false)
			}
		}
	}
}

func handleMount(
	ctx context.Context,
	getCfg func() *config.Config,
	n notify.Notifier,
	setCfg func(*config.Config),
	bus *webui.EventBus,
	evt watcher.MountEvent,
	cfgPath string,
	activeMounts *sync.Map,
	histLog *history.Log,
	importMu *sync.Mutex,
	forceReimport bool,
) {
	cfg := getCfg()
	slog.Info("mount detected", "mount_point", evt.MountPoint, "device", evt.Device)

	uuid, err := cardUUID(evt.Device, evt.MountPoint)
	if err != nil {
		slog.Warn("could not determine card UUID", "device", evt.Device, "error", err)
		return
	}
	activeMounts.Store(uuid, evt)
	bus.Publish(webui.ProgressEvent{
		Kind:       webui.ProgressKindCardDetected,
		CardUUID:   uuid,
		MountPoint: evt.MountPoint,
		FSType:     evt.FSType,
	})

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
		}
		if err := n.Notify(ctx, notify.Event{
			Kind:      notify.KindNewCardPending,
			CardUUID:  uuid,
			MountPath: evt.MountPoint,
			Time:      time.Now(),
			Detail:    "edit config to activate",
		}); err != nil {
			slog.Warn("notify: delivery failed", "kind", string(notify.KindNewCardPending), "error", err)
		}
		return
	}

	if importMu != nil {
		if !importMu.TryLock() {
			slog.Warn("import already in progress, skipping auto-trigger", "uuid", uuid)
			return
		}
		defer importMu.Unlock()
	}

	imp := importer.New(getCfg(), n)

	if !forceReimport && imp.AlreadyImported(evt.MountPoint) {
		slog.Info("card already imported, skipping", "uuid", uuid, "mount_point", evt.MountPoint)
		bus.Publish(webui.ProgressEvent{Kind: webui.ProgressKindCompleted, Owner: entry.Owner, CardUUID: uuid})
		return
	}

	if err := n.Notify(ctx, notify.Event{
		Kind:      notify.KindImportStarted,
		CardUUID:  uuid,
		Owner:     entry.Owner,
		CardLabel: entry.Label,
		MountPath: evt.MountPoint,
		Time:      time.Now(),
	}); err != nil {
		slog.Warn("notify: delivery failed", "kind", string(notify.KindImportStarted), "error", err)
	}

	bus.Publish(webui.ProgressEvent{Kind: webui.ProgressKindStarted, Owner: entry.Owner, CardUUID: uuid})
	imp.SetProgressCallback(func(imported, skipped, failed int, bytesCopied int64) {
		bus.Publish(webui.ProgressEvent{
			Kind:        webui.ProgressKindProgress,
			Owner:       entry.Owner,
			CardUUID:    uuid,
			Imported:    imported,
			Skipped:     skipped,
			Failed:      failed,
			BytesCopied: bytesCopied,
		})
	})
	start := time.Now()
	res, importErr := imp.Import(ctx, entry.Owner, evt.MountPoint, uuid)
	imp.SetProgressCallback(nil)
	elapsed := time.Since(start)

	if importErr != nil {
		_ = n.Notify(ctx, notify.Event{
			Kind:      notify.KindImportFailed,
			CardUUID:  uuid,
			Owner:     entry.Owner,
			CardLabel: entry.Label,
			MountPath: evt.MountPoint,
			Time:      time.Now(),
			Detail:    importErr.Error(),
		})
		bus.Publish(webui.ProgressEvent{Kind: webui.ProgressKindFailed, Owner: entry.Owner, CardUUID: uuid, Error: importErr.Error()})
		return
	}

	if err := n.Notify(ctx, notify.Event{
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
	}); err != nil {
		slog.Warn("notify: delivery failed", "kind", string(notify.KindImportCompleted), "error", err)
	}
	bus.Publish(webui.ProgressEvent{
		Kind:        webui.ProgressKindCompleted,
		Owner:       entry.Owner,
		CardUUID:    uuid,
		Total:       res.Total,
		Imported:    res.Imported,
		Skipped:     res.Skipped,
		Failed:      res.Failed,
		BytesCopied: res.BytesCopied,
	})
	_ = histLog.Append(history.Entry{
		UUID:        uuid,
		Owner:       entry.Owner,
		MountPath:   evt.MountPoint,
		StartedAt:   start,
		CompletedAt: time.Now().UTC(),
		Total:       res.Total,
		Imported:    res.Imported,
		Skipped:     res.Skipped,
		Failed:      res.Failed,
		BytesCopied: res.BytesCopied,
		HookOutput:  res.HookOutput,
	})

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

func checkBlkid() {
	if _, err := exec.LookPath("blkid"); err != nil {
		slog.Error("startup: blkid not found on PATH — card UUID detection will fail; install util-linux")
		os.Exit(1)
	}
}

func cardUUID(device, mountPoint string) (string, error) {
	// UUID covers most cards. exFAT SD cards formatted by cameras sometimes
	// have a zero Volume Serial Number, making blkid return empty. Fall back
	// through PARTUUID and LABEL before trying UUID.txt.
	for _, field := range []string{"UUID", "PARTUUID", "LABEL"} {
		v, ok := blkidField(device, field)
		if !ok {
			continue
		}
		if field != "UUID" {
			slog.Warn("cardUUID: UUID unavailable, using fallback identifier",
				slog.String("device", device),
				slog.String("field", field),
				slog.String("value", v),
			)
		}
		if field == "LABEL" {
			return "label:" + v, nil
		}
		return v, nil
	}

	// No blkid identifier found. Check for a UUID.txt file written on a
	// previous insertion, then try to create one. This is safe: it is a
	// normal filesystem write rather than a raw device write.
	uuidFile := filepath.Join(mountPoint, "UUID.txt")
	if data, err := os.ReadFile(uuidFile); err == nil {
		if uuid := strings.TrimSpace(string(data)); uuid != "" {
			slog.Info("cardUUID: using UUID from UUID.txt", slog.String("device", device), slog.String("uuid", uuid))
			return uuid, nil
		}
	}

	uuid, err := generateUUID()
	if err != nil {
		return "", fmt.Errorf("cardUUID: generate UUID: %w", err)
	}
	if err := os.WriteFile(uuidFile, []byte(uuid+"\n"), 0o644); err != nil {
		slog.Warn("cardUUID: card has no identifier and UUID.txt could not be written — card may be mounted read-only; reformat on a computer to assign a UUID",
			slog.String("device", device),
			slog.String("mount_point", mountPoint),
			slog.String("error", err.Error()),
		)
		return "", fmt.Errorf("cardUUID: no identifier for %s and UUID.txt write failed: %w", device, err)
	}
	slog.Warn("cardUUID: no identifier found; assigned new UUID via UUID.txt",
		slog.String("device", device),
		slog.String("mount_point", mountPoint),
		slog.String("uuid", uuid),
	)
	return uuid, nil
}

func generateUUID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40 // version 4
	b[8] = (b[8] & 0x3f) | 0x80 // variant bits
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

func blkidField(device, field string) (string, bool) {
	out, err := exec.Command("blkid", "-s", field, "-o", "value", device).Output()
	if err != nil {
		return "", false
	}
	v := strings.TrimSpace(string(out))
	return v, v != ""
}
