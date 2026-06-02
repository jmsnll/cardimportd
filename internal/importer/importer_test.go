package importer

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/binary"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jmsnll/cardimportd/internal/config"
	"github.com/jmsnll/cardimportd/internal/meta"
	"github.com/jmsnll/cardimportd/internal/notify"
)

func makeFile(t *testing.T, dir, name string, content []byte) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("makeFile %s: %v", name, err)
	}
	return path
}

func randomBytes(t *testing.T, n int) []byte {
	t.Helper()
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		t.Fatalf("rand.Read: %v", err)
	}
	return b
}

// discardNotifier silently drops all notifications.
type discardNotifier struct{}

func (d *discardNotifier) Notify(_ context.Context, _ notify.Event) error { return nil }

// makeConfig returns a minimal Config wired to importRoot.
func boolPtr(b bool) *bool { return &b }

func makeConfig(importRoot string) *config.Config {
	return &config.Config{
		ImportRoot: importRoot,
		FileExtensions: []string{
			".jpg", ".jpeg", ".raf", ".arw", ".mp4", ".mov",
		},
		Cards: make(map[string]config.CardEntry),
	}
}

// --- CopyVerified tests ---

func TestCopyVerified_Basic(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(t, 4096)
	src := makeFile(t, dir, "src.bin", data)
	dst := filepath.Join(dir, "dst.bin")

	n, _, err := CopyVerified(src, dst)
	if err != nil {
		t.Fatalf("CopyVerified: %v", err)
	}
	if n != int64(len(data)) {
		t.Errorf("n = %d, want %d", n, len(data))
	}
	got, _ := os.ReadFile(dst)
	if !bytes.Equal(got, data) {
		t.Error("dst content mismatch")
	}
	if _, err := os.Stat(dst + ".tmp"); !os.IsNotExist(err) {
		t.Error("tmp file still exists")
	}
}

func TestCopyVerified_MissingSrc(t *testing.T) {
	dir := t.TempDir()
	_, _, err := CopyVerified(filepath.Join(dir, "missing.bin"), filepath.Join(dir, "out.bin"))
	if err == nil {
		t.Fatal("expected error for missing src")
	}
}

// --- sameSecond / uniqueDst tests ---

func TestSameSecond(t *testing.T) {
	base := time.Date(2026, 5, 30, 14, 0, 0, 0, time.UTC)

	if !sameSecond(base, base) {
		t.Error("identical times should be sameSecond")
	}
	if !sameSecond(base, base.Add(500*time.Millisecond)) {
		t.Error("sub-second delta should be sameSecond")
	}
	if sameSecond(base, base.Add(time.Second)) {
		t.Error("1s apart should not be sameSecond")
	}
	if sameSecond(time.Time{}, base) || sameSecond(base, time.Time{}) {
		t.Error("zero time must never match")
	}
}

func TestUniqueDst(t *testing.T) {
	dir := t.TempDir()
	got, err := uniqueDst(dir, "DSCF0001.RAF")
	if err != nil {
		t.Fatalf("uniqueDst: %v", err)
	}
	if !strings.HasSuffix(got, "_2.RAF") {
		t.Errorf("uniqueDst = %q, want _2.RAF suffix", got)
	}
	makeFile(t, dir, "DSCF0001_2.RAF", []byte("x"))
	got, err = uniqueDst(dir, "DSCF0001.RAF")
	if err != nil {
		t.Fatalf("uniqueDst: %v", err)
	}
	if !strings.HasSuffix(got, "_3.RAF") {
		t.Errorf("uniqueDst after _2 exists = %q, want _3.RAF suffix", got)
	}
}

// --- checkDup tests ---

func TestCheckDup_NoExisting(t *testing.T) {
	dir := t.TempDir()
	src := makeFile(t, dir, "IMG001.jpg", randomBytes(t, 512))
	dst := filepath.Join(dir, "out", "IMG001.jpg")

	res, err := checkDup(src, dst, meta.Extract(src))
	if err != nil {
		t.Fatalf("checkDup: %v", err)
	}
	if res.action != dupCreate {
		t.Errorf("action = %v, want dupCreate", res.action)
	}
}

