// photomigrate reorganises an existing photo/video library into the cardimportd
// destination layout: {dst}/YYYY/MM/DD/{filename}.
//
// It uses EXIF DateTimeOriginal (with video container and mtime fallbacks) to
// determine the date, then performs a verified copy (SHA-256) followed by removal
// of the source file.
//
// Usage:
//
//	photomigrate -src /old/library -dst /new/library [-dry-run] [-verbose] [-workers N] [-ext .jpg,.raf,...]
package main

import (
	"crypto/sha256"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/log"
	"github.com/jmsnll/cardimportd/internal/importer"
	"github.com/jmsnll/cardimportd/internal/meta"
)

var defaultExts = []string{
	".jpg", ".jpeg",
	".raf",
	".arw", ".lrf",
	".cr3", ".cr2",
	".nef", ".nrw",
	".dng",
	".orf",
	".rw2",
	".heic", ".heif",
	".mp4", ".mov", ".mxf",
	".wav", ".aif",
}

// findXMPSidecar returns the path of the XMP sidecar for rawPath if one exists
// in the same directory (checks both .xmp and .XMP), or empty string otherwise.
func findXMPSidecar(rawPath string) string {
	stem := strings.TrimSuffix(rawPath, filepath.Ext(rawPath))
	for _, e := range []string{".xmp", ".XMP"} {
		if _, err := os.Stat(stem + e); err == nil {
			return stem + e
		}
	}
	return ""
}

