package meta

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
	"testing"
	"time"
)

// writeBox encodes a box with a 4-byte big-endian size, 4-byte type, and payload.
func writeBox(w io.Writer, boxType string, payload []byte) {
	size := uint32(8 + len(payload))
	var hdr [8]byte
	binary.BigEndian.PutUint32(hdr[0:4], size)
	copy(hdr[4:8], []byte(boxType))
	_, _ = w.Write(hdr[:])
	_, _ = w.Write(payload)
}

// buildMvhdV0 returns a 100-byte mvhd version-0 payload with the given creation_time.
func buildMvhdV0(creationTime uint32) []byte {
	payload := make([]byte, 100)
	payload[0] = 0
	binary.BigEndian.PutUint32(payload[4:8], creationTime)
	return payload
}

// buildMvhdV1 returns a 112-byte mvhd version-1 payload with the given creation_time.
func buildMvhdV1(creationTime uint64) []byte {
	payload := make([]byte, 112)
	payload[0] = 1
	binary.BigEndian.PutUint64(payload[4:12], creationTime)
	return payload
}

// buildMP4 wraps an mvhd payload in a moov box.
func buildMP4(mvhdPayload []byte) []byte {
	var mvhd bytes.Buffer
	writeBox(&mvhd, "mvhd", mvhdPayload)
	var moov bytes.Buffer
	writeBox(&moov, "moov", mvhd.Bytes())
	return moov.Bytes()
}

func TestExtractVideoMeta_Version0(t *testing.T) {
	refUnix := time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC).Unix()
	refMac := uint32(uint64(refUnix) + macEpochOffset)
	data := buildMP4(buildMvhdV0(refMac))
	got, ok := parseMoovMvhd(bytes.NewReader(data))
	if !ok {
		t.Fatal("parseMoovMvhd returned false, want true")
	}
	want := time.Unix(refUnix, 0).UTC()
	if !got.DateTimeOriginal.Equal(want) {
		t.Errorf("DateTimeOriginal = %v, want %v", got.DateTimeOriginal, want)
	}
	if got.Source != SourceVideoContainer {
		t.Errorf("Source = %v, want SourceVideoContainer", got.Source)
	}
}

func TestExtractVideoMeta_Version1(t *testing.T) {
	refUnix := time.Date(2019, 12, 31, 23, 59, 59, 0, time.UTC).Unix()
	refMac := uint64(refUnix) + macEpochOffset
	data := buildMP4(buildMvhdV1(refMac))
	got, ok := parseMoovMvhd(bytes.NewReader(data))
	if !ok {
		t.Fatal("parseMoovMvhd returned false, want true")
	}
	want := time.Unix(refUnix, 0).UTC()
	if !got.DateTimeOriginal.Equal(want) {
		t.Errorf("DateTimeOriginal = %v, want %v", got.DateTimeOriginal, want)
	}
}

func TestExtractVideoMeta_NoMoov(t *testing.T) {
	var b bytes.Buffer
	writeBox(&b, "ftyp", make([]byte, 8))
	_, ok := parseMoovMvhd(bytes.NewReader(b.Bytes()))
	if ok {
		t.Fatal("expected ok=false for stream with no moov box")
	}
}

func TestExtractVideoMeta_MacEpochUnderflow(t *testing.T) {
	data := buildMP4(buildMvhdV0(0))
	_, ok := parseMoovMvhd(bytes.NewReader(data))
	if ok {
		t.Fatal("expected ok=false for creation_time=0")
	}
}

// writeDirEntry writes a 12-byte IFD directory entry at buf[0:12].
func writeDirEntry(buf []byte, tag, typ uint16, count, valueOrOffset uint32) {
	binary.LittleEndian.PutUint16(buf[0:2], tag)
	binary.LittleEndian.PutUint16(buf[2:4], typ)
	binary.LittleEndian.PutUint32(buf[4:8], count)
	binary.LittleEndian.PutUint32(buf[8:12], valueOrOffset)
}

