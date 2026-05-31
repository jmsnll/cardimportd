// Package watcher polls /proc/mounts at a fixed interval and emits MountEvents
// whenever a USB mount point (prefix /volumeUSB) appears or disappears.
package watcher

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"
)

// Action describes whether a USB volume was mounted or unmounted.
type Action int

const (
	Mounted   Action = iota
	Unmounted
)

func (a Action) String() string {
	switch a {
	case Mounted:
		return "mounted"
	case Unmounted:
		return "unmounted"
	default:
		return "unknown"
	}
}

// MountEvent is emitted each time a /volumeUSB mount point appears or disappears.
type MountEvent struct {
	MountPoint string
	Device     string
	FSType     string
	Action     Action
}

type mountEntry struct {
	device     string
	mountPoint string
	fsType     string
}

// usbMountPrefix is the fallback prefix used when no watchPrefixes are configured.
const usbMountPrefix = "/volumeUSB"

// Watcher polls a mounts file and streams MountEvents for USB volumes.
type Watcher struct {
	procMountsPath string
	interval       time.Duration
	watchPrefixes  []string
	events         chan MountEvent
}

// New creates a Watcher that polls procMountsPath every interval.
// watchPrefixes is the list of mount-point prefixes to monitor; if empty it
// defaults to ["/volumeUSB"]. Pass "/proc/mounts" and 2*time.Second in production.
func New(procMountsPath string, interval time.Duration, watchPrefixes ...string) *Watcher {
	if len(watchPrefixes) == 0 {
		watchPrefixes = []string{usbMountPrefix}
	}
	return &Watcher{
		procMountsPath: procMountsPath,
		interval:       interval,
		watchPrefixes:  watchPrefixes,
		events:         make(chan MountEvent, 8),
	}
}

// Events returns the read-only channel of mount events.
func (w *Watcher) Events() <-chan MountEvent {
	return w.events
}

// maxBackoff is the upper bound on the exponential backoff delay.
const maxBackoff = 30 * time.Second

// Start begins polling. Blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	var snapshot map[string]mountEntry
	first := true
	consecutiveErrors := 0

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			current, err := w.readUSBMounts()
			if err != nil {
				consecutiveErrors++
				backoff := w.interval * (1 << consecutiveErrors)
				if backoff > maxBackoff {
					backoff = maxBackoff
				}
				slog.Warn("watcher: failed to read mounts file",
					slog.String("path", w.procMountsPath),
					slog.String("error", err.Error()),
				)
				if backoff > w.interval {
					slog.Warn("watcher: backing off due to repeated errors",
						slog.Duration("backoff", backoff),
					)
				}
				select {
				case <-ctx.Done():
					return ctx.Err()
				case <-time.After(backoff):
				}
				continue
			}

			consecutiveErrors = 0

			if first {
				snapshot = current
				first = false
				slog.Debug("watcher: initial snapshot populated",
					slog.Int("usb_mounts", len(snapshot)),
				)
				continue
			}

			w.diff(snapshot, current)
			snapshot = current
		}
	}
}

func (w *Watcher) readUSBMounts() (map[string]mountEntry, error) {
	f, err := os.Open(w.procMountsPath)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", w.procMountsPath, err)
	}
	defer f.Close()

	result := make(map[string]mountEntry)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		entry, ok := parseMountsLine(scanner.Text())
		if !ok {
			continue
		}
		if !w.isWatched(entry.mountPoint) {
			continue
		}
		result[entry.mountPoint] = entry
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", w.procMountsPath, err)
	}
	return result, nil
}

func (w *Watcher) diff(prev, curr map[string]mountEntry) {
	for mp, entry := range curr {
		if _, existed := prev[mp]; !existed {
			slog.Debug("watcher: usb volume mounted",
				slog.String("mount_point", mp),
				slog.String("device", entry.device),
			)
			w.send(MountEvent{MountPoint: mp, Device: entry.device, FSType: entry.fsType, Action: Mounted})
		}
	}
	for mp, entry := range prev {
		if _, still := curr[mp]; !still {
			slog.Debug("watcher: usb volume unmounted",
				slog.String("mount_point", mp),
				slog.String("device", entry.device),
			)
			w.send(MountEvent{MountPoint: mp, Device: entry.device, FSType: entry.fsType, Action: Unmounted})
		}
	}
}

func (w *Watcher) isWatched(mp string) bool {
	for _, prefix := range w.watchPrefixes {
		if strings.HasPrefix(mp, prefix) {
			return true
		}
	}
	return false
}

func (w *Watcher) send(evt MountEvent) {
	select {
	case w.events <- evt:
	default:
		slog.Warn("watcher: events channel full, dropping event",
			slog.String("mount_point", evt.MountPoint),
		)
	}
}

func parseMountsLine(line string) (mountEntry, bool) {
	line = strings.TrimSpace(line)
	if line == "" || strings.HasPrefix(line, "#") {
		return mountEntry{}, false
	}
	fields := strings.Fields(line)
	if len(fields) < 3 {
		return mountEntry{}, false
	}
	return mountEntry{device: fields[0], mountPoint: fields[1], fsType: fields[2]}, true
}
