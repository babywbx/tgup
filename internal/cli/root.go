package cli

import (
	"fmt"
	"io"
	"strings"
)

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
	case "mcp":
		return runMCP(args[1:], stdout, stderr)
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
		"  mcp serve",
		"  mcp schema",
	}
	_, _ = fmt.Fprintln(w, strings.Join(lines, "\n"))
}