func main() {
	src     := flag.String("src", "", "source directory to scan recursively (required)")
	dst     := flag.String("dst", "", "destination root; files land at {dst}/YYYY/MM/DD/ (required)")
	dryRun  := flag.Bool("dry-run", false, "print planned operations without executing")
	verbose := flag.Bool("verbose", false, "log each file operation")
	workers := flag.Int("workers", runtime.NumCPU(), "number of parallel copy workers")
	extFlag := flag.String("ext", "", "comma-separated extensions to include (default: same set as cardimportd)")
	flag.Parse()

	if *src == "" || *dst == "" {
		fmt.Fprintln(os.Stderr, "photomigrate: -src and -dst are required")
		flag.Usage()
		os.Exit(1)
	}

	logLevel := log.InfoLevel
	if *verbose {
		logLevel = log.DebugLevel
	}
	logger := log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: false,
		Level:           logLevel,
	})
	slog.SetDefault(slog.New(logger))

	srcReal := realPath(*src)
	dstReal := realPath(*dst)
	sep := string(os.PathSeparator)
	if srcReal == dstReal ||
		strings.HasPrefix(dstReal, srcReal+sep) ||
		strings.HasPrefix(srcReal, dstReal+sep) {
		logger.Fatal("-src and -dst must not overlap")
	}

	exts := buildExtSet(*extFlag)

	// ── Phase 1: scan ───────────────────────────────────────────────────────

	logger.Info("Scanning", "src", *src)

	var files []string
	walkErr := filepath.WalkDir(*src, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			logger.Warn("walk error", "path", path, "error", err)
			return nil
		}
		if d.IsDir() {
			if strings.HasPrefix(d.Name(), "@") {
				return filepath.SkipDir
			}
			return nil
		}
		if !exts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if walkErr != nil {
		logger.Fatal("directory walk failed", "error", walkErr)
	}

	total := len(files)
	logger.Info("Scan complete", "files", total)

	if total == 0 {
		logger.Info("Nothing to migrate")
		return
	}

	// ── Phase 2: process ────────────────────────────────────────────────────

	var (
		processed  atomic.Int64
		moved      atomic.Int64
		skipped    atomic.Int64
		failed     atomic.Int64
		bytesMoved atomic.Int64
	)

	work := make(chan string, *workers*4)
	start := time.Now()

	// Progress line — suppressed in verbose/dry-run since per-file logs already give visibility.
	var stopProgress chan struct{}
	if !*verbose && !*dryRun {
		stopProgress = make(chan struct{})
		go func() {
			ticker := time.NewTicker(200 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-ticker.C:
					n := processed.Load()
					pct := float64(n) / float64(total) * 100
					elapsed := time.Since(start).Seconds()
					rate := 0.0
					if elapsed > 0 {
						rate = float64(n) / elapsed
					}
					eta := ""
					if rate > 0 {
						remaining := time.Duration(float64(int64(total)-n)/rate) * time.Second
						eta = fmt.Sprintf("  eta %s", remaining.Round(time.Second))
					}
					fmt.Fprintf(os.Stderr, "\r  %5.1f%%  %d/%d  %.0f files/s%s  moved=%d  skipped=%d  failed=%d   ",
						pct, n, total, rate, eta,
						moved.Load(), skipped.Load(), failed.Load())
				case <-stopProgress:
					fmt.Fprintln(os.Stderr)
					return
				}
			}
		}()
	}

	var resolveMu sync.Mutex
	var wg sync.WaitGroup
	for range *workers {
		wg.Go(func() {
			for path := range work {
				m := meta.Extract(path)
				dstDir := filepath.Join(*dst,
					m.DateTimeOriginal.Format("2006"),
					m.DateTimeOriginal.Format("01"),
					m.DateTimeOriginal.Format("02"),
				)
				naiveDst := filepath.Join(dstDir, filepath.Base(path))

				resolveMu.Lock()
				op, finalDst, resolveErr := resolveAction(path, naiveDst)
				resolveMu.Unlock()
				if resolveErr != nil {
					logger.Error("could not resolve destination", "src", path, "error", resolveErr)
					failed.Add(1)
					processed.Add(1)
					continue
				}

				// Resolve XMP sidecar destination upfront using the same dstDir,
				// before the raw is moved, so the source file still exists.
				sidecar := findXMPSidecar(path)
				var sidecarOp action
				var finalSidecarDst string
				if sidecar != "" {
					naiveSidecarDst := filepath.Join(dstDir, filepath.Base(sidecar))
					resolveMu.Lock()
					var sidecarErr error
					sidecarOp, finalSidecarDst, sidecarErr = resolveAction(sidecar, naiveSidecarDst)
					resolveMu.Unlock()
					if sidecarErr != nil {
						logger.Warn("could not resolve sidecar destination", "src", sidecar, "error", sidecarErr)
						sidecar = ""
					}
				}

				if *verbose {
					switch op {
					case actionSkip:
						logger.Debug("skip", "src", path, "dst", finalDst)
					case actionRename:
						logger.Debug("move", "src", path, "dst", finalDst, "note", "renamed to avoid collision")
					default:
						logger.Debug("move", "src", path, "dst", finalDst)
					}
					if sidecar != "" {
						if sidecarOp == actionSkip {
							logger.Debug("skip sidecar", "src", sidecar, "dst", finalSidecarDst)
						} else {
							logger.Debug("move sidecar", "src", sidecar, "dst", finalSidecarDst)
						}
					}
				}

				if *dryRun {
					if op == actionSkip {
						skipped.Add(1)
					} else {
						moved.Add(1)
					}
					if sidecar != "" {
						if sidecarOp == actionSkip {
							skipped.Add(1)
						} else {
							moved.Add(1)
						}
					}
					processed.Add(1)
					continue
				}

				primaryOK := false
				switch op {
				case actionSkip:
					if err := os.Remove(path); err != nil {
						logger.Warn("remove source failed", "src", path, "error", err)
					}
					skipped.Add(1)
					primaryOK = true

				default:
					if err := os.MkdirAll(filepath.Dir(finalDst), 0o755); err != nil {
						logger.Error("mkdir failed", "dir", filepath.Dir(finalDst), "error", err)
						failed.Add(1)
						processed.Add(1)
						continue
					}
					n, _, copyErr := importer.CopyVerified(path, finalDst)
					if copyErr != nil {
						logger.Error("copy failed", "src", path, "dst", finalDst, "error", copyErr)
						failed.Add(1)
						processed.Add(1)
						continue
					}
					if err := os.Remove(path); err != nil {
						logger.Warn("remove source failed after copy", "src", path, "error", err)
					}
					moved.Add(1)
					bytesMoved.Add(n)
					primaryOK = true
				}

				if primaryOK && sidecar != "" {
					switch sidecarOp {
					case actionSkip:
						if err := os.Remove(sidecar); err != nil {
							logger.Warn("remove sidecar source failed", "src", sidecar, "error", err)
						}
						skipped.Add(1)
					default:
						if err := os.MkdirAll(filepath.Dir(finalSidecarDst), 0o755); err != nil {
							logger.Error("mkdir failed for sidecar", "dir", filepath.Dir(finalSidecarDst), "error", err)
							failed.Add(1)
						} else {
							n, _, copyErr := importer.CopyVerified(sidecar, finalSidecarDst)
							if copyErr != nil {
								logger.Error("copy sidecar failed", "src", sidecar, "dst", finalSidecarDst, "error", copyErr)
								failed.Add(1)
							} else {
								if err := os.Remove(sidecar); err != nil {
									logger.Warn("remove sidecar source failed after copy", "src", sidecar, "error", err)
								}
								moved.Add(1)
								bytesMoved.Add(n)
							}
						}
					}
				}

				processed.Add(1)
			}
		})
	}

	for _, path := range files {
		work <- path
	}
	close(work)
	wg.Wait()

	if stopProgress != nil {
		close(stopProgress)
	}

	elapsed := time.Since(start).Round(time.Second)
	gb := float64(bytesMoved.Load()) / (1 << 30)
	logger.Info("Complete",
		"moved", moved.Load(),
		"skipped", skipped.Load(),
		"failed", failed.Load(),
		"data", fmt.Sprintf("%.2f GB", gb),
		"elapsed", elapsed,
	)
	if *dryRun {
		logger.Info("Dry run — no files were modified")
	}
}

