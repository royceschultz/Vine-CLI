package server

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// RotatingLog is a write-closer that rotates log files when they exceed a max size.
type RotatingLog struct {
	mu       sync.Mutex
	path     string
	maxSize  int64
	maxFiles int
	file     *os.File
	size     int64
}

// NewRotatingLog creates a rotating log writer.
// maxSize is in bytes (default 10MB if 0). maxFiles is the number of rotated files to keep (default 3 if 0).
func NewRotatingLog(path string, maxSize int64, maxFiles int) (*RotatingLog, error) {
	if maxSize <= 0 {
		maxSize = 10 * 1024 * 1024 // 10MB
	}
	if maxFiles <= 0 {
		maxFiles = 3
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("creating log directory: %w", err)
	}

	rl := &RotatingLog{
		path:     path,
		maxSize:  maxSize,
		maxFiles: maxFiles,
	}

	if err := rl.openFile(); err != nil {
		return nil, err
	}

	return rl, nil
}

func (rl *RotatingLog) openFile() error {
	f, err := os.OpenFile(rl.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return fmt.Errorf("opening log file: %w", err)
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("stat log file: %w", err)
	}
	rl.file = f
	rl.size = info.Size()
	return nil
}

func (rl *RotatingLog) Write(p []byte) (int, error) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	if rl.size+int64(len(p)) > rl.maxSize {
		if err := rl.rotate(); err != nil {
			return 0, err
		}
	}

	n, err := rl.file.Write(p)
	rl.size += int64(n)
	return n, err
}

func (rl *RotatingLog) Close() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	if rl.file != nil {
		return rl.file.Close()
	}
	return nil
}

func (rl *RotatingLog) rotate() error {
	rl.file.Close()

	// Shift existing rotated files: .3 -> deleted, .2 -> .3, .1 -> .2, current -> .1
	for i := rl.maxFiles; i >= 1; i-- {
		src := rl.rotatedPath(i)
		if i == rl.maxFiles {
			os.Remove(src)
		} else {
			dst := rl.rotatedPath(i + 1)
			os.Rename(src, dst)
		}
	}

	// Rotate current file to .1
	os.Rename(rl.path, rl.rotatedPath(1))

	return rl.openFile()
}

func (rl *RotatingLog) rotatedPath(n int) string {
	return fmt.Sprintf("%s.%d", rl.path, n)
}
