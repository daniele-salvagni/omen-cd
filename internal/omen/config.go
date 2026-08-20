// Package omen implements a pull-based GitOps runner.
//
// It fast-forwards a git checkout, diffs the newly fetched commits against a
// persisted last-applied sha, matches changed files against user-defined path
// globs, and runs a shell command for each rule that matches.
package omen

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Host is the host-side config: what to sync and how.
// It lives outside the repo (typically /etc/omen/<instance>.yaml) and is
// owned by the machine, not the git tree.
type Host struct {
	Repo   string `yaml:"repo"`
	Dir    string `yaml:"dir"`
	Branch string `yaml:"branch"`
	Source string `yaml:"source"`
	SSHKey string `yaml:"ssh_key"`
}

// Spec is the in-repo config: what to run when the tracked branch advances.
// It lives in the repo (default .omen.yaml at the root) and travels with the
// code it deploys.
type Spec struct {
	Notify string `yaml:"notify"`
	Rules  []Rule `yaml:"rules"`
}

// Rule fires when any changed file matches one of Paths.
type Rule struct {
	Name  string   `yaml:"name"`
	Paths []string `yaml:"paths"`
	Run   string   `yaml:"run"`
}

// LoadHost reads and validates a host config, applying defaults.
func LoadHost(path string) (*Host, error) {
	var h Host
	if err := readYAML(path, &h); err != nil {
		return nil, err
	}
	if h.Repo == "" {
		return nil, fmt.Errorf("host config: repo is required")
	}
	if h.Dir == "" {
		return nil, fmt.Errorf("host config: dir is required")
	}
	if h.Branch == "" {
		h.Branch = "main"
	}
	if h.Source == "" {
		h.Source = ".omen.yaml"
	}
	return &h, nil
}

// LoadSpec reads and validates an in-repo sync spec.
func LoadSpec(path string) (*Spec, error) {
	var s Spec
	if err := readYAML(path, &s); err != nil {
		return nil, err
	}
	for i, r := range s.Rules {
		if r.Run == "" {
			return nil, fmt.Errorf("spec: rule %d: 'run' is required", i)
		}
		if len(r.Paths) == 0 {
			return nil, fmt.Errorf("spec: rule %d (%q): 'paths' is required", i, r.Name)
		}
	}
	return &s, nil
}

func readYAML(path string, dst any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := yaml.Unmarshal(b, dst); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
