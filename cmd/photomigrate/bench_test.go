package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// benchRawSize approximates a mid-range mirrorless RAW file (Fuji X-T5, Sony A7).
const benchRawSize = 32 << 20 // 32 MB

func makeBenchFile(tb testing.TB, size int64) string {
	tb.Helper()
	f, err := os.CreateTemp(tb.TempDir(), "bench*.arw")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := io.CopyN(f, rand.Reader, size); err != nil {
		tb.Fatal(err)
	}
	f.Close()
	return f.Name()
}

func copyBenchFile(tb testing.TB, src string) string {
	tb.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		tb.Fatal(err)
	}
	f, err := os.CreateTemp(tb.TempDir(), "bench-copy*.arw")
	if err != nil {
		tb.Fatal(err)
	}
	if _, err := f.Write(data); err != nil {
		tb.Fatal(err)
	}
	f.Close()
	return f.Name()
}

// resolveActionFullLock is the pre-optimisation behaviour: mutex held for the
// entire call including SHA-256. Used only as a benchmark baseline.
func resolveActionFullLock(srcPath, dstPath string, mu *sync.Mutex) (action, string, error) {
	mu.Lock()
	defer mu.Unlock()

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return 0, "", fmt.Errorf("stat src: %w", err)
	}
	dstInfo, err := os.Stat(dstPath)
	if os.IsNotExist(err) {
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
	ext := filepath.Ext(dstPath)
	stem := dstPath[:len(dstPath)-len(ext)]
	for i := 2; i <= 999; i++ {
		candidate := fmt.Sprintf("%s_%d%s", stem, i, ext)
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return actionRename, candidate, nil
		}
	}
	return 0, "", fmt.Errorf("no unique destination for %q", filepath.Base(dstPath))
}

// BenchmarkResolveAction_Create measures the fast path: dst absent, just a stat.
func BenchmarkResolveAction_Create(b *testing.B) {
	src := makeBenchFile(b, 4<<10)
	dir := b.TempDir()
	mu := new(sync.Mutex)
	b.ResetTimer()
	for i := range b.N {
		dst := filepath.Join(dir, fmt.Sprintf("%d.arw", i))
		if _, _, err := resolveAction(src, dst, mu); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResolveAction_Skip_Sequential measures the skip path (SHA-256 of both
// files) with a single goroutine — the baseline for the parallel benchmarks below.
func BenchmarkResolveAction_Skip_Sequential(b *testing.B) {
	src := makeBenchFile(b, benchRawSize)
	dst := copyBenchFile(b, src)
	mu := new(sync.Mutex)
	b.SetBytes(benchRawSize * 2)
	b.ResetTimer()
	for range b.N {
		if _, _, err := resolveAction(src, dst, mu); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResolveAction_Skip_Parallel_NarrowLock shows the current behaviour:
// SHA-256 runs outside the mutex so workers hash concurrently.
// Throughput should scale with GOMAXPROCS until I/O-bound.
func BenchmarkResolveAction_Skip_Parallel_NarrowLock(b *testing.B) {
	src := makeBenchFile(b, benchRawSize)
	dst := copyBenchFile(b, src)
	mu := new(sync.Mutex)
	b.SetBytes(benchRawSize * 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, _, err := resolveAction(src, dst, mu); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkResolveAction_Skip_Parallel_FullLock shows the pre-optimisation
// behaviour: mutex held throughout, serialising SHA-256 across all workers.
// Throughput should match the sequential baseline regardless of GOMAXPROCS.
func BenchmarkResolveAction_Skip_Parallel_FullLock(b *testing.B) {
	src := makeBenchFile(b, benchRawSize)
	dst := copyBenchFile(b, src)
	mu := new(sync.Mutex)
	b.SetBytes(benchRawSize * 2)
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, _, err := resolveActionFullLock(src, dst, mu); err != nil {
				b.Fatal(err)
			}
		}
	})
}
