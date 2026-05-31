package system_test

// Black-box system tests for cardimportd.
//
// TestMain builds the real binary once. Each test spins up a daemon process
// with isolated temp dirs, a fake blkid script, and a writable proc/mounts
// file, then observes the filesystem and HTTP API for side effects.

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// -----------------------------------------------------------------------
// TestMain – build once
// -----------------------------------------------------------------------

var binPath string

func TestMain(m *testing.M) {
	tmp, err := os.MkdirTemp("", "cardimportd-bin-*")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(tmp)

	binPath = filepath.Join(tmp, "cardimportd")
	cmd := exec.Command("go", "build", "-o", binPath,
		"../../cmd/cardimportd")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "build failed: %v\n", err)
		os.Exit(1)
	}
	os.Exit(m.Run())
}

// -----------------------------------------------------------------------
// harness – one isolated daemon per test
// -----------------------------------------------------------------------

type harness struct {
	t          *testing.T
	binDir     string // contains fake blkid
	mountPoint string // source dir; written into fake proc/mounts
	destDir    string
	mountsFile string
	cfgFile    string
	webuiPort  int
	proc       *exec.Cmd
	done       chan struct{}
}

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("freePort: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}

// newHarness sets up temp dirs and config. cardRegistered controls whether the
// card UUID starts as active (true) or pending (false) in the config.
func newHarness(t *testing.T, uuid string, cardRegistered bool) *harness {
	t.Helper()

	tmp := t.TempDir()
	mountPoint := filepath.Join(tmp, "mount")
	destDir := filepath.Join(tmp, "dest")
	for _, d := range []string{mountPoint, destDir} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}

	// Fake blkid: always echoes uuid regardless of device argument.
	binDir := filepath.Join(tmp, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir binDir: %v", err)
	}
	blkid := filepath.Join(binDir, "blkid")
	if err := os.WriteFile(blkid,
		[]byte("#!/bin/sh\necho '"+uuid+"'\n"), 0o755); err != nil {
		t.Fatalf("write blkid: %v", err)
	}

	// Empty proc/mounts file – tests append to it to simulate insertions.
	mountsFile := filepath.Join(tmp, "proc_mounts")
	if err := os.WriteFile(mountsFile, nil, 0o644); err != nil {
		t.Fatalf("write mounts: %v", err)
	}

	status := "pending"
	if cardRegistered {
		status = "active"
	}
	cfgFile := filepath.Join(tmp, "config.yaml")
	cfgYAML := fmt.Sprintf(`watch_paths:
  - %s
import_root: %s
cards:
  "%s":
    owner: TestOwner
    status: %s
file_extensions:
  - .jpg
  - .jpeg
log_path: %s
`, mountPoint, destDir, uuid, status, filepath.Join(tmp, "daemon.log"))
	if err := os.WriteFile(cfgFile, []byte(cfgYAML), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	return &harness{
		t:          t,
		binDir:     binDir,
		mountPoint: mountPoint,
		destDir:    destDir,
		mountsFile: mountsFile,
		cfgFile:    cfgFile,
		webuiPort:  freePort(t),
		done:       make(chan struct{}),
	}
}

func (h *harness) start() {
	h.t.Helper()

	h.proc = exec.Command(binPath,
		"--config", h.cfgFile,
		"--webui-port", fmt.Sprintf("%d", h.webuiPort),
		"--proc-mounts", h.mountsFile,
		"--poll-interval", "100ms",
	)
	h.proc.Env = append(os.Environ(),
		"PATH="+h.binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	h.proc.Stdout = os.Stderr
	h.proc.Stderr = os.Stderr

	if err := h.proc.Start(); err != nil {
		h.t.Fatalf("start daemon: %v", err)
	}
	go func() {
		h.proc.Wait()
		close(h.done)
	}()

	h.t.Cleanup(h.stop)

	if !waitFor(5*time.Second, func() bool {
		resp, err := http.Get(h.apiURL("/api/config"))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return resp.StatusCode == http.StatusOK
	}) {
		h.t.Fatal("daemon webui did not become ready")
	}
	// Give the watcher time to fire its initial snapshot tick (poll=100ms)
	// before any test appends a mount line. Without this the mount line lands
	// in the snapshot and never produces a Mounted event.
	time.Sleep(300 * time.Millisecond)
}

func (h *harness) stop() {
	if h.proc != nil && h.proc.Process != nil {
		h.proc.Process.Signal(os.Interrupt)
		select {
		case <-h.done:
		case <-time.After(3 * time.Second):
			h.proc.Process.Kill()
		}
	}
}

// triggerMount appends a mount line for mountPoint to the fake proc/mounts.
func (h *harness) triggerMount() {
	h.t.Helper()
	line := fmt.Sprintf("/dev/sdtest %s vfat rw 0 0\n", h.mountPoint)
	f, err := os.OpenFile(h.mountsFile, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		h.t.Fatalf("open mounts: %v", err)
	}
	defer f.Close()
	if _, err := f.WriteString(line); err != nil {
		h.t.Fatalf("write mount line: %v", err)
	}
}

func (h *harness) apiURL(path string) string {
	return fmt.Sprintf("http://127.0.0.1:%d%s", h.webuiPort, path)
}

func (h *harness) getJSON(path string, dst any) {
	h.t.Helper()
	resp, err := http.Get(h.apiURL(path))
	if err != nil {
		h.t.Fatalf("GET %s: %v", path, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		h.t.Fatalf("GET %s: status %d", path, resp.StatusCode)
	}
	if err := json.NewDecoder(resp.Body).Decode(dst); err != nil {
		h.t.Fatalf("GET %s decode: %v", path, err)
	}
}

func (h *harness) postJSON(path string, body any) int {
	h.t.Helper()
	data, _ := json.Marshal(body)
	resp, err := http.Post(h.apiURL(path), "application/json",
		bytes.NewReader(data))
	if err != nil {
		h.t.Fatalf("POST %s: %v", path, err)
	}
	resp.Body.Close()
	return resp.StatusCode
}

// -----------------------------------------------------------------------
// helpers
// -----------------------------------------------------------------------

func waitFor(timeout time.Duration, check func() bool) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if check() {
			return true
		}
		time.Sleep(100 * time.Millisecond)
	}
	return false
}

// testJPEG is a 104-byte minimal JPEG with:
//   DateTimeOriginal = 2024:03:15 10:30:00  (local)
//   Model            = FujiFilm X-T5
var testJPEG = []byte{
	0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x62, 0x45, 0x78, 0x69, 0x66, 0x00, 0x00,
	0x49, 0x49, 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00, 0x02, 0x00, 0x10, 0x01,
	0x02, 0x00, 0x0E, 0x00, 0x00, 0x00, 0x26, 0x00, 0x00, 0x00, 0x69, 0x87,
	0x04, 0x00, 0x01, 0x00, 0x00, 0x00, 0x34, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x46, 0x75, 0x6A, 0x69, 0x46, 0x69, 0x6C, 0x6D, 0x20, 0x58,
	0x2D, 0x54, 0x35, 0x00, 0x01, 0x00, 0x03, 0x90, 0x02, 0x00, 0x14, 0x00,
	0x00, 0x00, 0x46, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x32, 0x30,
	0x32, 0x34, 0x3A, 0x30, 0x33, 0x3A, 0x31, 0x35, 0x20, 0x31, 0x30, 0x3A,
	0x33, 0x30, 0x3A, 0x30, 0x30, 0x00, 0xFF, 0xD9,
}

func writeTestJPEG(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), testJPEG, 0o644); err != nil {
		t.Fatalf("write JPEG: %v", err)
	}
}

