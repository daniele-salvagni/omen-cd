package omen

import (
	"bytes"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// End-to-end tests that use a real git binary against a local bare repo.
// Skipped if git is missing (unlikely on a dev machine or CI).

func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}

// upstream sets up a bare "remote" repo and a scratch working tree used to
// author commits into it. It returns the bare repo path and a commit()
// helper that writes files and pushes them to the given branch.
func upstream(t *testing.T, branch string) (string, func(files map[string]string, msg string)) {
	t.Helper()
	root := t.TempDir()
	bare := filepath.Join(root, "remote.git")
	work := filepath.Join(root, "seed")
	mustGit(t, "", "init", "--bare", "-b", branch, bare)
	mustGit(t, "", "init", "-b", branch, work)
	mustGit(t, work, "config", "user.email", "t@t")
	mustGit(t, work, "config", "user.name", "t")
	mustGit(t, work, "remote", "add", "origin", bare)

	commit := func(files map[string]string, msg string) {
		t.Helper()
		for p, c := range files {
			full := filepath.Join(work, p)
			if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(full, []byte(c), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		mustGit(t, work, "add", "-A")
		mustGit(t, work, "commit", "-m", msg)
		mustGit(t, work, "push", "-u", "origin", branch)
	}
	return bare, commit
}

func mustGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

func TestSyncInitialSkipRecordsHead(t *testing.T) {
	requireGit(t)
	repo, commit := upstream(t, "main")
	commit(map[string]string{
		".omen.yaml": "rules:\n  - name: r\n    paths: [\"**\"]\n    run: touch /tmp/should-not-fire-omen-test-$$\n",
		"a/x.txt":    "hi",
	}, "init")

	work := filepath.Join(t.TempDir(), "checkout")
	state := filepath.Join(t.TempDir(), "state")
	h := &Host{Repo: repo, Dir: work, Branch: "main", Source: ".omen.yaml"}

	var buf bytes.Buffer
	err := Sync(h, Options{
		Instance:  "test",
		StatePath: state,
		Log:       log.New(&buf, "", 0),
		Out:       &buf,
	})
	if err != nil {
		t.Fatalf("Sync: %v\n%s", err, buf.String())
	}

	got, err := ReadState(state)
	if err != nil || got == "" {
		t.Fatalf("expected state written, got %q err=%v", got, err)
	}
	if !strings.Contains(buf.String(), "first run") {
		t.Fatalf("expected first-run log line, got:\n%s", buf.String())
	}
}

func TestSyncAppliesRuleOnNewCommit(t *testing.T) {
	requireGit(t)
	repo, commit := upstream(t, "main")

	// Marker path proves the rule fired.
	tmp := t.TempDir()
	marker := filepath.Join(tmp, "fired")

	commit(map[string]string{
		".omen.yaml": "rules:\n  - name: r\n    paths: [\"content/**\"]\n    run: touch " + marker + "\n",
		"content/a":  "1",
	}, "init")

	work := filepath.Join(tmp, "checkout")
	state := filepath.Join(tmp, "state")
	h := &Host{Repo: repo, Dir: work, Branch: "main", Source: ".omen.yaml"}

	// No state and no --apply-all would be a no-op first run. Force the
	// initial deploy the same way an operator does on a fresh install.
	if err := Sync(h, Options{Instance: "test", StatePath: state, ApplyAll: true, Log: log.New(io.Discard, "", 0), Out: io.Discard}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(marker); err != nil {
		t.Fatalf("expected marker at %s: %v", marker, err)
	}
}

func TestSyncNoAdvanceIsNoop(t *testing.T) {
	requireGit(t)
	repo, commit := upstream(t, "main")
	commit(map[string]string{
		".omen.yaml": "rules:\n  - name: r\n    paths: [\"**\"]\n    run: false\n",
	}, "init")

	tmp := t.TempDir()
	work := filepath.Join(tmp, "checkout")
	state := filepath.Join(tmp, "state")
	h := &Host{Repo: repo, Dir: work, Branch: "main", Source: ".omen.yaml"}

	// First run: no state, records HEAD, runs nothing.
	if err := Sync(h, Options{Instance: "t", StatePath: state, Log: log.New(io.Discard, "", 0), Out: io.Discard}); err != nil {
		t.Fatal(err)
	}
	// Second run: no advance, should be a no-op (would fail on rule 'false' otherwise).
	if err := Sync(h, Options{Instance: "t", StatePath: state, Log: log.New(io.Discard, "", 0), Out: io.Discard}); err != nil {
		t.Fatalf("second sync should be no-op, got %v", err)
	}
}

func TestSyncFailingRuleLeavesStateAtPrev(t *testing.T) {
	requireGit(t)
	repo, commit := upstream(t, "main")
	commit(map[string]string{".omen.yaml": "rules: []\n", "a": "1"}, "init")

	tmp := t.TempDir()
	work := filepath.Join(tmp, "checkout")
	state := filepath.Join(tmp, "state")
	h := &Host{Repo: repo, Dir: work, Branch: "main", Source: ".omen.yaml"}

	// First sync: record initial HEAD.
	if err := Sync(h, Options{Instance: "t", StatePath: state, Log: log.New(io.Discard, "", 0), Out: io.Discard}); err != nil {
		t.Fatal(err)
	}
	prev, _ := ReadState(state)

	// Push a commit with a failing rule.
	commit(map[string]string{
		".omen.yaml": "rules:\n  - name: bad\n    paths: [\"**\"]\n    run: false\n",
	}, "break")

	err := Sync(h, Options{Instance: "t", StatePath: state, Log: log.New(io.Discard, "", 0), Out: io.Discard})
	if err == nil {
		t.Fatal("expected rule failure")
	}
	after, _ := ReadState(state)
	if after != prev {
		t.Fatalf("state advanced despite failure: prev=%s after=%s", prev, after)
	}
}
