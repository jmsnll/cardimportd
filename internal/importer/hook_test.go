package importer

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestRunHook_Empty(t *testing.T) {
	if err := runHook("", "J", "/m", Result{}); err != nil {
		t.Errorf("empty hook: %v", err)
	}
}

func TestRunHook_Success(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	marker := filepath.Join(t.TempDir(), "ran")
	if err := runHook("touch "+marker, "J", "/m", Result{}); err != nil {
		t.Fatalf("runHook: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("hook did not run")
	}
}

func TestRunHook_EnvVars(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	out := filepath.Join(t.TempDir(), "out.txt")
	cmd := `printf '%s %d %d' "$CARDIMPORTD_OWNER" "$CARDIMPORTD_FILES_IMPORTED" "$CARDIMPORTD_BYTES_COPIED" > ` + out
	if err := runHook(cmd, "Alice", "/m", Result{Imported: 7, BytesCopied: 2048}); err != nil {
		t.Fatalf("runHook: %v", err)
	}
	data, _ := os.ReadFile(out)
	for _, want := range []string{"Alice", "7", "2048"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("output %q missing %q", data, want)
		}
	}
}

func TestRunHook_NonZeroExit(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	if err := runHook("exit 1", "J", "/m", Result{}); err == nil {
		t.Error("expected error for exit 1")
	}
}