func TestCheckDup_SkipSameSize(t *testing.T) {
	dir := t.TempDir()
	data := randomBytes(t, 1024)
	src := makeFile(t, dir, "src.jpg", data)
	dst := makeFile(t, dir, "dst.jpg", data)

	// Force same mtime so sameSecond returns true via mtime fallback.
	now := time.Now().Truncate(time.Second)
	os.Chtimes(src, now, now)
	os.Chtimes(dst, now, now)

	res, err := checkDup(src, dst, meta.Extract(src))
	if err != nil {
		t.Fatalf("checkDup: %v", err)
	}
	if res.action != dupSkip {
		t.Errorf("action = %v, want dupSkip (same size, same mtime)", res.action)
	}
}

func TestCheckDup_ReplaceSmallerExisting(t *testing.T) {
	dir := t.TempDir()
	src := makeFile(t, dir, "src.jpg", randomBytes(t, 2048))
	dst := makeFile(t, dir, "dst.jpg", randomBytes(t, 512))

	now := time.Now().Truncate(time.Second)
	os.Chtimes(src, now, now)
	os.Chtimes(dst, now, now)

	res, err := checkDup(src, dst, meta.Extract(src))
	if err != nil {
		t.Fatalf("checkDup: %v", err)
	}
	if res.action != dupReplace {
		t.Errorf("action = %v, want dupReplace (src larger, same mtime)", res.action)
	}
}

func TestCheckDup_RenameWhenTimestampsDiffer(t *testing.T) {
	dir := t.TempDir()
	src := makeFile(t, dir, "src.jpg", randomBytes(t, 512))
	dst := makeFile(t, dir, "dst.jpg", randomBytes(t, 512))

	// Give src and dst distinct mtimes so sameSecond returns false.
	past := time.Now().Add(-10 * time.Second).Truncate(time.Second)
	now := time.Now().Truncate(time.Second)
	os.Chtimes(dst, past, past)
	os.Chtimes(src, now, now)

	res, err := checkDup(src, dst, meta.Extract(src))
	if err != nil {
		t.Fatalf("checkDup: %v", err)
	}
	if res.action != dupRename {
		t.Errorf("action = %v, want dupRename (different timestamps)", res.action)
	}
	if res.dstPath == dst {
		t.Error("dstPath should be a new unique path, not the original dst")
	}
}

// --- Import integration tests ---

func TestImport_BasicWalk(t *testing.T) {
	cardDir := t.TempDir()
	dstRoot := t.TempDir()

	makeFile(t, cardDir, "DSCF0001.jpg", randomBytes(t, 512))
	makeFile(t, cardDir, "DSCF0002.jpg", randomBytes(t, 512))
	makeFile(t, cardDir, "readme.txt", []byte("ignore"))

	subDir := filepath.Join(cardDir, "DCIM", "100FUJI")
	os.MkdirAll(subDir, 0o755)
	makeFile(t, subDir, "DSCF0003.RAF", randomBytes(t, 1024))

	imp := New(makeConfig(dstRoot), &discardNotifier{})
	res, err := imp.Import(context.Background(), "James", cardDir, "TEST-UUID")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	if res.Failed != 0 {
		t.Errorf("Failed = %d, want 0", res.Failed)
	}
}

func TestImport_SkipUnsupportedExtensions(t *testing.T) {
	cardDir := t.TempDir()
	dstRoot := t.TempDir()
	makeFile(t, cardDir, "notes.txt", []byte("ignore"))
	makeFile(t, cardDir, "photo.jpg", randomBytes(t, 256))

	imp := New(makeConfig(dstRoot), &discardNotifier{})
	res, err := imp.Import(context.Background(), "James", cardDir, "TEST-UUID")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Total != 1 {
		t.Errorf("Total = %d, want 1", res.Total)
	}
}

func TestImport_ManifestWritten(t *testing.T) {
	cardDir, dstRoot := t.TempDir(), t.TempDir()
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 512))
	makeFile(t, cardDir, "b.jpg", randomBytes(t, 512))
	cfg := makeConfig(dstRoot)
	cfg.WriteManifest = boolPtr(true)
	res, err := New(cfg, &discardNotifier{}).Import(context.Background(), "James", cardDir, "")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Imported != 2 {
		t.Fatalf("Imported = %d", res.Imported)
	}
	mPath := filepath.Join(dstRoot, "James's Library", "cardimportd-manifest.txt")
	data, err := os.ReadFile(mPath)
	if err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	if lines := strings.Split(strings.TrimSpace(string(data)), "\n"); len(lines) != 2 {
		t.Errorf("manifest lines = %d, want 2", len(lines))
	}
}

