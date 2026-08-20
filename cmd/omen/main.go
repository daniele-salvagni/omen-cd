// Command omen is a pull-based GitOps runner.
//
// Usage:
//
//	omen [--config PATH] [--dry-run] [--apply-all]
//	omen init [--spec]
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

// dispatch routes a leading subcommand token (init, version) if present;
// anything else is treated as a sync invocation with flags.
func dispatch(args []string) error {
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		switch args[0] {
		case "init":
			return initCmd(args[1:])
		case "version":
			fmt.Println(versionString())
			return nil
		default:
			return fmt.Errorf("unknown command %q", args[0])
		}
	}
	return syncCmd(args)
}

func syncCmd(args []string) error {
	fs := flag.NewFlagSet("omen", flag.ContinueOnError)
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

func initCmd(args []string) error {
	fs := flag.NewFlagSet("omen init", flag.ContinueOnError)
	spec := fs.Bool("spec", false, "print a starter sync spec instead of a host config")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *spec {
		fmt.Print(omen.SpecTemplate)
	} else {
		fmt.Print(omen.HostTemplate)
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