func sha256sum(t *testing.T, path string) string {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("sha256 open %s: %v", path, err)
	}
	defer f.Close()
	h := sha256.New()
	io.Copy(h, f)
	return fmt.Sprintf("%x", h.Sum(nil))
}

// -----------------------------------------------------------------------
// tests
// -----------------------------------------------------------------------

// TestImportHappyPath inserts a card with one JPEG, waits for the import to
// complete, then asserts the file landed under YYYY/MM/DD with intact content.
func TestImportHappyPath(t *testing.T) {
	const uuid = "AABB-CCDD"
	h := newHarness(t, uuid, true)
	writeTestJPEG(t, h.mountPoint, "DSCF0001.JPG")
	h.start()
	h.triggerMount()

	wantRel := filepath.Join("TestOwner's Library", "2024", "03", "15", "DSCF0001.JPG")
	wantPath := filepath.Join(h.destDir, wantRel)

	if !waitFor(10*time.Second, func() bool {
		_, err := os.Stat(wantPath)
		return err == nil
	}) {
		t.Fatalf("file never appeared at %s", wantPath)
	}

	if sha256sum(t, wantPath) != sha256sum(t, filepath.Join(h.mountPoint, "DSCF0001.JPG")) {
		t.Error("destination SHA-256 does not match source")
	}
}

// TestImportMultipleFiles verifies all files in a card directory are imported.
func TestImportMultipleFiles(t *testing.T) {
	const uuid = "1122-3344"
	h := newHarness(t, uuid, true)
	writeTestJPEG(t, h.mountPoint, "DSCF0001.JPG")
	writeTestJPEG(t, h.mountPoint, "DSCF0002.JPG")
	writeTestJPEG(t, h.mountPoint, "DSCF0003.JPG")
	h.start()
	h.triggerMount()

	base := filepath.Join(h.destDir, "TestOwner's Library", "2024", "03", "15")
	if !waitFor(10*time.Second, func() bool {
		entries, err := os.ReadDir(base)
		return err == nil && len(entries) == 3
	}) {
		entries, _ := os.ReadDir(base)
		t.Fatalf("expected 3 files in dest, got %d", len(entries))
	}
}

