package importer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

type manifestEntry struct{ hash, relPath string }

func writeManifest(path string, entries []manifestEntry) error {
	if len(entries) == 0 {
		return nil
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("manifest: %w", err)
	}
	w := bufio.NewWriter(f)
	for _, e := range entries {
		_, _ = fmt.Fprintf(w, "%s  %s\n", e.hash, e.relPath)
	}
	flushErr := w.Flush()
	closeErr := f.Close()
	if flushErr != nil {
		return flushErr
	}
	return closeErr
}

func manifestFilePath(importRoot, owner string) string {
	return filepath.Join(importRoot, owner+"'s Library", "cardimportd-manifest.txt")
}
