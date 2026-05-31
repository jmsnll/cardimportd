package importer

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/meta"
	"github.com/jmsnll/cardimportd/internal/notify"
)

// Result holds aggregate counters for a completed import run.
type Result struct {
	Total        int
	Imported     int
	Skipped      int
	Failed       int
	MirrorFailed int
	BytesCopied  int64
}

const progressInterval = 10

// Importer orchestrates the import of all media files from a mounted card.
type Importer struct {
	cfg        *config.Config
	notifier   notify.Notifier
	onProgress func(imported, skipped, failed int, bytesCopied int64)
}

// New returns an Importer wired to cfg and notifier.
func New(cfg *config.Config, notifier notify.Notifier) *Importer {
	return &Importer{cfg: cfg, notifier: notifier}
}

// SetProgressCallback registers a function called every progressInterval files.
func (imp *Importer) SetProgressCallback(fn func(imported, skipped, failed int, bytesCopied int64)) {
	imp.onProgress = fn
}

// Import walks mountPath for supported files and imports them to the
// per-owner destination under cfg.ImportRoot.
func (imp *Importer) Import(ctx context.Context, owner, mountPath, cardUUID string) (Result, error) {
	var res Result
	var entries []manifestEntry

	if err := preflight(mountPath, imp.cfg.ImportRoot, imp.cfg.MinFreeGB); err != nil {
		return Result{}, fmt.Errorf("preflight: %w", err)
	}

	cardEntry, _ := imp.cfg.LookupCard(cardUUID)
	tmplStr := cardEntry.DestinationTemplate
	if tmplStr == "" {
		tmplStr = imp.cfg.DestinationTemplate
	}
	if tmplStr == "" {
		tmplStr = defaultDestTemplate
	}
	destTmpl, err := template.New("dest").Parse(tmplStr)
	if err != nil {
		return Result{}, fmt.Errorf("destination template: %w", err)
	}

	ext := make(map[string]bool, len(imp.cfg.FileExtensions))
	for _, e := range imp.cfg.FileExtensions {
		ext[strings.ToLower(e)] = true
	}

	err = filepath.WalkDir(mountPath, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			slog.Warn("importer: walk error", "path", path, "error", walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if !ext[strings.ToLower(filepath.Ext(path))] {
			return nil
		}

		res.Total++
		n, hash, dstPath, action, mirrFail, err := imp.importFile(ctx, owner, path, cardUUID, destTmpl)
		if err != nil {
			slog.Error("importer: failed", "src", path, "error", err)
			res.Failed++
			return nil
		}
		if mirrFail {
			res.MirrorFailed++
		}
		switch action {
		case dupSkip:
			res.Skipped++
		default:
			res.Imported++
			res.BytesCopied += n
			if imp.cfg.WriteManifest && hash != "" {
				if rel, relErr := filepath.Rel(imp.cfg.ImportRoot, dstPath); relErr == nil {
					entries = append(entries, manifestEntry{hash: hash, relPath: rel})
				}
			}
		}
		if imp.onProgress != nil && res.Total%progressInterval == 0 {
			imp.onProgress(res.Imported, res.Skipped, res.Failed, res.BytesCopied)
		}
		return nil
	})

	if err == nil && imp.cfg.WriteManifest && len(entries) > 0 {
		mPath := manifestFilePath(imp.cfg.ImportRoot, owner)
		if mErr := writeManifest(mPath, entries); mErr != nil {
			slog.Warn("importer: manifest write failed", "error", mErr)
		}
	}

	if err == nil && imp.cfg.PostImportHook != "" {
		if hookErr := runHook(imp.cfg.PostImportHook, owner, mountPath, res); hookErr != nil {
			slog.Warn("post-import hook failed", "error", hookErr)
		}
	}
	return res, err
}

func (imp *Importer) importFile(ctx context.Context, owner, srcPath, cardUUID string, destTmpl *template.Template) (n int64, hash string, dstPath string, action dupAction, mirrorFailed bool, err error) {
	m := meta.Extract(srcPath)

	dstDir, err := resolveDestDir(imp.cfg.ImportRoot, destTmpl, owner, cardUUID, m.CameraModel, m.DateTimeOriginal)
	if err != nil {
		return 0, "", "", 0, false, fmt.Errorf("resolve dest: %w", err)
	}
	naiveDst := filepath.Join(dstDir, filepath.Base(srcPath))

	dup, err := checkDup(srcPath, naiveDst, m)
	if err != nil {
		return 0, "", "", 0, false, fmt.Errorf("dedup check: %w", err)
	}

	if dup.action == dupSkip {
		slog.Debug("importer: skip (duplicate)", "src", srcPath, "dst", dup.dstPath)
		return 0, "", dup.dstPath, dupSkip, false, nil
	}

	if err := os.MkdirAll(filepath.Dir(dup.dstPath), 0o755); err != nil {
		return 0, "", "", 0, false, fmt.Errorf("mkdir: %w", err)
	}

	n, hash, err = copyVerified(srcPath, dup.dstPath)
	if err != nil {
		return 0, "", "", 0, false, err
	}

	slog.Info("importer: imported",
		"src", srcPath,
		"dst", dup.dstPath,
		"bytes", n,
		"action", dup.action,
		"meta_source", m.Source,
	)
	var mirrFail bool
	if imp.cfg.MirrorRoot != "" {
		if rel, relErr := filepath.Rel(imp.cfg.ImportRoot, dup.dstPath); relErr == nil {
			mirrorDst := filepath.Join(imp.cfg.MirrorRoot, rel)
			if mkErr := os.MkdirAll(filepath.Dir(mirrorDst), 0o755); mkErr != nil {
				slog.Warn("importer: mirror mkdir failed", "error", mkErr)
				mirrFail = true
			} else if _, _, cpErr := copyVerified(srcPath, mirrorDst); cpErr != nil {
				slog.Warn("importer: mirror copy failed", "error", cpErr)
				mirrFail = true
			}
		} else {
			slog.Warn("importer: mirror rel path failed", "error", relErr)
			mirrFail = true
		}
	}
	return n, hash, dup.dstPath, dup.action, mirrFail, nil
}