// buildMinimalJPEGWithEXIF constructs a minimal JFIF stream with an APP1/EXIF
// segment. The TIFF block is little-endian with:
//   - IFD0: Model (0x0110) and ExifIFD pointer (0x8769)
//   - ExifIFD: DateTimeOriginal (0x9003)
//
// model and dto must be null-terminated ASCII strings.
func buildMinimalJPEGWithEXIF(model, dto string) []byte {
	const (
		ifd0Off    = 8
		exifIFDOff = 38
		valAreaOff = 56
	)
	modelBytes := []byte(model)
	dtoBytes := []byte(dto)
	tiff := make([]byte, valAreaOff+len(modelBytes)+len(dtoBytes))
	// TIFF header: "II" (little-endian), magic 0x002A, IFD0 offset.
	copy(tiff[0:2], "II")
	binary.LittleEndian.PutUint16(tiff[2:4], 0x002A)
	binary.LittleEndian.PutUint32(tiff[4:8], ifd0Off)

	// IFD0: 2 entries (Model, ExifIFD pointer).
	binary.LittleEndian.PutUint16(tiff[ifd0Off:], 2)
	writeDirEntry(tiff[ifd0Off+2:], 0x0110, 2, uint32(len(modelBytes)), uint32(valAreaOff))
	writeDirEntry(tiff[ifd0Off+14:], 0x8769, 4, 1, uint32(exifIFDOff))
	binary.LittleEndian.PutUint32(tiff[ifd0Off+26:], 0)

	// ExifIFD: 1 entry (DateTimeOriginal).
	dtoOff := uint32(valAreaOff + len(modelBytes))
	binary.LittleEndian.PutUint16(tiff[exifIFDOff:], 1)
	writeDirEntry(tiff[exifIFDOff+2:], 0x9003, 2, uint32(len(dtoBytes)), dtoOff)
	binary.LittleEndian.PutUint32(tiff[exifIFDOff+14:], 0)

	// Value area.
	copy(tiff[valAreaOff:], modelBytes)
	copy(tiff[valAreaOff+len(modelBytes):], dtoBytes)

	// Wrap TIFF in APP1 segment.
	exifPfx := []byte("Exif")
	exifPfx = append(exifPfx, 0x00, 0x00)
	app1Body := append(exifPfx, tiff...)
	app1Len := uint16(len(app1Body) + 2)

	var j bytes.Buffer
	j.Write([]byte{0xFF, 0xD8})
	j.Write([]byte{0xFF, 0xE1})
	j.WriteByte(byte(app1Len >> 8))
	j.WriteByte(byte(app1Len))
	j.Write(app1Body)
	j.Write([]byte{0xFF, 0xD9})
	return j.Bytes()
}

// buildRAFFile constructs a synthetic RAF container. The JPEG payload is placed
// at offset 160, well beyond the 92-byte fixed header.
func buildRAFFile(jpegPayload []byte) []byte {
	const jpegStart = 160
	buf := make([]byte, jpegStart+len(jpegPayload))
	copy(buf[0:8], "FUJIFILM")
	copy(buf[8:15], "CCD-RAW")
	buf[15] = 0x00
	binary.BigEndian.PutUint32(buf[84:88], uint32(jpegStart))
	binary.BigEndian.PutUint32(buf[88:92], uint32(len(jpegPayload)))
	copy(buf[jpegStart:], jpegPayload)
	return buf
}

func TestExtractRAFMeta_ValidWithEXIF(t *testing.T) {
	model := "X-T4"
	modelNT := model + string([]byte{0})
	dto := "2022:03:10 08:30:00" + string([]byte{0})
	jpeg := buildMinimalJPEGWithEXIF(modelNT, dto)
	rafData := buildRAFFile(jpeg)

	f, err := os.CreateTemp(t.TempDir(), "*.raf")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	if _, err := f.Write(rafData); err != nil {
		t.Fatalf("write: %v", err)
	}
	path := f.Name()
	f.Close()

	got, ok := extractRAFMeta(path)
	if !ok {
		t.Fatal("extractRAFMeta() returned false, want true")
	}
	if got.CameraModel != model {
		t.Errorf("CameraModel = %q, want %q", got.CameraModel, model)
	}
	if got.DateTimeOriginal.Year() != 2022 {
		t.Errorf("Year = %d, want 2022", got.DateTimeOriginal.Year())
	}
	if got.Source != SourceEXIF {
		t.Errorf("Source = %v, want SourceEXIF", got.Source)
	}
}

