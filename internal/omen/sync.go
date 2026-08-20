package omen

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options controls one Sync invocation.
type Options struct {
	Instance  string      // used for state path, lock path, and OMEN_INSTANCE
	ApplyAll  bool        // treat every tracked file as changed (one-shot force)
	DryRun    bool        // report what would happen; write nothing, run nothing
	StatePath string      // override the derived state path (mainly for tests)
	Log       *log.Logger // control-plane messages
	Out       io.Writer   // stdout/stderr for git and rule shells
}

// Sync performs one deploy cycle: fetch, ff-only merge, diff, match, run.
// State advances only after every matching rule succeeds, so a mid-batch
// failure is retried on the next tick against the same diff.
func Sync(h *Host, opts Options) error {
	if opts.Log == nil {
		opts.Log = log.New(io.Discard, "", 0)
	}
	if opts.Out == nil {
		opts.Out = io.Discard
	}
	statePath := opts.StatePath
	if statePath == "" {
		statePath = DefaultStatePath(opts.Instance)
	}

	if !opts.DryRun {
		unlock, err := Lock(statePath + ".lock")
		if err != nil {
			return err
		}
		defer unlock()
	}

	g := NewGit(h.Dir, h.SSHKey, opts.Out)

	if !g.HasCheckout() {
		if opts.DryRun {
			opts.Log.Printf("would clone %s -> %s", h.Repo, h.Dir)
			return nil
		}
		opts.Log.Printf("cloning %s -> %s", h.Repo, h.Dir)
		if err := g.Clone(h.Repo, h.Branch); err != nil {
			return fmt.Errorf("clone: %w", err)
		}
	}

	prev, err := ReadState(statePath)
	if err != nil {
		return fmt.Errorf("read state: %w", err)
	}

	if err := g.Fetch(h.Branch); err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	// In dry-run we never touch the working tree, so we plan against the
	// remote tip; otherwise we ff-merge and plan against HEAD.
	headRef := "HEAD"
	if opts.DryRun {
		headRef = "origin/" + h.Branch
	} else if err := g.MergeFF(h.Branch); err != nil {
		return fmt.Errorf("ff-only merge failed (local diverged?): %w", err)
	}
	head, err := g.RevParse(headRef)
	if err != nil {
		return err
	}

	changed, skipInitial, err := plan(g, prev, head, opts.ApplyAll)
	if err != nil {
		return err
	}
	if skipInitial {
		opts.Log.Printf("first run, recording %s (use --apply-all to deploy now)", short(head))
		if opts.DryRun {
			return nil
		}
		return WriteState(statePath, head)
	}
	if len(changed) == 0 {
		return nil
	}

	spec, err := LoadSpec(filepath.Join(h.Dir, h.Source))
	if err != nil {
		return fmt.Errorf("load spec: %w", err)
	}

	opts.Log.Printf("commit %s -> %s (%d %s)", short(prev), short(head), len(changed), plural(len(changed), "file", "files"))

	ran, failed, err := runRules(spec.Rules, changed, h.Dir, opts)
	if err != nil {
		notify(spec, h.Dir, opts.Instance, head, "failed: "+failed, opts.Out, opts.Log)
		return err
	}

	if opts.DryRun {
		return nil
	}

	// Advance state on any real advance, even if no rule fired: otherwise
	// we would re-diff and re-check the same files on every tick forever.
	if prev != head {
		if err := WriteState(statePath, head); err != nil {
			return fmt.Errorf("write state: %w", err)
		}
	}
	if len(ran) > 0 {
		notify(spec, h.Dir, opts.Instance, head, "deployed: "+strings.Join(ran, ", "), opts.Out, opts.Log)
	} else {
		opts.Log.Printf("no rules matched (%d changed %s)", len(changed), plural(len(changed), "file", "files"))
	}
	return nil
}

// plan decides which files count as changed for this run. It returns
// (files, skipInitial, err). skipInitial means "first run with no state";
// callers should record HEAD and stop. Use --apply-all to force a full run
// against every tracked file.
func plan(g *Git, prev, head string, applyAll bool) ([]string, bool, error) {
	if applyAll {
		f, err := g.LsFiles()
		return f, false, err
	}
	if prev == "" {
		return nil, true, nil
	}
	if prev == head {
		return nil, false, nil
	}
	f, err := g.Diff(prev, head)
	return f, false, err
}

// runRules executes every rule whose globs match at least one changed file,
// in order. It stops at the first failure. On success failed is "".
func runRules(rules []Rule, changed []string, dir string, opts Options) (ran []string, failed string, err error) {
	for _, r := range rules {
		if !matchAny(r.Paths, changed) {
			continue
		}
		name := ruleName(r)
		if opts.DryRun {
			opts.Log.Printf("would run: %s", name)
			ran = append(ran, name)
			continue
		}
		opts.Log.Printf("running: %s", name)
		if err := shell(dir, r.Run, opts.Out); err != nil {
			return ran, name, fmt.Errorf("rule %q: %w", name, err)
		}
		ran = append(ran, name)
	}
	return ran, "", nil
}

func shell(dir, command string, out io.Writer) error {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = dir
	cmd.Stdout = out
	cmd.Stderr = out
	return cmd.Run()
}

// notify runs the spec's notify command with omen context in the environment.
// A broken notifier is logged and swallowed: it must never fail a good deploy.
func notify(s *Spec, dir, instance, sha, status string, out io.Writer, lg *log.Logger) {
	if s.Notify == "" {
		return
	}
	cmd := exec.Command("sh", "-c", s.Notify)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(),
		"OMEN_SHA="+sha,
		"OMEN_SHORT="+short(sha),
		"OMEN_STATUS="+status,
		"OMEN_INSTANCE="+instance,
	)
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		lg.Printf("notify failed: %v", err)
	}
}

func ruleName(r Rule) string {
	if r.Name != "" {
		return r.Name
	}
	return r.Run
}

func plural(n int, singular, pl string) string {
	if n == 1 {
		return singular
	}
	return pl
}

func short(sha string) string {
	if len(sha) > 7 {
		return sha[:7]
	}
	return sha
}

// DefaultStatePath returns the state file path for the given instance.
// Callers may override it via Options.StatePath (tests do).
func DefaultStatePath(instance string) string {
	if instance == "" {
		instance = "default"
	}
	return filepath.Join("/var/lib/omen", instance+".state")
}
