// Command omen is a pull-based GitOps runner.
//
// Usage:
//
//	omen [--config PATH] [--dry-run] [--apply-all]
//	omen init [host|spec]
//	omen unit [service|timer]
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

func main() {
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "omen:", err)
		os.Exit(1)
	}
}

const usage = `Usage:
  omen [--config PATH] [--dry-run] [--apply-all]
  omen init [host|spec]
  omen unit [service|timer]
  omen version

Bare invocation performs one sync using the given config.
`

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
	dryRun := fs.Bool("dry-run", false, "report what would happen; write nothing")
	applyAll := fs.Bool("apply-all", false, "run all matching rules against HEAD")
	if err := fs.Parse(args); err != nil {
		return err
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

// initCmd prints a starter template. Defaults to host if no argument given.
func initCmd(args []string) error {
	what := "host"
	if len(args) > 0 {
		what = args[0]
	}
	switch what {
	case "host":
		fmt.Print(omen.HostTemplate)
	case "spec":
		fmt.Print(omen.SpecTemplate)
	default:
		return fmt.Errorf("unknown template %q (want host or spec)", what)
	}
	return nil
}

// unitCmd prints a systemd unit file.
func unitCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: omen unit [service|timer]")
	}
	switch args[0] {
	case "service":
		fmt.Print(omen.ServiceUnit)
	case "timer":
		fmt.Print(omen.TimerUnit)
	default:
		return fmt.Errorf("unknown unit %q (want service or timer)", args[0])
	}
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