type action int

const (
	actionCreate action = iota
	actionSkip
	actionRename
)

// resolveAction decides what to do with a source file given its naive destination.
// - actionCreate  destination does not exist
// - actionSkip    destination exists with identical content (SHA-256 + size match)
// - actionRename  destination exists but content differs; finalDst is a unique path
func resolveAction(srcPath, dstPath string) (action, string, error) {
	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return 0, "", fmt.Errorf("stat src: %w", err)
	}

	dstInfo, err := os.Stat(dstPath)
	if errors.Is(err, os.ErrNotExist) {
		return actionCreate, dstPath, nil
	}
	if err != nil {
		return 0, "", fmt.Errorf("stat dst: %w", err)
	}

	if srcInfo.Size() == dstInfo.Size() {
		same, err := sameContent(srcPath, dstPath)
		if err != nil {
			return 0, "", fmt.Errorf("content compare: %w", err)
		}
		if same {
			return actionSkip, dstPath, nil
		}
	}

	ext  := filepath.Ext(dstPath)
	stem := strings.TrimSuffix(dstPath, ext)
	for i := 2; i <= 999; i++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return actionRename, candidate, nil
		}
	}
	return 0, "", fmt.Errorf("could not find unique destination for %q after 998 attempts", filepath.Base(dstPath))
}

func sameContent(a, b string) (bool, error) {
	ha, err := hashFile(a)
	if err != nil {
		return false, err
	}
	hb, err := hashFile(b)
	if err != nil {
		return false, err
	}
	return ha == hb, nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer func() { _ = f.Close() }()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

func realPath(p string) string {
	abs, err := filepath.Abs(p)
	if err != nil {
		return p
	}
	real, err := filepath.EvalSymlinks(abs)
	if err != nil {
		return abs
	}
	return real
}

func buildExtSet(extFlag string) map[string]bool {
	m := make(map[string]bool)
	if extFlag != "" {
		for e := range strings.SplitSeq(extFlag, ",") {
			e = strings.TrimSpace(e)
			if e != "" {
				m[strings.ToLower(e)] = true
			}
		}
		return m
	}
	for _, e := range defaultExts {
		m[e] = true
	}
	return m
}