// TestPendingCard verifies that an unregistered card is written to the config
// as pending and the import is skipped.
func TestPendingCard(t *testing.T) {
	const uuid = "UNKN-OWNN"
	// Config has no cards registered.
	tmp := t.TempDir()
	mountPoint := filepath.Join(tmp, "mount")
	destDir := filepath.Join(tmp, "dest")
	os.MkdirAll(mountPoint, 0o755)
	os.MkdirAll(destDir, 0o755)

	binDir := filepath.Join(tmp, "bin")
	os.MkdirAll(binDir, 0o755)
	os.WriteFile(filepath.Join(binDir, "blkid"),
		[]byte("#!/bin/sh\necho '"+uuid+"'\n"), 0o755)

	mountsFile := filepath.Join(tmp, "proc_mounts")
	os.WriteFile(mountsFile, nil, 0o644)

	cfgFile := filepath.Join(tmp, "config.yaml")
	cfgYAML := fmt.Sprintf(`watch_paths:
  - %s
import_root: %s
cards: {}
file_extensions:
  - .jpg
log_path: %s
`, mountPoint, destDir, filepath.Join(tmp, "daemon.log"))
	os.WriteFile(cfgFile, []byte(cfgYAML), 0o644)

	port := freePort(t)
	proc := exec.Command(binPath,
		"--config", cfgFile,
		"--webui-port", fmt.Sprintf("%d", port),
		"--proc-mounts", mountsFile,
		"--poll-interval", "100ms",
	)
	proc.Env = append(os.Environ(),
		"PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	proc.Stdout = os.Stderr
	proc.Stderr = os.Stderr
	if err := proc.Start(); err != nil {
		t.Fatalf("start: %v", err)
	}
	done := make(chan struct{})
	go func() { proc.Wait(); close(done) }()
	t.Cleanup(func() {
		proc.Process.Signal(os.Interrupt)
		select {
		case <-done:
		case <-time.After(3 * time.Second):
			proc.Process.Kill()
		}
	})

	// Wait for webui ready.
	webuiReady := waitFor(5*time.Second, func() bool {
		resp, err := http.Get(fmt.Sprintf("http://127.0.0.1:%d/api/config", port))
		if err != nil {
			return false
		}
		resp.Body.Close()
		return true
	})
	if !webuiReady {
		t.Fatal("webui never ready")
	}
	time.Sleep(300 * time.Millisecond) // let watcher take initial snapshot

	// Trigger mount with unknown card.
	line := fmt.Sprintf("/dev/sdtest %s vfat rw 0 0\n", mountPoint)
	f, _ := os.OpenFile(mountsFile, os.O_APPEND|os.O_WRONLY, 0o644)
	f.WriteString(line)
	f.Close()

	// Wait for config to be updated with the pending entry.
	apiURL := fmt.Sprintf("http://127.0.0.1:%d/api/cards", port)
	if !waitFor(5*time.Second, func() bool {
		resp, err := http.Get(apiURL)
		if err != nil {
			return false
		}
		defer resp.Body.Close()
		var cards map[string]map[string]any
		json.NewDecoder(resp.Body).Decode(&cards)
		entry, ok := cards[uuid]
		return ok && entry["status"] == "pending"
	}) {
		t.Fatalf("card %s never appeared as pending in /api/cards", uuid)
	}

	// Nothing should have been imported.
	entries, _ := os.ReadDir(destDir)
	if len(entries) != 0 {
		t.Errorf("expected dest to be empty for pending card, got %d entries", len(entries))
	}
}

// TestDedup verifies that a file already present at the destination is not
// overwritten when the same card is re-inserted.
func TestDedup(t *testing.T) {
	const uuid = "DEDU-PKEY"
	h := newHarness(t, uuid, true)
	writeTestJPEG(t, h.mountPoint, "DSCF0001.JPG")
	h.start()

	// First import.
	h.triggerMount()
	wantPath := filepath.Join(h.destDir, "TestOwner's Library", "2024", "03", "15", "DSCF0001.JPG")
	if !waitFor(10*time.Second, func() bool {
		_, err := os.Stat(wantPath)
		return err == nil
	}) {
		t.Fatal("first import: file never appeared")
	}

	// Record mtime of imported file.
	info1, _ := os.Stat(wantPath)

	// Remove mount line, then re-add to simulate re-insertion.
	os.WriteFile(h.mountsFile, nil, 0o644)
	time.Sleep(300 * time.Millisecond)
	h.triggerMount()
	time.Sleep(3 * time.Second)

	// File should still exist with the same mtime (not re-copied).
	info2, err := os.Stat(wantPath)
	if err != nil {
		t.Fatalf("file disappeared after second mount: %v", err)
	}
	if !info2.ModTime().Equal(info1.ModTime()) {
		t.Error("file mtime changed: file was unexpectedly re-copied")
	}
}

// -----------------------------------------------------------------------
// webui API tests
// -----------------------------------------------------------------------

// TestWebuiGetConfig verifies GET /api/config returns the current config.
func TestWebuiGetConfig(t *testing.T) {
	const uuid = "WEBU-CFG1"
	h := newHarness(t, uuid, true)
	h.start()

	var cfg map[string]any
	h.getJSON("/api/config", &cfg)

	if cfg["import_root"] == nil {
		t.Error("import_root missing from /api/config response")
	}
	if cfg["cards"] == nil {
		t.Error("cards missing from /api/config response")
	}
}

// TestWebuiGetCards verifies GET /api/cards returns the card list.
func TestWebuiGetCards(t *testing.T) {
	const uuid = "WEBU-CARD"
	h := newHarness(t, uuid, true)
	h.start()

	var cards map[string]any
	h.getJSON("/api/cards", &cards)

	if _, ok := cards[uuid]; !ok {
		t.Errorf("expected card %s in response, got keys: %v", uuid, cards)
	}
}

// TestWebuiUpdateCard verifies POST /api/cards/{uuid} activates a pending card.
func TestWebuiUpdateCard(t *testing.T) {
	const uuid = "WEBU-UPD1"
	h := newHarness(t, uuid, false) // starts as pending
	h.start()

	status := h.postJSON("/api/cards/"+uuid,
		map[string]string{"owner": "Alice", "status": "active"})
	if status != http.StatusOK {
		t.Fatalf("POST /api/cards/%s: want 200, got %d", uuid, status)
	}

	// Verify the change is reflected immediately.
	var cards map[string]map[string]string
	h.getJSON("/api/cards", &cards)
	entry, ok := cards[uuid]
	if !ok {
		t.Fatalf("card %s missing after update", uuid)
	}
	if entry["status"] != "active" {
		t.Errorf("status: want active, got %q", entry["status"])
	}
	if entry["owner"] != "Alice" {
		t.Errorf("owner: want Alice, got %q", entry["owner"])
	}
}

// TestManualImportTrigger verifies POST /api/cards/{uuid}/import triggers a re-import.
func TestManualImportTrigger(t *testing.T) {
	const uuid = "MANU-0001"
	h := newHarness(t, uuid, true)
	writeTestJPEG(t, h.mountPoint, "DSCF0001.JPG")
	h.start()
	h.triggerMount()
	wantPath := filepath.Join(h.destDir, "TestOwner's Library", "2024", "03", "15", "DSCF0001.JPG")
	if !waitFor(10*time.Second, func() bool { _, err := os.Stat(wantPath); return err == nil }) {
		t.Fatal("initial import never completed")
	}
	os.Remove(wantPath)
	status := h.postJSON("/api/cards/"+uuid+"/import", map[string]string{"mount_path": h.mountPoint})
	if status != http.StatusAccepted {
		t.Fatalf("POST trigger: want 202, got %d", status)
	}
	if !waitFor(10*time.Second, func() bool { _, err := os.Stat(wantPath); return err == nil }) {
		t.Fatal("manual re-import never completed")
	}
}

// TestWebuiDeleteCard verifies DELETE /api/cards/{uuid} removes a card.
func TestWebuiDeleteCard(t *testing.T) {
	const uuid = "WEBU-DEL1"
	h := newHarness(t, uuid, true)
	h.start()

	// Confirm it exists first.
	var before map[string]any
	h.getJSON("/api/cards", &before)
	if _, ok := before[uuid]; !ok {
		t.Fatalf("card %s not present before delete", uuid)
	}

	resp, err := http.NewRequest(http.MethodDelete, h.apiURL("/api/cards/"+uuid), nil)
	if err != nil {
		t.Fatalf("build DELETE request: %v", err)
	}
	result, err := http.DefaultClient.Do(resp)
	if err != nil {
		t.Fatalf("DELETE /api/cards/%s: %v", uuid, err)
	}
	result.Body.Close()
	if result.StatusCode != http.StatusOK {
		t.Fatalf("DELETE: want 200, got %d", result.StatusCode)
	}

	var after map[string]any
	h.getJSON("/api/cards", &after)
	if _, ok := after[uuid]; ok {
		t.Errorf("card %s still present after delete", uuid)
	}
}
