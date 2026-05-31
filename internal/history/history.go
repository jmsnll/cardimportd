package history

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Entry struct {
	UUID        string    `json:"uuid"`
	Owner       string    `json:"owner"`
	MountPath   string    `json:"mount_path"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at"`
	Total       int       `json:"total"`
	Imported    int       `json:"imported"`
	Skipped     int       `json:"skipped"`
	Failed      int       `json:"failed"`
	BytesCopied int64     `json:"bytes_copied"`
}

type Log struct {
	mu   sync.Mutex
	path string
}

func New(path string) *Log { return &Log{path: path} }

func (l *Log) Append(entry Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return fmt.Errorf("history mkdir: %w", err)
	}
	f, err := os.OpenFile(l.path, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("history open: %w", err)
	}
	defer f.Close()
	line, _ := json.Marshal(entry)
	_, err = fmt.Fprintf(f, "%s\n", line)
	return err
}

func (l *Log) Recent(n int) ([]Entry, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	f, err := os.Open(l.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("history open: %w", err)
	}
	defer f.Close()
	var entries []Entry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		b := scanner.Bytes()
		if len(b) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(b, &e); err != nil {
			slog.Warn("history: malformed line", "error", err)
			continue
		}
		entries = append(entries, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("history scan: %w", err)
	}
	for i, j := 0, len(entries)-1; i < j; i, j = i+1, j-1 {
		entries[i], entries[j] = entries[j], entries[i]
	}
	if n > 0 && len(entries) > n {
		entries = entries[:n]
	}
	return entries, nil
}
