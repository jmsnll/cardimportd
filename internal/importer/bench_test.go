package importer

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jmsnll/cardimportd/internal/meta"
)

func BenchmarkCopyVerified(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1 << 10},
		{"1MB", 1 << 20},
		{"100MB", 100 << 20},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			src := filepath.Join(b.TempDir(), "src.bin")
			if err := os.WriteFile(src, bytes.Repeat([]byte{0xAB}, tc.size), 0o644); err != nil {
				b.Fatal(err)
			}

			b.SetBytes(int64(tc.size))
			b.ReportAllocs()
			b.ResetTimer()

			for b.Loop() {
				dst := filepath.Join(b.TempDir(), fmt.Sprintf("dst_%d.bin", b.N))
				if _, _, err := copyVerified(src, dst); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkCheckDupNoExist(b *testing.B) {
	dir := b.TempDir()
	src := filepath.Join(dir, "src.jpg")
	if err := os.WriteFile(src, []byte("data"), 0o644); err != nil {
		b.Fatal(err)
	}
	dst := filepath.Join(dir, "dst.jpg")
	m := meta.FileMeta{DateTimeOriginal: time.Date(2024, 3, 15, 10, 30, 0, 0, time.Local)}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		checkDup(src, dst, m)
	}
}

func BenchmarkCheckDupSkip(b *testing.B) {
	dir := b.TempDir()
	src := filepath.Join(dir, "src.jpg")
	dst := filepath.Join(dir, "dst.jpg")
	payload := []byte("same content")
	if err := os.WriteFile(src, payload, 0o644); err != nil {
		b.Fatal(err)
	}
	if err := os.WriteFile(dst, payload, 0o644); err != nil {
		b.Fatal(err)
	}
	m := meta.FileMeta{DateTimeOriginal: time.Date(2024, 3, 15, 10, 30, 0, 0, time.Local)}

	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		checkDup(src, dst, m)
	}
}
