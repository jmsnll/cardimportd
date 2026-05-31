// video.go reads the creation_time field from the mvhd (Movie Header) box in
// MP4 and MOV files. The MP4 container uses a Mac epoch: seconds since
// 1904-01-01 00:00:00 UTC. We subtract macEpochOffset to convert to Unix time.
package meta

import (
	"encoding/binary"
	"io"
	"os"
	"time"
)

// macEpochOffset is the number of seconds between 1904-01-01 and 1970-01-01.
// Used to convert MP4 Mac-epoch timestamps to Unix timestamps.
const macEpochOffset = 2082844800

// extractVideoMeta parses the moov/mvhd box from an MP4 or MOV file and
// returns the creation_time as FileMeta.
func extractVideoMeta(path string) (FileMeta, bool) {
	f, err := os.Open(path)
	if err != nil {
		return FileMeta{}, false
	}
	defer func() { _ = f.Close() }()

	return parseMoovMvhd(f)
}

// parseMoovMvhd scans top-level ISO base media boxes in r until it finds the
// "moov" box, then scans inside it for "mvhd".
func parseMoovMvhd(r io.ReadSeeker) (FileMeta, bool) {
	moovData, ok := findBox(r, "moov")
	if !ok {
		return FileMeta{}, false
	}

	mvhdData, ok := findBoxInBytes(moovData, "mvhd")
	if !ok {
		return FileMeta{}, false
	}

	return parseMvhd(mvhdData)
}

// findBox scans forward in r looking for a top-level box with the given type.
// It returns the box payload (excluding the 8-byte header).
func findBox(r io.ReadSeeker, boxType string) ([]byte, bool) {
	header := make([]byte, 8)
	for {
		if _, err := io.ReadFull(r, header); err != nil {
			return nil, false
		}

		size := binary.BigEndian.Uint32(header[0:4])
		typ := string(header[4:8])

		if size < 8 {
			// Malformed or extended-size box; stop scanning.
			return nil, false
		}

		payloadSize := int64(size) - 8

		if typ == boxType {
			if payloadSize <= 0 {
				return []byte{}, true
			}
			payload := make([]byte, payloadSize)
			if _, err := io.ReadFull(r, payload); err != nil {
				return nil, false
			}
			return payload, true
		}

		// Skip this box.
		if _, err := r.Seek(payloadSize, io.SeekCurrent); err != nil {
			return nil, false
		}
	}
}

// findBoxInBytes scans data for a box with the given type and returns its
// payload. It mirrors findBox but operates on an in-memory byte slice.
func findBoxInBytes(data []byte, boxType string) ([]byte, bool) {
	for len(data) >= 8 {
		size := binary.BigEndian.Uint32(data[0:4])
		typ := string(data[4:8])

		if size < 8 || int(size) > len(data) {
			return nil, false
		}

		payload := data[8:size]

		if typ == boxType {
			return payload, true
		}

		data = data[size:]
	}

	return nil, false
}

// parseMvhd parses the mvhd box payload and extracts creation_time.
// The box payload starts with a 1-byte version followed by 3 bytes of flags.
//
//   - version 0: creation_time is a uint32 (seconds, Mac epoch)
//   - version 1: creation_time is a uint64 (seconds, Mac epoch)
func parseMvhd(payload []byte) (FileMeta, bool) {
	if len(payload) < 4 {
		return FileMeta{}, false
	}

	version := payload[0]
	// payload[1:4] are flags; skip them.

	var creationSecs uint64
	const versionSize = 4 // version (1) + flags (3)

	switch version {
	case 0:
		if len(payload) < versionSize+4 {
			return FileMeta{}, false
		}
		creationSecs = uint64(binary.BigEndian.Uint32(payload[versionSize : versionSize+4]))
	case 1:
		if len(payload) < versionSize+8 {
			return FileMeta{}, false
		}
		creationSecs = binary.BigEndian.Uint64(payload[versionSize : versionSize+8])
	default:
		return FileMeta{}, false
	}

	if creationSecs < macEpochOffset {
		return FileMeta{}, false
	}

	unixSecs := int64(creationSecs - macEpochOffset)
	dt := time.Unix(unixSecs, 0).UTC()

	return FileMeta{
		DateTimeOriginal: dt,
		Source:           SourceVideoContainer,
	}, true
}
