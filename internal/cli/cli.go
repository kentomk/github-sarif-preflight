package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/kentomk/github-sarif-preflight/internal/preflight"
)

const topHelp = `github-sarif-preflight checks SARIF against GitHub Code Scanning's consumer profile and the current checkout.

Usage:
  github-sarif-preflight check [--root PATH] [--format text|json] SARIF_FILE...
  github-sarif-preflight version
  github-sarif-preflight help

The check command reports GSP001 through GSP005 diagnostics without uploading
SARIF or reading source-file contents.

Exit codes:
  0  no actionable diagnostic
  1  one or more consumer-profile diagnostics
  2  invalid arguments, unreadable input, or malformed SARIF
`

const checkHelp = `github-sarif-preflight check

Usage:
  github-sarif-preflight check [--root PATH] [--format text|json] SARIF_FILE...

Options:
  --root PATH          repository checkout root (default ".")
  --format text|json   output format (default "text")
  -h, --help           show this help

Exit codes:
  0  no actionable diagnostic
  1  one or more GSP001 through GSP005 diagnostics
  2  invalid arguments, unreadable input, or malformed SARIF
`

func Run(args []string, stdout, stderr io.Writer, version string) int {
	if len(args) == 1 && isHelp(args[0]) {
		fmt.Fprint(stdout, topHelp)
		return 0
	}
	if len(args) == 1 && args[0] == "version" {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if len(args) == 0 || args[0] != "check" {
		fmt.Fprintln(stderr, "usage: github-sarif-preflight check [--root PATH] [--format text|json] SARIF_FILE...")
		return 2
	}
	if len(args) == 2 && isHelp(args[1]) {
		fmt.Fprint(stdout, checkHelp)
		return 0
	}

	flags := flag.NewFlagSet("check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", ".", "repository root")
	format := flags.String("format", "text", "output format: text or json")
	if err := flags.Parse(args[1:]); err != nil {
		return 2
	}
	if *format != "text" && *format != "json" {
		fmt.Fprintln(stderr, "format must be text or json")
		return 2
	}

	report, err := preflight.Analyze(*root, version, flags.Args())
	if err != nil {
		fmt.Fprintf(stderr, "input error: %v\n", err)
		return 2
	}
	if *format == "json" {
		encoder := json.NewEncoder(stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			fmt.Fprintf(stderr, "output error: %v\n", err)
			return 2
		}
	} else {
		for _, diagnostic := range report.Diagnostics {
			position := fmt.Sprintf("run[%d].result[%d]", diagnostic.Run, diagnostic.Result)
			if diagnostic.Location != nil {
				position += fmt.Sprintf(".location[%d]", *diagnostic.Location)
			}
			fields := []string{diagnostic.Input + ":" + position, diagnostic.ID, diagnostic.Severity, diagnostic.Message}
			if diagnostic.Path != "" {
				fields = append(fields, "path="+diagnostic.Path)
			}
			fmt.Fprintln(stdout, strings.Join(fields, " "))
		}
		for _, unknown := range report.Unknowns {
			position := fmt.Sprintf("run[%d].result[%d].location[%d]", unknown.Run, unknown.Result, unknown.Location)
			fmt.Fprintf(stdout, "%s:%s unknown %s\n", unknown.Input, position, unknown.Reason)
		}
		fmt.Fprintf(stdout, "summary: inputs=%d runs=%d results=%d diagnostics=%d unknowns=%d\n", report.Summary.Inputs, report.Summary.Runs, report.Summary.Results, report.Summary.Diagnostics, report.Summary.Unknowns)
	}
	if len(report.Diagnostics) > 0 {
		return 1
	}
	return 0
}

func isHelp(arg string) bool {
	return arg == "help" || arg == "-h" || arg == "--help"
}
