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
	".xmp",
}

func main() {
	src     := flag.String("src", "", "source directory to scan recursively (required)")
	dst     := flag.String("dst", "", "destination root; files land at {dst}/YYYY/MM/DD/ (required)")
	dryRun  := flag.Bool("dry-run", false, "print planned operations without executing")
	verbose := flag.Bool("verbose", false, "print each file operation even when not in dry-run")
	workers := flag.Int("workers", runtime.NumCPU(), "number of parallel copy workers")
	extFlag := flag.String("ext", "", "comma-separated extensions to include (default: same set as cardimportd)")
	flag.Parse()

	if *src == "" || *dst == "" {
		fmt.Fprintln(os.Stderr, "photomigrate: -src and -dst are required")
		flag.Usage()
		os.Exit(1)
	}

	srcReal := realPath(*src)
	dstReal := realPath(*dst)
	sep := string(os.PathSeparator)
	if srcReal == dstReal ||
		strings.HasPrefix(dstReal, srcReal+sep) ||
		strings.HasPrefix(srcReal, dstReal+sep) {
		fmt.Fprintln(os.Stderr, "photomigrate: -src and -dst must not overlap")
		os.Exit(1)
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	exts := buildExtSet(*extFlag)

	var (
		total      atomic.Int64
		moved      atomic.Int64
		skipped    atomic.Int64
		failed     atomic.Int64
		bytesMoved atomic.Int64
	)

	work := make(chan string, *workers*4)

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

				op, finalDst, err := resolveAction(path, naiveDst)
				if err != nil {
					slog.Error("could not resolve destination", "src", path, "error", err)
					failed.Add(1)
					continue
				}

				if *dryRun || *verbose {
					switch op {
					case actionSkip:
						fmt.Printf("SKIP  %s\n      (already at %s)\n", path, finalDst)
					case actionRename:
						fmt.Printf("MOVE  %s\n   -> %s  (renamed to avoid collision)\n", path, finalDst)
					default:
						fmt.Printf("MOVE  %s\n   -> %s\n", path, finalDst)
					}
				}

				if *dryRun {
					if op == actionSkip {
						skipped.Add(1)
					} else {
						moved.Add(1)
					}
					continue
				}

				switch op {
				case actionSkip:
					if err := os.Remove(path); err != nil {
						slog.Warn("remove source failed (already at dst)", "src", path, "error", err)
					}
					skipped.Add(1)

				default:
					if err := os.MkdirAll(filepath.Dir(finalDst), 0o755); err != nil {
						slog.Error("mkdir failed", "dir", filepath.Dir(finalDst), "error", err)
						failed.Add(1)
						continue
					}
					n, _, copyErr := importer.CopyVerified(path, finalDst)
					if copyErr != nil {
						slog.Error("copy failed", "src", path, "dst", finalDst, "error", copyErr)
						failed.Add(1)
						continue
					}
					if err := os.Remove(path); err != nil {
						slog.Warn("remove source failed after copy", "src", path, "error", err)
					}
					moved.Add(1)
					bytesMoved.Add(n)
				}
			}
		})
	}

	err := filepath.WalkDir(*src, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			slog.Warn("walk error", "path", path, "error", walkErr)
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if !exts[strings.ToLower(filepath.Ext(path))] {
			return nil
		}
		total.Add(1)
		work <- path
		return nil
	})

	close(work)
	wg.Wait()

	if err != nil {
		slog.Error("directory walk failed", "error", err)
		os.Exit(1)
	}

	fmt.Printf("\ntotal=%d  moved=%d  skipped=%d  failed=%d  bytes=%d\n",
		total.Load(), moved.Load(), skipped.Load(), failed.Load(), bytesMoved.Load())
	if *dryRun {
		fmt.Println("(dry-run: no files were modified)")
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
