package cli

import (
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/babywbx/tgup/internal/app"
	"github.com/babywbx/tgup/internal/config"
)

func runDemo(args []string, stdout io.Writer, stderr io.Writer) int {
	fs := flag.NewFlagSet("demo", flag.ContinueOnError)
	fs.SetOutput(stderr)

	var configPath string
	fs.StringVar(&configPath, "config", "", "path to config file")

	apiID := &intValue{}
	apiHash := &stringValue{}
	sessionPath := &stringValue{}
	sessionPathAlias := &stringValue{}

	fs.Var(apiID, "api-id", "telegram api id")
	fs.Var(apiHash, "api-hash", "telegram api hash")
	fs.Var(sessionPath, "session", "session file path")
	fs.Var(sessionPathAlias, "session-path", "session file path (alias)")

	if err := fs.Parse(args); err != nil {
		return 2
	}
	if len(fs.Args()) > 0 {
		fmt.Fprintf(stderr, "unexpected demo args: %s\n", strings.Join(fs.Args(), " "))
		return 2
	}

	if sessionPath.set && sessionPathAlias.set {
		fmt.Fprintln(stderr, "conflicting flags: --session and --session-path")
		return 2
	}
	if sessionPathAlias.set && !sessionPath.set {
		sessionPath = sessionPathAlias
	}

	cli := config.Overlay{}
	if apiID.set {
		cli.Telegram.APIID = apiID.ptr()
	}
	if apiHash.set {
		cli.Telegram.APIHash = apiHash.ptr()
	}
	if sessionPath.set {
		cli.Telegram.SessionPath = sessionPath.ptr()
	}

	if err := app.RunDemo(configPath, cli, stdout); err != nil {
		fmt.Fprintf(stderr, "demo failed: %v\n", err)
		return 1
	}
	return 0
}