func TestExtractRAFMeta_WrongMagic(t *testing.T) {
	modelNT := "X-T4" + string([]byte{0})
	dtoNT := "2022:03:10 08:30:00" + string([]byte{0})
	rafData := buildRAFFile(buildMinimalJPEGWithEXIF(modelNT, dtoNT))
	copy(rafData[0:8], "NOTFUJI!")

	f, err := os.CreateTemp(t.TempDir(), "*.raf")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write(rafData)
	path := f.Name()
	f.Close()

	_, ok := extractRAFMeta(path)
	if ok {
		t.Fatal("expected ok=false for wrong magic")
	}
}

func TestExtractRAFMeta_TruncatedHeader(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.raf")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write([]byte("FUJIFILM"))
	path := f.Name()
	f.Close()

	_, ok := extractRAFMeta(path)
	if ok {
		t.Fatal("expected ok=false for truncated header")
	}
}

func TestExtract_JPEGWithEXIF(t *testing.T) {
	modelNT := "Sony A7R V" + string([]byte{0})
	dtoNT := "2023:11:20 14:00:00" + string([]byte{0})
	data := buildMinimalJPEGWithEXIF(modelNT, dtoNT)

	f, err := os.CreateTemp(t.TempDir(), "*.jpg")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write(data)
	path := f.Name()
	f.Close()

	got := Extract(path)
	if got.Source != SourceEXIF {
		t.Errorf("Source = %v, want SourceEXIF", got.Source)
	}
	if got.DateTimeOriginal.Year() != 2023 {
		t.Errorf("Year = %d, want 2023", got.DateTimeOriginal.Year())
	}
}

func TestExtract_MP4WithMoov(t *testing.T) {
	refUnix := time.Date(2024, 6, 1, 9, 0, 0, 0, time.UTC).Unix()
	refMac := uint32(uint64(refUnix) + macEpochOffset)
	data := buildMP4(buildMvhdV0(refMac))

	f, err := os.CreateTemp(t.TempDir(), "*.mp4")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.Write(data)
	path := f.Name()
	f.Close()

	got := Extract(path)
	if got.Source != SourceVideoContainer {
		t.Errorf("Source = %v, want SourceVideoContainer", got.Source)
	}
	want := time.Unix(refUnix, 0).UTC()
	if !got.DateTimeOriginal.Equal(want) {
		t.Errorf("DateTimeOriginal = %v, want %v", got.DateTimeOriginal, want)
	}
}

func TestExtract_FallsBackToMtimeForUnknownExtension(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "*.txt")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.WriteString("no exif here")
	path := f.Name()
	f.Close()

	before := time.Now().Add(-time.Second)
	got := Extract(path)
	after := time.Now().Add(time.Second)

	if got.Source != SourceMtime {
		t.Errorf("Source = %v, want SourceMtime", got.Source)
	}
	if got.DateTimeOriginal.Before(before) || got.DateTimeOriginal.After(after) {
		t.Errorf("DateTimeOriginal %v not in expected range [%v, %v]",
			got.DateTimeOriginal, before, after)
	}
}

func TestExtract_DNGFallsBackGracefully(t *testing.T) {
	// Write random bytes as a .dng file — EXIF extraction will fail and the
	// function must fall back to mtime without panicking.
	f, err := os.CreateTemp(t.TempDir(), "*.dng")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write(make([]byte, 64)) // 64 zero bytes — not valid EXIF
	path := f.Name()
	f.Close()

	before := time.Now().Add(-time.Second)
	got := Extract(path)
	after := time.Now().Add(time.Second)

	if got.Source != SourceMtime {
		t.Errorf("Source = %v, want SourceMtime for invalid DNG content", got.Source)
	}
	if got.DateTimeOriginal.Before(before) || got.DateTimeOriginal.After(after) {
		t.Errorf("DateTimeOriginal %v not in expected mtime range [%v, %v]",
			got.DateTimeOriginal, before, after)
	}
}

func TestExtract_FallsBackToMtimeForMP4WithNoMoov(t *testing.T) {
	var b bytes.Buffer
	writeBox(&b, "ftyp", make([]byte, 8))

	f, err := os.CreateTemp(t.TempDir(), "*.mp4")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_, _ = f.Write(b.Bytes())
	path := f.Name()
	f.Close()

	got := Extract(path)
	if got.Source != SourceMtime {
		t.Errorf("Source = %v, want SourceMtime", got.Source)
	}
}
