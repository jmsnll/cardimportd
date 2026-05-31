package importer

import (
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"
	"syscall"
)

func preflight(mountPath, importRoot string, minFreeGB float64) error {
	sourceSize, err := dirSize(mountPath)
	if err != nil {
		slog.Warn("preflight: cannot estimate source size, skipping", "error", err)
		return nil
	}
	var stat syscall.Statfs_t
	if err := syscall.Statfs(importRoot, &stat); err != nil {
		slog.Warn("preflight: statfs failed, skipping", "error", err)
		return nil
	}
	available := stat.Bavail * uint64(stat.Bsize)
	required := uint64(float64(sourceSize) * 1.1)
	if minFreeGB > 0 {
		if floor := uint64(minFreeGB * (1 << 30)); floor > required {
			required = floor
		}
	}
	slog.Info("preflight: disk space check",
		"source_bytes", sourceSize, "available_bytes", available, "required_bytes", required)
	if available < required {
		return fmt.Errorf("insufficient disk space: need %d bytes, have %d on %s", required, available, importRoot)
	}
	return nil
}

func dirSize(path string) (int64, error) {
	var total int64
	err := filepath.WalkDir(path, func(_ string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if info, err := d.Info(); err == nil {
			total += info.Size()
		}
		return nil
	})
	return total, err
}
