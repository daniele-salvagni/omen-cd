package omen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

// ReadState returns the last-applied sha, or "" if no state exists yet.
func ReadState(path string) (string, error) {
	b, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(b)), nil
}

// WriteState persists sha atomically: write to a sibling tmp file, then rename.
// A crash mid-write leaves the previous state intact.
func WriteState(path, sha string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(sha+"\n"), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Lock takes an exclusive, non-blocking flock on path. It returns an unlock
// func that must be called (typically via defer). If another process already
// holds the lock, Lock fails immediately rather than queuing.
func Lock(path string) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return nil, err
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		f.Close()
		return nil, fmt.Errorf("another omen instance is running (lock %s): %w", path, err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		_ = f.Close()
	}, nil
}
