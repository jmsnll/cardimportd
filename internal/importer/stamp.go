package importer

import (
	"encoding/json"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stampFileName = ".cardimportd"

type importStamp struct {
	ImportedAt time.Time `json:"imported_at"`
	CardUUID   string    `json:"card_uuid"`
	Owner      string    `json:"owner"`
	FileCount  int       `json:"file_count"`
	RatedOnly  bool      `json:"rated_only"`
}

func readStamp(mountPath string) (importStamp, bool) {
	data, err := os.ReadFile(filepath.Join(mountPath, stampFileName))
	if err != nil {
		return importStamp{}, false
	}
	var s importStamp
	if err := json.Unmarshal(data, &s); err != nil {
		return importStamp{}, false
	}
	return s, true
}

func writeStamp(mountPath, cardUUID, owner string, fileCount int, ratedOnly bool) error {
	s := importStamp{
		ImportedAt: time.Now().UTC(),
		CardUUID:   cardUUID,
		Owner:      owner,
		FileCount:  fileCount,
		RatedOnly:  ratedOnly,
	}
	data, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(mountPath, stampFileName), data, 0o644)
}

// countMatchingFiles walks mountPath counting files whose extension is in ext,
// using the same logic as the Import walk so the count is always comparable.
func countMatchingFiles(mountPath string, ext map[string]bool) int {
	var n int
	if err := filepath.WalkDir(mountPath, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}
		if ext[strings.ToLower(filepath.Ext(path))] {
			n++
		}
		return nil
	}); err != nil {
		slog.Warn("importer: stamp walk error", "path", mountPath, "error", err)
	}
	return n
}
