package omen

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnvFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "env")
	content := `# a comment
FOO=bar
QUOTED="hello world"
SINGLE='hi'

BLANK=
`
	if err := os.WriteFile(p, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("FOO", "")
	t.Setenv("QUOTED", "")
	t.Setenv("SINGLE", "")
	t.Setenv("BLANK", "unset")

	if err := LoadEnvFile(p, true); err != nil {
		t.Fatal(err)
	}
	for k, want := range map[string]string{
		"FOO":    "bar",
		"QUOTED": "hello world",
		"SINGLE": "hi",
		"BLANK":  "",
	} {
		if got := os.Getenv(k); got != want {
			t.Errorf("%s: got %q, want %q", k, got, want)
		}
	}
}

func TestLoadEnvFileMissing(t *testing.T) {
	p := filepath.Join(t.TempDir(), "nope")
	if err := LoadEnvFile(p, false); err != nil {
		t.Fatalf("optional missing should be nil, got %v", err)
	}
	if err := LoadEnvFile(p, true); err == nil {
		t.Fatal("required missing should error")
	}
}

func TestLoadEnvFileMalformed(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "env")
	if err := os.WriteFile(p, []byte("no equals sign here\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := LoadEnvFile(p, true); err == nil {
		t.Fatal("expected parse error")
	}
}
