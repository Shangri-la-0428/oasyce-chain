package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}

	switch os.Args[1] {
	case "audit-home":
		if err := runAuditHome(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "audit-home: %v\n", err)
			os.Exit(1)
		}
	case "replay-v080":
		if err := runReplayV080(os.Args[2:]); err != nil {
			fmt.Fprintf(os.Stderr, "replay-v080: %v\n", err)
			os.Exit(1)
		}
	default:
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage:")
	fmt.Fprintln(os.Stderr, "  go run ./tools/v080_fixture_audit audit-home --home /path/to/node-home [--output report.json]")
	fmt.Fprintln(os.Stderr, "  go run ./tools/v080_fixture_audit replay-v080 --source-home /path/to/pre-upgrade-home [--working-home ./tmp/v080-replay] [--output report.json]")
}

func runAuditHome(args []string) error {
	fs := flag.NewFlagSet("audit-home", flag.ContinueOnError)
	home := fs.String("home", "", "copied node home to audit")
	output := fs.String("output", "", "optional report output path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *home == "" {
		return fmt.Errorf("--home is required")
	}

	report, err := loadAuditFromHome(*home)
	if err != nil {
		return err
	}
	return emitReport(report, *output)
}

func runReplayV080(args []string) error {
	fs := flag.NewFlagSet("replay-v080", flag.ContinueOnError)
	sourceHome := fs.String("source-home", "", "copied pre-upgrade VPS node home")
	workingHome := fs.String("working-home", "", "repo-local temp home copy for replay")
	output := fs.String("output", "", "optional report output path")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *sourceHome == "" {
		return fmt.Errorf("--source-home is required")
	}

	work := *workingHome
	if work == "" {
		work = filepath.Join("tmp", "v080-replay")
	}
	if err := os.RemoveAll(work); err != nil {
		return fmt.Errorf("clear working home: %w", err)
	}

	report, err := replayV080(*sourceHome, work)
	if err != nil {
		return err
	}
	return emitReport(report, *output)
}

func emitReport(report any, output string) error {
	if output == "" {
		return writeJSON(os.Stdout, report)
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return err
	}
	f, err := os.Create(output)
	if err != nil {
		return err
	}
	defer f.Close()
	return writeJSON(f, report)
}
