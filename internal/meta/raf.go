// raf.go handles Fuji RAF files, which embed a full JPEG at a fixed offset
// within the RAF container. The JPEG is extracted and its EXIF is parsed with
// the standard extractEXIFFromBytes helper.
package meta

import (
	"bytes"
	"encoding/binary"
	"io"
	"os"
)

// rafMagic is the 16-byte magic prefix present at the start of every RAF file.
var rafMagic = [16]byte{
	'F', 'U', 'J', 'I', 'F', 'I', 'L', 'M',
	'C', 'C', 'D', '-', 'R', 'A', 'W', 0x00,
}

// RAF header offsets (byte positions within the 92-byte fixed header).
const (
	rafJPEGOffsetPos = 84 // big-endian uint32: byte offset of the embedded JPEG
	rafJPEGLengthPos = 88 // big-endian uint32: byte length of the embedded JPEG
	rafHeaderSize    = 92
)

// extractRAFMeta extracts EXIF metadata from the embedded JPEG inside a
// Fuji RAF file.
func extractRAFMeta(path string) (FileMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return FileMeta{}, false
	}
	defer func() { _ = f.Close() }()

	header := make([]byte, rafHeaderSize)
	if _, err := io.ReadFull(f, header); err != nil {
		return FileMeta{}, false
	}

	// Verify the RAF magic prefix.
	var magic [16]byte
	copy(magic[:], header[:16])
	if magic != rafMagic {
		return FileMeta{}, false
	}

	jpegOffset := binary.BigEndian.Uint32(header[rafJPEGOffsetPos:])
	jpegLength := binary.BigEndian.Uint32(header[rafJPEGLengthPos:])

	if jpegLength == 0 {
		return FileMeta{}, false
	}

	if _, err := f.Seek(int64(jpegOffset), io.SeekStart); err != nil {
		return FileMeta{}, false
	}

	jpegData := make([]byte, jpegLength)
	if _, err := io.ReadFull(f, jpegData); err != nil {
		return FileMeta{}, false
	}

	return exifFromReader(bytes.NewReader(jpegData))
}