func TestImport_ManifestDisabled(t *testing.T) {
	cardDir, dstRoot := t.TempDir(), t.TempDir()
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 512))
	cfg := makeConfig(dstRoot)
	cfg.WriteManifest = boolPtr(false)
	New(cfg, &discardNotifier{}).Import(context.Background(), "James", cardDir, "")
	if _, err := os.Stat(filepath.Join(dstRoot, "James's Library", "cardimportd-manifest.txt")); !os.IsNotExist(err) {
		t.Error("manifest written when disabled")
	}
}

func TestImport_CustomTemplate(t *testing.T) {
	cardDir, dstRoot := t.TempDir(), t.TempDir()
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 512))
	cfg := makeConfig(dstRoot)
	cfg.DestinationTemplate = "{{ .Owner }}/shots/{{ .Year }}"
	res, err := New(cfg, &discardNotifier{}).Import(context.Background(), "James", cardDir, "UUID")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Imported != 1 {
		t.Fatalf("Imported = %d", res.Imported)
	}
	if entries, err := os.ReadDir(filepath.Join(dstRoot, "James", "shots")); err != nil || len(entries) == 0 {
		t.Error("custom template path not created")
	}
}

func TestImport_MirrorCopied(t *testing.T) {
	cardDir, dstRoot, mirrorRoot := t.TempDir(), t.TempDir(), t.TempDir()
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 512))
	cfg := makeConfig(dstRoot)
	cfg.MirrorRoot = mirrorRoot
	res, err := New(cfg, &discardNotifier{}).Import(context.Background(), "James", cardDir, "")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.MirrorFailed != 0 {
		t.Errorf("MirrorFailed = %d, want 0", res.MirrorFailed)
	}
	var found bool
	filepath.WalkDir(mirrorRoot, func(p string, d os.DirEntry, _ error) error {
		if !d.IsDir() {
			found = true
		}
		return nil
	})
	if !found {
		t.Error("file not in mirror")
	}
}

func TestImport_MirrorFailDoesNotAbortPrimary(t *testing.T) {
	cardDir, dstRoot := t.TempDir(), t.TempDir()
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 512))
	cfg := makeConfig(dstRoot)
	// Use a regular file as a path component so MkdirAll fails even as root.
	blockFile := filepath.Join(t.TempDir(), "block")
	if err := os.WriteFile(blockFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg.MirrorRoot = filepath.Join(blockFile, "mirror")
	res, err := New(cfg, &discardNotifier{}).Import(context.Background(), "James", cardDir, "")
	if err != nil {
		t.Fatalf("Import failed: %v", err)
	}
	if res.Imported != 1 {
		t.Errorf("Imported = %d, want 1", res.Imported)
	}
	if res.MirrorFailed != 1 {
		t.Errorf("MirrorFailed = %d, want 1", res.MirrorFailed)
	}
}

func TestImport_SecondRunSkipsDuplicates(t *testing.T) {
	cardDir := t.TempDir()
	dstRoot := t.TempDir()
	makeFile(t, cardDir, "DSCF0001.jpg", randomBytes(t, 512))

	cfg := makeConfig(dstRoot)
	imp := New(cfg, &discardNotifier{})

	res1, err := imp.Import(context.Background(), "James", cardDir, "TEST-UUID")
	if err != nil {
		t.Fatalf("first Import: %v", err)
	}
	if res1.Imported != 1 {
		t.Fatalf("first run: Imported = %d, want 1", res1.Imported)
	}

	res2, err := imp.Import(context.Background(), "James", cardDir, "TEST-UUID")
	if err != nil {
		t.Fatalf("second Import: %v", err)
	}
	if res2.Failed != 0 {
		t.Errorf("second run: Failed = %d, want 0", res2.Failed)
	}
	if res2.Total != 1 {
		t.Errorf("second run: Total = %d, want 1", res2.Total)
	}
}

