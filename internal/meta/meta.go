// Package meta extracts DateTimeOriginal and CameraModel from image and video
// files. It is used by the importer to determine the destination path and the
// dedup key.
//
// Fallback chain: EXIF -> video container (mvhd) -> file mtime.
// Extract never returns an error; worst-case Source is SourceMtime.
package meta
import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	exif "github.com/dsoprea/go-exif/v3"
	exifcommon "github.com/dsoprea/go-exif/v3/common"
)

// exifDateFormat is the EXIF datetime string layout defined by the EXIF spec.
const exifDateFormat = "2006:01:02 15:04:05"

// Source describes how DateTimeOriginal was obtained.
type Source int

const (
	SourceEXIF Source = iota
	SourceVideoContainer
	SourceMtime
)

// FileMeta holds the extracted metadata for a single file.
type FileMeta struct {
	DateTimeOriginal time.Time
	CameraModel      string
	Source           Source
}

// Extract returns FileMeta for the file at path.
// It tries EXIF first, then the video container header, then file mtime.
// It never returns an error.
func Extract(path string) FileMeta {
	ext := strings.ToLower(filepath.Ext(path))

	switch ext {
	case ".raf":
		if m, ok := extractRAFMeta(path); ok {
			return m
		}
		slog.Debug("meta: RAF EXIF extraction failed, falling back", "path", path)

	case ".jpg", ".jpeg", ".arw", ".tif", ".tiff":
		if m, ok := extractEXIFFromFile(path); ok {
			return m
		}
		slog.Debug("meta: EXIF extraction failed, falling back", "path", path, "ext", ext)

	case ".cr3", ".cr2", ".nef", ".nrw", ".dng", ".orf", ".rw2":
		if m, ok := extractEXIFFromFile(path); ok {
			return m
		}
		slog.Debug("meta: EXIF extraction failed, falling back", "path", path, "ext", ext)

	case ".heic", ".heif":
		if m, ok := extractEXIFFromFile(path); ok {
			return m
		}
		slog.Debug("meta: HEIF EXIF extraction failed, falling back", "path", path)

	case ".mp4", ".mov":
		if m, ok := extractVideoMeta(path); ok {
			return m
		}
		slog.Debug("meta: video container extraction failed, falling back", "path", path, "ext", ext)
	}

	return mtimeFallback(path)
}

// extractEXIFFromFile opens path and delegates EXIF parsing to extractEXIFFromBytes.
func extractEXIFFromFile(path string) (FileMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return FileMeta{}, false
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return FileMeta{}, false
	}

	return extractEXIFFromBytes(data)
}

// extractEXIFFromBytes parses EXIF from raw image bytes.
// It reads:
//   - Model (IFD0, tag 0x0110) -> FileMeta.CameraModel
//   - DateTimeOriginal (IFD/Exif, tag 0x9003) -> FileMeta.DateTimeOriginal
func extractEXIFFromBytes(data []byte) (FileMeta, bool) {
	rawExif, err := exif.SearchAndExtractExif(data)
	if err != nil {
		return FileMeta{}, false
	}

	im, err := exifcommon.NewIfdMappingWithStandard()
	if err != nil {
		return FileMeta{}, false
	}

	ti := exif.NewTagIndex()

	_, index, err := exif.Collect(im, ti, rawExif)
	if err != nil {
		return FileMeta{}, false
	}

	rootIfd := index.RootIfd

	model := readStringTag(rootIfd, 0x0110)

	exifIfd, err := rootIfd.ChildWithIfdPath(exifcommon.IfdExifStandardIfdIdentity)
	if err != nil {
		return FileMeta{}, false
	}

	dtoRaw := readStringTag(exifIfd, 0x9003)
	if dtoRaw == "" {
		return FileMeta{}, false
	}

	dt, err := time.ParseInLocation(exifDateFormat, dtoRaw, time.Local)
	if err != nil {
		return FileMeta{}, false
	}

	return FileMeta{
		DateTimeOriginal: dt,
		CameraModel:      model,
		Source:           SourceEXIF,
	}, true
}

// readStringTag returns the string value of the first matching tag in ifd, or
// the empty string if the tag is absent or not a string type.
func readStringTag(ifd *exif.Ifd, tagID uint16) string {
	entries, err := ifd.FindTagWithId(tagID)
	if err != nil || len(entries) == 0 {
		return ""
	}

	v, err := entries[0].Value()
	if err != nil {
		return ""
	}

	s, ok := v.(string)
	if !ok {
		return ""
	}

	return strings.TrimRight(s, "\x00")
}

// exifFromReader is used by raf.go to parse EXIF from an in-memory JPEG.
// It reads all bytes from r and delegates to extractEXIFFromBytes.
func exifFromReader(r *bytes.Reader) (FileMeta, bool) {
	data := make([]byte, r.Len())
	if _, err := io.ReadFull(r, data); err != nil {
		return FileMeta{}, false
	}

	return extractEXIFFromBytes(data)
}

// mtimeFallback returns a FileMeta whose DateTimeOriginal is the file mtime.
func mtimeFallback(path string) FileMeta {
	info, err := os.Stat(path)
	if err != nil {
		slog.Warn("meta: stat failed for mtime fallback", "path", path, "err", err)
		return FileMeta{
			DateTimeOriginal: time.Now(),
			Source:           SourceMtime,
		}
	}

	return FileMeta{
		DateTimeOriginal: info.ModTime(),
		Source:           SourceMtime,
	}
}
