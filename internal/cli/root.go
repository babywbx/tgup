package cli

import (
	"fmt"
	"io"
	"strings"
)

var (
	buildVersion = "dev"
	buildCommit  = "unknown"
	buildDate    = "unknown"
)

// SetVersion sets build info injected via ldflags.
func SetVersion(version, commit, date string) {
	buildVersion = version
	buildCommit = commit
	buildDate = date
}

// Run executes tgup CLI command handling and returns process exit code.
func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}

	switch args[0] {
	case "dry-run":
		return runDryRun(args[1:], stdout, stderr)
	case "login":
		return runLogin(args[1:], stdout, stderr)
	case "run":
		return runRun(args[1:], stdout, stderr)
	case "demo":
		return runDemo(args[1:], stdout, stderr)
	case "mcp":
		return runMCP(args[1:], stdout, stderr)
	case "version", "-v", "--version":
		fmt.Fprintf(stdout, "tgup %s (commit: %s, built: %s)\n", buildVersion, buildCommit, buildDate)
		return 0
	case "-h", "--help", "help":
		printUsage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command: %s\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	lines := []string{
		"usage: tgup <command> [flags]",
		"",
		"commands:",
		"  login",
		"  dry-run",
		"  run",
		"  demo",
		"  mcp serve",
		"  mcp schema",
		"  version",
	}
	_, _ = fmt.Fprintln(w, strings.Join(lines, "\n"))
}
