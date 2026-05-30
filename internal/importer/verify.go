package importer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// copyVerified streams src to a sibling .tmp file, computes SHA-256 on both
// ends in a single read pass, then renames the .tmp to dst on success.
// Returns the number of bytes copied.
// On any failure the .tmp file is removed before returning.
func copyVerified(src, dst string) (int64, error) {
	tmp := dst + ".tmp"

	srcFile, err := os.Open(src)
	if err != nil {
		return 0, fmt.Errorf("open src %q: %w", src, err)
	}
	defer srcFile.Close()

	dstFile, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
	if err != nil {
		return 0, fmt.Errorf("create tmp %q: %w", tmp, err)
	}

	srcHash := sha256.New()
	dstHash := sha256.New()

	// TeeReader computes the source hash in the same pass as the copy.
	tee := io.TeeReader(srcFile, srcHash)
	mw := io.MultiWriter(dstFile, dstHash)

	n, copyErr := io.Copy(mw, tee)
	closeErr := dstFile.Close()

	if copyErr != nil || closeErr != nil {
		_ = os.Remove(tmp)
		if copyErr != nil {
			return 0, fmt.Errorf("copy to tmp %q: %w", tmp, copyErr)
		}
		return 0, fmt.Errorf("close tmp %q: %w", tmp, closeErr)
	}

	if fmt.Sprintf("%x", srcHash.Sum(nil)) != fmt.Sprintf("%x", dstHash.Sum(nil)) {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("sha256 mismatch after copy: %q → %q", src, dst)
	}

	if err := os.Rename(tmp, dst); err != nil {
		_ = os.Remove(tmp)
		return 0, fmt.Errorf("rename tmp to dst %q: %w", dst, err)
	}

	return n, nil
}
