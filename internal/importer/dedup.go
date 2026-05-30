package importer

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jmsnll/cardimportd/internal/meta"
)

type dupAction int

const (
	dupCreate  dupAction = iota
	dupSkip
	dupReplace
	dupRename
)

type dedupResult struct {
	action  dupAction
	dstPath string
}

func checkDup(srcPath, dstPath string, srcMeta meta.FileMeta) (dedupResult, error) {
	info, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
		return dedupResult{action: dupCreate, dstPath: dstPath}, nil
	}
	if err != nil {
		return dedupResult{}, fmt.Errorf("stat %q: %w", dstPath, err)
	}

	existingMeta := meta.Extract(dstPath)

	if sameSecond(srcMeta.DateTimeOriginal, existingMeta.DateTimeOriginal) {
		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			return dedupResult{}, fmt.Errorf("stat src %q: %w", srcPath, err)
		}
		if srcInfo.Size() > info.Size() {
			return dedupResult{action: dupReplace, dstPath: dstPath}, nil
		}
		return dedupResult{action: dupSkip, dstPath: dstPath}, nil
	}

	unique := uniqueDst(filepath.Dir(dstPath), filepath.Base(dstPath))
	return dedupResult{action: dupRename, dstPath: unique}, nil
}

func sameSecond(a, b time.Time) bool {
	if a.IsZero() || b.IsZero() {
		return false
	}
	return a.Truncate(time.Second).Equal(b.Truncate(time.Second))
}

func uniqueDst(dir, filename string) string {
	ext := filepath.Ext(filename)
	stem := strings.TrimSuffix(filename, ext)
	for i := 2; ; i++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s_%d%s", stem, i, ext))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}
