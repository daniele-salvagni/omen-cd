// Command omen is a pull-based GitOps runner.
//
// Usage:
//
//	omen [--config PATH] [--env-file PATH] [--dry-run] [--apply-all]
//	omen init host [flags]
//	omen init spec
//	omen unit service --config PATH
//	omen unit timer   --config PATH
//	omen version
package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"github.com/daniele-salvagni/omen-cd/internal/omen"
)

const usage = `Usage:
  omen [--config PATH] [--env-file PATH] [--dry-run] [--apply-all]
  omen init host [flags]
  omen init spec
  omen unit service --config PATH
  omen unit timer   --config PATH
  omen version

Bare invocation performs one sync using the given config. If an env file
sits next to the config (same name, .env suffix), it is loaded automatically.
`

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "omen:", err)
		os.Exit(1)
	}
}

// dispatch routes a leading subcommand token if present; anything else is
// treated as a sync invocation with flags.
func dispatch(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "version", "-version", "--version":
			fmt.Println(versionString())
			return nil
		case "help", "-h", "-help", "--help":
			fmt.Print(usage)
			return nil
		}
	}
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "init":
			return initCmd(args[1:])
		case "unit":
			return unitCmd(args[1:])
		default:
			return fmt.Errorf("unknown command %q", args[0])
		}
	}
	return syncCmd(args)
}

func syncCmd(args []string) error {
	fs := flag.NewFlagSet("omen", flag.ContinueOnError)
	fs.Usage = func() { fmt.Fprint(os.Stderr, usage) }
	cfg := fs.String("config", "/etc/omen/omen.yaml", "path to host config")
	envFile := fs.String("env-file", "", "path to env file (KEY=VALUE); overrides the default derived from --config")
	dryRun := fs.Bool("dry-run", false, "report what would happen; write nothing")
	applyAll := fs.Bool("apply-all", false, "run all matching rules against HEAD")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if *envFile != "" {
		if err := omen.LoadEnvFile(*envFile, true); err != nil {
			return err
		}
	} else {
		convention := strings.TrimSuffix(*cfg, filepath.Ext(*cfg)) + ".env"
		if err := omen.LoadEnvFile(convention, false); err != nil {
			return err
		}
	}

	h, err := omen.LoadHost(*cfg)
	if err != nil {
		return err
	}
	return omen.Sync(h, omen.Options{
		Instance: instanceFromPath(*cfg),
		ApplyAll: *applyAll,
		DryRun:   *dryRun,
		Log:      log.New(os.Stderr, "", 0),
		Out:      os.Stderr,
	})
}

// initCmd handles `omen init host` and `omen init spec`.
func initCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: omen init [host|spec]")
	}
	sub, args := args[0], args[1:]

	switch sub {
	case "host":
		return initHost(args)
	case "spec":
		fmt.Print(omen.SpecTemplate)
		return nil
	default:
		return fmt.Errorf("unknown template %q (want host or spec)", sub)
	}
}

func initHost(args []string) error {
	fs := flag.NewFlagSet("omen init host", flag.ContinueOnError)
	var h omen.HostInit
	fs.StringVar(&h.Repo, "repo", "", "git repo URL")
	fs.StringVar(&h.Dir, "dir", "", "local checkout path")
	fs.StringVar(&h.Branch, "branch", "", "branch to track (default: main)")
	fs.StringVar(&h.Source, "source", "", "in-repo spec path (default: .omen.yaml)")
	fs.StringVar(&h.User, "user", "", "run service as this user (default: root)")
	fs.StringVar(&h.SSHKey, "ssh-key", "", "SSH private key path")
	fs.StringVar(&h.Interval, "interval", "", "timer cadence (default: 60s)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Print(omen.RenderHostInit(h))
	return nil
}

// unitCmd handles `omen unit service|timer --config PATH`.
func unitCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: omen unit [service|timer] --config PATH")
	}
	sub, args := args[0], args[1:]

	fs := flag.NewFlagSet("omen unit", flag.ContinueOnError)
	cfg := fs.String("config", "", "path to host config (required)")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *cfg == "" {
		return fmt.Errorf("omen unit %s: --config is required", sub)
	}

	h, err := omen.LoadHost(*cfg)
	if err != nil {
		return err
	}
	name := instanceFromPath(*cfg)

	var out string
	switch sub {
	case "service":
		out, err = omen.RenderServiceUnit(h, name, *cfg)
	case "timer":
		out, err = omen.RenderTimerUnit(h, name)
	default:
		return fmt.Errorf("unknown unit %q (want service or timer)", sub)
	}
	if err != nil {
		return err
	}
	fmt.Print(out)
	return nil
}

// instanceFromPath derives the instance name from a config path so
// /etc/omen/web.yaml becomes "web".
func instanceFromPath(p string) string {
	base := filepath.Base(p)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func versionString() string {
	if bi, ok := debug.ReadBuildInfo(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return "devel"
}
