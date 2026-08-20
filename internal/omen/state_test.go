package omen

import (
	"path/filepath"
	"testing"
)

func TestStateReadMissingReturnsEmpty(t *testing.T) {
	got, err := ReadState(filepath.Join(t.TempDir(), "nope"))
	if err != nil {
		t.Fatal(err)
	}
	if got != "" {
		t.Fatalf("got %q, want empty", got)
	}
}

func TestStateWriteThenRead(t *testing.T) {
	p := filepath.Join(t.TempDir(), "sub", "state") // exercises MkdirAll
	if err := WriteState(p, "deadbeef"); err != nil {
		t.Fatal(err)
	}
	got, err := ReadState(p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "deadbeef" {
		t.Fatalf("got %q, want deadbeef", got)
	}
}

func TestLockIsExclusive(t *testing.T) {
	p := filepath.Join(t.TempDir(), "l.lock")
	unlock, err := Lock(p)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	if _, err := Lock(p); err == nil {
		t.Fatal("second Lock should have failed")
	}
}

func TestLockReleaseAllowsReacquire(t *testing.T) {
	p := filepath.Join(t.TempDir(), "l.lock")
	unlock, err := Lock(p)
	if err != nil {
		t.Fatal(err)
	}
	unlock()
	unlock2, err := Lock(p)
	if err != nil {
		t.Fatal(err)
	}
	unlock2()
}