// buildJPEGWithRating builds a minimal JPEG/EXIF stream with IFD0 entries for
// Model (0x0110), Rating (0x4746 SHORT), ExifIFD pointer (0x8769), and
// DateTimeOriginal (0x9003). Used to create rated test images.
//
// IFD0 layout: count(2) + 3×12 entries + next-IFD(4) = 42 bytes
//   ifd0Off=8  exifIFDOff=50  valAreaOff=68
func buildJPEGWithRating(model, dto string, rating uint16) []byte {
	const (
		ifd0Off    = 8
		exifIFDOff = 50
		valAreaOff = 68
	)
	writeTIFFEntry := func(buf []byte, tag, typ uint16, count, val uint32) {
		binary.LittleEndian.PutUint16(buf[0:2], tag)
		binary.LittleEndian.PutUint16(buf[2:4], typ)
		binary.LittleEndian.PutUint32(buf[4:8], count)
		binary.LittleEndian.PutUint32(buf[8:12], val)
	}
	modelB := []byte(model)
	dtoB := []byte(dto)
	tiff := make([]byte, valAreaOff+len(modelB)+len(dtoB))

	copy(tiff[0:2], "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 0x002A)
	binary.LittleEndian.PutUint32(tiff[4:8], ifd0Off)

	binary.LittleEndian.PutUint16(tiff[ifd0Off:], 3)
	writeTIFFEntry(tiff[ifd0Off+2:], 0x0110, 2, uint32(len(modelB)), uint32(valAreaOff))
	writeTIFFEntry(tiff[ifd0Off+14:], 0x4746, 3, 1, uint32(rating))
	writeTIFFEntry(tiff[ifd0Off+26:], 0x8769, 4, 1, uint32(exifIFDOff))
	binary.LittleEndian.PutUint32(tiff[ifd0Off+38:], 0)

	dtoOff := uint32(valAreaOff + len(modelB))
	binary.LittleEndian.PutUint16(tiff[exifIFDOff:], 1)
	writeTIFFEntry(tiff[exifIFDOff+2:], 0x9003, 2, uint32(len(dtoB)), dtoOff)
	binary.LittleEndian.PutUint32(tiff[exifIFDOff+14:], 0)

	copy(tiff[valAreaOff:], modelB)
	copy(tiff[valAreaOff+len(modelB):], dtoB)

	exifPfx := []byte("Exif\x00\x00")
	app1Body := make([]byte, 0, len(exifPfx)+len(tiff))
	app1Body = append(app1Body, exifPfx...)
	app1Body = append(app1Body, tiff...)
	app1Len := uint16(len(app1Body) + 2)
	var j bytes.Buffer
	j.Write([]byte{0xFF, 0xD8, 0xFF, 0xE1})
	j.WriteByte(byte(app1Len >> 8))
	j.WriteByte(byte(app1Len))
	j.Write(app1Body)
	j.Write([]byte{0xFF, 0xD9})
	return j.Bytes()
}

func TestImport_RatedOnly(t *testing.T) {
	cardDir := t.TempDir()
	dstRoot := t.TempDir()

	// Unrated image — random bytes, EXIF extraction fails so Rating=0.
	makeFile(t, cardDir, "unrated.jpg", randomBytes(t, 512))
	// Rated image — valid EXIF with Rating=3.
	makeFile(t, cardDir, "rated.jpg", buildJPEGWithRating("Fuji X100VI\x00", "2025:06:01 10:00:00\x00", 3))
	// Video — always imported regardless of rated_only.
	makeFile(t, cardDir, "clip.mp4", randomBytes(t, 512))

	cfg := makeConfig(dstRoot)
	cfg.RatedOnly = true

	res, err := New(cfg, &discardNotifier{}).Import(context.Background(), "James", cardDir, "")
	if err != nil {
		t.Fatalf("Import: %v", err)
	}
	if res.Total != 3 {
		t.Errorf("Total = %d, want 3", res.Total)
	}
	if res.Imported != 2 {
		t.Errorf("Imported = %d, want 2 (rated image + video)", res.Imported)
	}
	if res.Skipped != 1 {
		t.Errorf("Skipped = %d, want 1 (unrated image)", res.Skipped)
	}
}

func TestImport_HookCalled(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	cardDir, dstRoot := t.TempDir(), t.TempDir()
	marker := filepath.Join(t.TempDir(), "ran")
	makeFile(t, cardDir, "a.jpg", randomBytes(t, 512))
	cfg := makeConfig(dstRoot)
	cfg.PostImportHook = "touch " + marker
	if _, err := New(cfg, &discardNotifier{}).Import(context.Background(), "J", cardDir, ""); err != nil {
		t.Fatalf("Import: %v", err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Error("hook not called")
	}
}
