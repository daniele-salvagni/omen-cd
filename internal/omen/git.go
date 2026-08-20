package omen

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Git wraps the git binary with a fixed working directory and SSH env.
// Every call shells out; there is no in-process git implementation on purpose.
type Git struct {
	Dir string
	Env []string
	Log io.Writer
}

// NewGit prepares a Git bound to dir. It resolves GIT_SSH_COMMAND as follows:
// respect an existing value in the environment; else derive from sshKey if
// set; else use a minimal command that just caps the connect timeout.
func NewGit(dir, sshKey string, log io.Writer) *Git {
	env := os.Environ()
	if _, set := lookupEnv(env, "GIT_SSH_COMMAND"); !set {
		cmd := "ssh -o ConnectTimeout=30"
		if sshKey != "" {
			cmd = fmt.Sprintf("ssh -i %s -o IdentitiesOnly=yes -o ConnectTimeout=30", sshKey)
		}
		env = append(env, "GIT_SSH_COMMAND="+cmd)
	}
	return &Git{Dir: dir, Env: env, Log: log}
}

// HasCheckout reports whether Dir contains a git working copy.
func (g *Git) HasCheckout() bool {
	_, err := os.Stat(filepath.Join(g.Dir, ".git"))
	return err == nil
}

func (g *Git) Clone(repo, branch string) error {
	if err := os.MkdirAll(filepath.Dir(g.Dir), 0o755); err != nil {
		return err
	}
	return g.exec("", "clone", "--branch", branch, repo, g.Dir)
}

func (g *Git) Fetch(branch string) error {
	return g.exec(g.Dir, "fetch", "--quiet", "origin", branch)
}

func (g *Git) MergeFF(branch string) error {
	return g.exec(g.Dir, "merge", "--ff-only", "--quiet", "origin/"+branch)
}

func (g *Git) RevParse(ref string) (string, error) {
	return g.capture("rev-parse", ref)
}

func (g *Git) Diff(a, b string) ([]string, error) {
	out, err := g.capture("diff", "--name-only", a, b)
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

func (g *Git) LsFiles() ([]string, error) {
	out, err := g.capture("ls-files")
	if err != nil {
		return nil, err
	}
	return splitLines(out), nil
}

// exec streams git's output to g.Log. Use for commands whose output we don't
// need to parse (clone, fetch, merge).
func (g *Git) exec(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	cmd.Env = g.Env
	cmd.Stdout = g.Log
	cmd.Stderr = g.Log
	return cmd.Run()
}

// capture returns stdout. stderr is preserved in the error on failure.
func (g *Git) capture(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = g.Dir
	cmd.Env = g.Env
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return strings.TrimSpace(stdout.String()), nil
}

func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

func lookupEnv(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, e := range env {
		if strings.HasPrefix(e, prefix) {
			return e[len(prefix):], true
		}
	}
	return "", false
}
