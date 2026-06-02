package watcher

import (
	"context"
	"os"
	"testing"
	"time"
)

const testInterval = 10 * time.Millisecond

func writeMounts(t *testing.T, path string, lines []string) {
	t.Helper()
	var b []byte
	for _, l := range lines {
		b = append(b, []byte(l+"\n")...)
	}
	if err := os.WriteFile(path, b, 0o600); err != nil {
		t.Fatalf("writeMounts: %v", err)
	}
}

func tempMounts(t *testing.T, lines []string) string {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "proc_mounts_*")
	if err != nil {
		t.Fatalf("tempMounts: %v", err)
	}
	path := f.Name()
	f.Close()
	writeMounts(t, path, lines)
	return path
}

func drainEvents(t *testing.T, ch <-chan MountEvent, timeout time.Duration) []MountEvent {
	t.Helper()
	deadline := time.NewTimer(timeout)
	defer deadline.Stop()
	var evts []MountEvent
	for {
		select {
		case e := <-ch:
			evts = append(evts, e)
		case <-deadline.C:
			return evts
		}
	}
}

func startWatcher(t *testing.T, w *Watcher) context.CancelFunc {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Errorf("watcher.Start returned unexpected error: %v", err)
		}
	}()
	return cancel
}

func waitForTick() { time.Sleep(5 * testInterval) }

func TestInitialSnapshotEmitsMountedEvents(t *testing.T) {
	path := tempMounts(t, []string{"/dev/sdb1 /volumeUSB1/usbshare1 exfat rw 0 0"})
	w := New(path, testInterval)
	cancel := startWatcher(t, w)
	defer cancel()
	waitForTick()
	evts := drainEvents(t, w.Events(), 2*testInterval)
	if len(evts) != 1 {
		t.Fatalf("expected 1 Mounted event for already-mounted volume, got %d: %v", len(evts), evts)
	}
	if evts[0].Action != Mounted {
		t.Errorf("expected Action=Mounted, got %v", evts[0].Action)
	}
	if evts[0].MountPoint != "/volumeUSB1/usbshare1" {
		t.Errorf("unexpected MountPoint: %q", evts[0].MountPoint)
	}
}

func TestMountedEventEmittedWhenUSBLineAdded(t *testing.T) {
	path := tempMounts(t, []string{"sysfs /sys sysfs rw 0 0"})
	w := New(path, testInterval)
	cancel := startWatcher(t, w)
	defer cancel()
	waitForTick()
	writeMounts(t, path, []string{
		"sysfs /sys sysfs rw 0 0",
		"/dev/sdb1 /volumeUSB1/usbshare1 exfat rw 0 0",
	})
	evts := drainEvents(t, w.Events(), 10*testInterval)
	if len(evts) != 1 {
		t.Fatalf("expected 1 Mounted event, got %d: %v", len(evts), evts)
	}
	e := evts[0]
	if e.Action != Mounted {
		t.Errorf("expected Action=Mounted, got %v", e.Action)
	}
	if e.MountPoint != "/volumeUSB1/usbshare1" {
		t.Errorf("unexpected MountPoint: %q", e.MountPoint)
	}
	if e.Device != "/dev/sdb1" {
		t.Errorf("unexpected Device: %q", e.Device)
	}
	if e.FSType != "exfat" {
		t.Errorf("unexpected FSType: %q", e.FSType)
	}
}

func TestUnmountedEventEmittedWhenUSBLineRemoved(t *testing.T) {
	path := tempMounts(t, []string{"/dev/sdb1 /volumeUSB1/usbshare1 exfat rw 0 0"})
	w := New(path, testInterval)
	cancel := startWatcher(t, w)
	defer cancel()
	// Drain the Mounted event emitted for the already-present card before
	// testing the removal path.
	waitForTick()
	drainEvents(t, w.Events(), 2*testInterval)
	// Remove the card; the next poll should emit an Unmounted event.
	writeMounts(t, path, []string{"sysfs /sys sysfs rw 0 0"})
	evts := drainEvents(t, w.Events(), 10*testInterval)
	if len(evts) != 1 {
		t.Fatalf("expected 1 Unmounted event, got %d: %v", len(evts), evts)
	}
	e := evts[0]
	if e.Action != Unmounted {
		t.Errorf("expected Action=Unmounted, got %v", e.Action)
	}
}

func TestNonUSBMountsAreIgnored(t *testing.T) {
	path := tempMounts(t, []string{"sysfs /sys sysfs rw 0 0"})
	w := New(path, testInterval)
	cancel := startWatcher(t, w)
	defer cancel()
	waitForTick()
	writeMounts(t, path, []string{
		"sysfs /sys sysfs rw 0 0",
		"/dev/sda1 /volume1 ext4 rw 0 0",
	})
	evts := drainEvents(t, w.Events(), 10*testInterval)
	if len(evts) != 0 {
		t.Errorf("expected no events for non-USB mounts, got %d: %v", len(evts), evts)
	}
}

func TestParseMountsLine(t *testing.T) {
	tests := []struct {
		name   string
		line   string
		want   mountEntry
		wantOK bool
	}{
		{
			name:   "standard line",
			line:   "/dev/sdb1 /volumeUSB1/usbshare1 exfat rw,relatime 0 0",
			want:   mountEntry{device: "/dev/sdb1", mountPoint: "/volumeUSB1/usbshare1", fsType: "exfat"},
			wantOK: true,
		},
		{name: "blank line", line: "", wantOK: false},
		{name: "comment line", line: "# comment", wantOK: false},
		{name: "too few fields", line: "/dev/sdb1 /mnt", wantOK: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseMountsLine(tc.line)
			if ok != tc.wantOK {
				t.Fatalf("parseMountsLine(%q): ok=%v, want %v", tc.line, ok, tc.wantOK)
			}
			if ok && got != tc.want {
				t.Errorf("parseMountsLine(%q) = %+v, want %+v", tc.line, got, tc.want)
			}
		})
	}
}
