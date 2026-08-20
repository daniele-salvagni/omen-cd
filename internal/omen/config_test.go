package omen

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLoadHostDefaults(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "host.yaml")
	writeFile(t, p, "repo: git@host:me/repo.git\ndir: /srv/repo\n")

	h, err := LoadHost(p)
	if err != nil {
		t.Fatal(err)
	}
	if h.Branch != "main" || h.Source != ".omen.yaml" {
		t.Fatalf("defaults not applied: %+v", h)
	}
}

func TestLoadHostRequiresRepoAndDir(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]string{
		"missing repo": "dir: /srv/repo\n",
		"missing dir":  "repo: r\n",
	}
	for name, content := range cases {
		t.Run(name, func(t *testing.T) {
			p := filepath.Join(dir, name+".yaml")
			writeFile(t, p, content)
			if _, err := LoadHost(p); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestLoadSpecRuleValidation(t *testing.T) {
	dir := t.TempDir()
	good := `rules:
  - name: ok
    paths: ["**"]
    run: echo hi
`
	badNoPaths := `rules:
  - run: echo hi
`
	badNoRun := `rules:
  - paths: ["**"]
`

	for _, tc := range []struct {
		name    string
		content string
		wantErr bool
	}{
		{"valid", good, false},
		{"missing paths", badNoPaths, true},
		{"missing run", badNoRun, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := filepath.Join(dir, tc.name+".yaml")
			writeFile(t, p, tc.content)
			_, err := LoadSpec(p)
			if (err != nil) != tc.wantErr {
				t.Fatalf("got err=%v wantErr=%v", err, tc.wantErr)
			}
		})
	}
}
